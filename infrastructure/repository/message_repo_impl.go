package repository

import (
	"job-connect/delivery/ws"
	"job-connect/domain"
	"time"

	"gorm.io/gorm"
)

type MessageRepository struct {
	db  *gorm.DB
	hub *ws.Hub
}

func NewMessageRepository(db *gorm.DB, hub *ws.Hub) *MessageRepository {
	return &MessageRepository{db: db, hub: hub}
}

func (r *MessageRepository) CreateMessage(input *domain.CreateMessageInput) (domain.Message, error) {

	// 1️⃣ find or create conversation
	var conv domain.Conversation

	err := r.db.
		Where(
			"(user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)",
			input.SenderID, input.ReceiverID,
			input.ReceiverID, input.SenderID,
		).
		First(&conv).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			conv = domain.Conversation{
				User1ID: input.SenderID,
				User2ID: input.ReceiverID,
			}
			if err := r.db.Create(&conv).Error; err != nil {
				return domain.Message{}, err
			}
		} else {
			return domain.Message{}, err
		}
	}

	// 2️⃣ create message
	message := domain.Message{
		SenderID:       input.SenderID,
		ReceiverID:     input.ReceiverID,
		ConversationID: conv.ID,
		Type:           input.Type,
		Text:           input.Text,
		ImageUrl:       input.ImageUrl,
		VideoUrl:       input.VideoUrl,
		Caption:        input.Caption,
		IsSeen:         false,
	}

	if err := r.db.Create(&message).Error; err != nil {
		return domain.Message{}, err
	}

	// 3️⃣ update conversation
	if err := r.db.
		Model(&domain.Conversation{}).
		Where("id = ?", conv.ID).
		Updates(map[string]interface{}{
			"last_message_id": message.ID,
			"updated_at":      time.Now(),
		}).Error; err != nil {
		return domain.Message{}, err
	}

	// 4️⃣ REAL-TIME PUSH (🔥 NEW)
	r.hub.SendToUser(input.ReceiverID, domain.WSMessageEvent{
		Type: "new_message",
		Data: message,
	})

	return message, nil
}

func (r *MessageRepository) GetMessagesByConversationID(conversationID uint) ([]domain.Message, error) {
	var messages []domain.Message

	err := r.db.
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Find(&messages).Error

	return messages, err
}

func (r *MessageRepository) GetConversationsByUserID(userID uint) ([]domain.ConversationResponse, error) {

	var conversations []domain.Conversation

	err := r.db.
		Where("user1_id = ? OR user2_id = ?", userID, userID).
		Order("updated_at DESC").
		Find(&conversations).Error

	if err != nil {
		return nil, err
	}

	// collect all other user IDs
	userIDMap := make(map[uint]bool)

	for _, c := range conversations {
		if c.User1ID == userID {
			userIDMap[c.User2ID] = true
		} else {
			userIDMap[c.User1ID] = true
		}
	}

	var userIDs []uint
	for id := range userIDMap {
		userIDs = append(userIDs, id)
	}

	// load all users in ONE query
	var users []domain.User
	r.db.Where("id IN ?", userIDs).Find(&users)

	// map users by ID
	userMap := make(map[uint]domain.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	var result []domain.ConversationResponse

	for _, c := range conversations {

		var otherUserID uint
		if c.User1ID == userID {
			otherUserID = c.User2ID
		} else {
			otherUserID = c.User1ID
		}

		var lastMsg domain.Message
		if c.LastMessageID != nil {
			r.db.Where("id = ?", *c.LastMessageID).First(&lastMsg)
		}

		var unseen int64
		r.db.Model(&domain.Message{}).
			Where("conversation_id = ? AND receiver_id = ? AND is_seen = false",
				c.ID, userID).
			Count(&unseen)

		result = append(result, domain.ConversationResponse{
			OtherUserID: otherUserID,
			LastMessage: lastMsg,
			UnseenCount: unseen,
			User:        userMap[otherUserID],
		})
	}

	return result, nil
}

func (r *MessageRepository) MarkMessageAsSeen(conversationID uint, userID uint) error {
	now := time.Now()

	err := r.db.
		Model(&domain.Message{}).
		Where("conversation_id = ? AND receiver_id = ? AND is_seen = false",
			conversationID, userID).
		Updates(map[string]interface{}{
			"is_seen": true,
			"seen_at": now,
		}).Error

	if err != nil {
		return err
	}

	//notify sender that messages were seen
	r.hub.SendToUser(userID, map[string]interface{}{
		"type": "messages_seen",
		"data": map[string]interface{}{
			"conversation_id": conversationID,
		},
	})

	return nil
}

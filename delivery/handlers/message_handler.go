package handlers

import (
	"encoding/json"
	"fmt"
	"job-connect/auth"
	"job-connect/domain"
	"job-connect/usecase"
	"net/http"
)

type MessageHandler struct {
	messageUsecase *usecase.MessageUsecase
}

type CreateMessageRequest struct {
	SenderID   uint    `json:"sender_id" example:"1"`
	ReceiverID uint    `json:"receiver_id" example:"2"`
	Type       string  `json:"type" example:"text"`
	Text       *string `json:"text,omitempty" example:"Hello"`
	ImageUrl   *string `json:"image_url,omitempty" example:"https://example.com/image.jpg"`
	VideoUrl   *string `json:"video_url,omitempty" example:"https://example.com/video.mp4"`
	Caption    *string `json:"caption,omitempty" example:"sample caption"`
}

type CreateMessageResponse struct {
	Message domain.Message `json:"message"`
}

type GetMessagesByConversationResponse struct {
	Messages []domain.Message `json:"messages"`
}

type GetConversationsResponse struct {
	Conversations []domain.ConversationResponse `json:"conversations"`
}

type MarkMessageAsSeenResponse struct {
	Message string `json:"message" example:"Messages marked as seen"`
}

func NewMessageHandler(messageUsecase *usecase.MessageUsecase) *MessageHandler {
	return &MessageHandler{messageUsecase: messageUsecase}
}

// CreateMessage godoc
// @Summary Create a new message
// @Description Create a new message in a conversation
// @Tags Messages
// @Accept json
// @Produce json
// @Param message body handlers.CreateMessageRequest true "Message input"
// @Success 200 {object} handlers.CreateMessageResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /messages [post]
func (h *MessageHandler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	var req CreateMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	input := domain.CreateMessageInput{
		SenderID:   req.SenderID,
		ReceiverID: req.ReceiverID,
		Type:       req.Type,
		Text:       req.Text,
		ImageUrl:   req.ImageUrl,
		VideoUrl:   req.VideoUrl,
		Caption:    req.Caption,
	}

	message, err := h.messageUsecase.CreateMessage(&input)
	if err != nil {
		http.Error(w, "Failed to create message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": message,
	})

}

// GetMessagesByConversationID godoc
// @Summary Get messages by conversation ID
// @Description Get all messages in a specific conversation
// @Tags Messages
// @Accept json
// @Produce json
// @Param conversation_id query int true "Conversation ID"
// @Success 200 {object} handlers.GetMessagesByConversationResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /messages [get]
func (h *MessageHandler) GetMessagesByConversationID(w http.ResponseWriter, r *http.Request) {
	convIDStr := r.URL.Query().Get("conversation_id")
	if convIDStr == "" {
		http.Error(w, "conversation_id is required", http.StatusBadRequest)
		return
	}

	var convID uint
	if _, err := fmt.Sscanf(convIDStr, "%d", &convID); err != nil {
		http.Error(w, "invalid conversation_id", http.StatusBadRequest)
		return
	}

	messages, err := h.messageUsecase.GetMessagesByConversationID(convID)
	if err != nil {
		http.Error(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": messages,
	})
}

// GetConversationsByUserID godoc
// @Summary Get conversations by user ID
// @Description Get all conversations for a specific user
// @Tags Messages
// @Accept json
// @Produce json
// @Success 200 {object} handlers.GetConversationsResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /messages/conversations [get]
func (h *MessageHandler) GetConversationsByUserID(w http.ResponseWriter, r *http.Request) {
	userIDStr, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID uint
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	conversations, err := h.messageUsecase.GetConversationsByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to get conversations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"conversations": conversations,
	})
}

// MarkMessageAsSeen godoc
// @Summary Mark messages as seen
// @Description Mark all messages in a conversation as seen for the current user
// @Tags Messages
// @Accept json
// @Produce json
// @Param conversation_id query int true "Conversation ID"
// @Success 200 {object} handlers.MarkMessageAsSeenResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /messages/seen [post]
func (h *MessageHandler) MarkMessageAsSeen(w http.ResponseWriter, r *http.Request) {
	convIDStr := r.URL.Query().Get("conversation_id")
	userIdStr, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if convIDStr == "" {
		http.Error(w, "conversation_id is required", http.StatusBadRequest)
		return
	}

	var convID uint
	if _, err := fmt.Sscanf(convIDStr, "%d", &convID); err != nil {
		http.Error(w, "invalid conversation_id", http.StatusBadRequest)
		return
	}

	var userID uint
	if _, err := fmt.Sscanf(userIdStr, "%d", &userID); err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	err = h.messageUsecase.MarkMessageAsSeen(convID, userID)
	if err != nil {
		http.Error(w, "Failed to mark messages as seen", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Messages marked as seen",
	})
}

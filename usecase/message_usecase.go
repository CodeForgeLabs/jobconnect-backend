package usecase

import "job-connect/domain"

type MessageUsecase struct {
	messageRepo domain.MessageRepository
}

func NewMessageUsecase(messageRepo domain.MessageRepository) *MessageUsecase {
	return &MessageUsecase{messageRepo: messageRepo}
}

func (m *MessageUsecase) CreateMessage(message *domain.CreateMessageInput) (domain.Message, error) {
	return m.messageRepo.CreateMessage(message)
}

func (m *MessageUsecase) GetMessagesByConversationID(conversationID uint) ([]domain.Message, error) {
	return m.messageRepo.GetMessagesByConversationID(conversationID)
}

func (m *MessageUsecase) GetConversationsByUserID(userID uint) ([]domain.ConversationResponse, error) {
	return m.messageRepo.GetConversationsByUserID(userID)
}

func (m *MessageUsecase) MarkMessageAsSeen(conversationID uint, userID uint) error {
	return m.messageRepo.MarkMessageAsSeen(conversationID, userID)
}

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

func NewMessageHandler(messageUsecase *usecase.MessageUsecase) *MessageHandler {
	return &MessageHandler{messageUsecase: messageUsecase}
}

// CreateMessage godoc
// @Summary Create a new message
// @Description Create a new message in a conversation
// @Tags Messages
// @Accept json
// @Produce json
// @Param message body domain.CreateMessageInput true "Message input"
// @Success 200 {object} domain.GenericResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /messages [post]
func (h *MessageHandler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateMessageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
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
// @Success 200 {object} domain.GenericResponse
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
// @Success 200 {object} domain.GenericResponse
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
// @Success 200 {object} domain.GenericResponse
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

package handlers

import (
	"encoding/json"
	"job-connect/auth"
	domain "job-connect/domain"
	"job-connect/usecase"
	"net/http"
)

type NotificationHandler struct {
	notificationUsecase *usecase.NotificationUsecase
}

func NewNotificationHandler(notificationUsecase *usecase.NotificationUsecase) *NotificationHandler {
	return &NotificationHandler{
		notificationUsecase: notificationUsecase,
	}
}

// CreateNotification godoc
// @Summary Create a new notification for a user
// @Description Create a new notification for a user with the specified details
// @Tags Notification
// @Accept json
// @Produce json
// @Param notification body domain.Notification true "Notification details"
// @Success 201 {object} domain.GenericResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /notifications [post]
func (h *NotificationHandler) CreateNotification(w http.ResponseWriter, r *http.Request) {
	var notification domain.Notification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := h.notificationUsecase.CreateNotification(&notification); err != nil {
		http.Error(w, "Failed to create notification", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(domain.GenericResponse{Message: "Notification created successfully"})
}

// GetNotificationsByUserID godoc
// @Summary Get notifications for a user
// @Description Retrieve all notifications for a user by their ID
// @Tags Notification
// @Accept json
// @Produce json
// @Param userId path int true "User ID"
// @Success 200 {object} domain.GenericResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /notifications/user/{userId} [get]
func (h *NotificationHandler) GetNotificationsByUserID(w http.ResponseWriter, r *http.Request) {
	userId, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	notifications, err := h.notificationUsecase.GetNotificationsByUserID(parseUint(userId))
	if err != nil {
		http.Error(w, "Failed to retrieve notifications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domain.GenericResponse{Data: notifications})
}

// MarkNotificationAsRead godoc
// @Summary Mark a notification as read
// @Description Mark a specific notification as read by its ID
// @Tags Notification
// @Accept json
// @Produce json
// @Param notificationId path int true "Notification ID"
// @Success 200 {object} domain.GenericResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /notifications/read [post]
func (h *NotificationHandler) MarkNotificationAsRead(w http.ResponseWriter, r *http.Request) {
	userIDStr, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.notificationUsecase.MarkNotificationAsRead(parseUint(userIDStr)); err != nil {
		http.Error(w, "Failed to mark notification as read", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(domain.GenericResponse{Message: "Notification marked as read"})
}

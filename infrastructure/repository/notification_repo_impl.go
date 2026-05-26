package repository

import (
	"job-connect/delivery/ws"
	"job-connect/domain"

	"gorm.io/gorm"
)

type notificationRepo struct {
	db  *gorm.DB
	hub *ws.Hub
}

func NewNotificationRepo(db *gorm.DB, hub *ws.Hub) *notificationRepo {
	return &notificationRepo{db: db, hub: hub}
}

func (r *notificationRepo) CreateNotification(notification *domain.Notification) error {

	r.hub.SendToUser(notification.UserID, domain.WSMessageEvent{
		Type: "new_notification",
		Data: notification,
	})
	return r.db.Create(notification).Error
}

func (r *notificationRepo) GetNotificationsByUserID(userID uint) ([]domain.Notification, error) {
	var notifications []domain.Notification
	err := r.db.Where("user_id = ?", userID).Order("created_at desc").Find(&notifications).Error
	return notifications, err
}

func (r *notificationRepo) MarkNotificationAsRead(userId uint) error {
	return r.db.Model(&domain.Notification{}).Where("user_id = ?", userId).Update("is_read", true).Error
}

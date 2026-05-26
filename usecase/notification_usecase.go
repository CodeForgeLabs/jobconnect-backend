package usecase

import "job-connect/domain"

type NotificationUsecase struct {
	notificationRepo domain.NotificationRepository
}

func NewNotificationUsecase(notificationRepo domain.NotificationRepository) *NotificationUsecase {
	return &NotificationUsecase{notificationRepo: notificationRepo}
}

func (n *NotificationUsecase) CreateNotification(notification *domain.Notification) error {
	return n.notificationRepo.CreateNotification(notification)
}

func (n *NotificationUsecase) GetNotificationsByUserID(userID uint) ([]domain.Notification, error) {
	return n.notificationRepo.GetNotificationsByUserID(userID)
}

func (n *NotificationUsecase) MarkNotificationAsRead(userId uint) error {
	return n.notificationRepo.MarkNotificationAsRead(userId)
}

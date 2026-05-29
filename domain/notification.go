package domain

import (
	"time"
)

type NotificationType string

const (
	NotifyConnectsPurchased    NotificationType = "CONNECTS_PURCHASED"
	NotifyConnectRefunded      NotificationType = "CONNECTS_REFUNDED"
	NotifyProposalStatus       NotificationType = "PROPOSAL_STATUS_CHANGED"
	NotifyMilestoneStatus      NotificationType = "MILESTONE_STATUS_CHANGED"
	NotifyContractCreated      NotificationType = "CONTRACT_CREATED"
	NotifyContractStatus       NotificationType = "CONTRACT_STATUS_CHANGED"
	NotifyWeeklyPaymentRelased NotificationType = "WEEKLY_PAYMENT_RELASED"
	NotifyJobDeleted           NotificationType = "JOB_DELETED"
)

type Notification struct {
	ID      uint             `gorm:"primaryKey" json:"id"`
	UserID  uint             `gorm:"not null;index" json:"user_id"` // Who receives the notification
	Type    NotificationType `gorm:"type:varchar(50);not null" json:"type"`
	Title   string           `gorm:"type:varchar(255);not null" json:"title"`
	Message string           `gorm:"type:text;not null" json:"message"`
	IsRead  bool             `gorm:"default:false;index" json:"is_read"`

	// Context fields (Optional, but highly recommended for frontend routing)
	ContractID *uint `json:"contract_id,omitempty"`
	ProposalID *uint `json:"proposal_id,omitempty"`
	JobID      *uint `json:"job_id,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type NotificationRepository interface {
	CreateNotification(notification *Notification) error
	GetNotificationsByUserID(userID uint) ([]Notification, error)
	MarkNotificationAsRead(userId uint) error
}

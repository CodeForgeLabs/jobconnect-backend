package domain

import "time"

type Message struct {
	ID uint `gorm:"primaryKey"`

	SenderID   uint `gorm:"index;not null"`
	ReceiverID uint `gorm:"index;not null"`

	ConversationID uint `gorm:"index;not null"`

	Type string `gorm:"type:varchar(20)"`

	Text     *string
	ImageUrl *string
	VideoUrl *string
	Caption  *string

	IsSeen bool `gorm:"default:false"`
	SeenAt *time.Time

	IsEdited bool `gorm:"default:false"`
	EditedAt *time.Time

	IsDeleted bool `gorm:"default:false"`
	DeletedAt *time.Time

	CreatedAt time.Time
}

type Conversation struct {
	ID uint `gorm:"primaryKey"`

	User1ID uint `gorm:"index;not null"`
	User2ID uint `gorm:"index;not null"`

	LastMessageID *uint

	UpdatedAt time.Time
}
type ConversationResponse struct {
	OtherUserID uint
	LastMessage Message
	UnseenCount int64
	User        User
}

type MessageRepository interface {
	CreateMessage(message *CreateMessageInput) (Message, error)
	GetMessagesByConversationID(conversationID uint) ([]Message, error)
	GetConversationsByUserID(userID uint) ([]ConversationResponse, error)
	MarkMessageAsSeen(conversationID uint, userID uint) error
}

type CreateMessageInput struct {
	SenderID   uint
	ReceiverID uint
	Type       string
	Text       *string
	ImageUrl   *string
	VideoUrl   *string
	Caption    *string
}

type MessageResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type GenericResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type WSMessageEvent struct {
	Type string `json:"type"` // "new_message"
	Data any    `json:"data"`
}

type WSConversationUpdate struct {
	Type string               `json:"type"` // "conversation_update"
	Data ConversationResponse `json:"data"`
}

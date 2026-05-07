package domain

import "time"

type Review struct {
	ID uint `gorm:"primaryKey" json:"id"`

	ContractID   uint `gorm:"not null;uniqueIndex" json:"contract_id"`
	ClientID     uint `gorm:"not null;index" json:"client_id"`
	FreelancerID uint `gorm:"not null;index" json:"freelancer_id"`

	Rating int    `gorm:"not null" json:"rating"`
	Note   string `gorm:"type:text;not null" json:"note"`

	FreelancerReply *string `gorm:"type:text" json:"freelancer_reply,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type ReviewListResult struct {
	AverageRating float64
	ReviewCount   int64
	Reviews       []*Review
}

type ReviewRepository interface {
	CreateReview(contractID, clientID uint, rating int, note string) (*Review, error)
	GetReviewByID(id uint) (*Review, error)
	UpdateReview(reviewID, clientID uint, rating *int, note *string) (*Review, error)
	UpdateFreelancerReply(reviewID, freelancerID uint, reply *string) (*Review, error)
	ListReviewsByFreelancerID(freelancerID uint) (*ReviewListResult, error)
}

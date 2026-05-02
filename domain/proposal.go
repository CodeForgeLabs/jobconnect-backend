package domain

import "time"

type ProposalStatus string

const (
	ProposalPending  ProposalStatus = "PENDING"
	ProposalInvited  ProposalStatus = "INVITED"
	ProposalRejected ProposalStatus = "REJECTED"
)

type Proposal struct {
	ID uint `gorm:"primaryKey" json:"id"`

	JobID      uint `gorm:"not null;index" json:"job_id"`
	JobOwnerID uint `gorm:"not null;index" json:"job_owner_id"`
	SenderID   uint `gorm:"not null;index" json:"sender_id"`

	Description string `gorm:"type:text;not null" json:"description"`

	Status ProposalStatus `gorm:"type:varchar(20);default:'PENDING'" json:"status"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
type ProposalWithUserResponse struct {
	ProposalID  uint           `json:"proposal_id"`
	JobID       uint           `json:"job_id"`
	SenderID    uint           `json:"sender_id"`
	Description string         `json:"description"`
	Status      ProposalStatus `json:"status"`

	UserID       uint   `json:"user_id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Email        string `json:"email"`
	Headline     string `json:"headline"`
	Skills       string `json:"skills"`
	ProfileImage string `json:"profile_picture_url"`
}

type ProposalRepository interface {
	CreateProposal(proposal *Proposal) error
	GetProposalByID(id uint) (*Proposal, error)
	UpdateProposal(proposal *Proposal) error
	DeleteProposal(id uint) error
	ListProposalsByJobID(jobID uint) ([]*ProposalWithUserResponse, error)
}

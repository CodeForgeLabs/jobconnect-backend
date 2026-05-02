package domain

import "time"

type ContractType string
type ContractStatus string

const (
	ContractFixed  ContractType = "FIXED"
	ContractHourly ContractType = "HOURLY"

	ContractActive    ContractStatus = "ACTIVE"
	ContractCompleted ContractStatus = "COMPLETED"
	ContractCancelled ContractStatus = "CANCELLED"
	ContractPaused    ContractStatus = "PAUSED"
)

type Contract struct {
	ID uint `gorm:"primaryKey" json:"id"`

	JobID      uint `gorm:"not null;index" json:"job_id"`
	ProposalID uint `gorm:"not null;index" json:"proposal_id"`

	ClientID     uint `gorm:"not null;index" json:"client_id"`
	FreelancerID uint `gorm:"not null;index" json:"freelancer_id"`

	Type ContractType `gorm:"type:varchar(20);not null" json:"type"`

	Title       string `gorm:"type:varchar(255)" json:"title"`
	Description string `gorm:"type:text" json:"description"`

	// hourly
	HourlyRate      *float64 `gorm:"type:decimal(10,2)" json:"hourly_rate,omitempty"`
	WeeklyHourLimit *int     `json:"weekly_hour_limit,omitempty"`

	// fixed
	TotalBudget *float64 `gorm:"type:decimal(10,2)" json:"total_budget,omitempty"`

	Status ContractStatus `gorm:"type:varchar(20);default:'ACTIVE'" json:"status"`

	StartDate time.Time  `json:"start_date"`
	EndDate   *time.Time `json:"end_date,omitempty"`

	CreatedAt  time.Time           `gorm:"autoCreateTime"`
	UpdatedAt  time.Time           `gorm:"autoUpdateTime"`
	Milestones []ContractMilestone `json:"milestones" gorm:"-"`
}
type ContractWithDetailsResponse struct {
	ContractID uint `json:"contract_id"`

	JobID uint `json:"job_id"`

	ClientID     uint `json:"client_id"`
	FreelancerID uint `json:"freelancer_id"`

	Type ContractType `json:"type"`

	Title       string `json:"title"`
	Description string `json:"description"`

	HourlyRate      *float64 `json:"hourly_rate,omitempty"`
	WeeklyHourLimit *int     `json:"weekly_hour_limit,omitempty"`
	TotalBudget     *float64 `json:"total_budget,omitempty"`

	Status ContractStatus `json:"status"`

	StartDate time.Time  `json:"start_date"`
	EndDate   *time.Time `json:"end_date,omitempty"`

	ClientFirstName    string `json:"client_first_name"`
	ClientLastName     string `json:"client_last_name"`
	ClientEmail        string `json:"client_email"`
	ClientHeadline     string `json:"client_headline"`
	ClientSkills       string `json:"client_skills"`
	ClientProfileImage string `json:"client_profile_picture_url"`

	FreelancerFirstName    string `json:"freelancer_first_name"`
	FreelancerLastName     string `json:"freelancer_last_name"`
	FreelancerEmail        string `json:"freelancer_email"`
	FreelancerHeadline     string `json:"freelancer_headline"`
	FreelancerSkills       string `json:"freelancer_skills"`
	FreelancerProfileImage string `json:"freelancer_profile_picture_url"`

	ProposalDescription string `json:"proposal_description"`

	JobTitle string `json:"job_title"`
}

type ContractMilestoneStatus string

const (
	MilestonePending           ContractMilestoneStatus = "PENDING"
	MilestoneInProgress        ContractMilestoneStatus = "IN_PROGRESS"
	MilestoneSubmitted         ContractMilestoneStatus = "SUBMITTED"
	MilestoneRevisionRequested ContractMilestoneStatus = "REVISION_REQUESTED"
	MilestoneApproved          ContractMilestoneStatus = "APPROVED"
	MilestonePaid              ContractMilestoneStatus = "PAID"
)

type ContractMilestone struct {
	ID uint `gorm:"primaryKey"`

	ContractID uint `gorm:"not null;index"`

	Description string `gorm:"type:text"`

	Amount float64 `gorm:"type:decimal(10,2)"`

	Status ContractMilestoneStatus `gorm:"type:varchar(30);default:'PENDING'"`

	SubmissionURL string `json:"submission_url,omitempty"` // optional file link

	ClientFeedback  string     `gorm:"type:text"`
	WorkDescription string     `gorm:"type:text"`
	DueDate         *time.Time `json:"due_date,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type TimeLog struct {
	ID uint `gorm:"primaryKey"`

	ContractID   uint `gorm:"not null;index"`
	FreelancerID uint `gorm:"not null;index"`

	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`

	TotalHours float64 `gorm:"type:decimal(10,2)"`

	IsPaid bool `gorm:"default:false"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}
type MyContractResponse struct {
	ContractID uint `json:"contract_id"`

	JobID uint `json:"job_id"`

	ClientID     uint `json:"client_id"`
	FreelancerID uint `json:"freelancer_id"`

	Type ContractType `json:"type"`

	Title       string `json:"title"`
	Description string `json:"description"`

	HourlyRate      *float64 `json:"hourly_rate,omitempty"`
	WeeklyHourLimit *int     `json:"weekly_hour_limit,omitempty"`
	TotalBudget     *float64 `json:"total_budget,omitempty"`

	Status ContractStatus `json:"status"`

	StartDate time.Time  `json:"start_date"`
	EndDate   *time.Time `json:"end_date,omitempty"`

	// ✅ FIXED names (must match SQL aliases)
	ClientFirstName         string `json:"client_first_name"`
	ClientLastName          string `json:"client_last_name"`
	ClientEmail             string `json:"client_email"`
	ClientHeadline          string `json:"client_headline"`
	ClientSkills            string `json:"client_skills"`
	ClientProfilePictureURL string `json:"client_profile_picture_url"`

	FreelancerFirstName         string `json:"freelancer_first_name"`
	FreelancerLastName          string `json:"freelancer_last_name"`
	FreelancerEmail             string `json:"freelancer_email"`
	FreelancerHeadline          string `json:"freelancer_headline"`
	FreelancerSkills            string `json:"freelancer_skills"`
	FreelancerProfilePictureURL string `json:"freelancer_profile_picture_url"`

	ProposalDescription string `json:"proposal_description"`

	JobTitle   string              `json:"job_title"`
	Milestones []ContractMilestone `json:"milestones" gorm:"-"`
}
type SubmitMilestoneRequest struct {
	ContractID          uint   `json:"contract_id"`
	MilestoneID         uint   `json:"milestone_id"`
	Description         string `json:"description"`
	MilestoneProjectURL string `json:"milestone_project_url"`
}
type ContractRepository interface {
	CreateContract(jobId, freelancerId string) error
	GetMyContracts(userID uint) ([]*MyContractResponse, error)
	GetContractByID(id uint) (*MyContractResponse, error)
	SubmitMilestone(request *SubmitMilestoneRequest) error
	ModifyStatus(milestoneId uint, newStatus ContractMilestoneStatus) error
	ModifyContractStatus(contractId uint, newStatus ContractStatus) error

	// log time for hourly contracts (not implemented yet)
	StartWorkSession(contractId, freelancerId uint) error
	EndWorkSession(contractId, freelancerId uint) error
	FetchTimeLogs(contractId, freelancerId uint) ([]*TimeLog, error)
	FetchTimeElapsed(contractId, freelancerId uint) (float64, error)
	FetchWeeklyHours(contractId, freelancerId uint) (float64, error)
}

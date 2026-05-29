package domain

import "time"

type JobType string
type ExperienceLevel string
type WorkMode string
type JobStatus string

const (
	JobTypeHourly JobType = "HOURLY"
	JobTypeFixed  JobType = "FIXED"

	WorkModeRemote WorkMode = "REMOTE"
	WorkModeOnsite WorkMode = "ONSITE"
	WorkModeHybrid WorkMode = "HYBRID"

	ExpEntry        ExperienceLevel = "ENTRY"
	ExpIntermediate ExperienceLevel = "INTERMEDIATE"
	ExpSenior       ExperienceLevel = "SENIOR"

	StatusOpen   JobStatus = "OPEN"
	StatusClosed JobStatus = "CLOSED"
	StatusDraft  JobStatus = "DRAFT"
)

type Job struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// relationships
	CreatedBy uint `gorm:"not null;index" json:"created_by"`

	// core info
	Title       string `gorm:"type:varchar(255);not null" json:"title"`
	Description string `gorm:"type:text;not null" json:"description"`

	Category string `gorm:"type:varchar(100)" json:"category"`

	// classification
	JobType         JobType         `gorm:"type:varchar(20);not null" json:"job_type"`
	ExperienceLevel ExperienceLevel `gorm:"type:varchar(20)" json:"experience_level"`
	WorkMode        WorkMode        `gorm:"type:varchar(20)" json:"work_mode"`

	// hourly job fields
	HourlyRate     *float64 `gorm:"type:decimal(10,2)" json:"hourly_rate,omitempty"`
	MaxWeeklyHours *int     `json:"max_weekly_hours,omitempty"`

	// fixed job fields
	Budget *float64 `gorm:"type:decimal(10,2)" json:"budget,omitempty"`

	// location (only for onsite/hybrid)
	Location string `gorm:"type:varchar(255)" json:"location,omitempty"`

	// company / private
	CompanyName string `gorm:"type:varchar(255)" json:"company_name,omitempty"`
	IsPrivate   bool   `gorm:"default:false" json:"is_private"`

	// skills (simple version)
	Skills string `gorm:"type:text" json:"skills"` // comma-separated OR JSON later

	// status + tracking
	Status            JobStatus `gorm:"type:varchar(20);default:'OPEN'" json:"status"`
	ApplicationsCount int       `gorm:"default:0" json:"applications_count"`

	Deadline   *time.Time  `json:"deadline,omitempty"`
	Milestones []Milestone `gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE;" json:"milestones,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type Milestone struct {
	ID uint `gorm:"primaryKey" json:"id"`

	JobID uint `gorm:"index;not null" json:"job_id"`

	Description string     `gorm:"type:text" json:"description"`
	Amount      float64    `gorm:"type:decimal(10,2)" json:"amount"`
	IsPaid      bool       `gorm:"default:false" json:"is_paid"`
	Deadline    *time.Time `json:"deadline,omitempty"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

type JobRepository interface {
	CreateJob(job *Job) error
	GetJobByID(id uint) (*Job, error)
	UpdateJob(job *Job) error
	DeleteJob(id uint) error
	ListJobs(filter JobFilter) ([]*Job, error)
	ListMyJobs(userID uint) ([]*Job, error)
}

type JobFilter struct {
	Title           string
	Company         string
	Location        string
	Category        string
	JobType         JobType
	ExperienceLevel ExperienceLevel
	WorkMode        WorkMode
	Status          JobStatus
	Skills          []string // filter by skills (comma-separated or JSON)
	BudgetMin       *float64
	HourlyRateMin   *float64
}

func ParseJobType(s string) JobType {
	switch s {
	case "HOURLY":
		return JobTypeHourly
	case "FIXED":
		return JobTypeFixed
	default:
		return ""
	}
}

func ParseWorkMode(s string) WorkMode {
	switch s {
	case "REMOTE":
		return WorkModeRemote
	case "ONSITE":
		return WorkModeOnsite
	case "HYBRID":
		return WorkModeHybrid
	default:
		return ""
	}
}

func ParseExperienceLevel(s string) ExperienceLevel {
	switch s {
	case "ENTRY":
		return ExpEntry
	case "INTERMEDIATE":
		return ExpIntermediate
	case "SENIOR":
		return ExpSenior
	default:
		return ""
	}
}

func ParseJobStatus(s string) JobStatus {
	switch s {
	case "OPEN":
		return StatusOpen
	case "CLOSED":
		return StatusClosed
	case "DRAFT":
		return StatusDraft
	default:
		return ""
	}
}

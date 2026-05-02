package domain

import "time"

type Role string
type Availability string

const (
	RoleClient     Role = "CLIENT"
	RoleFreelancer Role = "FREELANCER"
	RoleAdmin      Role = "ADMIN"

	AvailabilityFullTime     Availability = "FULLTIME"
	AvailabilityPartTime     Availability = "PARTTIME"
	AvailabilityContract     Availability = "CONTRACT"
	AvailabilityNotAvailable Availability = "NOT_AVAILABLE"
)

type User struct {
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// Registration (required)
	Role      Role   `gorm:"column:role;type:varchar(50);not null" json:"role"`
	FirstName string `gorm:"column:first_name;type:varchar(100);not null" json:"first_name"`
	LastName  string `gorm:"column:last_name;type:varchar(100);not null" json:"last_name"`
	Email     string `gorm:"column:email;type:varchar(255);unique;not null" json:"email"`
	Password  string `gorm:"column:password;type:varchar(255);not null" json:"-"`

	// Optional onboarding/profile
	Headline          string       `gorm:"column:headline;type:varchar(255)" json:"headline"`
	Bio               string       `gorm:"column:bio;type:text" json:"bio"`
	Skills            string       `gorm:"column:skills;type:text" json:"skills"` // store as comma-separated or JSON
	HourlyRate        float64      `gorm:"column:hourly_rate;type:decimal(10,2)" json:"hourly_rate"`
	Availability      Availability `gorm:"column:availability;type:varchar(50)" json:"availability"`
	Location          string       `gorm:"column:location;type:varchar(255)" json:"location"`
	PhoneNumber       string       `gorm:"column:phone_number;type:varchar(50)" json:"phone_number"`
	ProfilePictureURL string       `gorm:"column:profile_picture_url;type:text" json:"profile_picture_url"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

type UserRepository interface {
	Login(email, password string) (*User, error)
	CreateUser(user *User) error
	GetUserByID(id uint) (*User, error)
	GetUserByEmail(email string) (*User, error)
	UpdateUser(user *User) error
	DeleteUser(id uint) error
}

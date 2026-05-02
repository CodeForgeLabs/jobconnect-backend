package domain

import "time"

type PortfolioItem struct {
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	UserID uint `gorm:"column:user_id;not null;index" json:"user_id"`

	Title       string     `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Description string     `gorm:"column:description;type:text;not null" json:"description"`
	ImageURL    string     `gorm:"column:image_url;type:text;not null" json:"image_url"`
	TechStack   string     `gorm:"column:tech_stack;type:text" json:"tech_stack"`
	StartDate   time.Time  `gorm:"column:start_date;type:date;not null" json:"start_date"`
	EndDate     *time.Time `gorm:"column:end_date;type:date" json:"end_date,omitempty"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

type PortfolioRepository interface {
	CreatePortfolioItem(item *PortfolioItem) error
	ListPortfolioByUserID(userID uint) ([]*PortfolioItem, error)
	GetPortfolioItemByID(id uint) (*PortfolioItem, error)
	UpdatePortfolioItem(item *PortfolioItem) error
	DeletePortfolioItem(id uint) error
}

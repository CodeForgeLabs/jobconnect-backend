package repository

import (
	"job-connect/domain"

	"gorm.io/gorm"
)

type PortfolioRepository struct {
	db *gorm.DB
}

func NewPortfolioRepository(db *gorm.DB) *PortfolioRepository {
	return &PortfolioRepository{db: db}
}

func (r *PortfolioRepository) CreatePortfolioItem(item *domain.PortfolioItem) error {
	return r.db.Create(item).Error
}

func (r *PortfolioRepository) ListPortfolioByUserID(userID uint) ([]*domain.PortfolioItem, error) {
	var items []*domain.PortfolioItem
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PortfolioRepository) GetPortfolioItemByID(id uint) (*domain.PortfolioItem, error) {
	var item domain.PortfolioItem
	err := r.db.First(&item, id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *PortfolioRepository) UpdatePortfolioItem(item *domain.PortfolioItem) error {
	return r.db.Save(item).Error
}

func (r *PortfolioRepository) DeletePortfolioItem(id uint) error {
	return r.db.Delete(&domain.PortfolioItem{}, id).Error
}

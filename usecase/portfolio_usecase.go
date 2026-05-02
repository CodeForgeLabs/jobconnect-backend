package usecase

import "job-connect/domain"

type PortfolioUsecase struct {
	portfolioRepo domain.PortfolioRepository
}

func NewPortfolioUsecase(portfolioRepo domain.PortfolioRepository) *PortfolioUsecase {
	return &PortfolioUsecase{portfolioRepo: portfolioRepo}
}

func (p *PortfolioUsecase) CreatePortfolioItem(item *domain.PortfolioItem) error {
	return p.portfolioRepo.CreatePortfolioItem(item)
}

func (p *PortfolioUsecase) ListPortfolioByUserID(userID uint) ([]*domain.PortfolioItem, error) {
	return p.portfolioRepo.ListPortfolioByUserID(userID)
}

func (p *PortfolioUsecase) GetPortfolioItemByID(id uint) (*domain.PortfolioItem, error) {
	return p.portfolioRepo.GetPortfolioItemByID(id)
}

func (p *PortfolioUsecase) UpdatePortfolioItem(item *domain.PortfolioItem) error {
	return p.portfolioRepo.UpdatePortfolioItem(item)
}

func (p *PortfolioUsecase) DeletePortfolioItem(id uint) error {
	return p.portfolioRepo.DeletePortfolioItem(id)
}

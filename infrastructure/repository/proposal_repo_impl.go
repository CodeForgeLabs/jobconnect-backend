package repository

import (
	"fmt"
	"job-connect/domain"

	"gorm.io/gorm"
)

type ProposalRepository struct {
	db *gorm.DB
}

func NewProposalRepository(db *gorm.DB) *ProposalRepository {
	return &ProposalRepository{db: db}
}

func (r *ProposalRepository) CreateProposal(proposal *domain.Proposal) error {

	tx := r.db.Begin()

	// 1. create proposal
	if err := tx.Create(proposal).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 2. increment job application count
	if err := tx.Model(&domain.Job{}).
		Where("id = ?", proposal.JobID).
		UpdateColumn(
			"applications_count",
			gorm.Expr("applications_count + ?", 1),
		).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *ProposalRepository) GetProposalByID(id uint) (*domain.Proposal, error) {
	var proposal domain.Proposal
	err := r.db.First(&proposal, id).Error
	if err != nil {
		return nil, err
	}
	return &proposal, nil
}

func (r *ProposalRepository) ListProposalsByJobID(jobID uint) ([]*domain.ProposalWithUserResponse, error) {
	var results []*domain.ProposalWithUserResponse

	err := r.db.Table("proposals").
		Select(`
			proposals.id as proposal_id,
			proposals.job_id,
			proposals.sender_id,
			proposals.description,
			proposals.status,
			users.id as user_id,
			users.first_name,
			users.last_name,
			users.email,
			users.headline,
			users.skills,
			users.profile_picture_url
		`).
		Joins("JOIN users ON users.id = proposals.sender_id").
		Where("proposals.job_id = ?", jobID).
		Scan(&results).Error

	return results, err
}

func (r *ProposalRepository) UpdateProposal(proposal *domain.Proposal) error {
	return r.db.Save(proposal).Error
}

func (r *ProposalRepository) DeleteProposal(id uint) error {
	return r.db.Delete(&domain.Proposal{}, id).Error
}

func (r *ProposalRepository) ListMyProposals(userID uint) ([]*domain.Proposal, error) {
	var proposals []*domain.Proposal
	err := r.db.Where("sender_id = ?", userID).Find(&proposals).Error
	fmt.Println("error", err)
	return proposals, err
}

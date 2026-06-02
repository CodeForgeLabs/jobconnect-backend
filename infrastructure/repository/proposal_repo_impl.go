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
	// fetch job and check if it is private
	// if it is private check who is invited if that user is the invited one create the propsal otherwise return error
	var job domain.Job
	err := r.db.First(&job, proposal.JobID).Error
	if err != nil {
		return fmt.Errorf("job not found")
	}

	if job.IsPrivate {
		if job.InvitedUserId == 0 {
			return fmt.Errorf("this job is private but no user is invited yet")
		}
		if job.InvitedUserId != proposal.SenderID {
			return fmt.Errorf("you are not invited to this job")
		}
	}
	tx := r.db.Begin()
	// check if the freelancer already applied if yes reject the second
	// Check if the sender has already applied for the job
	var existingProposal domain.Proposal
	err = r.db.Where("job_id = ? AND sender_id = ?", proposal.JobID, proposal.SenderID).First(&existingProposal).Error
	if err == nil {
		// A proposal already exists for this job by this sender
		return fmt.Errorf("sender has already applied for this job")
	}

	// Check freelancer connects
	var user domain.User
	if err := tx.First(&user, proposal.SenderID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("user not found")
	}

	const requiredConnects = 10

	if user.Connect < requiredConnects {
		tx.Rollback()
		return fmt.Errorf("insufficient connects: %d required", requiredConnects)
	}
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
	// Deduct connects
	if err := tx.Model(&domain.User{}).
		Where("id = ?", proposal.SenderID).
		UpdateColumn(
			"connect",
			gorm.Expr("connect - ?", requiredConnects),
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

package repository

import (
	"errors"
	"fmt"
	"job-connect/domain"
	"strconv"
	"time"

	"gorm.io/gorm"
)

type ContractRepository struct {
	db *gorm.DB
}

func NewContractRepository(db *gorm.DB) *ContractRepository {
	return &ContractRepository{db: db}
}

func (r *ContractRepository) CreateContract(jobId, freelancerId string, clientID uint) error {
	// parse IDs
	jobIDUint, err := strconv.ParseUint(jobId, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid job id")
	}

	freelancerIDUint, err := strconv.ParseUint(freelancerId, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid freelancer id")
	}

	jobID := uint(jobIDUint)
	freelancerID := uint(freelancerIDUint)

	// =========================
	// FETCH JOB
	// =========================
	var job domain.Job
	if err := r.db.Preload("Milestones").First(&job, jobID).Error; err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	if job.CreatedBy != clientID {
		return domain.ErrForbidden
	}

	// =========================
	// FETCH PROPOSAL
	// =========================
	var proposal domain.Proposal
	if err := r.db.
		Where("job_id = ? AND sender_id = ?", jobID, freelancerID).
		First(&proposal).Error; err != nil {
		return fmt.Errorf("proposal not found: %w", err)
	}

	if proposal.Status == domain.ProposalRejected {
		return domain.ErrInvalidState
	}

	var existingContract domain.Contract
	err = r.db.Where("job_id = ? OR proposal_id = ?", jobID, proposal.ID).First(&existingContract).Error
	if err == nil {
		return domain.ErrConflict
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// =========================
	// DETERMINE CONTRACT TYPE
	// =========================
	var contractType domain.ContractType
	if job.JobType == domain.JobTypeFixed {
		contractType = domain.ContractFixed
	} else {
		contractType = domain.ContractHourly
	}

	// =========================
	// BUILD CONTRACT
	// =========================
	contract := domain.Contract{
		JobID:        job.ID,
		ProposalID:   proposal.ID,
		ClientID:     job.CreatedBy,
		FreelancerID: freelancerID,
		Type:         contractType,
		Title:        job.Title,
		Description:  job.Description,
		Status:       domain.ContractActive,
	}

	// hourly contract fields
	if contractType == domain.ContractHourly {
		contract.HourlyRate = job.HourlyRate
		contract.WeeklyHourLimit = job.MaxWeeklyHours
	}

	// fixed contract fields
	if contractType == domain.ContractFixed {
		contract.TotalBudget = job.Budget
	}

	// =========================
	// TRANSACTION START
	// =========================
	tx := r.db.Begin()

	// create contract
	if err := tx.Create(&contract).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create contract: %w", err)
	}

	// =========================
	// CREATE CONTRACT MILESTONES
	// ONLY FOR FIXED CONTRACTS
	// =========================
	if contractType == domain.ContractFixed {
		for _, milestone := range job.Milestones {
			contractMilestone := domain.ContractMilestone{
				ContractID:  contract.ID,
				Description: milestone.Description,
				Amount:      milestone.Amount,
				Status:      domain.MilestonePending,
			}

			if err := tx.Create(&contractMilestone).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to create contract milestone: %w", err)
			}
		}
	}

	// =========================
	// UPDATE PROPOSAL STATUS
	// =========================
	if err := tx.Model(&proposal).
		Update("status", domain.ProposalHired).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update proposal status: %w", err)
	}

	// =========================
	// CLOSE JOB
	// =========================
	if err := tx.Model(&job).
		Update("status", domain.StatusClosed).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to close job: %w", err)
	}

	// commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit contract creation: %w", err)
	}

	return nil
}

func (r *ContractRepository) DeleteContract(id uint) error {
	return r.db.Delete(&domain.Contract{}, id).Error
}

func (r *ContractRepository) GetMyContracts(userID uint) ([]*domain.MyContractResponse, error) {
	contracts := make([]*domain.MyContractResponse, 0)

	err := r.db.
		Table("contracts").
		Select(`
			contracts.id AS contract_id,
			contracts.job_id,
			contracts.client_id,
			contracts.freelancer_id,
			contracts.type,
			contracts.title,
			contracts.description,
			contracts.hourly_rate,
			contracts.weekly_hour_limit,
			contracts.total_budget,
			contracts.status,
			contracts.start_date,
			contracts.end_date,

			proposals.description AS proposal_description,
			jobs.title AS job_title,

			client.first_name AS client_first_name,
			client.last_name AS client_last_name,
			client.email AS client_email,
			client.headline AS client_headline,
			client.skills AS client_skills,
			client.profile_picture_url AS client_profile_picture_url,

			freelancer.first_name AS freelancer_first_name,
			freelancer.last_name AS freelancer_last_name,
			freelancer.email AS freelancer_email,
			freelancer.headline AS freelancer_headline,
			freelancer.skills AS freelancer_skills,
			freelancer.profile_picture_url AS freelancer_profile_picture_url
		`).
		Joins("LEFT JOIN proposals ON proposals.id = contracts.proposal_id").
		Joins("LEFT JOIN jobs ON jobs.id = contracts.job_id").
		Joins("LEFT JOIN users AS client ON client.id = contracts.client_id").
		Joins("LEFT JOIN users AS freelancer ON freelancer.id = contracts.freelancer_id").
		Where("contracts.client_id = ? OR contracts.freelancer_id = ?", userID, userID).
		Scan(&contracts).Error

	if err != nil {
		return nil, err
	}

	var contractIDs []uint
	for _, c := range contracts {
		contractIDs = append(contractIDs, c.ContractID)
	}

	if len(contractIDs) == 0 {
		return contracts, nil
	}

	var milestones []domain.ContractMilestone

	err = r.db.
		Where("contract_id IN ?", contractIDs).
		Find(&milestones).Error

	if err != nil {
		return nil, err
	}

	milestoneMap := make(map[uint][]domain.ContractMilestone)
	for _, m := range milestones {
		milestoneMap[m.ContractID] = append(milestoneMap[m.ContractID], m)
	}

	for _, c := range contracts {
		c.Milestones = milestoneMap[c.ContractID]
	}

	return contracts, nil
}

func (r *ContractRepository) GetContractByID(id uint) (*domain.MyContractResponse, error) {
	var contract domain.MyContractResponse

	err := r.db.
		Table("contracts").
		Select(`
			contracts.id AS contract_id,
			contracts.job_id,
			contracts.client_id,
			contracts.freelancer_id,
			contracts.type,
			contracts.title,
			contracts.description,
			contracts.hourly_rate,
			contracts.weekly_hour_limit,
			contracts.total_budget,
			contracts.status,
			contracts.start_date,
			contracts.end_date,

			proposals.description AS proposal_description,
			jobs.title AS job_title,

			client.first_name AS client_first_name,
			client.last_name AS client_last_name,
			client.email AS client_email,
			client.headline AS client_headline,
			client.skills AS client_skills,
			client.profile_picture_url AS client_profile_picture_url,

			freelancer.first_name AS freelancer_first_name,
			freelancer.last_name AS freelancer_last_name,
			freelancer.email AS freelancer_email,
			freelancer.headline AS freelancer_headline,
			freelancer.skills AS freelancer_skills,
			freelancer.profile_picture_url AS freelancer_profile_picture_url
		`).
		Joins("LEFT JOIN proposals ON proposals.id = contracts.proposal_id").
		Joins("LEFT JOIN jobs ON jobs.id = contracts.job_id").
		Joins("LEFT JOIN users AS client ON client.id = contracts.client_id").
		Joins("LEFT JOIN users AS freelancer ON freelancer.id = contracts.freelancer_id").
		Where("contracts.id = ?", id).
		Take(&contract).Error

	if err != nil {
		return nil, err
	}

	var milestones []domain.ContractMilestone

	err = r.db.
		Where("contract_id = ?", id).
		Order("created_at ASC").
		Find(&milestones).Error

	if err != nil {
		return nil, err
	}

	contract.Milestones = milestones

	return &contract, nil
}

func (r *ContractRepository) SubmitMilestone(request *domain.SubmitMilestoneRequest) error {
	// later we will notify the client
	return r.db.Model(&domain.ContractMilestone{}).
		Where("id = ? AND contract_id = ?", request.MilestoneID, request.ContractID).
		Updates(map[string]interface{}{
			"work_description": request.Description,
			"submission_url":   request.MilestoneProjectURL,
			"status":           domain.MilestoneSubmitted,
		}).Error
}

func (r *ContractRepository) ModifyStatus(milestoneId uint, newStatus domain.ContractMilestoneStatus) error {
	// later if the status is approved we will release the payment to the freelancer
	return r.db.Model(&domain.ContractMilestone{}).
		Where("id = ?", milestoneId).
		Update("status", newStatus).Error
}

func (r *ContractRepository) ModifyContractStatus(contractId, actorUserID uint, newStatus domain.ContractStatus) error {
	var contract domain.Contract
	if err := r.db.First(&contract, contractId).Error; err != nil {
		return err
	}

	if contract.ClientID != actorUserID {
		return domain.ErrForbidden
	}

	if contract.Status == domain.ContractCompleted && newStatus != domain.ContractCompleted {
		return domain.ErrInvalidState
	}

	updates := map[string]interface{}{
		"status": newStatus,
	}

	if newStatus == domain.ContractCompleted && contract.EndDate == nil {
		now := time.Now()
		updates["end_date"] = &now
	}

	return r.db.Model(&contract).Updates(updates).Error
}

func (r *ContractRepository) StartWorkSession(contractId, freelancerId uint) error {
	log := &domain.TimeLog{
		ContractID:   contractId,
		FreelancerID: freelancerId,
		StartTime:    time.Now(),
		IsPaid:       false,
	}

	return r.db.Create(log).Error
}

func (r *ContractRepository) EndWorkSession(contractId, freelancerId uint) error {
	var log domain.TimeLog

	// get latest active session
	err := r.db.
		Where("contract_id = ? AND freelancer_id = ? AND end_time IS NULL",
			contractId, freelancerId).
		Order("start_time DESC").
		First(&log).Error

	if err != nil {
		return err
	}

	now := time.Now()
	log.EndTime = &now

	duration := now.Sub(log.StartTime).Hours()
	log.TotalHours = duration

	return r.db.Save(&log).Error
}

func (r *ContractRepository) FetchTimeLogs(contractId, freelancerId uint) ([]*domain.TimeLog, error) {
	var logs []*domain.TimeLog

	err := r.db.
		Where("contract_id = ? AND freelancer_id = ?", contractId, freelancerId).
		Order("start_time DESC").
		Find(&logs).Error

	if err != nil {
		return nil, err
	}

	return logs, nil
}

func (r *ContractRepository) FetchTimeElapsed(contractId, freelancerId uint) (float64, error) {
	var logs []domain.TimeLog

	err := r.db.
		Where("contract_id = ? AND freelancer_id = ?", contractId, freelancerId).
		Find(&logs).Error

	if err != nil {
		return 0, err
	}

	var total float64

	for _, log := range logs {
		if log.EndTime != nil {
			total += log.TotalHours
		} else {
			total += time.Since(log.StartTime).Hours()
		}
	}

	return total, nil
}

func (r *ContractRepository) FetchWeeklyHours(contractId, freelancerId uint) (float64, error) {
	var logs []domain.TimeLog

	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	err := r.db.
		Where("contract_id = ? AND freelancer_id = ? AND start_time >= ?",
			contractId, freelancerId, sevenDaysAgo).
		Find(&logs).Error

	if err != nil {
		return 0, err
	}

	var total float64

	for _, log := range logs {
		if log.EndTime != nil {
			total += log.TotalHours
		} else {
			total += time.Since(log.StartTime).Hours()
		}
	}

	return total, nil
}

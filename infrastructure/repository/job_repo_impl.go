package repository

import (
	"fmt"
	"job-connect/domain"

	"gorm.io/gorm"
)

type JobRepository struct {
	db *gorm.DB
}

func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) CreateJob(job *domain.Job) error {
	if job.Budget != nil {
		userId := job.CreatedBy
		// check wallet amount
		// if wallet amount < userId return don't have enough amount
		var wallet domain.Wallet
		err := r.db.Where("user_id = ?", userId).First(&wallet).Error
		if err != nil {
			return fmt.Errorf("failed to retrieve wallet: %w", err)
		}

		if float64(wallet.BalanceMinor) < *job.Budget {
			return fmt.Errorf("insufficient funds: wallet balance is %.2f, but job budget is %.2f", float64(wallet.BalanceMinor), *job.Budget)
		}

		// deduct the amount from the user's wallet
		err = r.db.Model(&domain.Wallet{}).
			Where("user_id = ?", userId).
			Update("balance_minor", gorm.Expr("balance_minor - ?", *job.Budget)).Error
		if err != nil {
			return fmt.Errorf("failed to deduct amount from wallet: %w", err)
		}

		tx := domain.WalletTransaction{
			WalletID:    wallet.ID,
			TxRef:       fmt.Sprintf("job_creation_%d_%d", userId, job.ID),
			Type:        domain.TxEscrow,
			Status:      domain.TxSuccess,
			AmountMinor: int64(*job.Budget), // convert to minor unit
			Description: fmt.Sprintf("Payment for creating job ID %d", job.ID),
			Provider:    "Internal",
		}

		err = r.db.Create(&tx).Error
		if err != nil {
			return fmt.Errorf("failed to create wallet transaction: %w", err)
		}

	}

	err := r.db.Create(job).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *JobRepository) GetJobByID(id uint) (*domain.Job, error) {
	var job domain.Job

	err := r.db.
		Preload("Milestones").
		First(&job, id).Error

	if err != nil {
		return nil, err
	}

	return &job, nil
}

func (r *JobRepository) UpdateJob(job *domain.Job) error {

	tx := r.db.Begin()

	// update job
	if err := tx.Save(job).Error; err != nil {
		tx.Rollback()
		return err
	}

	// delete old milestones
	if err := tx.Where("job_id = ?", job.ID).Delete(&domain.Milestone{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// re-insert milestones
	for i := range job.Milestones {
		job.Milestones[i].JobID = job.ID
	}

	if len(job.Milestones) > 0 {
		if err := tx.Create(&job.Milestones).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (r *JobRepository) DeleteJob(id uint) error {
	// Start a transaction since we are touching multiple tables
	tx := r.db.Begin()

	// 1. Fetch all proposals associated with this job to find who applied
	var proposals []domain.Proposal
	if err := tx.Where("job_id = ?", id).Find(&proposals).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to fetch proposals for job: %w", err)
	}

	// 2. Refund 10 connects to each applicant and queue notifications
	notificationsToDispatch := make([]domain.Notification, 0, len(proposals))

	for _, proposal := range proposals {
		// Increment the user's connect count by 10
		err := tx.Model(&domain.User{}).
			Where("id = ?", proposal.SenderID).
			Update("connect", gorm.Expr("connect + ?", 10)).Error
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to refund connects for user %d: %w", proposal.SenderID, err)
		}

		// Prepare a notification struct for this specific user
		notif := domain.Notification{
			UserID:     proposal.SenderID,
			Type:       domain.NotifyConnectRefunded,
			Title:      "Connects Refunded",
			Message:    "The job you applied to was deleted by the client. We have returned your 10 connects.",
			JobID:      &id,
			ProposalID: &proposal.ID,
			IsRead:     false,
		}

		// Save the notification to the database within the transaction
		if err := tx.Create(&notif).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to save notification for user %d: %w", proposal.SenderID, err)
		}

		// Keep a record to send out via WebSockets after a successful commit
		notificationsToDispatch = append(notificationsToDispatch, notif)
	}

	// 3. Delete the proposals associated with the job (or let GORM handle cascade if configured)
	if err := tx.Where("job_id = ?", id).Delete(&domain.Proposal{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to clean up job proposals: %w", err)
	}

	// 4. Finally, delete the actual job record
	if err := tx.Delete(&domain.Job{}, id).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete job: %w", err)
	}

	// Commit the entire chain of actions cleanly
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit job deletion and refunds: %w", err)
	}

	return nil
}
func (r *JobRepository) ListJobs(filter domain.JobFilter) ([]*domain.Job, error) {
	var jobs []*domain.Job

	query := r.db.Model(&domain.Job{})

	// ======================
	// TEXT SEARCH FILTERS
	// ======================
	if filter.Title != "" {
		query = query.Where("title ILIKE ?", "%"+filter.Title+"%")
	}

	if filter.Company != "" {
		query = query.Where("company_name ILIKE ?", "%"+filter.Company+"%")
	}

	if filter.Location != "" {
		query = query.Where("location ILIKE ?", "%"+filter.Location+"%")
	}

	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}

	// ======================
	// ENUM FILTERS
	// ======================
	if filter.JobType != "" {
		query = query.Where("job_type = ?", filter.JobType)
	}

	if filter.ExperienceLevel != "" {
		query = query.Where("experience_level = ?", filter.ExperienceLevel)
	}

	if filter.WorkMode != "" {
		query = query.Where("work_mode = ?", filter.WorkMode)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	// ======================
	// BUDGET FILTERS
	// ======================
	if filter.BudgetMin != nil {
		query = query.Where("budget >= ?", *filter.BudgetMin)
	}

	if filter.HourlyRateMin != nil {
		query = query.Where("hourly_rate >= ?", *filter.HourlyRateMin)
	}

	// ======================
	// SKILLS FILTER (IMPORTANT)
	// ======================
	if len(filter.Skills) > 0 {
		for _, skill := range filter.Skills {
			query = query.Where("skills ILIKE ?", "%"+skill+"%")
		}
	}
	query = query.Where("status = ?", domain.StatusOpen).Order("created_at DESC")
	// EXECUTE
	// ======================
	err := query.
		Preload("Milestones").
		Order("created_at DESC").
		Find(&jobs).Error
	if err != nil {
		return nil, err
	}

	return jobs, nil
}

func (r *JobRepository) ListMyJobs(userID uint) ([]*domain.Job, error) {
	var jobs []*domain.Job

	// retrive all my jobs i posted so far
	err := r.db.Where("created_by = ?", userID).
		Preload("Milestones").
		Order("created_at DESC").
		Find(&jobs).Error
	if err != nil {
		return nil, err
	}

	return jobs, nil
}

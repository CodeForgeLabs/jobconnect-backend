package repository

import (
	"fmt"
	"job-connect/domain"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

type JobRepository struct {
	db               *gorm.DB
	notificationRepo domain.NotificationRepository
}

func NewJobRepository(db *gorm.DB, notificationRepo domain.NotificationRepository) *JobRepository {
	return &JobRepository{db: db, notificationRepo: notificationRepo}
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

		txRef := generateTxRef(wallet.ID)
		tx := domain.WalletTransaction{
			WalletID:    wallet.ID,
			TxRef:       fmt.Sprintf("job_creation_%d_%d_%s", userId, job.ID, txRef),
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
	var job domain.Job
	if err := tx.First(&job, id).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("job not found: %w", err)
	}

	var wallet domain.Wallet

	if err := tx.
		Where("user_id = ?", job.CreatedBy).
		First(&wallet).Error; err != nil {

		tx.Rollback()
		return fmt.Errorf("client wallet not found: %w", err)
	}

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

	// 5 refund the balance for the client
	if err := tx.Model(&wallet).
		Update(
			"balance_minor",
			gorm.Expr("balance_minor + ?", job.Budget),
		).Error; err != nil {

		tx.Rollback()
		return fmt.Errorf("failed to refund client wallet: %w", err)
	}
	txRef := generateTxRef(wallet.ID)
	walletTx := domain.WalletTransaction{
		WalletID:    wallet.ID,
		TxRef:       fmt.Sprintf("job_creation_%d_%d_%s", wallet.ID, job.ID, txRef),
		Type:        domain.TxRefund,
		Status:      domain.TxSuccess,
		AmountMinor: int64(*job.Budget),
		Description: "Refund for deleted job",
		Provider:    "SYSTEM",
	}

	if err := tx.Create(&walletTx).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create refund transaction: %w", err)
	}
	// notify the client
	clientNotif := domain.Notification{
		UserID:  job.CreatedBy,
		Type:    domain.NotifyJobDeleted,
		Title:   "Job Deleted",
		Message: fmt.Sprintf("Your job '%s' was deleted. We have refunded your wallet with the original budget amount.", job.Title),
		JobID:   &id,
		IsRead:  false,
	}

	if err := tx.Create(&clientNotif).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create client notification: %w", err)
	}
	// After successful commit, dispatch all notifications via WebSockets
	for _, notif := range notificationsToDispatch {
		r.notificationRepo.CreateNotification(&notif) // This will also send via WebSocket
	}
	r.notificationRepo.CreateNotification(&clientNotif) // Notify the client as well
	// Commit the entire chain of actions cleanly
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit job deletion and refunds: %w", err)
	}

	return nil
}
func (r *JobRepository) ListJobs(filter domain.JobFilter) ([]*domain.Job, error) {
	var jobs []*domain.Job

	query := r.db.Model(&domain.Job{}).
		Where("status = ?", domain.StatusOpen).
		Where("is_private = ?", false)

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

func (r *JobRepository) ListRecommendedJobs(filter domain.JobFilter) ([]*domain.Job, error) {
	var jobs []*domain.Job

	query := r.db.Model(&domain.Job{})

	// ======================
	// ONLY PUBLIC OPEN JOBS
	// ======================
	query = query.
		Where("status = ?", domain.StatusOpen).
		Where("is_private = ?", false)

	// ======================
	// TEXT FILTERS
	// ======================
	if filter.Title != "" {
		query = query.Where(
			"title ILIKE ?",
			"%"+filter.Title+"%",
		)
	}

	if filter.Company != "" {
		query = query.Where(
			"company_name ILIKE ?",
			"%"+filter.Company+"%",
		)
	}

	if filter.Location != "" {
		query = query.Where(
			"location ILIKE ?",
			"%"+filter.Location+"%",
		)
	}

	if filter.Category != "" {
		query = query.Where(
			"category = ?",
			filter.Category,
		)
	}

	// ======================
	// ENUM FILTERS
	// ======================
	if filter.JobType != "" {
		query = query.Where(
			"job_type = ?",
			filter.JobType,
		)
	}

	if filter.ExperienceLevel != "" {
		query = query.Where(
			"experience_level = ?",
			filter.ExperienceLevel,
		)
	}

	if filter.WorkMode != "" {
		query = query.Where(
			"work_mode = ?",
			filter.WorkMode,
		)
	}

	// ======================
	// BUDGET FILTERS
	// ======================
	if filter.BudgetMin != nil {
		query = query.Where(
			"budget >= ?",
			*filter.BudgetMin,
		)
	}

	if filter.HourlyRateMin != nil {
		query = query.Where(
			"hourly_rate >= ?",
			*filter.HourlyRateMin,
		)
	}

	// ======================
	// SKILL FILTERS
	// ======================
	if len(filter.Skills) > 0 {
		for _, skill := range filter.Skills {
			query = query.Where(
				"skills ILIKE ?",
				"%"+skill+"%",
			)
		}
	}

	// ======================
	// LOAD JOBS
	// ======================
	if err := query.
		Preload("Milestones").
		Find(&jobs).Error; err != nil {
		return nil, err
	}

	// ======================
	// NO RECOMMENDATION MODE
	// ======================
	if filter.RecommendedFor == nil {
		sort.Slice(jobs, func(i, j int) bool {
			return jobs[i].CreatedAt.After(
				jobs[j].CreatedAt,
			)
		})

		return jobs, nil
	}

	// ======================
	// LOAD USER
	// ======================
	var user domain.User

	if err := r.db.
		First(&user, *filter.RecommendedFor).
		Error; err != nil {
		return nil, err
	}

	// ======================
	// USER SKILLS
	// ======================
	userSkills := []string{}

	if user.Skills != "" {
		for _, skill := range strings.Split(
			user.Skills,
			",",
		) {
			userSkills = append(
				userSkills,
				strings.TrimSpace(
					strings.ToLower(skill),
				),
			)
		}
	}

	// ======================
	// USER PROPOSALS
	// ======================
	var proposals []domain.Proposal

	if err := r.db.
		Where("sender_id = ?", user.ID).
		Find(&proposals).Error; err != nil {
		return nil, err
	}

	proposedJobs := make(map[uint]bool)

	for _, proposal := range proposals {
		proposedJobs[proposal.JobID] = true
	}

	// ======================
	// SCORE JOBS
	// ======================
	type scoredJob struct {
		job   *domain.Job
		score int
	}

	var scoredJobs []scoredJob

	for _, job := range jobs {

		if job.CreatedBy == user.ID {
			continue
		}

		score := 0

		// ----------
		// Skill score
		// ----------
		jobSkills := []string{}

		if job.Skills != "" {
			for _, skill := range strings.Split(
				job.Skills,
				",",
			) {
				jobSkills = append(
					jobSkills,
					strings.TrimSpace(
						strings.ToLower(skill),
					),
				)
			}
		}

		for _, userSkill := range userSkills {
			for _, jobSkill := range jobSkills {
				if userSkill == jobSkill {
					score += 10
				}
			}
		}

		// ----------
		// Location bonus
		// ----------
		if user.Location != "" &&
			strings.EqualFold(
				strings.TrimSpace(user.Location),
				strings.TrimSpace(job.Location),
			) {
			score += 5
		}

		// ----------
		// Remote bonus
		// ----------
		if job.WorkMode == domain.WorkModeRemote {
			score += 3
		}

		// ----------
		// Recent jobs bonus
		// ----------
		if time.Since(job.CreatedAt) < 72*time.Hour {
			score += 5
		}

		// ----------
		// Already applied penalty
		// ----------
		if proposedJobs[job.ID] {
			score -= 20
		}

		scoredJobs = append(
			scoredJobs,
			scoredJob{
				job:   job,
				score: score,
			},
		)
	}

	// ======================
	// SORT BY SCORE
	// ======================
	sort.Slice(
		scoredJobs,
		func(i, j int) bool {

			if scoredJobs[i].score ==
				scoredJobs[j].score {

				return scoredJobs[i].
					job.CreatedAt.After(
					scoredJobs[j].
						job.CreatedAt,
				)
			}

			return scoredJobs[i].score >
				scoredJobs[j].score
		},
	)

	// ======================
	// EXTRACT JOBS
	// ======================
	recommendedJobs := make(
		[]*domain.Job,
		0,
		len(scoredJobs),
	)

	for _, item := range scoredJobs {
		recommendedJobs = append(
			recommendedJobs,
			item.job,
		)
	}

	return recommendedJobs, nil
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

func (r *JobRepository) ListJobByClientId(clientID uint) ([]*domain.Job, error) {
	var jobs []*domain.Job
	fmt.Println("^^^^^^^^^^^^^^^^^^^^^^^^^^")
	// retrive all my jobs i posted so far
	err := r.db.Where("created_by = ?", clientID).
		Preload("Milestones").
		Order("created_at DESC").
		Find(&jobs).Error
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return jobs, nil
}

func (r *JobRepository) InviteUserToJob(jobID uint, userID uint, clientId uint) error {
	// check first if the job is private and belongs to the client and also he doesn't invited someone else before
	var job domain.Job
	err := r.db.First(&job, jobID).Error
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	if !job.IsPrivate {
		return fmt.Errorf("cannot invite user to a public job")
	}

	if job.CreatedBy != clientId {
		return fmt.Errorf("only the client who created the job can invite users")
	}

	if job.InvitedUserId != 0 {
		return fmt.Errorf("a user has already been invited to this job")
	}
	// Update the job record to set the invited_user_id
	err = r.db.Model(&domain.Job{}).
		Where("id = ?", jobID).
		Update("invited_user_id", userID).Error
	if err != nil {
		return fmt.Errorf("failed to invite user to job: %w", err)
	}

	// Optionally, you can also create a notification for the invited user here
	notification := domain.Notification{
		UserID:  userID,
		Type:    domain.NotifyInvitationToJob,
		Title:   "You've been invited to a job",
		Message: fmt.Sprintf("You have been invited to apply for the job '%s'. Check it out!", job.Title),
		JobID:   &jobID,
	}

	err = r.notificationRepo.CreateNotification(&notification)
	if err != nil {
		return fmt.Errorf("failed to create notification for invited user: %w", err)
	}

	return nil
}

func (r *JobRepository) GetGotInvitedJobs(userID uint) ([]*domain.Job, error) {
	var jobs []*domain.Job

	err := r.db.Where("invited_user_id = ?", userID).
		Preload("Milestones").
		Order("created_at DESC").
		Find(&jobs).Error
	if err != nil {
		return nil, err
	}

	return jobs, nil
}

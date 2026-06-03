package repository

import (
	"errors"
	"fmt"
	"job-connect/chapa"
	"job-connect/domain"
	"strconv"
	"time"

	"gorm.io/gorm"
)

type ContractRepository struct {
	db               *gorm.DB
	notificationRepo domain.NotificationRepository
}

func NewContractRepository(db *gorm.DB, notificationRepo domain.NotificationRepository) *ContractRepository {
	return &ContractRepository{db: db, notificationRepo: notificationRepo}
}

func (r *ContractRepository) CreateContract(jobId, freelancerId string, clientID uint) error {
	// check first if we have already created the contract between this this freelancerId and client id or if this client id is hired someone for this job then return an error
	var econtract domain.Contract
	err := r.db.
		Where("job_id = ?", jobId).
		First(&econtract).
		Error

	if err == nil {
		return fmt.Errorf("a contract already exists for this job")
	}
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
				DeadLine:    milestone.Deadline,
			}

			if err := tx.Create(&contractMilestone).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to create contract milestone: %w", err)
			}
		}
	}

	// =========================
	// UPDATE SELECTED PROPOSAL
	// =========================
	if err := tx.Model(&proposal).
		Update("status", domain.ProposalHired).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update proposal status: %w", err)
	}

	// =========================
	// REJECT OTHER PROPOSALS
	// =========================
	if err := tx.Model(&domain.Proposal{}).
		Where("job_id = ? AND id <> ?", job.ID, proposal.ID).
		Where("status IN ?", []domain.ProposalStatus{
			domain.ProposalPending,
			domain.ProposalInvited,
		}).
		Update("status", domain.ProposalRejected).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to reject other proposals: %w", err)
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
	// 1. Create Notification for Freelancer
	freelancerNotif := domain.Notification{
		UserID:     freelancerID,
		Type:       domain.NotifyContractCreated,
		Title:      "Congratulations! You've been hired",
		Message:    fmt.Sprintf("Your proposal for '%s' was accepted. The contract is now active.", contract.Title),
		ContractID: &contract.ID,
		ProposalID: &proposal.ID,
		JobID:      &job.ID,
		IsRead:     false,
	}

	_ = r.notificationRepo.CreateNotification(
		&freelancerNotif,
	)

	// 2. Create Notification for Client
	clientNotif := domain.Notification{
		UserID:     clientID,
		Type:       domain.NotifyContractCreated,
		Title:      "Contract started successfully",
		Message:    fmt.Sprintf("You have successfully started a contract with the freelancer for '%s'.", contract.Title),
		ContractID: &contract.ID,
		ProposalID: &proposal.ID,
		JobID:      &job.ID,
		IsRead:     false,
	}
	_ = r.notificationRepo.CreateNotification(&clientNotif)
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
	err := r.db.Model(&domain.ContractMilestone{}).
		Where("id = ? AND contract_id = ?", request.MilestoneID, request.ContractID).
		Updates(map[string]interface{}{
			"work_description": request.Description,
			"submission_url":   request.MilestoneProjectURL,
			"status":           domain.MilestoneSubmitted,
		}).Error
	if err != nil {
		return fmt.Errorf("failed to submit milestone updates: %w", err)
	}
	// 3. Create the database notification for the Client
	var contract domain.Contract
	if err := r.db.First(&contract, request.ContractID).Error; err != nil {
		return fmt.Errorf("contract not found for notification: %w", err)
	}
	clientNotif := domain.Notification{
		UserID:     contract.ClientID, // Retrieved dynamically from the contract record
		Type:       domain.NotifyMilestoneStatus,
		Title:      "Milestone Work Submitted",
		Message:    fmt.Sprintf("The work for milestone on contract '%s' has been submitted for your approval.", contract.Title),
		ContractID: &contract.ID,
		JobID:      &contract.JobID,
		ProposalID: &contract.ProposalID,
		IsRead:     false,
	}
	_ = r.notificationRepo.CreateNotification(&clientNotif)

	// 4. Create a database notification for the Freelancer as confirmation
	freelancerNotif := domain.Notification{
		UserID:     contract.FreelancerID,
		Type:       domain.NotifyMilestoneStatus,
		Title:      "Milestone Submitted Successfully",
		Message:    fmt.Sprintf("Your submission for contract '%s' was sent to the client.", contract.Title),
		ContractID: &contract.ID,
		JobID:      &contract.JobID,
		ProposalID: &contract.ProposalID,
		IsRead:     false,
	}
	_ = r.notificationRepo.CreateNotification(&freelancerNotif)
	return nil
}

func (r *ContractRepository) ModifyStatus(milestoneId uint, newStatus domain.ContractMilestoneStatus, feedback string) error {
	// 1. Find the milestone first to get its ContractID
	var milestone domain.ContractMilestone
	if err := r.db.First(&milestone, milestoneId).Error; err != nil {
		return fmt.Errorf("milestone not found: %w", err)
	}

	// 2. Fetch the parent Contract to get the FreelancerID and Title context
	var contract domain.Contract
	if err := r.db.First(&contract, milestone.ContractID).Error; err != nil {
		return fmt.Errorf("parent contract not found: %w", err)
	}

	// 3. Update the milestone status in the database
	err := r.db.Model(&domain.ContractMilestone{}).
		Where("id = ?", milestoneId).
		Updates(map[string]interface{}{
			"status":          newStatus,
			"client_feedback": feedback,
		}).Error
	if err != nil {
		return fmt.Errorf("failed to update milestone status: %w", err)
	}

	if newStatus == domain.MilestoneApproved {
		var wallet domain.Wallet
		err := r.db.Where("user_id = ?", contract.FreelancerID).First(&wallet).Error
		if err != nil {
			return fmt.Errorf("failed to retrieve wallet: %w", err)
		}

		// calculate the amount in minor unit (cents)
		amountMinor := int64(milestone.Amount)

		err = r.db.Model(&domain.Wallet{}).
			Where("user_id = ?", contract.FreelancerID).
			Update("balance_minor", gorm.Expr("balance_minor + ?", amountMinor)).Error
		if err != nil {
			return fmt.Errorf("failed to update wallet balance: %w", err)
		}

		tx := domain.WalletTransaction{
			WalletID:    wallet.ID,
			TxRef:       fmt.Sprintf("milestone_payment_%d_%d", contract.FreelancerID, milestone.ID),
			Type:        domain.TxPayment,
			Status:      domain.TxSuccess,
			AmountMinor: amountMinor,
			Description: fmt.Sprintf("Payment for milestone on contract '%s'", contract.Title),
			Provider:    "Internal",
		}

		err = r.db.Create(&tx).Error
		if err != nil {
			return fmt.Errorf("failed to create wallet transaction: %w", err)
		}
	}

	// 4. Customize the notification title and message based on the new status
	var title, message string
	switch newStatus {
	case domain.MilestoneRevisionRequested:
		title = "Revision Requested"
		message = fmt.Sprintf("The client requested changes on your milestone for contract '%s'.", contract.Title)
	case domain.MilestoneApproved, domain.MilestonePaid:
		title = "Milestone Approved & Payment Released"
		message = fmt.Sprintf("Great news! Your milestone for contract '%s' has been approved, and your payment has been released from escrow.", contract.Title)
	default:
		title = "Milestone Status Updated"
		message = fmt.Sprintf("Your milestone status for contract '%s' has been updated to %s.", contract.Title, newStatus)
	}

	// 5. Save the notification to the database for the Freelancer
	freelancerNotif := domain.Notification{
		UserID:     contract.FreelancerID, // Retrieved safely through the milestone -> contract connection
		Type:       domain.NotifyMilestoneStatus,
		Title:      title,
		Message:    message,
		ContractID: &contract.ID,
		JobID:      &contract.JobID,
		ProposalID: &contract.ProposalID,
		IsRead:     false,
	}
	_ = r.notificationRepo.CreateNotification(&freelancerNotif)

	return nil
}

// func (r *ContractRepository) ModifyContractStatus(contractId, actorUserID uint, newStatus domain.ContractStatus) error {
// 	var contract domain.Contract
// 	if err := r.db.First(&contract, contractId).Error; err != nil {
// 		return err
// 	}

// 	if contract.ClientID != actorUserID {
// 		return domain.ErrForbidden
// 	}

// 	if contract.Status == domain.ContractCompleted && newStatus != domain.ContractCompleted {
// 		return domain.ErrInvalidState
// 	}

// 	updates := map[string]interface{}{
// 		"status": newStatus,
// 	}

// 	if newStatus == domain.ContractCompleted && contract.EndDate == nil {
// 		now := time.Now()
// 		updates["end_date"] = &now
// 	}

// 	// 1. Execute the status updates in the database
// 	if err := r.db.Model(&contract).Updates(updates).Error; err != nil {
// 		return err
// 	}

// 	// 2. Customize the notification content based on the new contract status
// 	var title, message string
// 	switch newStatus {
// 	case domain.ContractCompleted:
// 		title = "Contract Completed!"
// 		message = fmt.Sprintf("The client has marked your contract '%s' as completed. Great job!", contract.Title)
// 	case "CANCELLED": // Use your exact enum string/value if you have a cancelled status
// 		title = "Contract Cancelled"
// 		message = fmt.Sprintf("The contract '%s' has been cancelled by the client.", contract.Title)
// 	default:
// 		title = "Contract Status Updated"
// 		message = fmt.Sprintf("The status of your contract '%s' has been updated to %s.", contract.Title, newStatus)
// 	}

// 	// 3. Save the notification to the database for the Freelancer
// 	freelancerNotif := domain.Notification{
// 		UserID:     contract.FreelancerID, // The freelancer tied to this contract
// 		Type:       domain.NotifyContractStatus,
// 		Title:      title,
// 		Message:    message,
// 		ContractID: &contract.ID,
// 		JobID:      &contract.JobID,
// 		ProposalID: &contract.ProposalID,
// 		IsRead:     false,
// 	}
// 	_ = r.notificationRepo.CreateNotification(&freelancerNotif)

// 	return nil
// }

func (r *ContractRepository) ModifyContractStatus(contractId, actorUserID uint, newStatus domain.ContractStatus) error {
	// Start a database transaction to ensure data integrity
	txCtx := r.db.Begin()
	if txCtx.Error != nil {
		return txCtx.Error
	}
	// Defer a rollback which will execute if we return early due to an error
	defer func() {
		if r := recover(); r != nil {
			txCtx.Rollback()
		}
	}()

	var contract domain.Contract
	if err := txCtx.First(&contract, contractId).Error; err != nil {
		txCtx.Rollback()
		return err
	}

	if contract.ClientID != actorUserID {
		txCtx.Rollback()
		return domain.ErrForbidden
	}

	if contract.Status == domain.ContractCompleted && newStatus != domain.ContractCompleted {
		txCtx.Rollback()
		return domain.ErrInvalidState
	}

	updates := map[string]interface{}{
		"status": newStatus,
	}

	if newStatus == domain.ContractCompleted && contract.EndDate == nil {
		now := time.Now()
		updates["end_date"] = &now
	}

	// 1. Execute the status updates in the database
	if err := txCtx.Model(&contract).Updates(updates).Error; err != nil {
		txCtx.Rollback()
		return err
	}

	// 2. Only refund milestones if the new status is CANCELLED
	if newStatus == domain.ContractCancelled {
		var milestones []domain.ContractMilestone
		if err := txCtx.Where("contract_id = ?", contractId).Find(&milestones).Error; err != nil {
			txCtx.Rollback()
			return err
		}

		var clientWallet domain.Wallet
		if err := txCtx.Where("user_id = ?", contract.ClientID).
			First(&clientWallet).Error; err != nil {
			txCtx.Rollback()
			return err
		}

		var freelancerWallet domain.Wallet
		if err := txCtx.Where("user_id = ?", contract.FreelancerID).
			First(&freelancerWallet).Error; err != nil {
			txCtx.Rollback()
			return err
		}

		for _, m := range milestones {

			// already settled
			if m.Status == domain.MilestoneApproved ||
				m.Status == domain.MilestonePaid {
				continue
			}

			amountMinor := int64(m.Amount)

			var refundToClient int64
			var payToFreelancer int64

			switch m.Status {

			case domain.MilestonePending:
				refundToClient = amountMinor
				payToFreelancer = 0

			case domain.MilestoneInProgress,
				domain.MilestoneSubmitted,
				domain.MilestoneRevisionRequested:

				refundToClient = amountMinor / 2
				payToFreelancer = amountMinor - refundToClient

			default:
				refundToClient = amountMinor
			}

			// =====================
			// REFUND CLIENT
			// =====================
			if refundToClient > 0 {

				if err := txCtx.Model(&clientWallet).
					Update(
						"balance_minor",
						gorm.Expr(
							"balance_minor + ?",
							refundToClient,
						),
					).Error; err != nil {
					txCtx.Rollback()
					return err
				}

				clientTx := domain.WalletTransaction{
					WalletID: clientWallet.ID,
					TxRef: fmt.Sprintf(
						"CONTRACT-CANCEL-REFUND-%d-%d",
						contract.ID,
						time.Now().UnixNano(),
					),
					Type:        domain.TxRefund,
					Status:      domain.TxSuccess,
					AmountMinor: refundToClient,
					Description: fmt.Sprintf(
						"Refund for cancelled contract %d milestone %d",
						contract.ID,
						m.ID,
					),
					Provider: "SYSTEM",
				}

				if err := txCtx.Create(&clientTx).Error; err != nil {
					txCtx.Rollback()
					return err
				}
			}

			// =====================
			// PAY FREELANCER
			// =====================
			if payToFreelancer > 0 {

				if err := txCtx.Model(&freelancerWallet).
					Update(
						"balance_minor",
						gorm.Expr(
							"balance_minor + ?",
							payToFreelancer,
						),
					).Error; err != nil {
					txCtx.Rollback()
					return err
				}

				freelancerTx := domain.WalletTransaction{
					WalletID: freelancerWallet.ID,
					TxRef: fmt.Sprintf(
						"CONTRACT-CANCEL-PAYOUT-%d-%d",
						contract.ID,
						time.Now().UnixNano(),
					),
					Type:        domain.TxPayment,
					Status:      domain.TxSuccess,
					AmountMinor: payToFreelancer,
					Description: fmt.Sprintf(
						"50%% compensation for cancelled contract %d milestone %d",
						contract.ID,
						m.ID,
					),
					Provider: "SYSTEM",
				}

				if err := txCtx.Create(&freelancerTx).Error; err != nil {
					txCtx.Rollback()
					return err
				}
			}

			// =====================
			// MARK MILESTONE
			// =====================
			if err := txCtx.Model(&domain.ContractMilestone{}).
				Where("id = ?", m.ID).
				Update(
					"status",
					domain.MilestonePaid,
				).Error; err != nil {
				txCtx.Rollback()
				return err
			}
		}
	}

	// Commit the database transaction if everything up to this point succeeded
	if err := txCtx.Commit().Error; err != nil {
		return err
	}

	// 3. Customize the notification content based on the new contract status
	var title, message string
	switch newStatus {
	case domain.ContractCompleted:
		title = "Contract Completed!"
		message = fmt.Sprintf("The client has marked your contract '%s' as completed. Great job!", contract.Title)
	case "CANCELLED":
		title = "Contract Cancelled"
		message = fmt.Sprintf("The contract '%s' has been cancelled by the client.", contract.Title)
	default:
		title = "Contract Status Updated"
		message = fmt.Sprintf("The status of your contract '%s' has been updated to %s.", contract.Title, newStatus)
	}

	// 4. Save the notification to the database for the Freelancer
	freelancerNotif := domain.Notification{
		UserID:     contract.FreelancerID,
		Type:       domain.NotifyContractStatus,
		Title:      title,
		Message:    message,
		ContractID: &contract.ID,
		JobID:      &contract.JobID,
		ProposalID: &contract.ProposalID,
		IsRead:     false,
	}
	_ = r.notificationRepo.CreateNotification(&freelancerNotif)

	return nil
}
func (r *ContractRepository) StartWorkSession(contractId, freelancerId uint) (string, error) {
	var freelancer domain.User
	var contract domain.Contract
	var client domain.User
	if err := r.db.First(&freelancer, freelancerId).Error; err != nil {
		return "", fmt.Errorf("freelancer not found: %w", err)
	}
	if err := r.db.First(&contract, contractId).Error; err != nil {
		return "", fmt.Errorf("client not found: %w", err)
	}
	if err := r.db.First(&client, contract.ClientID).Error; err != nil {
		return "", fmt.Errorf("client not found: %w", err)
	}
	if contract.Status == domain.ContractCompleted || contract.Status == domain.ContractCancelled {
		return "", fmt.Errorf("cannot start work session on a completed or cancelled contract")
	}
	// 1. Create time log
	log := &domain.TimeLog{
		ContractID:   contractId,
		FreelancerID: freelancerId,
		StartTime:    time.Now(),
		IsPaid:       false,
	}

	if err := r.db.Create(log).Error; err != nil {
		return "", err
	}

	// 2. Create calendar invite
	calendarService := chapa.NewGoogleCalendarInviteService()

	now := time.Now()
	start := now
	end := now.Add(3 * time.Hour)
	fmt.Println("********************************************")
	fmt.Println(client.Email)
	fmt.Println(freelancer.Email)
	result, err := calendarService.CreateInvite(
		chapa.CalendarInviteInput{
			Summary:     "Work Session Started",
			Description: "Freelancer work monitoring session",

			AttendeeEmails: []string{
				client.Email,
				freelancer.Email,
			},

			StartAt: start.Format(time.RFC3339),
			EndAt:   end.Format(time.RFC3339),
		},
	)

	if err != nil {
		return "", err
	}

	// 3. return meeting link
	return result.MeetLink, nil
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

func (r *ContractRepository) FetchTimeLogs(contractId uint) ([]*domain.TimeLog, error) {
	var logs []*domain.TimeLog

	err := r.db.
		Where("contract_id = ?", contractId).
		Order("start_time DESC").
		Find(&logs).Error

	if err != nil {
		return nil, err
	}

	return logs, nil
}

func (r *ContractRepository) FetchTimeElapsed(contractId uint) (float64, error) {
	var logs []domain.TimeLog

	err := r.db.
		Where("contract_id = ?", contractId).
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

func (r *ContractRepository) FetchWeeklyHours(contractId uint) (float64, error) {
	var logs []domain.TimeLog

	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	err := r.db.
		Where("contract_id = ? AND start_time >= ?",
			contractId, sevenDaysAgo).
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

func (r *ContractRepository) FetchWeeklyWorkLogs(
	contractId uint,
) ([]*domain.WeeklyWorkLogResponse, error) {

	var logs []domain.TimeLog

	// Fetch all logs for the contract
	err := r.db.
		Where("contract_id = ?", contractId).
		Order("start_time ASC").
		Find(&logs).Error

	if err != nil {
		return nil, err
	}

	// week map
	weekMap := make(map[string]*domain.WeeklyWorkLogResponse)

	for _, log := range logs {

		// -------------------------
		// WEEK INFO
		// -------------------------
		year, week := log.StartTime.ISOWeek()

		weekKey := fmt.Sprintf("%d-%d", year, week)

		// Calculate week start (Monday)
		weekday := int(log.StartTime.Weekday())

		// Go starts Sunday=0
		if weekday == 0 {
			weekday = 7
		}

		weekStart := log.StartTime.AddDate(0, 0, -(weekday - 1))
		weekEnd := weekStart.AddDate(0, 0, 6)

		// -------------------------
		// CREATE WEEK IF NOT EXISTS
		// -------------------------
		if _, exists := weekMap[weekKey]; !exists {
			weekMap[weekKey] = &domain.WeeklyWorkLogResponse{
				WeekNumber: week,
				WeekStart:  weekStart.Format("2006-01-02"),
				WeekEnd:    weekEnd.Format("2006-01-02"),
				Days:       []domain.DayWorkLogResponse{},
			}
		}

		weekResponse := weekMap[weekKey]

		// -------------------------
		// FIND DAY
		// -------------------------
		dayName := log.StartTime.Weekday().String()
		date := log.StartTime.Format("2006-01-02")

		var dayResponse *domain.DayWorkLogResponse

		for i := range weekResponse.Days {
			if weekResponse.Days[i].Date == date {
				dayResponse = &weekResponse.Days[i]
				break
			}
		}

		// -------------------------
		// CREATE DAY IF NOT EXISTS
		// -------------------------
		if dayResponse == nil {

			newDay := domain.DayWorkLogResponse{
				Day:        dayName,
				Date:       date,
				TotalHours: 0,
				Sessions:   []domain.WorkSessionResponse{},
			}

			weekResponse.Days = append(weekResponse.Days, newDay)

			dayResponse = &weekResponse.Days[len(weekResponse.Days)-1]
		}

		// -------------------------
		// CREATE SESSION
		// -------------------------
		session := domain.WorkSessionResponse{
			ID:         log.ID,
			StartTime:  log.StartTime,
			EndTime:    log.EndTime,
			TotalHours: log.TotalHours,
			IsPaid:     log.IsPaid,
		}

		// append session
		dayResponse.Sessions = append(dayResponse.Sessions, session)

		// update totals
		dayResponse.TotalHours += log.TotalHours
		weekResponse.TotalHours += log.TotalHours
	}

	// -------------------------
	// CONVERT MAP TO SLICE
	// -------------------------
	result := make([]*domain.WeeklyWorkLogResponse, 0)

	for _, week := range weekMap {
		result = append(result, week)
	}

	return result, nil
}

func (r *ContractRepository) PayWeeklyLogs(
	request domain.PayWeeklyLogsRequest,
) error {

	// -----------------------------------
	// FETCH CONTRACT
	// -----------------------------------
	var contract domain.Contract

	if err := r.db.
		First(&contract, request.ContractID).Error; err != nil {

		return err
	}

	if contract.Type != domain.ContractHourly {
		return fmt.Errorf("contract is not hourly")
	}

	if contract.HourlyRate == nil {
		return fmt.Errorf("hourly rate missing")
	}

	// -----------------------------------
	// FETCH UNPAID LOGS
	// -----------------------------------
	var logs []domain.TimeLog

	if err := r.db.
		Where("contract_id = ? AND is_paid = ?",
			request.ContractID,
			false,
		).
		Find(&logs).Error; err != nil {

		return err
	}

	var selectedLogs []domain.TimeLog

	var totalHours float64

	for _, log := range logs {

		year, week := log.StartTime.ISOWeek()

		if year == request.Year &&
			week == request.WeekNumber {

			selectedLogs = append(selectedLogs, log)
			totalHours += log.TotalHours
		}
	}

	if len(selectedLogs) == 0 {
		return fmt.Errorf("no unpaid logs found")
	}

	// -----------------------------------
	// CALCULATE PAYMENT
	// -----------------------------------
	totalAmount := totalHours * (*contract.HourlyRate)

	// convert to minor unit
	totalAmountMinor := int64(totalAmount)

	// -----------------------------------
	// START DB TRANSACTION
	// -----------------------------------
	tx := r.db.Begin()

	// -----------------------------------
	// FETCH WALLETS
	// -----------------------------------
	var clientWallet domain.Wallet
	var freelancerWallet domain.Wallet

	if err := tx.
		Where("user_id = ?", contract.ClientID).
		First(&clientWallet).Error; err != nil {

		tx.Rollback()
		return err
	}

	if err := tx.
		Where("user_id = ?", contract.FreelancerID).
		First(&freelancerWallet).Error; err != nil {

		tx.Rollback()
		return err
	}

	// -----------------------------------
	// CHECK CLIENT BALANCE
	// -----------------------------------
	if clientWallet.BalanceMinor < totalAmountMinor {

		tx.Rollback()
		return fmt.Errorf("insufficient balance")
	}

	// -----------------------------------
	// UPDATE WALLET BALANCES
	// -----------------------------------
	clientWallet.BalanceMinor -= totalAmountMinor

	freelancerWallet.BalanceMinor += totalAmountMinor

	if err := tx.Save(&clientWallet).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Save(&freelancerWallet).Error; err != nil {
		tx.Rollback()
		return err
	}

	// -----------------------------------
	// MARK LOGS AS PAID
	// -----------------------------------
	var logIDs []uint

	for _, log := range selectedLogs {
		logIDs = append(logIDs, log.ID)
	}

	if err := tx.
		Model(&domain.TimeLog{}).
		Where("id IN ?", logIDs).
		Update("is_paid", true).Error; err != nil {

		tx.Rollback()
		return err
	}

	// -----------------------------------
	// CREATE CLIENT TRANSACTION
	// -----------------------------------
	clientTx := domain.WalletTransaction{
		WalletID: clientWallet.ID,

		// TxRef: fmt.Sprintf(
		// 	"WEEKLY-PAY-%d-%d-%d-client",
		// 	request.ContractID,
		// 	request.Year,
		// 	request.WeekNumber,
		// ),
		TxRef: fmt.Sprintf(
			"WEEKLY-PAY-%d-%d-%d-%d-client",
			request.ContractID,
			request.Year,
			request.WeekNumber,
			time.Now().UnixNano(),
		),

		Type:        domain.TxPayment,
		Status:      domain.TxSuccess,
		AmountMinor: totalAmountMinor,
		Description: fmt.Sprintf(
			"Weekly payment for contract #%d week %d",
			request.ContractID,
			request.WeekNumber,
		),
		Provider: "INTERNAL",
	}

	if err := tx.Create(&clientTx).Error; err != nil {
		tx.Rollback()
		return err
	}

	// -----------------------------------
	// CREATE FREELANCER TRANSACTION
	// -----------------------------------
	freelancerTx := domain.WalletTransaction{
		WalletID: freelancerWallet.ID,

		// TxRef: fmt.Sprintf(
		// 	"WEEKLY-PAY-%d-%d-%d-freelancer",
		// 	request.ContractID,
		// 	request.Year,
		// 	request.WeekNumber,
		// ),
		TxRef: fmt.Sprintf(
			"WEEKLY-PAY-%d-%d-%d-%d-freelancer",
			request.ContractID,
			request.Year,
			request.WeekNumber,
			time.Now().UnixNano(),
		),

		Type:        domain.TxPayment,
		Status:      domain.TxSuccess,
		AmountMinor: totalAmountMinor,
		Description: fmt.Sprintf(
			"Received weekly payment for contract #%d week %d",
			request.ContractID,
			request.WeekNumber,
		),
		Provider: "INTERNAL",
	}

	if err := tx.Create(&freelancerTx).Error; err != nil {
		tx.Rollback()
		return err
	}

	// -----------------------------------
	// COMMIT
	// -----------------------------------
	if err := tx.Commit().Error; err != nil {
		return err
	}

	// -----------------------------------
	// CREATE NOTIFICATION FOR FREELANCER
	// -----------------------------------
	freelancerNotif := domain.Notification{
		UserID: contract.FreelancerID,

		Type: domain.NotifyWeeklyPaymentRelased,

		Title: "Weekly Payment Received",

		Message: fmt.Sprintf(
			"You received payment for week %d on contract '%s'.",
			request.WeekNumber,
			contract.Title,
		),

		ContractID: &contract.ID,
		ProposalID: &contract.ProposalID,
		JobID:      &contract.JobID,

		IsRead: false,
	}

	_ = r.notificationRepo.CreateNotification(
		&freelancerNotif,
	)

	// -----------------------------------
	// CREATE NOTIFICATION FOR CLIENT
	// -----------------------------------
	clientNotif := domain.Notification{
		UserID: contract.ClientID,

		Type: domain.NotifyWeeklyPaymentRelased,

		Title: "Weekly Payment Sent",

		Message: fmt.Sprintf(
			"You successfully paid week %d for contract '%s'.",
			request.WeekNumber,
			contract.Title,
		),

		ContractID: &contract.ID,
		ProposalID: &contract.ProposalID,
		JobID:      &contract.JobID,

		IsRead: false,
	}

	_ = r.notificationRepo.CreateNotification(
		&clientNotif,
	)

	return nil
}

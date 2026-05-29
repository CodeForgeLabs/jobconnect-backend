package repository

import (
	"context"
	"fmt"
	"job-connect/chapa"
	"job-connect/domain"
	"time"

	"gorm.io/gorm"
)

type walletRepo struct {
	db               *gorm.DB
	notificationRepo domain.NotificationRepository
	chapaClient      *chapa.Client
}

func NewWalletRepo(db *gorm.DB, notificationRepo domain.NotificationRepository, chapaClient *chapa.Client) *walletRepo {
	return &walletRepo{db: db, notificationRepo: notificationRepo, chapaClient: chapaClient}
}

func (r *walletRepo) GetOrCreate(userID uint) (domain.Wallet, error) {
	var wallet domain.Wallet

	err := r.db.Where("user_id = ?", userID).First(&wallet).Error
	if err == nil {
		return wallet, nil
	}

	if err == gorm.ErrRecordNotFound {
		wallet = domain.Wallet{
			UserID:       userID,
			BalanceMinor: 0,
			Currency:     "ETB",
		}
		err = r.db.Create(&wallet).Error
	}

	return wallet, err
}

func (r *walletRepo) CreateTransaction(tx domain.WalletTransaction) (domain.WalletTransaction, error) {
	tx.TxRef = generateTxRef(tx.WalletID)
	err := r.db.Create(&tx).Error
	return tx, err
}

func (r *walletRepo) UpdateTransactionStatus(txRef string, status domain.TransactionStatus, externalRef string) error {
	err := r.db.
		Model(&domain.WalletTransaction{}).
		Where("tx_ref = ?", txRef).
		Updates(map[string]interface{}{
			"status":       status,
			"external_ref": externalRef,
		}).Error
	if err != nil {
		return err
	}
	if status == domain.TxSuccess {
		var tx domain.WalletTransaction
		if err := r.db.Where("tx_ref = ?", txRef).First(&tx).Error; err != nil {
			return err
		}
		println("*********************************")
		println(tx.AmountMinor)
		var wallet domain.Wallet
		if err := r.db.First(&wallet, tx.WalletID).Error; err != nil {
			return err
		}

		// var balanceUpdate int64
		// if tx.Type == domain.TxDeposit {
		// 	balanceUpdate = tx.AmountMinor
		// } else if tx.Type == domain.TxPayment || tx.Type == domain.TxWithdraw {
		// 	balanceUpdate = -tx.AmountMinor
		// }

		return r.db.Model(&domain.Wallet{}).
			Where("id = ?", wallet.ID).
			UpdateColumn(
				"balance_minor",
				gorm.Expr("balance_minor + ?", tx.AmountMinor),
			).Error
	}

	return nil
}

func (r *walletRepo) GetTransactionByTxRef(txRef string) (domain.WalletTransaction, error) {
	var tx domain.WalletTransaction
	err := r.db.Where("tx_ref = ?", txRef).First(&tx).Error
	return tx, err
}
func generateTxRef(walletID uint) string {
	return fmt.Sprintf(
		"wallet-%d-%d",
		walletID,
		time.Now().UnixNano(),
	)
}

func (r *walletRepo) FetchTransactionsByWalletID(walletID uint) ([]domain.WalletTransaction, error) {
	var txs []domain.WalletTransaction
	err := r.db.Where("wallet_id = ?", walletID).Order("created_at desc").Find(&txs).Error
	return txs, err
}

func (r *walletRepo) BuyConnect(amount int, userId uint) (bool, error) {
	tx := r.db.Begin()

	// Find wallet
	var wallet domain.Wallet
	if err := tx.Where("user_id = ?", userId).First(&wallet).Error; err != nil {
		tx.Rollback()
		return false, err
	}

	// Calculate total cost (1 connect = 10 birr -> 1000 minor units if 10 birr is 1000 cents)
	// Adjust this multiplier based on whether your '10' is already in Birr or Cents/Minor units.
	totalCost := amount * 10

	// Check balance
	if int(wallet.BalanceMinor) < totalCost {
		tx.Rollback()
		return false, fmt.Errorf("insufficient wallet balance")
	}

	// Deduct wallet balance
	if err := tx.Model(&wallet).
		Update("balance_minor", int(wallet.BalanceMinor)-totalCost).Error; err != nil {
		tx.Rollback()
		return false, err
	}

	// ==========================================
	// NEW: RECORD WALLET TRANSACTION
	// ==========================================
	txRef := fmt.Sprintf("TX-CONN-%d-%d", userId, time.Now().UnixNano())

	transaction := domain.WalletTransaction{
		WalletID:    wallet.ID,
		TxRef:       txRef,
		Type:        "DEBIT",      // Or your domain.TransactionTypeDebit enum
		Status:      "SUCCESSFUL", // Or your domain.TransactionStatusSuccess enum
		AmountMinor: int64(totalCost),
		Description: fmt.Sprintf("Purchased %d connects", amount),
		Provider:    "INTERNAL", // Internal wallet exchange, not Chapa
		ExternalRef: "",         // No external reference needed
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return false, fmt.Errorf("failed to record wallet transaction: %w", err)
	}

	// Update user connect field
	if err := tx.Model(&domain.User{}).
		Where("id = ?", userId).
		Update("connect", gorm.Expr("connect + ?", amount)).Error; err != nil {
		tx.Rollback()
		return false, err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return false, err
	}

	// ==========================================
	// NOTIFICATIONS (AFTER COMMIT)
	// ==========================================

	// 1. Save persistent database notification
	notif := domain.Notification{
		UserID:  userId,
		Type:    domain.NotifyConnectsPurchased, // "CONNECTS_PURCHASED"
		Title:   "Connects Purchased Successfully",
		Message: fmt.Sprintf("You have successfully purchased %d connects for %d minor units!", amount, totalCost),
		IsRead:  false,
	}
	_ = r.notificationRepo.CreateNotification(&notif)

	return true, nil
}

func (r *walletRepo) WithdrawBalance(request domain.TransferRequest, ctx context.Context) (bool, error) {
	// first check the amount in the wallet, if it is there reduce it
	// then add the transaction to the wallet transaction table
	/*

				type Wallet struct {
					ID uint `gorm:"primaryKey"`

					UserID uint `gorm:"uniqueIndex;not null"`

					BalanceMinor int64 `gorm:"default:0"` // store in cents/birr minor unit

					Currency string `gorm:"type:varchar(10);default:'ETB'"`

					CreatedAt time.Time
					UpdatedAt time.Time
				}

				type TransactionType string
				type TransactionStatus string

				const (
					TxDeposit  TransactionType = "DEPOSIT"
					TxPayment  TransactionType = "PAYMENT"
					TxWithdraw TransactionType = "WITHDRAW"
					TxEscrow   TransactionType = "ESCROW"
				)

				const (
					TxPending TransactionStatus = "PENDING"
					TxSuccess TransactionStatus = "SUCCESS"
					TxFailed  TransactionStatus = "FAILED"
				)

				type WalletTransaction struct {
					ID uint `gorm:"primaryKey"`

					WalletID uint `gorm:"index;not null"`

					TxRef string `gorm:"uniqueIndex;not null"`

					Type   TransactionType   `gorm:"type:varchar(20)"`
					Status TransactionStatus `gorm:"type:varchar(20);default:'PENDING'"`

					AmountMinor int64 `gorm:"not null"` // always minor unit

					Description string

					Provider string `gorm:"type:varchar(20)"` // CHAPA

					ExternalRef string `gorm:"type:varchar(100)"`

					CreatedAt time.Time
					UpdatedAt time.Time
				}
			type TransferRequest struct {
			Amount    string `json:"amount"`
			Currency  string `json:"currency"`
			BankCode  string `json:"bank_code"`
			AccountNo string `json:"account_number"`
			UserId    uint   `json:"user_id"`
		}

	*/

	r.db.Transaction(func(tx *gorm.DB) error {
		var wallet domain.Wallet
		if err := tx.Where("user_id = ?", request.UserId).First(&wallet).Error; err != nil {
			return err
		}

		amountMinor, err := parseAmountToMinor(request.Amount)
		if err != nil {
			return fmt.Errorf("invalid amount format: %w", err)
		}

		if wallet.BalanceMinor < amountMinor {
			return fmt.Errorf("insufficient balance")
		}

		// Deduct balance immediately to prevent double spending
		if err := tx.Model(&wallet).
			Update("balance_minor", wallet.BalanceMinor-amountMinor).Error; err != nil {
			return err
		}

		walletTx := domain.WalletTransaction{
			WalletID:    wallet.ID,
			TxRef:       fmt.Sprintf("TX-WITHDRAW-%d-%d", request.UserId, time.Now().UnixNano()),
			Type:        domain.TxWithdraw,
			Status:      domain.TxPending,
			AmountMinor: amountMinor,
			Description: fmt.Sprintf("Withdrawal to %s %s", request.BankCode, request.AccountNo),
			Provider:    "CHAPA",
		}

		if err := tx.Create(&walletTx).Error; err != nil {
			return err
		}

		// Initiate transfer with Chapa
		success, err := r.chapaClient.InitiateTransfer(ctx, chapa.TransferRequest{
			Amount:    request.Amount,
			Currency:  request.Currency,
			BankCode:  request.BankCode,
			AccountNo: request.AccountNo,
		})
		fmt.Println("**************************************************")
		fmt.Println(success)
		fmt.Println(err)
		if err != nil || success != "" {
			tx.Model(&walletTx).Updates(map[string]interface{}{
				"status": domain.TxFailed,
			})
			tx.Model(&wallet).Update("balance_minor", wallet.BalanceMinor) // refund balance
		}

		tx.Model(&walletTx).Updates(map[string]interface{}{
			"status": domain.TxSuccess,
		})

		return nil
	})

	return true, nil

}

func parseAmountToMinor(amountStr string) (int64, error) {
	var major float64
	_, err := fmt.Sscanf(amountStr, "%f", &major)
	if err != nil {
		return 0, err
	}
	return int64(major), nil // Convert to minor units (e.g., cents)
}

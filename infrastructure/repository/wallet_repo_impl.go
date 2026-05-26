package repository

import (
	"fmt"
	"job-connect/domain"
	"time"

	"gorm.io/gorm"
)

type walletRepo struct {
	db               *gorm.DB
	notificationRepo domain.NotificationRepository
}

func NewWalletRepo(db *gorm.DB, notificationRepo domain.NotificationRepository) *walletRepo {
	return &walletRepo{db: db, notificationRepo: notificationRepo}
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

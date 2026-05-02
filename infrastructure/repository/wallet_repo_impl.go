package repository

import (
	"fmt"
	"job-connect/domain"
	"time"

	"gorm.io/gorm"
)

type walletRepo struct {
	db *gorm.DB
}

func NewWalletRepo(db *gorm.DB) *walletRepo {
	return &walletRepo{db: db}
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

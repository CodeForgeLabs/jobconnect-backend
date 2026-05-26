package domain

import "time"

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

type WalletRepository interface {
	GetOrCreate(userID uint) (Wallet, error)

	CreateTransaction(tx WalletTransaction) (WalletTransaction, error)

	UpdateTransactionStatus(txRef string, status TransactionStatus, externalRef string) error
	GetTransactionByTxRef(txRef string) (WalletTransaction, error)
	FetchTransactionsByWalletID(walletID uint) ([]WalletTransaction, error)
	BuyConnect(amount int, userId uint) (bool, error)
}

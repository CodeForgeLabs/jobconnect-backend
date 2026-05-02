package usecase

import "job-connect/domain"

type WalletUsecase struct {
	walletRepo domain.WalletRepository
}

func NewWalletUsecase(walletRepo domain.WalletRepository) *WalletUsecase {
	return &WalletUsecase{walletRepo: walletRepo}
}

func (w *WalletUsecase) GetOrCreateWallet(userID uint) (domain.Wallet, error) {
	return w.walletRepo.GetOrCreate(userID)
}

func (w *WalletUsecase) CreateTransaction(tx domain.WalletTransaction) (domain.WalletTransaction, error) {

	return w.walletRepo.CreateTransaction(tx)
}

func (w *WalletUsecase) UpdateTransactionStatus(txRef string, status domain.TransactionStatus, externalRef string) error {
	return w.walletRepo.UpdateTransactionStatus(txRef, status, externalRef)
}

func (w *WalletUsecase) GetTransactionByTxRef(txRef string) (domain.WalletTransaction, error) {
	return w.walletRepo.GetTransactionByTxRef(txRef)
}

func (w *WalletUsecase) FetchTransactionsByWalletID(walletID uint) ([]domain.WalletTransaction, error) {
	return w.walletRepo.FetchTransactionsByWalletID(walletID)
}

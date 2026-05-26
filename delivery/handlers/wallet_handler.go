package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"job-connect/auth"
	"job-connect/chapa"
	domain "job-connect/domain"
	"job-connect/usecase"
	"net/http"
)

type WalletHandler struct {
	walletUsecase *usecase.WalletUsecase
	chapaClient   *chapa.Client
}
type BuyConnectInput struct {
	Amount int `json:"amount"`
}

func NewWalletHandler(walletUsecase *usecase.WalletUsecase, chapaClient *chapa.Client) *WalletHandler {
	return &WalletHandler{
		walletUsecase: walletUsecase,
		chapaClient:   chapaClient,
	}
}

type CreateDepositInput struct {
	UserID      uint
	AmountMinor int64
	Description string
	Email       string
	Phone       string
}

// GetOrCreateWallet godoc
// @Summary Get or create a wallet for the authenticated user
// @Description Retrieve the wallet for the authenticated user, or create one if it doesn't exist
// @Tags Wallet
// @Accept json
// @Produce json
// @Success 200 {object} domain.GenericResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /wallet/balance [get]
func (h *WalletHandler) GetOrCreateWallet(w http.ResponseWriter, r *http.Request) {
	userId, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	wallet, err := h.walletUsecase.GetOrCreateWallet(parseUint(userId))
	if err != nil {
		http.Error(w, "Failed to get or create wallet", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"wallet": wallet,
	})
}

// CreateTransaction godoc
// @Summary Create a new wallet transaction and initialize payment
// @Description Create a new wallet transaction for the authenticated user and initialize payment with Chapa
// @Tags Wallet
// @Accept json
// @Produce json
// @Param input body CreateDepositInput true "Create Deposit Input"
// @Success 200 {object} domain.GenericResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /wallet/transaction [post]
func (h *WalletHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var input CreateDepositInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	userId, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	wallet, err := h.walletUsecase.GetOrCreateWallet(parseUint(userId))
	if err != nil {
		http.Error(w, "Failed to get or create wallet", http.StatusInternalServerError)
		return
	}
	tx, err := h.walletUsecase.CreateTransaction(domain.WalletTransaction{
		WalletID:    wallet.ID,
		AmountMinor: input.AmountMinor,
		Description: input.Description,
		Status:      domain.TxPending,
	})
	if err != nil {
		http.Error(w, "Failed to create transaction", http.StatusInternalServerError)
		return
	}

	paymentURL, err := h.chapaClient.InitializePayment(r.Context(), chapa.PaymentRequest{
		Amount:      float64(input.AmountMinor),
		TxRef:       tx.TxRef,
		Description: input.Description,
		CallbackURL: "https://jobconnect-backend-4qq7.onrender.com/api/v1/wallet/transaction/update",
		// ReturnURL:   "https://mezgebesibhat.vercel.app/thank-you",
	})
	if err != nil {
		http.Error(w, "Failed to initialize payment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if paymentURL == "" {
		http.Error(w, "empty payment url from chapa", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"payment_url": paymentURL,
	})
}

func (h *WalletHandler) UpdateTransactionStatus(w http.ResponseWriter, r *http.Request) {
	println("**********************************************************")
	println("🔥 CHAPA CALLBACK HIT")
	println("**********************************************************")

	// Read full raw body first
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	// Debug everything
	fmt.Println("METHOD:", r.Method)
	fmt.Println("URL:", r.URL.String())
	fmt.Println("HEADERS:", r.Header)
	fmt.Println("RAW BODY:", string(bodyBytes))

	// Reset body so JSON decoder can read it again
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	type ChapaCallback struct {
		TxRef     string `json:"trx_ref"`
		Status    string `json:"status"`
		Reference string `json:"ref_id"`
		Amount    string `json:"amount"`
		Currency  string `json:"currency"`
	}

	var payload ChapaCallback

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	println("Received Chapa callback:", payload.TxRef, payload.Status, payload.Reference)
	// 1️⃣ find transaction
	tx, err := h.walletUsecase.GetTransactionByTxRef(payload.TxRef)
	if err != nil {
		http.Error(w, "transaction not found", http.StatusNotFound)
		return
	}

	// 2️⃣ prevent double processing (idempotency)
	if tx.Status == domain.TxSuccess {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("already processed"))
		return
	}

	// 3️⃣ update status based on chapa
	if payload.Status == "success" {

		// mark transaction success
		err = h.walletUsecase.UpdateTransactionStatus(
			payload.TxRef,
			domain.TxSuccess,
			payload.Reference,
		)
		if err != nil {
			http.Error(w, "failed updating transaction", http.StatusInternalServerError)
			return
		}

	} else {
		// mark failed
		h.walletUsecase.UpdateTransactionStatus(
			payload.TxRef,
			domain.TxFailed,
			payload.Reference,
		)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// FetchTransactions godoc
// @Summary Fetch wallet transactions for the authenticated user
// @Description Retrieve all wallet transactions for the authenticated user's wallet
// @Tags Wallet
// @Accept json
// @Produce json
// @Success 200 {object} domain.GenericResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /wallet/transactions [get]
func (h *WalletHandler) FetchTransactions(w http.ResponseWriter, r *http.Request) {
	userId, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	wallet, err := h.walletUsecase.GetOrCreateWallet(parseUint(userId))
	if err != nil {
		http.Error(w, "Failed to get or create wallet", http.StatusInternalServerError)
		return
	}
	txs, err := h.walletUsecase.FetchTransactionsByWalletID(wallet.ID)
	if err != nil {
		http.Error(w, "Failed to fetch transactions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transactions": txs,
	})
}

// BuyConnect godoc
// @Summary Buy Connects for the authenticated user
// @Description Purchase Connects for the authenticated user by charging their wallet
// @Tags Wallet
// @Accept json
// @Produce json
// @Param input body BuyConnectInput true "Buy Connect Input"
// @Success 200 {object} domain.GenericResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /wallet/buy-connect [post]
func (h *WalletHandler) BuyConnect(w http.ResponseWriter, r *http.Request) {

	var input BuyConnectInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	userId, _, err := auth.GetUserFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	success, err := h.walletUsecase.BuyConnect(input.Amount, parseUint(userId))
	if err != nil || !success {
		http.Error(w, "Failed to buy connect", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Connect purchased successfully",
	})

}

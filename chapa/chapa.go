package chapa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	SecretKey string
	BaseURL   string
}

func NewClient(secretKey, baseURL string) *Client {
	return &Client{
		SecretKey: secretKey,
		BaseURL:   baseURL,
	}
}

type chapaInitResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		CheckoutURL string `json:"checkout_url"`
	} `json:"data"`
}

type PaymentRequest struct {
	Amount      float64
	TxRef       string
	Description string
	CallbackURL string
	ReturnURL   string
}

func (c *Client) InitializePayment(ctx context.Context, req PaymentRequest) (string, error) {

	payload := map[string]any{
		"amount":       fmt.Sprintf("%.2f", req.Amount),
		"currency":     "ETB",
		"tx_ref":       req.TxRef,
		"callback_url": req.CallbackURL,
		"return_url":   req.ReturnURL,
	}

	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(ctx,
		"POST",
		c.BaseURL+"/v1/transaction/initialize",
		bytes.NewBuffer(body),
	)
	if err != nil {
		println(err.Error())
		return "", err
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.SecretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		println(err.Error())
		return "", err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		println(resp.StatusCode)
		return "", fmt.Errorf("chapa http error: %d", resp.StatusCode)
	}

	fmt.Println("STATUS CODE:", resp.StatusCode)
	fmt.Println("RESPONSE BODY:", string(bodyBytes))

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("chapa http error: %d", resp.StatusCode)
	}

	var result chapaInitResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", err
	}

	if result.Status != "success" {
		return "", fmt.Errorf("chapa failed: %s", result.Message)
	}

	return result.Data.CheckoutURL, nil
}

type TransferRequest struct {
	Amount    string
	Currency  string
	BankCode  string
	AccountNo string
}
type TransferResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Reference string `json:"reference"`
	} `json:"data"`
}

func (c *Client) InitiateTransfer(ctx context.Context, req TransferRequest) (string, error) {
	payload := map[string]any{
		"amount":         req.Amount,
		"currency":       req.Currency,
		"bank_code":      req.BankCode,
		"account_number": req.AccountNo,
	}

	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.BaseURL+"/v1/transfers",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.SecretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return "", fmt.Errorf("chapa transfer error: %s", string(respBody))
	}

	var result TransferResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	fmt.Println("*****************")
	fmt.Println(result)
	fmt.Println("***********")
	if result.Status != "success" {
		return "", fmt.Errorf("transfer failed: %s", result.Message)
	}

	return result.Data.Reference, nil
}

func (c *Client) GetBanks(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(
		ctx,
		"GET",
		c.BaseURL+"/v1/banks",
		nil,
	)

	req.Header.Set("Authorization", "Bearer "+c.SecretKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))

	return nil
}

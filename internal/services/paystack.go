package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tehilla-22/b2b-api/internal/config"
)

type PaystackService struct {
	cfg *config.Config
	client *http.Client
}

type PaystackInitializeRequest struct {
	Email       string                 `json:"email"`
	Amount      int64                  `json:"amount"`
	Currency    string                 `json:"currency"`
	Reference   string                 `json:"reference"`
	CallbackURL string                 `json:"callback_url"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type PaystackInitializeResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		AuthorizationURL string `json:"authorization_url"`
		AccessCode       string `json:"access_code"`
		Reference        string `json:"reference"`
	} `json:"data"`
}

type PaystackVerifyResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Status   string `json:"status"`
		Amount   float64 `json:"amount"`
		Currency string `json:"currency"`
		Channel  string `json:"channel"`
		PaidAt   string `json:"paid_at"`
		Metadata map[string]interface{} `json:"metadata"`
	} `json:"data"`
}

func NewPaystackService(cfg *config.Config) *PaystackService {
	return &PaystackService{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *PaystackService) InitializeTransaction(email string, amount float64, currency string, reference string, metadata map[string]interface{}) (*PaystackInitializeResponse, error) {
	if s.cfg.PaystackSecretKey == "" {
		return nil, errors.New("paystack not configured")
	}

	body := PaystackInitializeRequest{
		Email:       email,
		Amount:      int64(amount * 100),
		Currency:    currency,
		Reference:   reference,
		CallbackURL: s.cfg.PaystackCallback,
		Metadata:    metadata,
	}

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "https://api.paystack.co/transaction/initialize", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.cfg.PaystackSecretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("paystack request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result PaystackInitializeResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("paystack parse error: %w", err)
	}

	if !result.Status {
		return nil, fmt.Errorf("paystack error: %s", result.Message)
	}

	return &result, nil
}

func (s *PaystackService) VerifyTransaction(reference string) (*PaystackVerifyResponse, error) {
	if s.cfg.PaystackSecretKey == "" {
		return nil, errors.New("paystack not configured")
	}

	req, _ := http.NewRequest("GET", fmt.Sprintf("https://api.paystack.co/transaction/verify/%s", reference), nil)
	req.Header.Set("Authorization", "Bearer "+s.cfg.PaystackSecretKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("paystack verify request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result PaystackVerifyResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("paystack parse error: %w", err)
	}

	if !result.Status {
		return nil, fmt.Errorf("paystack error: %s", result.Message)
	}

	return &result, nil
}

func (s *PaystackService) VerifyWebhookSignature(signature string, body []byte) bool {
	if signature == "" || s.cfg.PaystackSecretKey == "" {
		return false
	}
	mac := hmac.New(sha512.New, []byte(s.cfg.PaystackSecretKey))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (s *PaystackService) Transfer(amount float64, recipient string, reference string) error {
	if s.cfg.PaystackSecretKey == "" {
		return errors.New("paystack not configured")
	}

	payload := map[string]interface{}{
		"source":    "balance",
		"amount":    int64(amount * 100),
		"recipient": recipient,
		"reference": reference,
	}

	jsonBody, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.paystack.co/transfer", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.cfg.PaystackSecretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("paystack transfer failed: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

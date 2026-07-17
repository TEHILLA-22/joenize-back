package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/tehilla-22/b2b-api/internal/config"
	"github.com/tehilla-22/b2b-api/internal/database"
	"github.com/tehilla-22/b2b-api/internal/middleware"
	"github.com/tehilla-22/b2b-api/internal/models"
	"github.com/tehilla-22/b2b-api/internal/services"
	"github.com/tehilla-22/b2b-api/internal/utils"
)

type PaymentHandler struct {
	paystack *services.PaystackService
	cfg      *config.Config
}

func NewPaymentHandler(cfg *config.Config, paystack *services.PaystackService) *PaymentHandler {
	return &PaymentHandler{cfg: cfg, paystack: paystack}
}

func (h *PaymentHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var wallet models.Wallet
	err := database.DB.First(&wallet, "user_id = ?", userID).Error
	if err != nil {
		wallet = models.Wallet{
			UserID:   uuid.MustParse(userID),
			Balance:  0,
			Currency: "USD",
		}
		database.DB.Create(&wallet)
	}

	utils.JSON(w, http.StatusOK, wallet)
}

func (h *PaymentHandler) InitializePayment(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var body struct {
		OrderID string  `json:"order_id"`
		Amount  float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "User not found")
		return
	}

	reference := uuid.New().String()

	metadata := map[string]interface{}{
		"user_id":  userID,
		"order_id": body.OrderID,
		"type":     "order_payment",
	}

	result, err := h.paystack.InitializeTransaction(user.Email, body.Amount, "NGN", reference, metadata)
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadGateway, "Payment initialization failed: "+err.Error())
		return
	}

	payment := models.Payment{
		UserID:    uuid.MustParse(userID),
		Reference: reference,
		Amount:    body.Amount,
		Status:    "pending",
	}
	if body.OrderID != "" {
		oid, _ := uuid.Parse(body.OrderID)
		payment.OrderID = &oid
	}
	database.DB.Create(&payment)

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"authorization_url": result.Data.AuthorizationURL,
		"reference":         reference,
	})
}

func (h *PaymentHandler) VerifyPayment(w http.ResponseWriter, r *http.Request) {
	reference := r.URL.Query().Get("reference")
	if reference == "" {
		utils.ErrorJSON(w, http.StatusBadRequest, "Reference is required")
		return
	}

	result, err := h.paystack.VerifyTransaction(reference)
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadGateway, "Verification failed: "+err.Error())
		return
	}

	var payment models.Payment
	if err := database.DB.First(&payment, "reference = ?", reference).Error; err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "Payment not found")
		return
	}

	if result.Data.Status == "success" {
		payment.Status = "success"
		payment.Channel = result.Data.Channel

		var wallet models.Wallet
		if err := database.DB.First(&wallet, "user_id = ?", payment.UserID).Error; err == nil {
			wallet.Balance += payment.Amount
			database.DB.Save(&wallet)
		}

		if payment.OrderID != nil {
			database.DB.Model(&models.Order{}).Where("id = ?", payment.OrderID).Update("status", "paid")
			database.DB.Model(&models.Invoice{}).Where("order_id = ?", payment.OrderID).Update("status", "paid")
		}
	} else {
		payment.Status = "failed"
	}

	database.DB.Save(&payment)
	utils.JSON(w, http.StatusOK, payment)
}

func (h *PaymentHandler) PaystackWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	if !h.paystack.VerifyWebhookSignature(r.Header.Get("x-paystack-signature"), body) {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Invalid signature")
		return
	}

	var event struct {
		Event string `json:"event"`
		Data  struct {
			Reference string `json:"reference"`
			Status    string `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&event); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	if event.Event == "charge.success" {
		database.DB.Model(&models.Payment{}).
			Where("reference = ?", event.Data.Reference).
			Update("status", "success")
	}

	w.WriteHeader(http.StatusOK)
}

package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/tehilla-22/b2b-api/internal/config"
	"github.com/tehilla-22/b2b-api/internal/database"
	"github.com/tehilla-22/b2b-api/internal/middleware"
	"github.com/tehilla-22/b2b-api/internal/models"
	"github.com/tehilla-22/b2b-api/internal/services"
	"github.com/tehilla-22/b2b-api/internal/utils"
)

type SellerHandler struct {
	paystack       *services.PaystackService
	emailService   *services.EmailService
	cfg            *config.Config
	notificationHandler *NotificationHandler
}

func NewSellerHandler(cfg *config.Config, paystack *services.PaystackService, emailService *services.EmailService, notifHandler *NotificationHandler) *SellerHandler {
	return &SellerHandler{
		cfg:                 cfg,
		paystack:            paystack,
		emailService:        emailService,
		notificationHandler: notifHandler,
	}
}

type SellerOnboardingInput struct {
	Amount float64 `json:"amount"`
}

type SellerOnboardingResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	Reference        string `json:"reference"`
}

func (h *SellerHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var input struct {
		BusinessName    string `json:"business_name"`
		BusinessType    string `json:"business_type"`
		BusinessAddress string `json:"business_address"`
		TaxID           string `json:"tax_id"`
		PhoneNumber     string `json:"phone_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if input.BusinessName != "" {
		updates["business_name"] = input.BusinessName
	}
	if input.BusinessType != "" {
		updates["business_type"] = input.BusinessType
	}
	if input.BusinessAddress != "" {
		updates["business_address"] = input.BusinessAddress
	}
	if input.TaxID != "" {
		updates["tax_id"] = input.TaxID
	}
	if input.PhoneNumber != "" {
		updates["phone_number"] = input.PhoneNumber
	}

	if err := database.DB.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Could not update profile")
		return
	}

	var user models.User
	database.DB.First(&user, "id = ?", userID)
	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"detail": "Profile updated",
		"user":   user,
	})
}

func (h *SellerHandler) InitializeOnboarding(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "User not found")
		return
	}

	if user.IsSeller {
		utils.ErrorJSON(w, http.StatusBadRequest, "Already a seller")
		return
	}

	var input SellerOnboardingInput
	if r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
			return
		}
	}

	input.Amount = 8000.00

	reference := "SELLER-" + uuid.New().String()

	metadata := map[string]interface{}{
		"user_id": userID,
		"type":    "seller_onboarding",
	}

	result, err := h.paystack.InitializeTransaction(user.Email, input.Amount, "NGN", reference, metadata)
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadGateway, "Payment initialization failed: "+err.Error())
		return
	}

	payment := models.Payment{
		UserID:    uuid.MustParse(userID),
		Reference: reference,
		Amount:    input.Amount,
		Status:    "pending",
		Metadata:  `{"type": "seller_onboarding"}`,
	}
	database.DB.Create(&payment)

	utils.JSON(w, http.StatusOK, SellerOnboardingResponse{
		AuthorizationURL: result.Data.AuthorizationURL,
		Reference:        reference,
	})
}

func (h *SellerHandler) VerifyOnboarding(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	reference := r.URL.Query().Get("reference")
	if reference == "" {
		utils.ErrorJSON(w, http.StatusBadRequest, "Reference is required")
		return
	}

	result, err := h.paystack.VerifyTransaction(reference)
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadGateway, "Verification failed")
		return
	}

	var payment models.Payment
	if err := database.DB.First(&payment, "reference = ?", reference).Error; err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "Payment not found")
		return
	}

	if result.Data.Status != "success" {
		payment.Status = "failed"
		database.DB.Save(&payment)
		utils.ErrorJSON(w, http.StatusBadRequest, "Payment was not successful")
		return
	}

	payment.Status = "success"
	payment.Channel = result.Data.Channel
	now := time.Now()
	payment.PaidAt = &now
	database.DB.Save(&payment)

	database.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"is_seller":  true,
		"seller_paid": true,
	})

	h.notificationHandler.Notify(userID, "Seller onboarding complete", "Your seller onboarding payment has been confirmed. You can now create products and manage your storefront.", "success")

	var userForEmail models.User
	database.DB.Select("email").First(&userForEmail, "id = ?", userID)
	go h.emailService.SendOnboardingConfirmation(userForEmail.Email)

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"detail":    "Seller onboarding complete",
		"is_seller": true,
	})
}

func (h *SellerHandler) DashboardSummary(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var totalProducts int64
	database.DB.Model(&models.Product{}).Where("seller_id = ?", userID).Count(&totalProducts)

	var totalOrders int64
	database.DB.Model(&models.Order{}).Where("seller_id = ?", userID).Count(&totalOrders)

	var pendingOrders int64
	database.DB.Model(&models.Order{}).Where("seller_id = ? AND status IN ?", userID, []string{"pending", "confirmed", "processing"}).Count(&pendingOrders)

	var shippedOrders int64
	database.DB.Model(&models.Shipment{}).Where("seller_id = ? AND status NOT IN ?", userID, []string{"delivered", "cancelled"}).Count(&shippedOrders)

	var totalRevenue float64
	database.DB.Model(&models.Order{}).Where("seller_id = ? AND status = ?", userID, "completed").Select("COALESCE(SUM(total_amount), 0)").Scan(&totalRevenue)

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"total_products":   totalProducts,
		"total_orders":     totalOrders,
		"pending_orders":   pendingOrders,
		"active_shipments": shippedOrders,
		"total_revenue":    totalRevenue,
	})
}

func (h *SellerHandler) EscrowStatus(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var escrows []models.Escrow
	database.DB.Preload("Order").
		Where("seller_id = ?", userID).
		Order("created_at DESC").
		Find(&escrows)

	type EscrowStatus struct {
		ID          string  `json:"id"`
		OrderID     string  `json:"order_id"`
		OrderNumber string  `json:"order_number"`
		Amount      float64 `json:"amount"`
		Status      string  `json:"status"`
		CreatedAt   string  `json:"created_at"`
		ReleasedAt  *string `json:"released_at,omitempty"`
	}

	results := make([]EscrowStatus, 0)
	for _, e := range escrows {
		status := EscrowStatus{
			ID:          e.ID.String(),
			OrderID:     e.OrderID.String(),
			OrderNumber: e.Order.OrderNumber,
			Amount:      e.Amount,
			Status:      e.Status,
			CreatedAt:   e.CreatedAt.Format(time.RFC3339),
		}
		if e.ReleasedAt != nil {
			s := e.ReleasedAt.Format(time.RFC3339)
			status.ReleasedAt = &s
		}
		results = append(results, status)
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
		"count":   len(results),
	})
}

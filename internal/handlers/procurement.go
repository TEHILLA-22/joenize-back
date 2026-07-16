package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tehilla-22/b2b-api/internal/database"
	"github.com/tehilla-22/b2b-api/internal/middleware"
	"github.com/tehilla-22/b2b-api/internal/models"
	"github.com/tehilla-22/b2b-api/internal/utils"
	"gorm.io/gorm"
)

type ProcurementHandler struct{}

func NewProcurementHandler() *ProcurementHandler {
	return &ProcurementHandler{}
}

func (h *ProcurementHandler) ListRFQs(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	isSeller, _ := r.Context().Value(middleware.IsSellerKey).(bool)

	query := database.DB.Model(&models.RFQ{})
	if isSeller {
		query = query.Where("EXISTS (SELECT 1 FROM rfq_items WHERE rfq_items.rfq_id = rfqs.id AND rfq_items.product_id IN (SELECT id FROM products WHERE seller_id = ?))", userID)
	} else {
		query = query.Where("buyer_id = ?", userID)
	}

	var total int64
	query.Count(&total)

	var rfqs []models.RFQ
	query.Preload("Items").Preload("Quotes").Preload("Buyer", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, username, email")
	}).Order("created_at DESC").Find(&rfqs)

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"results": rfqs,
		"count":   int(total),
	})
}

func (h *ProcurementHandler) CreateRFQ(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var input struct {
		Title string `json:"title"`
		Notes string `json:"notes"`
		Items []struct {
			ProductID string `json:"product_id"`
			Quantity  int    `json:"quantity"`
			Notes     string `json:"notes"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	rfq := models.RFQ{
		BuyerID: uuid.MustParse(userID),
		Title:   input.Title,
		Notes:   input.Notes,
		Status:  "open",
	}

	for _, item := range input.Items {
		rfq.Items = append(rfq.Items, models.RFQItem{
			ProductID: uuid.MustParse(item.ProductID),
			Quantity:  item.Quantity,
			Notes:     item.Notes,
		})
	}

	database.DB.Create(&rfq)
	database.DB.Preload("Items").First(&rfq, rfq.ID)
	utils.JSON(w, http.StatusCreated, rfq)
}

func (h *ProcurementHandler) AddToCart(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var body struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if body.Quantity < 1 {
		body.Quantity = 1
	}

	var existing models.CartItem
	result := database.DB.Where("user_id = ? AND product_id = ?", userID, body.ProductID).First(&existing)
	if result.Error == nil {
		existing.Quantity += body.Quantity
		database.DB.Save(&existing)
		utils.JSON(w, http.StatusOK, existing)
		return
	}

	item := models.CartItem{
		UserID:    uuid.MustParse(userID),
		ProductID: uuid.MustParse(body.ProductID),
		Quantity:  body.Quantity,
	}
	database.DB.Create(&item)
	database.DB.Preload("Product").First(&item, item.ID)
	utils.JSON(w, http.StatusCreated, item)
}

func (h *ProcurementHandler) ListCart(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var items []models.CartItem
	database.DB.Preload("Product.Category").Preload("Product.Images").
		Where("user_id = ?", userID).Order("created_at DESC").Find(&items)

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"results": items,
		"count":   len(items),
	})
}

func (h *ProcurementHandler) RemoveFromCart(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	itemID := r.URL.Query().Get("item_id")

	result := database.DB.Where("id = ? AND user_id = ?", itemID, userID).Delete(&models.CartItem{})
	if result.RowsAffected == 0 {
		utils.ErrorJSON(w, http.StatusNotFound, "Cart item not found")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{"detail": "Item removed from cart"})
}

func (h *ProcurementHandler) CreateQuote(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var input struct {
		RFQID  string  `json:"rfq_id"`
		Amount float64 `json:"amount"`
		Notes  string  `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var rfq models.RFQ
	if err := database.DB.First(&rfq, "id = ?", input.RFQID).Error; err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "RFQ not found")
		return
	}

	quote := models.Quote{
		RFQID:    rfq.ID,
		SellerID: uuid.MustParse(userID),
		Amount:   input.Amount,
		Currency: "USD",
		Notes:    input.Notes,
		Status:   "pending",
	}
	database.DB.Create(&quote)
	database.DB.Preload("Seller", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, username, email")
	}).First(&quote, quote.ID)

	utils.JSON(w, http.StatusCreated, quote)
}

func (h *ProcurementHandler) RespondToQuote(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	quoteID := chi.URLParam(r, "id")

	var input struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var quote models.Quote
	if err := database.DB.Preload("RFQ").First(&quote, "id = ?", quoteID).Error; err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "Quote not found")
		return
	}

	if quote.RFQ.BuyerID != uuid.MustParse(userID) {
		utils.ErrorJSON(w, http.StatusForbidden, "Only the buyer can respond to this quote")
		return
	}

	if input.Action == "accept" {
		quote.Status = "accepted"
		database.DB.Save(&quote)
		database.DB.Model(&models.RFQ{}).Where("id = ?", quote.RFQID).Update("status", "accepted")
		utils.JSON(w, http.StatusOK, map[string]string{"detail": "Quote accepted"})
	} else if input.Action == "reject" {
		quote.Status = "rejected"
		database.DB.Save(&quote)
		utils.JSON(w, http.StatusOK, map[string]string{"detail": "Quote rejected"})
	} else {
		utils.ErrorJSON(w, http.StatusBadRequest, "Action must be 'accept' or 'reject'")
	}
}

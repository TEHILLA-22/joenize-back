package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

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
		query = query.Where(`
			is_private = false OR
			EXISTS (SELECT 1 FROM rfq_invitations WHERE rfq_invitations.rfq_id = rfqs.id AND rfq_invitations.supplier_id = ?)
		`, userID)
	} else {
		query = query.Where("buyer_id = ?", userID)
	}

	status := r.URL.Query().Get("status")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var rfqs []models.RFQ
	query.Preload("Items").Preload("Category").Preload("Buyer", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, username, email")
	}).Order("created_at DESC").Find(&rfqs)

	for i := range rfqs {
		filterQuotesForUser(&rfqs[i], userID)
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"results": rfqs,
		"count":   int(total),
	})
}

func (h *ProcurementHandler) ListRFQMarket(w http.ResponseWriter, r *http.Request) {
	query := database.DB.Model(&models.RFQ{}).Where("status = ? AND is_private = ?", "open", false)

	category := r.URL.Query().Get("category")
	if category != "" {
		query = query.Where("category_id = ?", category)
	}

	search := r.URL.Query().Get("search")
	if search != "" {
		query = query.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ?", "%"+strings.ToLower(search)+"%", "%"+strings.ToLower(search)+"%")
	}

	var total int64
	query.Count(&total)

	var rfqs []models.RFQ
	query.Preload("Category").Preload("Buyer", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, username, email")
	}).Order("created_at DESC").Limit(50).Find(&rfqs)

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"results": rfqs,
		"count":   int(total),
	})
}

func (h *ProcurementHandler) GetRFQ(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	rfqID := chi.URLParam(r, "id")

	var rfq models.RFQ
	if err := database.DB.Preload("Items").Preload("Category").Preload("Buyer", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, username, email")
	}).Preload("Images").Preload("Invitations.Supplier", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, username, email")
	}).First(&rfq, "id = ?", rfqID).Error; err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "RFQ not found")
		return
	}

	if rfq.BuyerID.String() != userID {
		if rfq.IsPrivate {
			var count int64
			database.DB.Model(&models.RFQInvitation{}).Where("rfq_id = ? AND supplier_id = ?", rfqID, userID).Count(&count)
			if count == 0 {
				utils.ErrorJSON(w, http.StatusNotFound, "RFQ not found")
				return
			}
		}

		database.DB.Where("rfq_id = ? AND seller_id = ?", rfqID, userID).Preload("Seller", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, username, email")
		}).Find(&rfq.Quotes)
	} else {
		database.DB.Where("rfq_id = ?", rfqID).Preload("Seller", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, username, email")
		}).Find(&rfq.Quotes)
	}

	if rfq.BuyerID.String() != userID {
		rfq.Invitations = nil
	}

	utils.JSON(w, http.StatusOK, rfq)
}

func (h *ProcurementHandler) CreateRFQ(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var input struct {
		Title             string   `json:"title"`
		Description       string   `json:"description"`
		CategoryID        string   `json:"category_id"`
		TargetPrice       float64  `json:"target_price"`
		Quantity          int      `json:"quantity"`
		Unit              string   `json:"unit"`
		Notes             string   `json:"notes"`
		IsPrivate         bool     `json:"is_private"`
		InvitedSupplierIDs []string `json:"invited_supplier_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	rfq := models.RFQ{
		BuyerID:     uuid.MustParse(userID),
		Title:       input.Title,
		Description: input.Description,
		TargetPrice: input.TargetPrice,
		Quantity:    input.Quantity,
		Unit:        input.Unit,
		Notes:       input.Notes,
		IsPrivate:   input.IsPrivate,
		Status:      "open",
	}

	if input.CategoryID != "" {
		cid, err := uuid.Parse(input.CategoryID)
		if err == nil {
			rfq.CategoryID = &cid
		}
	}

	if input.Quantity < 1 {
		rfq.Quantity = 1
	}

	database.DB.Create(&rfq)

	if input.IsPrivate && len(input.InvitedSupplierIDs) > 0 {
		for _, sid := range input.InvitedSupplierIDs {
			suuid, err := uuid.Parse(sid)
			if err != nil {
				continue
			}
			database.DB.Create(&models.RFQInvitation{
				RFQID:      rfq.ID,
				SupplierID: suuid,
			})
		}
	}

	database.DB.Preload("Category").Preload("Images").Preload("Invitations.Supplier", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, username, email")
	}).First(&rfq, rfq.ID)

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

	if rfq.IsPrivate {
		var count int64
		database.DB.Model(&models.RFQInvitation{}).Where("rfq_id = ? AND supplier_id = ?", input.RFQID, userID).Count(&count)
		if count == 0 {
			utils.ErrorJSON(w, http.StatusForbidden, "You are not invited to quote on this RFQ")
			return
		}
	}

	var existing int64
	database.DB.Model(&models.Quote{}).Where("rfq_id = ? AND seller_id = ?", input.RFQID, userID).Count(&existing)
	if existing > 0 {
		utils.ErrorJSON(w, http.StatusConflict, "You have already submitted a quote for this RFQ")
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

func (h *ProcurementHandler) ListSuppliers(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")

	query := database.DB.Model(&models.User{}).Where("is_seller = ?", true)
	if search != "" {
		query = query.Where("LOWER(username) LIKE ? OR LOWER(email) LIKE ? OR LOWER(business_name) LIKE ?",
			"%"+strings.ToLower(search)+"%", "%"+strings.ToLower(search)+"%", "%"+strings.ToLower(search)+"%")
	}

	var users []models.User
	query.Select("id, username, email, business_name").Limit(20).Find(&users)

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"results": users,
		"count":   len(users),
	})
}

func (h *ProcurementHandler) UploadRFQImage(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	rfqID := chi.URLParam(r, "id")

	var rfq models.RFQ
	if err := database.DB.First(&rfq, "id = ? AND buyer_id = ?", rfqID, userID).Error; err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "RFQ not found")
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "File too large")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "No image file provided")
		return
	}
	defer file.Close()

	url, err := uploadFile(file, header, "rfqs")
	if err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to upload image")
		return
	}

	image := models.RFQImage{
		RFQID: rfq.ID,
		URL:   url,
	}
	database.DB.Create(&image)

	database.DB.Preload("Images").First(&rfq, rfq.ID)
	utils.JSON(w, http.StatusCreated, rfq)
}

func filterQuotesForUser(rfq *models.RFQ, userID string) {
	if rfq.BuyerID.String() != userID {
		var filtered []models.Quote
		for _, q := range rfq.Quotes {
			if q.SellerID.String() == userID {
				filtered = append(filtered, q)
			}
		}
		rfq.Quotes = filtered
	}
}

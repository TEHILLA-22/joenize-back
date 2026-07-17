package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tehilla-22/b2b-api/internal/database"
	"github.com/tehilla-22/b2b-api/internal/middleware"
	"github.com/tehilla-22/b2b-api/internal/models"
	"github.com/tehilla-22/b2b-api/internal/utils"
	"gorm.io/gorm"
)

type OrderHandler struct {
	notifHandler *NotificationHandler
}

func NewOrderHandler(notifHandler *NotificationHandler) *OrderHandler {
	return &OrderHandler{notifHandler: notifHandler}
}

func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	isSeller, _ := r.Context().Value(middleware.IsSellerKey).(bool)

	query := database.DB.Model(&models.Order{})
	if isSeller {
		query = query.Where("seller_id = ?", userID)
	} else {
		query = query.Where("buyer_id = ?", userID)
	}

	status := r.URL.Query().Get("status")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 { page = 1 }
	if limit < 1 || limit > 100 { limit = 20 }
	offset := (page - 1) * limit

	var orders []models.Order
	query.Preload("Items").Preload("Invoice").Preload("Shipment").Preload("Escrow").
		Preload("Buyer", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, username, email")
		}).
		Offset(offset).Limit(limit).Order("created_at DESC").Find(&orders)

	count := int(total)
	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"results": orders,
		"count":   count,
		"page":    page,
		"total_pages": (count + limit - 1) / limit,
	})
}

func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	id := chi.URLParam(r, "id")

	var order models.Order
	err := database.DB.Preload("Items").Preload("Invoice").Preload("Shipment.Events").Preload("Escrow").Preload("Payments").
		First(&order, "id = ? AND (buyer_id = ? OR seller_id = ?)", id, userID, userID).Error
	if err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "Order not found")
		return
	}
	utils.JSON(w, http.StatusOK, order)
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var input struct {
		Items           []struct {
			ProductID string `json:"product_id"`
			Quantity  int    `json:"quantity"`
		} `json:"items"`
		ShippingAddress string `json:"shipping_address"`
		Notes           string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(input.Items) == 0 {
		utils.ErrorJSON(w, http.StatusBadRequest, "At least one item is required")
		return
	}

	var sellerID uuid.UUID
	var subtotal float64
	var orderItems []models.OrderItem

	for _, item := range input.Items {
		var product models.Product
		if err := database.DB.First(&product, "id = ?", item.ProductID).Error; err != nil {
			utils.ErrorJSON(w, http.StatusBadRequest, fmt.Sprintf("Product %s not found", item.ProductID))
			return
		}

		if sellerID == uuid.Nil {
			sellerID = product.SellerID
		} else if sellerID != product.SellerID {
			utils.ErrorJSON(w, http.StatusBadRequest, "All items must be from the same seller")
			return
		}

		total := product.Price * float64(item.Quantity)
		subtotal += total

		orderItems = append(orderItems, models.OrderItem{
			ProductID:  product.ID,
			Name:       product.Name,
			Quantity:   item.Quantity,
			UnitPrice:  product.Price,
			TotalPrice: total,
		})
	}

	orderNumber := fmt.Sprintf("ORD-%s-%d", time.Now().Format("20060102"), uuid.New().ID())
	totalAmount := subtotal

	order := models.Order{
		BuyerID:         uuid.MustParse(userID),
		SellerID:        sellerID,
		OrderNumber:     orderNumber,
		Status:          "pending",
		Subtotal:        subtotal,
		TotalAmount:     totalAmount,
		Currency:        "USD",
		Notes:           input.Notes,
		ShippingAddress: input.ShippingAddress,
		Items:           orderItems,
	}

	database.DB.Create(&order)

	dueDate := time.Now().Add(30 * 24 * time.Hour)
	database.DB.Create(&models.Invoice{
		OrderID:       order.ID,
		InvoiceNumber: fmt.Sprintf("INV-%s", orderNumber),
		Amount:        totalAmount,
		Status:        "pending",
		DueDate:       &dueDate,
	})

	if h.notifHandler != nil {
		h.notifHandler.Notify(sellerID.String(), "New order received", fmt.Sprintf("Order %s has been placed for $%.2f", orderNumber, totalAmount), "success")
		h.notifHandler.Notify(userID, "Order placed", fmt.Sprintf("Order %s has been placed successfully.", orderNumber), "success")
	}

	database.DB.Preload("Items").Preload("Invoice").First(&order, order.ID)
	utils.JSON(w, http.StatusCreated, order)
}

func (h *OrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	isSeller, _ := r.Context().Value(middleware.IsSellerKey).(bool)
	id := chi.URLParam(r, "id")

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var order models.Order
	query := database.DB.First(&order, "id = ?", id)
	if isSeller {
		query = database.DB.First(&order, "id = ? AND seller_id = ?", id, userID)
	}
	if query.Error != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "Order not found")
		return
	}

	validTransitions := map[string][]string{
		"pending":     {"confirmed", "cancelled"},
		"confirmed":   {"processing", "cancelled"},
		"processing":  {"shipped", "cancelled"},
		"shipped":     {"delivered"},
		"delivered":   {"completed"},
	}

	if allowed, ok := validTransitions[order.Status]; ok {
		valid := false
		for _, s := range allowed {
			if s == body.Status { valid = true; break }
		}
		if !valid {
			utils.ErrorJSON(w, http.StatusBadRequest, fmt.Sprintf("Cannot transition from %s to %s", order.Status, body.Status))
			return
		}
	}

	order.Status = body.Status
	database.DB.Save(&order)
	utils.JSON(w, http.StatusOK, order)
}

func (h *OrderHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	isSeller, _ := r.Context().Value(middleware.IsSellerKey).(bool)

	query := database.DB.Model(&models.Invoice{}).
		Joins("JOIN orders ON orders.id = invoices.order_id")

	if isSeller {
		query = query.Where("orders.seller_id = ?", userID)
	} else {
		query = query.Where("orders.buyer_id = ?", userID)
	}

	var total int64
	query.Count(&total)

	var invoices []models.Invoice
	query.Preload("Order").Order("created_at DESC").Find(&invoices)

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"results": invoices,
		"count":   int(total),
	})
}

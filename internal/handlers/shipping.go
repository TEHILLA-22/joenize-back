package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tehilla-22/b2b-api/internal/database"
	"github.com/tehilla-22/b2b-api/internal/middleware"
	"github.com/tehilla-22/b2b-api/internal/models"
	"github.com/tehilla-22/b2b-api/internal/utils"
)

type ShippingHandler struct{}

func NewShippingHandler() *ShippingHandler {
	return &ShippingHandler{}
}

func (h *ShippingHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	isSeller, _ := r.Context().Value(middleware.IsSellerKey).(bool)

	query := database.DB.Model(&models.Shipment{})
	if isSeller {
		query = query.Where("seller_id = ?", userID)
	} else {
		query = query.Where("buyer_id = ?", userID)
	}

	var total int64
	query.Count(&total)

	var shipments []models.Shipment
	query.Preload("Events").Preload("Order").Order("created_at DESC").Find(&shipments)

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"results": shipments,
		"count":   int(total),
	})
}

func (h *ShippingHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var body struct {
		OrderID       string `json:"order_id"`
		TrackingNumber string `json:"tracking_number"`
		Carrier        string `json:"carrier"`
		Origin         string `json:"origin"`
		Destination    string `json:"destination"`
		EstimatedDays  int    `json:"estimated_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var order models.Order
	if err := database.DB.First(&order, "id = ? AND seller_id = ?", body.OrderID, userID).Error; err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "Order not found or not yours")
		return
	}

	now := time.Now()
	shipment := models.Shipment{
		OrderID:       order.ID,
		SellerID:      uuid.MustParse(userID),
		BuyerID:       order.BuyerID,
		TrackingNumber: body.TrackingNumber,
		Carrier:        body.Carrier,
		Status:         "shipped",
		Origin:         body.Origin,
		Destination:    body.Destination,
		EstimatedDays:  body.EstimatedDays,
		ShippedAt:      &now,
	}

	database.DB.Create(&shipment)

	database.DB.Create(&models.ShipmentEvent{
		ShipmentID:  shipment.ID,
		Status:      "shipped",
		Location:    body.Origin,
		Description: "Package shipped",
		Timestamp:   now,
	})

	database.DB.Model(&order).Update("status", "shipped")
	database.DB.Preload("Events").First(&shipment, shipment.ID)

	utils.JSON(w, http.StatusCreated, shipment)
}

func (h *ShippingHandler) UpdateTracking(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var body struct {
		Status      string `json:"status"`
		Location    string `json:"location"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var shipment models.Shipment
	if err := database.DB.First(&shipment, "id = ?", id).Error; err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "Shipment not found")
		return
	}

	shipment.Status = body.Status
	if body.Status == "delivered" {
		now := time.Now()
		shipment.DeliveredAt = &now
		database.DB.Model(&models.Order{}).Where("id = ?", shipment.OrderID).Update("status", "delivered")
	}
	database.DB.Save(&shipment)

	database.DB.Create(&models.ShipmentEvent{
		ShipmentID:  shipment.ID,
		Status:      body.Status,
		Location:    body.Location,
		Description: body.Description,
		Timestamp:   time.Now(),
	})

	database.DB.Preload("Events").First(&shipment, shipment.ID)
	utils.JSON(w, http.StatusOK, shipment)
}

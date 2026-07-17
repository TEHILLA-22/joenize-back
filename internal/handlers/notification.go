package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/tehilla-22/b2b-api/internal/database"
	"github.com/tehilla-22/b2b-api/internal/middleware"
	"github.com/tehilla-22/b2b-api/internal/models"
	"github.com/tehilla-22/b2b-api/internal/utils"
)

type NotificationHandler struct {
	clients map[string][]chan string
}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{
		clients: make(map[string][]chan string),
	}
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var notifications []models.Notification
	database.DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(20).Find(&notifications)

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"results": notifications,
		"count":   len(notifications),
	})
}

func (h *NotificationHandler) Stream(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey)
	if userID == nil {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan string, 10)
	uid := userID.(string)
	h.clients[uid] = append(h.clients[uid], ch)

	notify := r.Context().Done()
	go func() {
		<-notify
		h.removeClient(uid, ch)
	}()

	for msg := range ch {
		fmt.Fprintf(w, "data: %s\n\n", msg)
		flusher.Flush()
	}
}

func (h *NotificationHandler) removeClient(userID string, ch chan string) {
	clients := h.clients[userID]
	for i, c := range clients {
		if c == ch {
			h.clients[userID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}
}

func (h *NotificationHandler) Notify(userID, title, body, tone string) {
	database.DB.Create(&models.Notification{
		UserID: uuid.MustParse(userID),
		Title:  title,
		Body:   body,
		Tone:   tone,
	})

	msg, _ := json.Marshal(map[string]string{
		"id":    fmt.Sprintf("%d", len(h.clients[userID])),
		"title": title,
		"body":  body,
		"tone":  tone,
	})

	for _, ch := range h.clients[userID] {
		select {
		case ch <- string(msg):
		default:
		}
	}
}

func (h *NotificationHandler) Broadcast(userID, title, body, tone string) {
	msg, _ := json.Marshal(map[string]string{
		"id":    fmt.Sprintf("%d", len(h.clients[userID])),
		"title": title,
		"body":  body,
		"tone":  tone,
	})

	for _, ch := range h.clients[userID] {
		select {
		case ch <- string(msg):
		default:
		}
	}
}

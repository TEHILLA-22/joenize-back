package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tehilla-22/b2b-api/internal/config"
	"github.com/tehilla-22/b2b-api/internal/middleware"
	"github.com/tehilla-22/b2b-api/internal/services"
	"github.com/tehilla-22/b2b-api/internal/utils"
)

type UserHandler struct {
	authService *services.AuthService
	cfg         *config.Config
}

func NewUserHandler(cfg *config.Config, authService *services.AuthService) *UserHandler {
	return &UserHandler{cfg: cfg, authService: authService}
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			utils.ErrorJSON(w, http.StatusBadRequest, "Failed to parse form")
			return
		}

		updates := make(map[string]interface{})

		if phone := r.FormValue("phone_number"); phone != "" {
			updates["phone_number"] = phone
		}

		file, header, err := r.FormFile("profile_photo")
		if err == nil {
			defer file.Close()
			photoURL, err := uploadFile(file, header, "profiles")
			if err == nil {
				updates["profile_photo"] = photoURL
			}
		}

		user, err := h.authService.UpdateUser(userID, updates)
		if err != nil {
			utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to update profile")
			return
		}
		utils.JSON(w, http.StatusOK, user)
		return
	}

	var body struct {
		PhoneNumber string `json:"phone_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if body.PhoneNumber != "" {
		updates["phone_number"] = body.PhoneNumber
	}

	user, err := h.authService.UpdateUser(userID, updates)
	if err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}
	utils.JSON(w, http.StatusOK, user)
}

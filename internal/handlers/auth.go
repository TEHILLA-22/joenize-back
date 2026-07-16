package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	// "github.com/google/uuid"
	"github.com/tehilla-22/b2b-api/internal/config"
	"github.com/tehilla-22/b2b-api/internal/middleware"
	"github.com/tehilla-22/b2b-api/internal/services"
	"github.com/tehilla-22/b2b-api/internal/utils"
)

type AuthHandler struct {
	authService  *services.AuthService
	emailService *services.EmailService
	cfg          *config.Config
}

func NewAuthHandler(cfg *config.Config, authService *services.AuthService, emailService *services.EmailService) *AuthHandler {
	return &AuthHandler{cfg: cfg, authService: authService, emailService: emailService}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input services.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))

	if input.Username == "" || input.Email == "" || input.Password == "" {
		utils.ErrorJSON(w, http.StatusUnprocessableEntity, "Username, email, and password are required")
		return
	}

	user, err := h.authService.Register(input)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			utils.ErrorJSON(w, http.StatusConflict, "A user with this email or username already exists")
			return
		}
		utils.ErrorJSON(w, http.StatusInternalServerError, "Could not create account")
		return
	}

	token, _ := h.authService.GenerateVerificationToken(user.ID.String())
	go h.emailService.SendVerificationEmail(user.Email, token)

	utils.JSON(w, http.StatusCreated, map[string]string{
		"detail": "Account created successfully. Please check your email to verify your account.",
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input services.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	input.Email = strings.TrimSpace(strings.ToLower(input.Email))

	result, err := h.authService.Login(input)
	if err != nil {
		utils.ErrorJSON(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	refreshToken, err := h.authService.GenerateRefreshToken(result.User.ID.String())
	if err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Could not complete login")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.AppEnv == "production",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 3600,
	})

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"access":        result.AccessToken,
		"refresh_token": refreshToken,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "User not found")
		return
	}

	utils.JSON(w, http.StatusOK, user)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		utils.ErrorJSON(w, http.StatusUnauthorized, "No refresh token")
		return
	}

	accessToken, err := h.authService.RefreshToken(cookie.Value)
	if err != nil {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"access": accessToken,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.AppEnv == "production",
		MaxAge:   -1,
	})

	utils.JSON(w, http.StatusOK, map[string]string{"detail": "Logged out successfully"})
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.authService.VerifyEmail(body.Token); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{"detail": "Email verified successfully"})
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Credential string `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if body.Credential == "" {
		utils.ErrorJSON(w, http.StatusBadRequest, "Credential is required")
		return
	}

	info, err := h.authService.VerifyGoogleToken(body.Credential)
	if err != nil {
		utils.ErrorJSON(w, http.StatusUnauthorized, err.Error())
		return
	}

	result, err := h.authService.LoginOrCreateGoogleUser(info)
	if err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Could not complete Google login")
		return
	}

	refreshToken, err := h.authService.GenerateRefreshToken(result.User.ID.String())
	if err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Could not complete login")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.AppEnv == "production",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 3600,
	})

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"access":        result.AccessToken,
		"refresh_token": refreshToken,
		"user":          result.User,
	})
}

package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tehilla-22/b2b-api/internal/config"
	"github.com/tehilla-22/b2b-api/internal/database"
	"github.com/tehilla-22/b2b-api/internal/models"
	"github.com/tehilla-22/b2b-api/internal/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type GoogleUserInfo struct {
	Sub      string `json:"sub"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
}

type AuthService struct {
	cfg *config.Config
}

func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{cfg: cfg}
}

type RegisterInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResult struct {
	AccessToken string       `json:"access"`
	User        *models.User `json:"user,omitempty"`
}

func (s *AuthService) Register(input RegisterInput) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: string(hashedPassword),
		IsBuyer:      true,
	}

	result := database.DB.Create(user)
	if result.Error != nil {
		return nil, result.Error
	}

	wallet := &models.Wallet{
		UserID:   user.ID,
		Balance:  0,
		Currency: "USD",
	}
	database.DB.Create(wallet)

	return user, nil
}

func (s *AuthService) Login(input LoginInput) (*AuthResult, error) {
	var user models.User
	result := database.DB.Where("email = ?", input.Email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid credentials")
		}
		return nil, result.Error
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !user.IsVerified {
		return nil, errors.New("email not verified. Please verify your email before logging in")
	}

	accessToken, err := utils.GenerateAccessToken(
		s.cfg.JWTSecret, s.cfg.JWTAccessExpiry,
		user.ID.String(), user.Email, user.IsSeller, user.IsBuyer,
	)
	if err != nil {
		return nil, err
	}

	return &AuthResult{AccessToken: accessToken, User: &user}, nil
}

func (s *AuthService) RefreshToken(tokenStr string) (string, error) {
	userID, err := utils.ValidateRefreshToken(s.cfg.JWTSecret, tokenStr)
	if err != nil {
		return "", errors.New("invalid or expired refresh token")
	}

	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		return "", errors.New("user not found")
	}

	accessToken, err := utils.GenerateAccessToken(
		s.cfg.JWTSecret, s.cfg.JWTAccessExpiry,
		user.ID.String(), user.Email, user.IsSeller, user.IsBuyer,
	)
	if err != nil {
		return "", err
	}

	return accessToken, nil
}

func (s *AuthService) GenerateRefreshToken(userID string) (string, error) {
	return utils.GenerateRefreshToken(s.cfg.JWTSecret, s.cfg.JWTRefreshExpiry, userID)
}

func (s *AuthService) GetUserByID(userID string) (*models.User, error) {
	var user models.User
	err := database.DB.Preload("Organizations").First(&user, "id = ?", userID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *AuthService) UpdateUser(userID string, updates map[string]interface{}) (*models.User, error) {
	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}

	if phone, ok := updates["phone_number"]; ok {
		user.PhoneNumber = phone.(string)
	}
	if photo, ok := updates["profile_photo"]; ok {
		user.ProfilePhoto = photo.(string)
	}

	database.DB.Save(&user)
	return &user, nil
}

func (s *AuthService) VerifyEmail(token string) error {
	claims, err := utils.ValidateToken(s.cfg.JWTSecret, token)
	if err != nil {
		return errors.New("invalid or expired verification token")
	}

	var user models.User
	if err := database.DB.First(&user, "id = ?", claims.UserID).Error; err != nil {
		return errors.New("user not found")
	}

	user.IsVerified = true
	database.DB.Save(&user)
	return nil
}

func (s *AuthService) VerifyGoogleToken(credential string) (*GoogleUserInfo, error) {
	resp, err := http.Get(fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", credential))
	if err != nil {
		return nil, errors.New("failed to verify Google token")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid Google token")
	}

	body, _ := io.ReadAll(resp.Body)
	var info GoogleUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, errors.New("failed to parse Google response")
	}

	if info.Email == "" {
		return nil, errors.New("email not provided by Google")
	}

	return &info, nil
}

func (s *AuthService) LoginOrCreateGoogleUser(info *GoogleUserInfo) (*AuthResult, error) {
	user, err := s.GetUserByEmail(info.Email)
	if err != nil {
		username := info.Name
		if username == "" {
			username = info.Email[:len(info.Email)-10]
		}

		user = &models.User{
			Username:   username,
			Email:      info.Email,
			IsBuyer:    true,
			IsVerified: true,
		}
		if result := database.DB.Create(user); result.Error != nil {
			return nil, result.Error
		}

		wallet := &models.Wallet{
			UserID:   user.ID,
			Balance:  0,
			Currency: "USD",
		}
		database.DB.Create(wallet)
	}

	accessToken, err := utils.GenerateAccessToken(
		s.cfg.JWTSecret, s.cfg.JWTAccessExpiry,
		user.ID.String(), user.Email, user.IsSeller, user.IsBuyer,
	)
	if err != nil {
		return nil, err
	}

	return &AuthResult{AccessToken: accessToken, User: user}, nil
}

func (s *AuthService) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := database.DB.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *AuthService) GenerateVerificationToken(userID string) (string, error) {
	return utils.GenerateAccessToken(
		s.cfg.JWTSecret, 24*time.Hour,
		userID, "", false, false,
	)
}

package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppName string
	AppPort int
	AppEnv  string

	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string

	JWTSecret        string
	JWTAccessExpiry  time.Duration
	JWTRefreshExpiry time.Duration

	PaystackSecretKey string
	PaystackPublicKey string
	PaystackCallback  string

	FrontendURL   string
	GoogleClientID string

	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string
}

func Load() *Config {
	return &Config{
		AppName: getEnv("APP_NAME", "Joenize B2B API"),
		AppPort: getEnvInt("APP_PORT", 8000),
		AppEnv:  getEnv("APP_ENV", "development"),

		DBHost:     getEnv("DATABASE_HOST", "localhost"),
		DBPort:     getEnvInt("DATABASE_PORT", 5432),
		DBName:     getEnv("DATABASE_NAME", "joenize"),
		DBUser:     getEnv("DATABASE_USER", "postgres"),
		DBPassword: getEnv("DATABASE_PASSWORD", "postgres"),
		DBSSLMode:  getEnv("DATABASE_SSLMODE", "disable"),

		JWTSecret:        getEnv("JWT_SECRET", "change-me"),
		JWTAccessExpiry:  getDuration("JWT_ACCESS_EXPIRY", 15*time.Minute),
		JWTRefreshExpiry: getDuration("JWT_REFRESH_EXPIRY", 7*24*time.Hour),

		PaystackSecretKey: os.Getenv("PAYSTACK_SECRET_KEY"),
		PaystackPublicKey: os.Getenv("PAYSTACK_PUBLIC_KEY"),
		PaystackCallback:  getEnv("PAYSTACK_CALLBACK_URL", "http://localhost:3000/seller-onboarding"),

		FrontendURL:   getEnv("FRONTEND_URL", "http://localhost:3000"),
		GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"),

		SMTPHost: getEnv("SMTP_HOST", ""),
		SMTPPort: getEnvInt("SMTP_PORT", 587),
		SMTPUser: os.Getenv("SMTP_USER"),
		SMTPPass: os.Getenv("SMTP_PASS"),
		SMTPFrom: getEnv("SMTP_FROM", "noreply@joenize.com"),
	}
}

func (c *Config) DSN() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

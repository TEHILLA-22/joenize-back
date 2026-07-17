package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/tehilla-22/b2b-api/internal/utils"
	"github.com/go-chi/chi/v5"
)

var allowedOrigins []string

func InitCORS(r chi.Router, origins string) {
	for _, o := range strings.Split(origins, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			allowedOrigins = append(allowedOrigins, o)
		}
	}
	r.Use(CORS)
}

type contextKey string

const UserIDKey contextKey = "user_id"
const UserEmailKey contextKey = "user_email"
const IsSellerKey contextKey = "is_seller"
const IsBuyerKey contextKey = "is_buyer"

func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.ErrorJSON(w, http.StatusUnauthorized, "Authorization header required")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				utils.ErrorJSON(w, http.StatusUnauthorized, "Invalid authorization format")
				return
			}

			claims, err := utils.ValidateToken(secret, parts[1])
			if err != nil {
				utils.ErrorJSON(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
			ctx = context.WithValue(ctx, IsSellerKey, claims.IsSeller)
			ctx = context.WithValue(ctx, IsBuyerKey, claims.IsBuyer)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func SellerOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isSeller, _ := r.Context().Value(IsSellerKey).(bool)
		if !isSeller {
			utils.ErrorJSON(w, http.StatusForbidden, "Seller access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		allowed := ""
		for _, o := range allowedOrigins {
			if o == origin {
				allowed = origin
				break
			}
		}
		if allowed == "" && len(allowedOrigins) > 0 {
			allowed = allowedOrigins[0]
		}

		if allowed != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowed)
		}
		if allowed == origin {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Organization-ID")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

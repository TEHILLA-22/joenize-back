package utils

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Detail string      `json:"detail,omitempty"`
	Error  string      `json:"error,omitempty"`
	Data   interface{} `json:"data,omitempty"`
	Access string      `json:"access,omitempty"`
	User   interface{} `json:"user,omitempty"`
	Count  *int        `json:"count,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func Success(w http.ResponseWriter, status int, data interface{}) {
	JSON(w, status, data)
}

func ErrorJSON(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"detail": message})
}

func ValidationError(w http.ResponseWriter, errors map[string]string) {
	JSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
		"errors": errors,
	})
}

// Package httputil はHTTPレスポンス生成のユーティリティを提供する。
package httputil

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse はエラーレスポンスの形式。
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON はJSONレスポンスを返す。
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// Error はエラーレスポンスを返す。
func Error(w http.ResponseWriter, status int, code string, message string) {
	JSON(w, status, ErrorResponse{
		Code:    code,
		Message: message,
	})
}

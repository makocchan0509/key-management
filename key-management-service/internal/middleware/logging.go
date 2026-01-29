// Package middleware はHTTPミドルウェアを提供する。
package middleware

import (
	"log/slog"
	"time"
)

// AuditLog は監査ログの構造体。
type AuditLog struct {
	Operation  string `json:"operation"`
	TenantID   string `json:"tenant_id"`
	Generation uint   `json:"generation,omitempty"`
	Result     string `json:"result"`
	Timestamp  string `json:"timestamp"`
}

// WriteAuditLog は監査ログを出力する。
func WriteAuditLog(operation string, tenantID string, generation uint, result string) {
	slog.Info("key operation completed",
		"operation", operation,
		"tenant_id", tenantID,
		"generation", generation,
		"result", result,
		"timestamp", time.Now().UTC().Format(time.RFC3339),
	)
}

// Package handler はHTTPハンドラを提供する。
package handler

import (
	"encoding/base64"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"key-management-service/internal/domain"
	"key-management-service/internal/middleware"
	"key-management-service/internal/usecase"
	"key-management-service/pkg/httputil"
)

var tenantIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// KeyHandler はHTTPハンドラを提供する。
type KeyHandler struct {
	service *usecase.KeyService
}

// NewKeyHandler は新しいKeyHandlerを生成する。
func NewKeyHandler(service *usecase.KeyService) *KeyHandler {
	return &KeyHandler{service: service}
}

func validateTenantID(tenantID string) error {
	if tenantID == "" {
		return domain.ErrInvalidTenantID
	}
	if len(tenantID) > 64 {
		return domain.ErrInvalidTenantID
	}
	if !tenantIDRegex.MatchString(tenantID) {
		return domain.ErrInvalidTenantID
	}
	return nil
}

func validateGeneration(genStr string) (uint, error) {
	gen, err := strconv.ParseUint(genStr, 10, 32)
	if err != nil || gen < 1 {
		return 0, domain.ErrInvalidGeneration
	}
	return uint(gen), nil
}

// KeyMetadataResponse は鍵メタデータのレスポンス形式。
type KeyMetadataResponse struct {
	TenantID   string `json:"tenant_id"`
	Generation uint   `json:"generation"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

// KeyResponse は鍵のレスポンス形式。
type KeyResponse struct {
	TenantID   string `json:"tenant_id"`
	Generation uint   `json:"generation"`
	Key        string `json:"key"`
}

// KeyListResponse は鍵一覧のレスポンス形式。
type KeyListResponse struct {
	Keys []KeyMetadataResponse `json:"keys"`
}

// CreateKey は新しい暗号鍵を生成する。
func (h *KeyHandler) CreateKey(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if err := validateTenantID(tenantID); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_TENANT_ID", "invalid tenant ID format")
		return
	}

	metadata, err := h.service.CreateKey(r.Context(), tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrKeyAlreadyExists) {
			middleware.WriteAuditLog("CREATE_KEY", tenantID, 0, "FAILED")
			httputil.Error(w, http.StatusConflict, "KEY_ALREADY_EXISTS", "key already exists for this tenant")
			return
		}
		middleware.WriteAuditLog("CREATE_KEY", tenantID, 0, "FAILED")
		httputil.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	middleware.WriteAuditLog("CREATE_KEY", tenantID, metadata.Generation, "SUCCESS")
	httputil.JSON(w, http.StatusCreated, KeyMetadataResponse{
		TenantID:   metadata.TenantID,
		Generation: metadata.Generation,
		Status:     string(metadata.Status),
		CreatedAt:  metadata.CreatedAt.Format(time.RFC3339),
	})
}

// GetCurrentKey は現在有効な鍵を取得する。
func (h *KeyHandler) GetCurrentKey(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if err := validateTenantID(tenantID); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_TENANT_ID", "invalid tenant ID format")
		return
	}

	key, err := h.service.GetCurrentKey(r.Context(), tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrKeyNotFound) {
			middleware.WriteAuditLog("GET_CURRENT_KEY", tenantID, 0, "FAILED")
			httputil.Error(w, http.StatusNotFound, "KEY_NOT_FOUND", "key not found for this tenant")
			return
		}
		middleware.WriteAuditLog("GET_CURRENT_KEY", tenantID, 0, "FAILED")
		httputil.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	middleware.WriteAuditLog("GET_CURRENT_KEY", tenantID, key.Generation, "SUCCESS")
	httputil.JSON(w, http.StatusOK, KeyResponse{
		TenantID:   key.TenantID,
		Generation: key.Generation,
		Key:        base64.StdEncoding.EncodeToString(key.Key),
	})
}

// GetKeyByGeneration は指定された世代の鍵を取得する。
func (h *KeyHandler) GetKeyByGeneration(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if err := validateTenantID(tenantID); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_TENANT_ID", "invalid tenant ID format")
		return
	}

	genStr := chi.URLParam(r, "generation")
	generation, err := validateGeneration(genStr)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_GENERATION", "invalid generation number")
		return
	}

	key, err := h.service.GetKeyByGeneration(r.Context(), tenantID, generation)
	if err != nil {
		if errors.Is(err, domain.ErrKeyNotFound) {
			middleware.WriteAuditLog("GET_KEY_BY_GENERATION", tenantID, generation, "FAILED")
			httputil.Error(w, http.StatusNotFound, "KEY_NOT_FOUND", "key not found for this tenant and generation")
			return
		}
		if errors.Is(err, domain.ErrKeyDisabled) {
			middleware.WriteAuditLog("GET_KEY_BY_GENERATION", tenantID, generation, "FAILED")
			httputil.Error(w, http.StatusGone, "KEY_DISABLED", "key has been disabled")
			return
		}
		middleware.WriteAuditLog("GET_KEY_BY_GENERATION", tenantID, generation, "FAILED")
		httputil.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	middleware.WriteAuditLog("GET_KEY_BY_GENERATION", tenantID, generation, "SUCCESS")
	httputil.JSON(w, http.StatusOK, KeyResponse{
		TenantID:   key.TenantID,
		Generation: key.Generation,
		Key:        base64.StdEncoding.EncodeToString(key.Key),
	})
}

// RotateKey は鍵をローテーションする。
func (h *KeyHandler) RotateKey(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if err := validateTenantID(tenantID); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_TENANT_ID", "invalid tenant ID format")
		return
	}

	metadata, err := h.service.RotateKey(r.Context(), tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrKeyNotFound) {
			middleware.WriteAuditLog("ROTATE_KEY", tenantID, 0, "FAILED")
			httputil.Error(w, http.StatusNotFound, "KEY_NOT_FOUND", "key not found for this tenant")
			return
		}
		middleware.WriteAuditLog("ROTATE_KEY", tenantID, 0, "FAILED")
		httputil.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	middleware.WriteAuditLog("ROTATE_KEY", tenantID, metadata.Generation, "SUCCESS")
	httputil.JSON(w, http.StatusCreated, KeyMetadataResponse{
		TenantID:   metadata.TenantID,
		Generation: metadata.Generation,
		Status:     string(metadata.Status),
		CreatedAt:  metadata.CreatedAt.Format(time.RFC3339),
	})
}

// ListKeys は鍵一覧を取得する。
func (h *KeyHandler) ListKeys(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if err := validateTenantID(tenantID); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_TENANT_ID", "invalid tenant ID format")
		return
	}

	keys, err := h.service.ListKeys(r.Context(), tenantID)
	if err != nil {
		middleware.WriteAuditLog("LIST_KEYS", tenantID, 0, "FAILED")
		httputil.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	middleware.WriteAuditLog("LIST_KEYS", tenantID, 0, "SUCCESS")
	response := KeyListResponse{
		Keys: make([]KeyMetadataResponse, len(keys)),
	}
	for i, k := range keys {
		response.Keys[i] = KeyMetadataResponse{
			TenantID:   k.TenantID,
			Generation: k.Generation,
			Status:     string(k.Status),
			CreatedAt:  k.CreatedAt.Format(time.RFC3339),
		}
	}
	httputil.JSON(w, http.StatusOK, response)
}

// DisableKey は鍵を無効化する。
func (h *KeyHandler) DisableKey(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if err := validateTenantID(tenantID); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_TENANT_ID", "invalid tenant ID format")
		return
	}

	genStr := chi.URLParam(r, "generation")
	generation, err := validateGeneration(genStr)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_GENERATION", "invalid generation number")
		return
	}

	err = h.service.DisableKey(r.Context(), tenantID, generation)
	if err != nil {
		if errors.Is(err, domain.ErrKeyNotFound) {
			middleware.WriteAuditLog("DISABLE_KEY", tenantID, generation, "FAILED")
			httputil.Error(w, http.StatusNotFound, "KEY_NOT_FOUND", "key not found for this tenant and generation")
			return
		}
		if errors.Is(err, domain.ErrKeyAlreadyDisabled) {
			middleware.WriteAuditLog("DISABLE_KEY", tenantID, generation, "FAILED")
			httputil.Error(w, http.StatusConflict, "KEY_ALREADY_DISABLED", "key is already disabled")
			return
		}
		middleware.WriteAuditLog("DISABLE_KEY", tenantID, generation, "FAILED")
		httputil.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	middleware.WriteAuditLog("DISABLE_KEY", tenantID, generation, "SUCCESS")
	w.WriteHeader(http.StatusAccepted)
}

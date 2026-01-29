package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"key-management-service/internal/domain"
	"key-management-service/internal/usecase"
)

// mockKeyRepository はテスト用のモックリポジトリ。
type mockKeyRepository struct {
	existsResult     bool
	existsErr        error
	createErr        error
	findByGenResult  *domain.EncryptionKey
	findByGenErr     error
	findLatestResult *domain.EncryptionKey
	findLatestErr    error
	findAllResult    []*domain.EncryptionKey
	findAllErr       error
	maxGenResult     uint
	maxGenErr        error
	updateStatusErr  error
	createdKeys      []*domain.EncryptionKey
}

func (m *mockKeyRepository) ExistsByTenantID(ctx context.Context, tenantID string) (bool, error) {
	return m.existsResult, m.existsErr
}

func (m *mockKeyRepository) Create(ctx context.Context, key *domain.EncryptionKey) error {
	if m.createErr != nil {
		return m.createErr
	}
	key.CreatedAt = time.Now()
	m.createdKeys = append(m.createdKeys, key)
	return nil
}

func (m *mockKeyRepository) FindByTenantIDAndGeneration(ctx context.Context, tenantID string, generation uint) (*domain.EncryptionKey, error) {
	return m.findByGenResult, m.findByGenErr
}

func (m *mockKeyRepository) FindLatestActiveByTenantID(ctx context.Context, tenantID string) (*domain.EncryptionKey, error) {
	return m.findLatestResult, m.findLatestErr
}

func (m *mockKeyRepository) FindAllByTenantID(ctx context.Context, tenantID string) ([]*domain.EncryptionKey, error) {
	return m.findAllResult, m.findAllErr
}

func (m *mockKeyRepository) GetMaxGeneration(ctx context.Context, tenantID string) (uint, error) {
	return m.maxGenResult, m.maxGenErr
}

func (m *mockKeyRepository) UpdateStatus(ctx context.Context, id string, status domain.KeyStatus) error {
	return m.updateStatusErr
}

// mockKMSClient はテスト用のモックKMSクライアント。
type mockKMSClient struct {
	encryptResult []byte
	encryptErr    error
	decryptResult []byte
	decryptErr    error
}

func (m *mockKMSClient) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	if m.encryptErr != nil {
		return nil, m.encryptErr
	}
	return append([]byte("encrypted:"), plaintext...), nil
}

func (m *mockKMSClient) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	if m.decryptErr != nil {
		return nil, m.decryptErr
	}
	if m.decryptResult != nil {
		return m.decryptResult, nil
	}
	return []byte("decrypted-key"), nil
}

func setupHandler(repo *mockKeyRepository, kms *mockKMSClient) *KeyHandler {
	service := usecase.NewKeyService(repo, kms)
	return NewKeyHandler(service)
}

func TestCreateKey_Success(t *testing.T) {
	repo := &mockKeyRepository{existsResult: false}
	kms := &mockKMSClient{}
	h := setupHandler(repo, kms)

	req := httptest.NewRequest(http.MethodPost, "/v1/tenants/tenant-001/keys", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", "tenant-001")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.CreateKey(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("want status 201, got %d", rec.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["tenant_id"] != "tenant-001" {
		t.Errorf("want tenant_id tenant-001, got %v", resp["tenant_id"])
	}
}

func TestCreateKey_InvalidTenantID(t *testing.T) {
	repo := &mockKeyRepository{}
	kms := &mockKMSClient{}
	h := setupHandler(repo, kms)

	req := httptest.NewRequest(http.MethodPost, "/v1/tenants/invalid@tenant/keys", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", "invalid@tenant")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.CreateKey(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want status 400, got %d", rec.Code)
	}
}

func TestCreateKey_AlreadyExists(t *testing.T) {
	repo := &mockKeyRepository{existsResult: true}
	kms := &mockKMSClient{}
	h := setupHandler(repo, kms)

	req := httptest.NewRequest(http.MethodPost, "/v1/tenants/tenant-001/keys", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", "tenant-001")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.CreateKey(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("want status 409, got %d", rec.Code)
	}
}

func TestGetCurrentKey_Success(t *testing.T) {
	repo := &mockKeyRepository{
		findLatestResult: &domain.EncryptionKey{
			TenantID:     "tenant-001",
			Generation:   3,
			EncryptedKey: []byte("encrypted"),
			Status:       domain.KeyStatusActive,
		},
	}
	kms := &mockKMSClient{decryptResult: []byte("plain-key")}
	h := setupHandler(repo, kms)

	req := httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-001/keys/current", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", "tenant-001")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.GetCurrentKey(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want status 200, got %d", rec.Code)
	}
}

func TestGetCurrentKey_NotFound(t *testing.T) {
	repo := &mockKeyRepository{findLatestResult: nil}
	kms := &mockKMSClient{}
	h := setupHandler(repo, kms)

	req := httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-001/keys/current", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", "tenant-001")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.GetCurrentKey(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("want status 404, got %d", rec.Code)
	}
}

func TestDisableKey_Success(t *testing.T) {
	repo := &mockKeyRepository{
		findByGenResult: &domain.EncryptionKey{
			ID:         "key-id",
			TenantID:   "tenant-001",
			Generation: 1,
			Status:     domain.KeyStatusActive,
		},
	}
	kms := &mockKMSClient{}
	h := setupHandler(repo, kms)

	req := httptest.NewRequest(http.MethodDelete, "/v1/tenants/tenant-001/keys/1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", "tenant-001")
	rctx.URLParams.Add("generation", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.DisableKey(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("want status 202, got %d", rec.Code)
	}
}

func TestDisableKey_AlreadyDisabled(t *testing.T) {
	repo := &mockKeyRepository{
		findByGenResult: &domain.EncryptionKey{
			ID:         "key-id",
			TenantID:   "tenant-001",
			Generation: 1,
			Status:     domain.KeyStatusDisabled,
		},
	}
	kms := &mockKMSClient{}
	h := setupHandler(repo, kms)

	req := httptest.NewRequest(http.MethodDelete, "/v1/tenants/tenant-001/keys/1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", "tenant-001")
	rctx.URLParams.Add("generation", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.DisableKey(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("want status 409, got %d", rec.Code)
	}
}

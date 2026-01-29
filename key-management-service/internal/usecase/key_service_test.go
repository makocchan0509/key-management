package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"key-management-service/internal/domain"
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
	if m.encryptResult != nil {
		return m.encryptResult, nil
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

func TestKeyService_CreateKey_Success(t *testing.T) {
	repo := &mockKeyRepository{existsResult: false}
	kms := &mockKMSClient{}
	svc := NewKeyService(repo, kms)

	metadata, err := svc.CreateKey(context.Background(), "tenant-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metadata.TenantID != "tenant-001" {
		t.Errorf("want tenant_id tenant-001, got %s", metadata.TenantID)
	}
	if metadata.Generation != 1 {
		t.Errorf("want generation 1, got %d", metadata.Generation)
	}
	if metadata.Status != domain.KeyStatusActive {
		t.Errorf("want status active, got %s", metadata.Status)
	}
	if len(repo.createdKeys) != 1 {
		t.Errorf("want 1 created key, got %d", len(repo.createdKeys))
	}
}

func TestKeyService_CreateKey_AlreadyExists(t *testing.T) {
	repo := &mockKeyRepository{existsResult: true}
	kms := &mockKMSClient{}
	svc := NewKeyService(repo, kms)

	_, err := svc.CreateKey(context.Background(), "tenant-001")
	if !errors.Is(err, domain.ErrKeyAlreadyExists) {
		t.Errorf("want ErrKeyAlreadyExists, got %v", err)
	}
}

func TestKeyService_GetCurrentKey_Success(t *testing.T) {
	repo := &mockKeyRepository{
		findLatestResult: &domain.EncryptionKey{
			TenantID:     "tenant-001",
			Generation:   3,
			EncryptedKey: []byte("encrypted"),
			Status:       domain.KeyStatusActive,
		},
	}
	kms := &mockKMSClient{decryptResult: []byte("plain-key")}
	svc := NewKeyService(repo, kms)

	key, err := svc.GetCurrentKey(context.Background(), "tenant-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if key.TenantID != "tenant-001" {
		t.Errorf("want tenant_id tenant-001, got %s", key.TenantID)
	}
	if key.Generation != 3 {
		t.Errorf("want generation 3, got %d", key.Generation)
	}
	if string(key.Key) != "plain-key" {
		t.Errorf("want key plain-key, got %s", string(key.Key))
	}
}

func TestKeyService_GetCurrentKey_NotFound(t *testing.T) {
	repo := &mockKeyRepository{findLatestResult: nil}
	kms := &mockKMSClient{}
	svc := NewKeyService(repo, kms)

	_, err := svc.GetCurrentKey(context.Background(), "tenant-001")
	if !errors.Is(err, domain.ErrKeyNotFound) {
		t.Errorf("want ErrKeyNotFound, got %v", err)
	}
}

func TestKeyService_GetKeyByGeneration_Success(t *testing.T) {
	repo := &mockKeyRepository{
		findByGenResult: &domain.EncryptionKey{
			TenantID:     "tenant-001",
			Generation:   2,
			EncryptedKey: []byte("encrypted"),
			Status:       domain.KeyStatusActive,
		},
	}
	kms := &mockKMSClient{decryptResult: []byte("plain-key")}
	svc := NewKeyService(repo, kms)

	key, err := svc.GetKeyByGeneration(context.Background(), "tenant-001", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if key.Generation != 2 {
		t.Errorf("want generation 2, got %d", key.Generation)
	}
}

func TestKeyService_GetKeyByGeneration_Disabled(t *testing.T) {
	repo := &mockKeyRepository{
		findByGenResult: &domain.EncryptionKey{
			TenantID:   "tenant-001",
			Generation: 2,
			Status:     domain.KeyStatusDisabled,
		},
	}
	kms := &mockKMSClient{}
	svc := NewKeyService(repo, kms)

	_, err := svc.GetKeyByGeneration(context.Background(), "tenant-001", 2)
	if !errors.Is(err, domain.ErrKeyDisabled) {
		t.Errorf("want ErrKeyDisabled, got %v", err)
	}
}

func TestKeyService_RotateKey_Success(t *testing.T) {
	repo := &mockKeyRepository{maxGenResult: 2}
	kms := &mockKMSClient{}
	svc := NewKeyService(repo, kms)

	metadata, err := svc.RotateKey(context.Background(), "tenant-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metadata.Generation != 3 {
		t.Errorf("want generation 3, got %d", metadata.Generation)
	}
}

func TestKeyService_RotateKey_NoExistingKey(t *testing.T) {
	repo := &mockKeyRepository{maxGenResult: 0}
	kms := &mockKMSClient{}
	svc := NewKeyService(repo, kms)

	_, err := svc.RotateKey(context.Background(), "tenant-001")
	if !errors.Is(err, domain.ErrKeyNotFound) {
		t.Errorf("want ErrKeyNotFound, got %v", err)
	}
}

func TestKeyService_ListKeys_Success(t *testing.T) {
	repo := &mockKeyRepository{
		findAllResult: []*domain.EncryptionKey{
			{TenantID: "tenant-001", Generation: 1, Status: domain.KeyStatusActive},
			{TenantID: "tenant-001", Generation: 2, Status: domain.KeyStatusDisabled},
		},
	}
	kms := &mockKMSClient{}
	svc := NewKeyService(repo, kms)

	keys, err := svc.ListKeys(context.Background(), "tenant-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("want 2 keys, got %d", len(keys))
	}
}

func TestKeyService_DisableKey_Success(t *testing.T) {
	repo := &mockKeyRepository{
		findByGenResult: &domain.EncryptionKey{
			ID:         "key-id",
			TenantID:   "tenant-001",
			Generation: 1,
			Status:     domain.KeyStatusActive,
		},
	}
	kms := &mockKMSClient{}
	svc := NewKeyService(repo, kms)

	err := svc.DisableKey(context.Background(), "tenant-001", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKeyService_DisableKey_AlreadyDisabled(t *testing.T) {
	repo := &mockKeyRepository{
		findByGenResult: &domain.EncryptionKey{
			ID:         "key-id",
			TenantID:   "tenant-001",
			Generation: 1,
			Status:     domain.KeyStatusDisabled,
		},
	}
	kms := &mockKMSClient{}
	svc := NewKeyService(repo, kms)

	err := svc.DisableKey(context.Background(), "tenant-001", 1)
	if !errors.Is(err, domain.ErrKeyAlreadyDisabled) {
		t.Errorf("want ErrKeyAlreadyDisabled, got %v", err)
	}
}

// Package usecase はアプリケーションのユースケースを実装する。
package usecase

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"key-management-service/internal/domain"
)

const keySize = 32 // AES-256 = 256 bits = 32 bytes

var tracer = otel.Tracer("key-management-service")

// KeyRepository はデータアクセスのインターフェース。
type KeyRepository interface {
	ExistsByTenantID(ctx context.Context, tenantID string) (bool, error)
	Create(ctx context.Context, key *domain.EncryptionKey) error
	FindByTenantIDAndGeneration(ctx context.Context, tenantID string, generation uint) (*domain.EncryptionKey, error)
	FindLatestActiveByTenantID(ctx context.Context, tenantID string) (*domain.EncryptionKey, error)
	FindAllByTenantID(ctx context.Context, tenantID string) ([]*domain.EncryptionKey, error)
	GetMaxGeneration(ctx context.Context, tenantID string) (uint, error)
	UpdateStatus(ctx context.Context, id string, status domain.KeyStatus) error
}

// KMSClient は暗号化/復号のインターフェース。
type KMSClient interface {
	Encrypt(ctx context.Context, plaintext []byte) ([]byte, error)
	Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)
}

// KeyService は暗号鍵に関するビジネスロジックを提供する。
type KeyService struct {
	repo      KeyRepository
	kmsClient KMSClient
}

// NewKeyService は新しいKeyServiceを生成する。
func NewKeyService(repo KeyRepository, kmsClient KMSClient) *KeyService {
	return &KeyService{
		repo:      repo,
		kmsClient: kmsClient,
	}
}

// generateAESKey はAES-256鍵を生成する。
func generateAESKey() ([]byte, error) {
	key := make([]byte, keySize)
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("generating random key: %w", err)
	}
	return key, nil
}

// CreateKey は指定されたテナントに対して新しい暗号鍵を生成する。
func (s *KeyService) CreateKey(ctx context.Context, tenantID string) (*domain.KeyMetadata, error) {
	ctx, span := tracer.Start(ctx, "KeyService.CreateKey",
		trace.WithAttributes(
			attribute.String("tenant.id", tenantID),
		),
	)
	defer span.End()

	// 既存チェック
	exists, err := s.repo.ExistsByTenantID(ctx, tenantID)
	if err != nil {
		span.RecordError(err)
		slog.ErrorContext(ctx, "failed to check existing key",
			"operation", "create_key",
			"tenant_id", tenantID,
			"error", err,
		)
		return nil, fmt.Errorf("checking existing key: %w", err)
	}
	if exists {
		slog.WarnContext(ctx, "key already exists",
			"operation", "create_key",
			"tenant_id", tenantID,
		)
		return nil, domain.ErrKeyAlreadyExists
	}

	// AES-256鍵を生成
	plainKey, err := generateAESKey()
	if err != nil {
		return nil, err
	}

	// KMSで暗号化
	encryptedKey, err := s.kmsClient.Encrypt(ctx, plainKey)
	if err != nil {
		span.RecordError(err)
		slog.ErrorContext(ctx, "failed to encrypt key",
			"operation", "create_key",
			"tenant_id", tenantID,
			"error", err,
		)
		return nil, fmt.Errorf("encrypting key: %w", err)
	}

	// DBに保存
	key := &domain.EncryptionKey{
		TenantID:     tenantID,
		Generation:   1,
		EncryptedKey: encryptedKey,
		Status:       domain.KeyStatusActive,
	}
	if err := s.repo.Create(ctx, key); err != nil {
		span.RecordError(err)
		slog.ErrorContext(ctx, "failed to create key in database",
			"operation", "create_key",
			"tenant_id", tenantID,
			"error", err,
		)
		return nil, fmt.Errorf("creating key: %w", err)
	}

	span.SetAttributes(attribute.Int("key.generation", 1))
	return &domain.KeyMetadata{
		TenantID:   key.TenantID,
		Generation: key.Generation,
		Status:     key.Status,
		CreatedAt:  key.CreatedAt,
	}, nil
}

// GetCurrentKey は指定されたテナントの現在有効な鍵を取得する。
func (s *KeyService) GetCurrentKey(ctx context.Context, tenantID string) (*domain.Key, error) {
	ctx, span := tracer.Start(ctx, "KeyService.GetCurrentKey",
		trace.WithAttributes(
			attribute.String("tenant.id", tenantID),
		),
	)
	defer span.End()

	key, err := s.repo.FindLatestActiveByTenantID(ctx, tenantID)
	if err != nil {
		span.RecordError(err)
		slog.ErrorContext(ctx, "failed to find current key",
			"operation", "get_current_key",
			"tenant_id", tenantID,
			"error", err,
		)
		return nil, fmt.Errorf("finding current key: %w", err)
	}
	if key == nil {
		slog.WarnContext(ctx, "key not found",
			"operation", "get_current_key",
			"tenant_id", tenantID,
		)
		return nil, domain.ErrKeyNotFound
	}

	// KMSで復号
	plainKey, err := s.kmsClient.Decrypt(ctx, key.EncryptedKey)
	if err != nil {
		span.RecordError(err)
		slog.ErrorContext(ctx, "failed to decrypt key",
			"operation", "get_current_key",
			"tenant_id", tenantID,
			"error", err,
		)
		return nil, fmt.Errorf("decrypting key: %w", err)
	}

	span.SetAttributes(attribute.Int("key.generation", int(key.Generation)))
	return &domain.Key{
		TenantID:   key.TenantID,
		Generation: key.Generation,
		Key:        plainKey,
	}, nil
}

// GetKeyByGeneration は指定されたテナント・世代の鍵を取得する。
func (s *KeyService) GetKeyByGeneration(ctx context.Context, tenantID string, generation uint) (*domain.Key, error) {
	ctx, span := tracer.Start(ctx, "KeyService.GetKeyByGeneration",
		trace.WithAttributes(
			attribute.String("tenant.id", tenantID),
			attribute.Int("key.generation", int(generation)),
		),
	)
	defer span.End()

	key, err := s.repo.FindByTenantIDAndGeneration(ctx, tenantID, generation)
	if err != nil {
		span.RecordError(err)
		slog.ErrorContext(ctx, "failed to find key by generation",
			"operation", "get_key_by_generation",
			"tenant_id", tenantID,
			"generation", generation,
			"error", err,
		)
		return nil, fmt.Errorf("finding key: %w", err)
	}
	if key == nil {
		slog.WarnContext(ctx, "key not found",
			"operation", "get_key_by_generation",
			"tenant_id", tenantID,
			"generation", generation,
		)
		return nil, domain.ErrKeyNotFound
	}
	if key.Status == domain.KeyStatusDisabled {
		slog.WarnContext(ctx, "key is disabled",
			"operation", "get_key_by_generation",
			"tenant_id", tenantID,
			"generation", generation,
		)
		return nil, domain.ErrKeyDisabled
	}

	// KMSで復号
	plainKey, err := s.kmsClient.Decrypt(ctx, key.EncryptedKey)
	if err != nil {
		span.RecordError(err)
		slog.ErrorContext(ctx, "failed to decrypt key",
			"operation", "get_key_by_generation",
			"tenant_id", tenantID,
			"generation", generation,
			"error", err,
		)
		return nil, fmt.Errorf("decrypting key: %w", err)
	}

	return &domain.Key{
		TenantID:   key.TenantID,
		Generation: key.Generation,
		Key:        plainKey,
	}, nil
}

// RotateKey は指定されたテナントに対して新しい世代の鍵を生成する。
func (s *KeyService) RotateKey(ctx context.Context, tenantID string) (*domain.KeyMetadata, error) {
	ctx, span := tracer.Start(ctx, "KeyService.RotateKey",
		trace.WithAttributes(
			attribute.String("tenant.id", tenantID),
		),
	)
	defer span.End()

	// 既存鍵の確認
	maxGen, err := s.repo.GetMaxGeneration(ctx, tenantID)
	if err != nil {
		span.RecordError(err)
		slog.ErrorContext(ctx, "failed to get max generation",
			"operation", "rotate_key",
			"tenant_id", tenantID,
			"error", err,
		)
		return nil, fmt.Errorf("getting max generation: %w", err)
	}
	if maxGen == 0 {
		slog.WarnContext(ctx, "key not found for rotation",
			"operation", "rotate_key",
			"tenant_id", tenantID,
		)
		return nil, domain.ErrKeyNotFound
	}

	// AES-256鍵を生成
	plainKey, err := generateAESKey()
	if err != nil {
		return nil, err
	}

	// KMSで暗号化
	encryptedKey, err := s.kmsClient.Encrypt(ctx, plainKey)
	if err != nil {
		span.RecordError(err)
		slog.ErrorContext(ctx, "failed to encrypt key",
			"operation", "rotate_key",
			"tenant_id", tenantID,
			"error", err,
		)
		return nil, fmt.Errorf("encrypting key: %w", err)
	}

	// DBに保存
	newGen := maxGen + 1
	key := &domain.EncryptionKey{
		TenantID:     tenantID,
		Generation:   newGen,
		EncryptedKey: encryptedKey,
		Status:       domain.KeyStatusActive,
	}
	if err := s.repo.Create(ctx, key); err != nil {
		span.RecordError(err)
		slog.ErrorContext(ctx, "failed to create rotated key in database",
			"operation", "rotate_key",
			"tenant_id", tenantID,
			"error", err,
		)
		return nil, fmt.Errorf("creating key: %w", err)
	}

	span.SetAttributes(attribute.Int("key.generation", int(newGen)))
	return &domain.KeyMetadata{
		TenantID:   key.TenantID,
		Generation: key.Generation,
		Status:     key.Status,
		CreatedAt:  key.CreatedAt,
	}, nil
}

// ListKeys は指定されたテナントの全世代の鍵メタデータを取得する。
func (s *KeyService) ListKeys(ctx context.Context, tenantID string) ([]*domain.KeyMetadata, error) {
	ctx, span := tracer.Start(ctx, "KeyService.ListKeys",
		trace.WithAttributes(
			attribute.String("tenant.id", tenantID),
		),
	)
	defer span.End()

	keys, err := s.repo.FindAllByTenantID(ctx, tenantID)
	if err != nil {
		span.RecordError(err)
		slog.ErrorContext(ctx, "failed to find all keys",
			"operation", "list_keys",
			"tenant_id", tenantID,
			"error", err,
		)
		return nil, fmt.Errorf("finding keys: %w", err)
	}

	metadata := make([]*domain.KeyMetadata, len(keys))
	for i, k := range keys {
		metadata[i] = &domain.KeyMetadata{
			TenantID:   k.TenantID,
			Generation: k.Generation,
			Status:     k.Status,
			CreatedAt:  k.CreatedAt,
		}
	}
	return metadata, nil
}

// DisableKey は指定されたテナント・世代の鍵を無効化する。
func (s *KeyService) DisableKey(ctx context.Context, tenantID string, generation uint) error {
	ctx, span := tracer.Start(ctx, "KeyService.DisableKey",
		trace.WithAttributes(
			attribute.String("tenant.id", tenantID),
			attribute.Int("key.generation", int(generation)),
		),
	)
	defer span.End()

	key, err := s.repo.FindByTenantIDAndGeneration(ctx, tenantID, generation)
	if err != nil {
		span.RecordError(err)
		slog.ErrorContext(ctx, "failed to find key for disable",
			"operation", "disable_key",
			"tenant_id", tenantID,
			"generation", generation,
			"error", err,
		)
		return fmt.Errorf("finding key: %w", err)
	}
	if key == nil {
		slog.WarnContext(ctx, "key not found",
			"operation", "disable_key",
			"tenant_id", tenantID,
			"generation", generation,
		)
		return domain.ErrKeyNotFound
	}
	if key.Status == domain.KeyStatusDisabled {
		slog.WarnContext(ctx, "key is already disabled",
			"operation", "disable_key",
			"tenant_id", tenantID,
			"generation", generation,
		)
		return domain.ErrKeyAlreadyDisabled
	}

	if err := s.repo.UpdateStatus(ctx, key.ID, domain.KeyStatusDisabled); err != nil {
		span.RecordError(err)
		slog.ErrorContext(ctx, "failed to update key status",
			"operation", "disable_key",
			"tenant_id", tenantID,
			"generation", generation,
			"error", err,
		)
		return fmt.Errorf("updating status: %w", err)
	}

	return nil
}

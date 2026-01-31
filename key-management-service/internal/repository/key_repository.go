// Package repository はデータアクセス層の実装を提供する。
package repository

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"key-management-service/internal/domain"
)

// EncryptionKeyModel はgorm用のモデル定義。
type EncryptionKeyModel struct {
	ID           string    `gorm:"type:char(36);primaryKey"`
	TenantID     string    `gorm:"type:varchar(64);not null;uniqueIndex:uk_tenant_generation;index:idx_tenant_id;index:idx_tenant_status"`
	Generation   uint      `gorm:"not null;uniqueIndex:uk_tenant_generation"`
	EncryptedKey []byte    `gorm:"type:blob;not null"`
	Status       string    `gorm:"type:enum('active','disabled');not null;default:'active';index:idx_tenant_status"`
	CreatedAt    time.Time `gorm:"type:datetime(6);not null;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"type:datetime(6);not null;autoUpdateTime"`
}

// TableName はテーブル名を返す。
func (EncryptionKeyModel) TableName() string {
	return "encryption_keys"
}

// BeforeCreate はレコード作成前にUUIDを生成する。
func (e *EncryptionKeyModel) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	return nil
}

// toDomain はモデルをドメインエンティティに変換する。
func (e *EncryptionKeyModel) toDomain() *domain.EncryptionKey {
	return &domain.EncryptionKey{
		ID:           e.ID,
		TenantID:     e.TenantID,
		Generation:   e.Generation,
		EncryptedKey: e.EncryptedKey,
		Status:       domain.KeyStatus(e.Status),
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}

// KeyRepository はデータアクセスを提供する。
type KeyRepository struct {
	db *gorm.DB
}

// NewKeyRepository は新しいKeyRepositoryを生成する。
func NewKeyRepository(db *gorm.DB) *KeyRepository {
	return &KeyRepository{db: db}
}

// ExistsByTenantID は指定されたテナントに鍵が存在するか確認する。
func (r *KeyRepository) ExistsByTenantID(ctx context.Context, tenantID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&EncryptionKeyModel{}).
		Where("tenant_id = ?", tenantID).
		Count(&count).Error
	if err != nil {
		slog.ErrorContext(ctx, "failed to count keys by tenant_id",
			"operation", "exists_by_tenant_id",
			"tenant_id", tenantID,
			"error", err,
		)
		return false, err
	}
	return count > 0, nil
}

// Create は新しい暗号鍵を保存する。
func (r *KeyRepository) Create(ctx context.Context, key *domain.EncryptionKey) error {
	model := &EncryptionKeyModel{
		ID:           key.ID,
		TenantID:     key.TenantID,
		Generation:   key.Generation,
		EncryptedKey: key.EncryptedKey,
		Status:       string(key.Status),
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		slog.ErrorContext(ctx, "failed to create key",
			"operation", "create",
			"tenant_id", key.TenantID,
			"generation", key.Generation,
			"error", err,
		)
		return err
	}
	// gormで設定された値をドメインエンティティに反映
	key.ID = model.ID
	key.CreatedAt = model.CreatedAt
	key.UpdatedAt = model.UpdatedAt
	return nil
}

// FindByTenantIDAndGeneration は指定されたテナント・世代の鍵を取得する。
func (r *KeyRepository) FindByTenantIDAndGeneration(ctx context.Context, tenantID string, generation uint) (*domain.EncryptionKey, error) {
	var model EncryptionKeyModel
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND generation = ?", tenantID, generation).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "failed to find key",
			"operation", "find_by_tenant_id_and_generation",
			"tenant_id", tenantID,
			"generation", generation,
			"error", err,
		)
		return nil, err
	}
	return model.toDomain(), nil
}

// FindLatestActiveByTenantID は指定されたテナントの最新有効鍵を取得する。
func (r *KeyRepository) FindLatestActiveByTenantID(ctx context.Context, tenantID string) (*domain.EncryptionKey, error) {
	var model EncryptionKeyModel
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, string(domain.KeyStatusActive)).
		Order("generation DESC").
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "failed to find latest active key",
			"operation", "find_latest_active_by_tenant_id",
			"tenant_id", tenantID,
			"error", err,
		)
		return nil, err
	}
	return model.toDomain(), nil
}

// FindAllByTenantID は指定されたテナントの全鍵を取得する。
func (r *KeyRepository) FindAllByTenantID(ctx context.Context, tenantID string) ([]*domain.EncryptionKey, error) {
	var models []EncryptionKeyModel
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("generation ASC").
		Find(&models).Error
	if err != nil {
		slog.ErrorContext(ctx, "failed to find all keys by tenant_id",
			"operation", "find_all_by_tenant_id",
			"tenant_id", tenantID,
			"error", err,
		)
		return nil, err
	}

	keys := make([]*domain.EncryptionKey, len(models))
	for i, m := range models {
		keys[i] = m.toDomain()
	}
	return keys, nil
}

// GetMaxGeneration は指定されたテナントの最大世代番号を取得する。
func (r *KeyRepository) GetMaxGeneration(ctx context.Context, tenantID string) (uint, error) {
	var maxGen *uint
	err := r.db.WithContext(ctx).
		Model(&EncryptionKeyModel{}).
		Where("tenant_id = ?", tenantID).
		Select("MAX(generation)").
		Scan(&maxGen).Error
	if err != nil {
		slog.ErrorContext(ctx, "failed to get max generation",
			"operation", "get_max_generation",
			"tenant_id", tenantID,
			"error", err,
		)
		return 0, err
	}
	if maxGen == nil {
		return 0, nil
	}
	return *maxGen, nil
}

// UpdateStatus は指定されたIDの鍵のステータスを更新する。
func (r *KeyRepository) UpdateStatus(ctx context.Context, id string, status domain.KeyStatus) error {
	err := r.db.WithContext(ctx).
		Model(&EncryptionKeyModel{}).
		Where("id = ?", id).
		Update("status", string(status)).Error
	if err != nil {
		slog.ErrorContext(ctx, "failed to update status",
			"operation", "update_status",
			"id", id,
			"status", status,
			"error", err,
		)
		return err
	}
	return nil
}

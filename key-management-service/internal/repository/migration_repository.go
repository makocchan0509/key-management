package repository

import (
	"context"
	"log/slog"
	"time"

	"key-management-service/internal/domain"

	"gorm.io/gorm"
)

// SchemaMigrationModel はschema_migrationsテーブルのモデル。
type SchemaMigrationModel struct {
	Version   string    `gorm:"column:version;primaryKey;type:varchar(14)"`
	AppliedAt time.Time `gorm:"column:applied_at;not null;autoCreateTime"`
}

// TableName はテーブル名を指定。
func (SchemaMigrationModel) TableName() string {
	return "schema_migrations"
}

// MigrationRepository はマイグレーション履歴を管理するリポジトリ。
type MigrationRepository struct {
	db *gorm.DB
}

// NewMigrationRepository は新しいMigrationRepositoryを生成する。
func NewMigrationRepository(db *gorm.DB) *MigrationRepository {
	return &MigrationRepository{db: db}
}

// FindAllApplied は適用済みマイグレーション一覧を取得する。
func (r *MigrationRepository) FindAllApplied(ctx context.Context) ([]*domain.Migration, error) {
	var models []SchemaMigrationModel
	if err := r.db.WithContext(ctx).Order("version ASC").Find(&models).Error; err != nil {
		slog.ErrorContext(ctx, "failed to find all applied migrations",
			"operation", "find_all_applied",
			"error", err,
		)
		return nil, err
	}

	migrations := make([]*domain.Migration, len(models))
	for i, model := range models {
		migrations[i] = &domain.Migration{
			Version:   model.Version,
			AppliedAt: &model.AppliedAt,
			Status:    domain.MigrationStatusApplied,
		}
	}

	return migrations, nil
}

// RecordMigration はマイグレーション適用履歴を記録する。
func (r *MigrationRepository) RecordMigration(ctx context.Context, version string) error {
	model := &SchemaMigrationModel{
		Version: version,
	}
	err := r.db.WithContext(ctx).Create(model).Error
	if err != nil {
		slog.ErrorContext(ctx, "failed to record migration",
			"operation", "record_migration",
			"version", version,
			"error", err,
		)
		return err
	}
	return nil
}

// IsMigrationApplied はマイグレーションが適用済みか確認する。
func (r *MigrationRepository) IsMigrationApplied(ctx context.Context, version string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&SchemaMigrationModel{}).Where("version = ?", version).Count(&count).Error; err != nil {
		slog.ErrorContext(ctx, "failed to check if migration is applied",
			"operation", "is_migration_applied",
			"version", version,
			"error", err,
		)
		return false, err
	}
	return count > 0, nil
}

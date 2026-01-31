package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"key-management-service/internal/domain"

	"gorm.io/gorm"
)

// MigrationRepository はマイグレーション履歴を管理するリポジトリのインターフェース。
type MigrationRepository interface {
	FindAllApplied(ctx context.Context) ([]*domain.Migration, error)
	RecordMigration(ctx context.Context, version string) error
	IsMigrationApplied(ctx context.Context, version string) (bool, error)
}

// MigrationService はマイグレーション実行のビジネスロジックを提供する。
type MigrationService struct {
	repo          MigrationRepository
	db            *gorm.DB
	migrationsDir string
}

// NewMigrationService は新しいMigrationServiceを生成する。
func NewMigrationService(repo MigrationRepository, db *gorm.DB, migrationsDir string) *MigrationService {
	return &MigrationService{
		repo:          repo,
		db:            db,
		migrationsDir: migrationsDir,
	}
}

// scanMigrationFiles はmigrationsディレクトリから.sqlファイルをスキャンする。
func (s *MigrationService) scanMigrationFiles(ctx context.Context) ([]*domain.Migration, error) {
	entries, err := os.ReadDir(s.migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []*domain.Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version, name, err := parseMigrationFileName(entry.Name())
		if err != nil {
			return nil, err
		}

		filePath := filepath.Join(s.migrationsDir, entry.Name())
		migrations = append(migrations, &domain.Migration{
			Version:  version,
			Name:     name,
			FilePath: filePath,
			Status:   domain.MigrationStatusPending,
		})
	}

	// バージョン順にソート
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// parseMigrationFileName はファイル名からバージョンと名前を抽出する。
// ファイル名のフォーマット: {version}_{name}.sql (例: 001_create_users.sql)
func parseMigrationFileName(filename string) (version, name string, err error) {
	// .sql拡張子を除去
	nameWithoutExt := strings.TrimSuffix(filename, ".sql")

	// アンダースコアで分割
	parts := strings.SplitN(nameWithoutExt, "_", 2)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("%w: %s (expected format: {version}_{name}.sql)", domain.ErrInvalidMigrationFile, filename)
	}

	version = parts[0]
	name = parts[1]

	return version, name, nil
}

// ApplyMigrations は未適用マイグレーションを番号順に実行する。
func (s *MigrationService) ApplyMigrations(ctx context.Context) (int, error) {
	// 全マイグレーションファイルをスキャン
	allMigrations, err := s.scanMigrationFiles(ctx)
	if err != nil {
		slog.Error("failed to scan migration files",
			"operation", "apply_migrations",
			"error", err,
		)
		return 0, err
	}

	// 未適用マイグレーションをフィルタリング
	var pendingMigrations []*domain.Migration
	for _, migration := range allMigrations {
		applied, err := s.repo.IsMigrationApplied(ctx, migration.Version)
		if err != nil {
			slog.ErrorContext(ctx, "failed to check migration status",
				"operation", "apply_migrations",
				"version", migration.Version,
				"error", err,
			)
			return 0, fmt.Errorf("failed to check migration status: %w", err)
		}
		if !applied {
			pendingMigrations = append(pendingMigrations, migration)
		}
	}

	if len(pendingMigrations) == 0 {
		return 0, nil
	}

	// 各マイグレーションを実行
	appliedCount := 0
	for _, migration := range pendingMigrations {
		if err := s.applyMigration(ctx, migration); err != nil {
			slog.ErrorContext(ctx, "failed to apply migration",
				"operation", "apply_migrations",
				"version", migration.Version,
				"error", err,
			)
			return appliedCount, fmt.Errorf("%w: version %s: %v", domain.ErrMigrationFailed, migration.Version, err)
		}
		appliedCount++
	}

	return appliedCount, nil
}

// applyMigration は単一のマイグレーションを実行する。
func (s *MigrationService) applyMigration(ctx context.Context, migration *domain.Migration) error {
	// SQLファイルを読み込み
	sqlBytes, err := os.ReadFile(migration.FilePath)
	if err != nil {
		slog.ErrorContext(ctx, "failed to read migration file",
			"operation", "apply_migration",
			"version", migration.Version,
			"file_path", migration.FilePath,
			"error", err,
		)
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// トランザクション内で実行
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// SQL実行
		if err := tx.Exec(string(sqlBytes)).Error; err != nil {
			slog.ErrorContext(ctx, "failed to execute migration SQL",
				"operation", "apply_migration",
				"version", migration.Version,
				"error", err,
			)
			return fmt.Errorf("failed to execute migration SQL: %w", err)
		}

		// 履歴を記録（トランザクション内で実行するため、同じtxを使用）
		model := struct {
			Version string `gorm:"column:version;primaryKey;type:varchar(14)"`
		}{
			Version: migration.Version,
		}
		if err := tx.Table("schema_migrations").Create(&model).Error; err != nil {
			slog.ErrorContext(ctx, "failed to record migration in schema_migrations",
				"operation", "apply_migration",
				"version", migration.Version,
				"error", err,
			)
			return fmt.Errorf("failed to record migration: %w", err)
		}

		return nil
	})
}

// GetMigrationStatus は現在のマイグレーション状況を取得する。
func (s *MigrationService) GetMigrationStatus(ctx context.Context) ([]*domain.Migration, error) {
	// 全マイグレーションファイルをスキャン
	allMigrations, err := s.scanMigrationFiles(ctx)
	if err != nil {
		return nil, err
	}

	// 適用済みマイグレーション履歴を取得
	appliedMigrations, err := s.repo.FindAllApplied(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch applied migrations",
			"operation", "get_migration_status",
			"error", err,
		)
		return nil, fmt.Errorf("failed to fetch applied migrations: %w", err)
	}

	// 適用済みマイグレーションのマップを作成
	appliedMap := make(map[string]*domain.Migration)
	for _, migration := range appliedMigrations {
		appliedMap[migration.Version] = migration
	}

	// ステータスを設定
	for _, migration := range allMigrations {
		if applied, exists := appliedMap[migration.Version]; exists {
			migration.Status = domain.MigrationStatusApplied
			migration.AppliedAt = applied.AppliedAt
		}
	}

	return allMigrations, nil
}

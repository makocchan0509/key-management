package usecase

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"key-management-service/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// mockMigrationRepository はテスト用のモック。
type mockMigrationRepository struct {
	appliedMigrations map[string]*domain.Migration
	recordError       error
}

func newMockMigrationRepository() *mockMigrationRepository {
	return &mockMigrationRepository{
		appliedMigrations: make(map[string]*domain.Migration),
	}
}

func (m *mockMigrationRepository) FindAllApplied(ctx context.Context) ([]*domain.Migration, error) {
	var result []*domain.Migration
	for _, migration := range m.appliedMigrations {
		result = append(result, migration)
	}
	return result, nil
}

func (m *mockMigrationRepository) RecordMigration(ctx context.Context, version string) error {
	if m.recordError != nil {
		return m.recordError
	}
	now := time.Now()
	m.appliedMigrations[version] = &domain.Migration{
		Version:   version,
		AppliedAt: &now,
		Status:    domain.MigrationStatusApplied,
	}
	return nil
}

func (m *mockMigrationRepository) IsMigrationApplied(ctx context.Context, version string) (bool, error) {
	_, exists := m.appliedMigrations[version]
	return exists, nil
}

// setupTestMigrationsDir はテスト用のmigrationsディレクトリを作成する。
func setupTestMigrationsDir(t *testing.T) string {
	t.Helper()

	// 一時ディレクトリを作成
	tmpDir := t.TempDir()
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatalf("failed to create migrations dir: %v", err)
	}

	// テスト用のマイグレーションファイルを作成
	files := map[string]string{
		"001_create_users.sql":    "CREATE TABLE users (id INT);",
		"002_create_posts.sql":    "CREATE TABLE posts (id INT);",
		"003_create_comments.sql": "CREATE TABLE comments (id INT);",
	}

	for filename, content := range files {
		filePath := filepath.Join(migrationsDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test migration file: %v", err)
		}
	}

	return migrationsDir
}

// setupTestDB はテスト用のインメモリSQLiteデータベースを作成する。
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// schema_migrationsテーブルを作成
	if err := db.Exec("CREATE TABLE schema_migrations (version VARCHAR(14) PRIMARY KEY, applied_at DATETIME)").Error; err != nil {
		t.Fatalf("failed to create schema_migrations table: %v", err)
	}

	return db
}

func TestMigrationService_ApplyMigrations(t *testing.T) {
	ctx := context.Background()
	migrationsDir := setupTestMigrationsDir(t)
	db := setupTestDB(t)
	repo := newMockMigrationRepository()

	service := NewMigrationService(repo, db, migrationsDir)

	// マイグレーションを実行
	count, err := service.ApplyMigrations(ctx)
	if err != nil {
		t.Fatalf("ApplyMigrations failed: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 migrations applied, got %d", count)
	}

	// テーブルが作成されたか確認
	tables := []string{"users", "posts", "comments"}
	for _, table := range tables {
		var count int64
		if err := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count).Error; err != nil {
			t.Errorf("failed to check table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("table %s was not created", table)
		}
	}
}

func TestMigrationService_ApplyMigrations_AlreadyApplied(t *testing.T) {
	ctx := context.Background()
	migrationsDir := setupTestMigrationsDir(t)
	db := setupTestDB(t)
	repo := newMockMigrationRepository()

	// 既にマイグレーションが適用済みと設定
	now := time.Now()
	repo.appliedMigrations["001"] = &domain.Migration{
		Version:   "001",
		AppliedAt: &now,
		Status:    domain.MigrationStatusApplied,
	}
	repo.appliedMigrations["002"] = &domain.Migration{
		Version:   "002",
		AppliedAt: &now,
		Status:    domain.MigrationStatusApplied,
	}

	service := NewMigrationService(repo, db, migrationsDir)

	// マイグレーションを実行
	count, err := service.ApplyMigrations(ctx)
	if err != nil {
		t.Fatalf("ApplyMigrations failed: %v", err)
	}

	// 未適用のマイグレーションのみ実行される
	if count != 1 {
		t.Errorf("expected 1 migration applied, got %d", count)
	}
}

func TestMigrationService_ApplyMigrations_Error(t *testing.T) {
	ctx := context.Background()
	migrationsDir := setupTestMigrationsDir(t)
	db := setupTestDB(t)
	repo := newMockMigrationRepository()

	service := NewMigrationService(repo, db, migrationsDir)

	// 不正なSQLファイルを作成
	invalidFile := filepath.Join(migrationsDir, "004_invalid.sql")
	if err := os.WriteFile(invalidFile, []byte("INVALID SQL SYNTAX;"), 0644); err != nil {
		t.Fatalf("failed to create invalid migration file: %v", err)
	}

	// マイグレーションを実行（エラーが発生することを期待）
	_, err := service.ApplyMigrations(ctx)
	if err == nil {
		t.Error("expected error for invalid SQL, but got nil")
	}
}

func TestMigrationService_GetMigrationStatus(t *testing.T) {
	ctx := context.Background()
	migrationsDir := setupTestMigrationsDir(t)
	db := setupTestDB(t)
	repo := newMockMigrationRepository()

	// 一部のマイグレーションを適用済みと設定
	now := time.Now()
	repo.appliedMigrations["001"] = &domain.Migration{
		Version:   "001",
		AppliedAt: &now,
		Status:    domain.MigrationStatusApplied,
	}

	service := NewMigrationService(repo, db, migrationsDir)

	// マイグレーションステータスを取得
	migrations, err := service.GetMigrationStatus(ctx)
	if err != nil {
		t.Fatalf("GetMigrationStatus failed: %v", err)
	}

	if len(migrations) != 3 {
		t.Errorf("expected 3 migrations, got %d", len(migrations))
	}

	// 001はapplied, 002と003はpending
	expectedStatuses := map[string]domain.MigrationStatus{
		"001": domain.MigrationStatusApplied,
		"002": domain.MigrationStatusPending,
		"003": domain.MigrationStatusPending,
	}

	for _, migration := range migrations {
		expectedStatus, exists := expectedStatuses[migration.Version]
		if !exists {
			t.Errorf("unexpected migration version: %s", migration.Version)
			continue
		}

		if migration.Status != expectedStatus {
			t.Errorf("migration %s: expected status %s, got %s", migration.Version, expectedStatus, migration.Status)
		}
	}
}

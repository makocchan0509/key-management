package repository

import (
	"context"
	"testing"

	"key-management-service/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB はテスト用のインメモリSQLiteデータベースを作成する。
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// encryption_keysテーブルを作成（SQLite用にENUM→TEXT変換）
	sql := `
		CREATE TABLE encryption_keys (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			generation INTEGER NOT NULL,
			encrypted_key BLOB NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, generation)
		);
		CREATE INDEX idx_tenant_id ON encryption_keys(tenant_id);
		CREATE INDEX idx_tenant_status ON encryption_keys(tenant_id, status);
	`

	if err := db.Exec(sql).Error; err != nil {
		t.Fatalf("failed to create encryption_keys table: %v", err)
	}

	return db
}

func TestKeyRepository_ExistsByTenantID(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := NewKeyRepository(db)

	// テストデータを挿入
	if err := db.Exec("INSERT INTO encryption_keys (id, tenant_id, generation, encrypted_key, status) VALUES (?, ?, ?, ?, ?)",
		"test-id-1", "tenant-1", 1, []byte("encrypted-key-1"), "active").Error; err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// テナントに鍵が存在する場合
	exists, err := repo.ExistsByTenantID(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("ExistsByTenantID failed: %v", err)
	}
	if !exists {
		t.Error("expected exists=true, got false")
	}

	// テナントに鍵が存在しない場合
	exists, err = repo.ExistsByTenantID(ctx, "tenant-2")
	if err != nil {
		t.Fatalf("ExistsByTenantID failed: %v", err)
	}
	if exists {
		t.Error("expected exists=false, got true")
	}
}

func TestKeyRepository_Create(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := NewKeyRepository(db)

	// 正常系: 鍵が作成される
	key := &domain.EncryptionKey{
		TenantID:     "tenant-1",
		Generation:   1,
		EncryptedKey: []byte("encrypted-key-1"),
		Status:       domain.KeyStatusActive,
	}

	if err := repo.Create(ctx, key); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// UUID自動生成を確認
	if key.ID == "" {
		t.Error("expected ID to be generated, got empty")
	}

	// タイムスタンプ反映を確認
	if key.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set, got zero value")
	}
	if key.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set, got zero value")
	}

	// データベースに保存されたことを確認
	var count int64
	if err := db.Model(&EncryptionKeyModel{}).Where("tenant_id = ?", "tenant-1").Count(&count).Error; err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestKeyRepository_FindByTenantIDAndGeneration(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := NewKeyRepository(db)

	// テストデータを挿入
	if err := db.Exec("INSERT INTO encryption_keys (id, tenant_id, generation, encrypted_key, status) VALUES (?, ?, ?, ?, ?)",
		"test-id-1", "tenant-1", 1, []byte("encrypted-key-1"), "active").Error; err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// 鍵が存在する場合
	key, err := repo.FindByTenantIDAndGeneration(ctx, "tenant-1", 1)
	if err != nil {
		t.Fatalf("FindByTenantIDAndGeneration failed: %v", err)
	}
	if key == nil {
		t.Fatal("expected key, got nil")
	}
	if key.TenantID != "tenant-1" {
		t.Errorf("expected tenant_id=tenant-1, got %s", key.TenantID)
	}
	if key.Generation != 1 {
		t.Errorf("expected generation=1, got %d", key.Generation)
	}

	// 鍵が存在しない場合
	key, err = repo.FindByTenantIDAndGeneration(ctx, "tenant-2", 1)
	if err != nil {
		t.Fatalf("FindByTenantIDAndGeneration failed: %v", err)
	}
	if key != nil {
		t.Errorf("expected nil, got %+v", key)
	}
}

func TestKeyRepository_FindLatestActiveByTenantID(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := NewKeyRepository(db)

	// テストデータを挿入
	testData := []struct {
		id         string
		generation uint
		status     string
	}{
		{"test-id-1", 1, "active"},
		{"test-id-2", 2, "active"},
		{"test-id-3", 3, "disabled"},
	}

	for _, data := range testData {
		if err := db.Exec("INSERT INTO encryption_keys (id, tenant_id, generation, encrypted_key, status) VALUES (?, ?, ?, ?, ?)",
			data.id, "tenant-1", data.generation, []byte("encrypted-key"), data.status).Error; err != nil {
			t.Fatalf("failed to insert test data: %v", err)
		}
	}

	// 最新有効鍵を返す（generation=2）
	key, err := repo.FindLatestActiveByTenantID(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("FindLatestActiveByTenantID failed: %v", err)
	}
	if key == nil {
		t.Fatal("expected key, got nil")
	}
	if key.Generation != 2 {
		t.Errorf("expected generation=2, got %d", key.Generation)
	}
	if key.Status != domain.KeyStatusActive {
		t.Errorf("expected status=active, got %s", key.Status)
	}

	// 鍵がない場合
	key, err = repo.FindLatestActiveByTenantID(ctx, "tenant-2")
	if err != nil {
		t.Fatalf("FindLatestActiveByTenantID failed: %v", err)
	}
	if key != nil {
		t.Errorf("expected nil, got %+v", key)
	}
}

func TestKeyRepository_FindAllByTenantID(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := NewKeyRepository(db)

	// テストデータを挿入（順不同）
	testData := []uint{3, 1, 2}
	for _, gen := range testData {
		if err := db.Exec("INSERT INTO encryption_keys (id, tenant_id, generation, encrypted_key, status) VALUES (?, ?, ?, ?, ?)",
			"test-id-"+string(rune(gen)), "tenant-1", gen, []byte("encrypted-key"), "active").Error; err != nil {
			t.Fatalf("failed to insert test data: %v", err)
		}
	}

	// 複数鍵を世代順に返す
	keys, err := repo.FindAllByTenantID(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("FindAllByTenantID failed: %v", err)
	}
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}

	// 世代順にソートされていることを確認
	expectedGenerations := []uint{1, 2, 3}
	for i, key := range keys {
		if key.Generation != expectedGenerations[i] {
			t.Errorf("keys[%d]: expected generation=%d, got %d", i, expectedGenerations[i], key.Generation)
		}
	}

	// 鍵がない場合
	keys, err = repo.FindAllByTenantID(ctx, "tenant-2")
	if err != nil {
		t.Fatalf("FindAllByTenantID failed: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected empty slice, got %d keys", len(keys))
	}
}

func TestKeyRepository_GetMaxGeneration(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := NewKeyRepository(db)

	// テストデータを挿入
	for gen := uint(1); gen <= 3; gen++ {
		if err := db.Exec("INSERT INTO encryption_keys (id, tenant_id, generation, encrypted_key, status) VALUES (?, ?, ?, ?, ?)",
			"test-id-"+string(rune(gen)), "tenant-1", gen, []byte("encrypted-key"), "active").Error; err != nil {
			t.Fatalf("failed to insert test data: %v", err)
		}
	}

	// 鍵がある場合
	maxGen, err := repo.GetMaxGeneration(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("GetMaxGeneration failed: %v", err)
	}
	if maxGen != 3 {
		t.Errorf("expected maxGen=3, got %d", maxGen)
	}

	// 鍵がない場合
	maxGen, err = repo.GetMaxGeneration(ctx, "tenant-2")
	if err != nil {
		t.Fatalf("GetMaxGeneration failed: %v", err)
	}
	if maxGen != 0 {
		t.Errorf("expected maxGen=0, got %d", maxGen)
	}
}

func TestKeyRepository_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := NewKeyRepository(db)

	// テストデータを挿入
	testID := "test-id-1"
	if err := db.Exec("INSERT INTO encryption_keys (id, tenant_id, generation, encrypted_key, status) VALUES (?, ?, ?, ?, ?)",
		testID, "tenant-1", 1, []byte("encrypted-key"), "active").Error; err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// ステータスを更新
	if err := repo.UpdateStatus(ctx, testID, domain.KeyStatusDisabled); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// 更新されたことを確認
	var model EncryptionKeyModel
	if err := db.Where("id = ?", testID).First(&model).Error; err != nil {
		t.Fatalf("failed to fetch updated record: %v", err)
	}
	if model.Status != string(domain.KeyStatusDisabled) {
		t.Errorf("expected status=disabled, got %s", model.Status)
	}
}

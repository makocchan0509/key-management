# 設計書

## アーキテクチャ概要

既存のレイヤードアーキテクチャに準拠し、各レイヤーでエラーログを出力する。

```
Handler → Usecase → Repository → Database
   ↓         ↓          ↓           ↓
  (既存)   (新規)     (新規)      (新規)
監査ログ  エラーログ  エラーログ   エラーログ

        → KMS
           ↓
         (新規)
        エラーログ
```

**原則**:
- エラーが発生した箇所で即座にログ出力
- エラーは上位レイヤーに伝播（既存のfmt.Errorf()は維持）
- ログ出力とエラー伝播の両方を行う

## コンポーネント設計

### 1. infrastructureレイヤー (`internal/infra/`)

#### 1-1. `kms.go`

**責務**:
- Cloud KMS API呼び出し時のエラーログ出力

**実装の要点**:
- `Encrypt()`メソッド: 暗号化エラー時にslog.Error出力
- `Decrypt()`メソッド: 復号エラー時にslog.Error出力
- ログに含める情報:
  - "operation": "kms_encrypt" / "kms_decrypt"
  - "error": エラーメッセージ
  - "key_name": KMS鍵のリソース名
- **絶対に出力しない情報**: plaintext（平文鍵）、ciphertext（暗号文）

**変更箇所**:
```go
// 変更前
func (c *KMSClient) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
    resp, err := c.client.Encrypt(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("encrypting: %w", err)
    }
    return resp.Ciphertext, nil
}

// 変更後
func (c *KMSClient) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
    resp, err := c.client.Encrypt(ctx, req)
    if err != nil {
        slog.ErrorContext(ctx, "failed to encrypt with KMS",
            "operation", "kms_encrypt",
            "key_name", c.keyName,
            "error", err,
        )
        return nil, fmt.Errorf("encrypting: %w", err)
    }
    return resp.Ciphertext, nil
}
```

#### 1-2. `database.go`

**責務**:
- データベース接続エラーのログ出力

**実装の要点**:
- `NewDB()`関数: 接続エラー時にslog.Error出力
- ログに含める情報:
  - "operation": "db_init"
  - "error": エラーメッセージ
- **絶対に出力しない情報**: DSN（パスワード含む）

**変更箇所**:
```go
// 変更後
func NewDB(dsn string) (*gorm.DB, error) {
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{...})
    if err != nil {
        slog.Error("failed to open database connection",
            "operation", "db_init",
            "error", err,
        )
        return nil, err
    }

    sqlDB, err := db.DB()
    if err != nil {
        slog.Error("failed to get underlying sql.DB",
            "operation", "db_init",
            "error", err,
        )
        return nil, err
    }
    // ...
}
```

### 2. usecaseレイヤー (`internal/usecase/`)

#### 2-1. `key_service.go`

**責務**:
- ビジネスロジック実行時のエラーログ出力
- ドメインエラーとシステムエラーの区別

**実装の要点**:
- **ドメインエラー**: slog.Warn（ビジネスルール違反、ユーザーの誤操作）
  - `domain.ErrKeyNotFound`, `domain.ErrKeyAlreadyExists`等
- **システムエラー**: slog.Error（インフラ障害、予期しないエラー）
  - リポジトリエラー、KMSエラー、その他の技術的エラー
- ログに含める情報:
  - "operation": メソッド名（"create_key", "get_current_key"等）
  - "tenant_id": テナントID
  - "generation": 世代番号（該当する場合）
  - "error": エラーメッセージ
- **絶対に出力しない情報**: plainKey（平文鍵）

**変更箇所の例**:
```go
// CreateKey()の例
func (s *KeyService) CreateKey(ctx context.Context, tenantID string) (*domain.KeyMetadata, error) {
    exists, err := s.repo.ExistsByTenantID(ctx, tenantID)
    if err != nil {
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
    // ... 以下同様のパターンでログ追加
}
```

**対象メソッド**:
- `CreateKey()`: 鍵生成
- `GetCurrentKey()`: 現在の鍵取得
- `GetKeyByGeneration()`: 特定世代の鍵取得
- `RotateKey()`: 鍵ローテーション
- `ListKeys()`: 鍵一覧取得
- `DisableKey()`: 鍵無効化

#### 2-2. `migration_service.go`

**責務**:
- マイグレーション実行時のエラーログ出力

**実装の要点**:
- ファイル読み込みエラー、SQL実行エラーをslog.Errorで出力
- ログに含める情報:
  - "operation": "apply_migrations" / "get_migration_status"
  - "migration_file": ファイル名
  - "error": エラーメッセージ

### 3. repositoryレイヤー (`internal/repository/`)

#### 3-1. `key_repository.go`

**責務**:
- データアクセス時のエラーログ出力

**実装の要点**:
- **gorm.ErrRecordNotFoundは例外**: これは通常フローのため、ログ出力せずnilを返す
- それ以外のエラー（接続エラー、SQL構文エラー等）はslog.Errorで出力
- ログに含める情報:
  - "operation": メソッド名（"exists_by_tenant_id", "create"等）
  - "tenant_id": テナントID（該当する場合）
  - "generation": 世代番号（該当する場合）
  - "error": エラーメッセージ

**変更箇所の例**:
```go
// ExistsByTenantID()の例
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

// FindByTenantIDAndGeneration()の例
func (r *KeyRepository) FindByTenantIDAndGeneration(ctx context.Context, tenantID string, generation uint) (*domain.EncryptionKey, error) {
    var model EncryptionKeyModel
    err := r.db.WithContext(ctx).
        Where("tenant_id = ? AND generation = ?", tenantID, generation).
        First(&model).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            // 通常フロー: ログ出力不要
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
```

**対象メソッド**:
- `ExistsByTenantID()`
- `Create()`
- `FindByTenantIDAndGeneration()`
- `FindLatestActiveByTenantID()`
- `FindAllByTenantID()`
- `GetMaxGeneration()`
- `UpdateStatus()`

#### 3-2. `migration_repository.go`

**責務**:
- マイグレーション管理のデータアクセス時のエラーログ出力

**実装の要点**:
- SQL実行エラー、トランザクションエラーをslog.Errorで出力
- ログに含める情報:
  - "operation": メソッド名
  - "migration_version": バージョン番号
  - "error": エラーメッセージ

## データフロー

### エラー発生時のログ出力フロー

```
1. エラー発生
2. 発生箇所でslog.Error/Warnを使ってログ出力
   - コンテキスト情報（tenant_id等）を構造化ログとして記録
   - エラーメッセージを記録
3. fmt.Errorf()でエラーをラップして上位レイヤーに伝播（既存の動作を維持）
4. 最終的にhandlerレイヤーで監査ログとして記録（既存）
```

## エラーハンドリング戦略

### ログレベルの使い分け

| エラー種別 | ログレベル | 例 |
|-----------|----------|-----|
| ドメインエラー | slog.Warn | ErrKeyNotFound, ErrKeyAlreadyExists |
| システムエラー | slog.Error | DB接続エラー、KMSエラー、SQL実行エラー |

### ログフォーマット

**標準フォーマット**:
```go
slog.ErrorContext(ctx, "エラーメッセージ",
    "operation", "操作名",
    "tenant_id", tenantID,  // コンテキスト情報
    "error", err,           // エラー内容
)
```

**Contextを使用する理由**:
- `slog.ErrorContext(ctx, ...)`を使用することで、トレースIDなどのコンテキスト情報を自動的にログに含めることができる

## テスト戦略

### ユニットテスト

**テスト方針**:
- エラーログ追加による既存テストへの影響を最小化
- ログ出力の検証は行わない（ログは副作用であり、テストの主目的ではない）
- エラー伝播の動作が変わっていないことを確認

**対応が必要なテスト**:
- 既存のユニットテストが全て通ることを確認
- 新たなテストケース追加は不要（ログ出力は副作用のため）

### 統合テスト

- Cloud Run環境でエラーログがCloud Loggingに出力されることを手動確認
- エラーログに期待する情報（tenant_id等）が含まれることを確認

## 依存ライブラリ

新しいライブラリの追加は不要。既存のライブラリを使用:
- `log/slog`: Go標準ライブラリの構造化ロギング

## ディレクトリ構造

変更されるファイル:
```
key-management-service/internal/
├── infra/
│   ├── kms.go           # 修正: エラーログ追加
│   └── database.go      # 修正: エラーログ追加
├── usecase/
│   ├── key_service.go       # 修正: エラーログ追加
│   └── migration_service.go # 修正: エラーログ追加
└── repository/
    ├── key_repository.go       # 修正: エラーログ追加
    └── migration_repository.go # 修正: エラーログ追加
```

## 実装の順序

1. **infrastructureレイヤー**: 外部依存（KMS、DB）のエラーログ実装
2. **repositoryレイヤー**: データアクセスのエラーログ実装
3. **usecaseレイヤー**: ビジネスロジックのエラーログ実装
4. **テスト実行**: 既存テストが全て通ることを確認

この順序により、下位レイヤーから上位レイヤーへと段階的に実装できる。

## セキュリティ考慮事項

### 機密情報の保護

**絶対にログに出力してはいけない情報**:
- 平文の暗号鍵（plainKey、plaintext）
- 暗号化された鍵（ciphertext）※必要性が低く、ログサイズ肥大化のため
- データベース接続文字列（DSN）※パスワードが含まれる

**ログに出力して良い情報**:
- tenant_id
- generation
- operation名
- エラーメッセージ
- KMS鍵のリソース名

### ログレビュー

実装後、以下を確認:
- [ ] grepで"plainKey"、"plaintext"を検索し、slogに渡していないことを確認
- [ ] 開発ガイドラインの「鍵の平文をログに出力しない」に準拠していることを確認

## パフォーマンス考慮事項

- slogは構造化ログを効率的に処理するため、パフォーマンス影響は最小限
- エラー時のみログ出力するため、正常フローへの影響なし
- ログレベルがINFO以上に設定されていても、ErrorとWarnは出力される

## 将来の拡張性

今回の実装により、以下の将来的な拡張が容易になる:
- Cloud Logging Alertsによるエラー監視
- ログベースのメトリクス作成
- エラー率のダッシュボード化
- 特定エラーの自動リトライ機能

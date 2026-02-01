# リポジトリ構造定義書 (Repository Structure Document)

## プロジェクト構造

```
key-management-service/
├── cmd/
│   ├── server/                      # APIサーバーエントリポイント
│   │   └── main.go
│   └── keyctl/                      # CLIツールエントリポイント
│       ├── main.go
│       └── migrate.go               # マイグレーションコマンド
├── internal/
│   ├── domain/                      # ドメインモデル・ビジネスルール
│   │   ├── key.go
│   │   ├── migration.go             # マイグレーションドメインモデル
│   │   └── errors.go
│   ├── usecase/                     # アプリケーションロジック
│   │   ├── key_service.go
│   │   └── migration_service.go     # マイグレーションサービス
│   ├── handler/                     # HTTPハンドラ
│   │   ├── key_handler.go
│   │   └── router.go
│   ├── repository/                  # データアクセス実装
│   │   ├── key_repository.go
│   │   └── migration_repository.go  # マイグレーションリポジトリ
│   ├── infra/                       # 外部サービス接続
│   │   ├── database.go
│   │   ├── kms.go
│   │   ├── logger.go                # トレース連携ロガー
│   │   └── tracer.go
│   └── middleware/                  # HTTPミドルウェア
│       ├── logging.go
│       └── tracing.go
├── pkg/                             # 外部公開可能な共通パッケージ
│   └── httputil/
│       └── response.go
├── config/
│   └── config.go                    # 設定構造体・読み込み
├── migrations/
│   └── 001_create_encryption_keys.sql
├── api/
│   └── openapi.yaml                 # OpenAPI定義
├── docs/                            # ドキュメント
│   ├── product-requirements.md
│   ├── functional-design.md
│   ├── architecture.md
│   ├── repository-structure.md
│   └── development-guidelines.md
├── .env.example                     # 環境変数テンプレート
├── .gitignore
├── go.mod
├── go.sum
├── Makefile
└── Dockerfile
```

## ディレクトリ詳細

### cmd/ (エントリポイント)

**役割**: アプリケーションのエントリポイントを配置する。各サブディレクトリが1つの実行バイナリに対応する。

**配置ファイル**:
- `main.go`: アプリケーション起動処理のみ。DI・サーバー起動・シグナルハンドリングを行う

**命名規則**:
- サブディレクトリ名はバイナリ名と一致させる

**依存関係**:
- 依存可能: `internal/`, `pkg/`, `config/`
- 依存禁止: 外部から`cmd/`へのインポート
- `cmd/`内のコードは薄く保ち、ロジックを持たせない

**例**:
```
cmd/
├── server/                  # APIサーバー → バイナリ名: server
│   └── main.go
└── keyctl/                  # CLIツール → バイナリ名: keyctl
    └── main.go
```

#### cmd/server/main.go

```go
// cmd/server/main.go
package main

import (
    "context"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/joho/godotenv"

    "key-management-service/config"
    "key-management-service/internal/handler"
    "key-management-service/internal/infra"
    "key-management-service/internal/repository"
    "key-management-service/internal/usecase"
)

func main() {
    ctx := context.Background()

    // .envファイルを読み込む（存在しない場合は無視）
    // 既存の環境変数は上書きしない
    _ = godotenv.Load()

    // 設定読み込み
    cfg := config.Load()

    // ログレベル設定
    var logLevel slog.Level
    switch cfg.LogLevel {
    case "DEBUG":
        logLevel = slog.LevelDebug
    case "WARN":
        logLevel = slog.LevelWarn
    case "ERROR":
        logLevel = slog.LevelError
    default:
        logLevel = slog.LevelInfo
    }

    // トレーサー初期化（ロガー設定の前に実行）
    tp, err := infra.InitTracer(ctx, cfg)
    if err != nil {
        slog.Error("failed to init tracer", "error", err)
        os.Exit(1)
    }
    if tp != nil {
        defer func() {
            if err := tp.Shutdown(ctx); err != nil {
                slog.Error("failed to shutdown tracer", "error", err)
            }
        }()
    }

    // トレース情報付きロガーを設定
    infra.SetupLogger(cfg, logLevel)

    // DB初期化
    if cfg.DatabaseURL == "" {
        slog.Error("DATABASE_URL is not set")
        os.Exit(1)
    }
    db, err := infra.NewDB(cfg.DatabaseURL, cfg)
    if err != nil {
        slog.Error("failed to init database", "error", err)
        os.Exit(1)
    }

    // KMSクライアント初期化
    kmsClient, err := infra.NewKMSClient(ctx)
    if err != nil {
        slog.Error("failed to init KMS client", "error", err)
        os.Exit(1)
    }
    defer func() {
        if closeErr := kmsClient.Close(); closeErr != nil {
            slog.Error("failed to close KMS client", "error", closeErr)
        }
    }()

    // DI
    repo := repository.NewKeyRepository(db)
    service := usecase.NewKeyService(repo, kmsClient)
    h := handler.NewKeyHandler(service)
    router := handler.NewRouter(h, cfg)

    // サーバー起動
    server := &http.Server{
        Addr:    ":" + cfg.Port,
        Handler: router,
    }

    // Graceful shutdown
    go func() {
        sigCh := make(chan os.Signal, 1)
        signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
        <-sigCh

        slog.Info("shutting down server...")
        shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
        defer cancel()
        if err := server.Shutdown(shutdownCtx); err != nil {
            slog.Error("server shutdown error", "error", err)
        }
    }()

    slog.Info("starting server", "port", cfg.Port)
    if err := server.ListenAndServe(); err != http.ErrServerClosed {
        slog.Error("server error", "error", err)
        os.Exit(1)
    }
    slog.Info("server stopped")
}
```

#### cmd/keyctl/main.go

```go
// cmd/keyctl/main.go
package main

import (
    "os"

    "github.com/spf13/cobra"
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "keyctl",
        Short: "Key Management Service CLI",
    }

    // グローバルフラグ
    rootCmd.PersistentFlags().String("api-url", "", "API endpoint URL")
    rootCmd.PersistentFlags().String("output", "text", "Output format: text, json")
    rootCmd.PersistentFlags().Duration("timeout", 30*time.Second, "Request timeout")

    // サブコマンド登録
    rootCmd.AddCommand(createCmd())
    rootCmd.AddCommand(getCmd())
    rootCmd.AddCommand(rotateCmd())
    rootCmd.AddCommand(listCmd())
    rootCmd.AddCommand(disableCmd())
    rootCmd.AddCommand(versionCmd())

    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### internal/ (プロジェクト内部コード)

Goコンパイラが外部パッケージからのimportを禁止するディレクトリ。プロジェクト固有のコードはすべてここに配置する。

#### internal/domain/ (ドメインモデル)

**役割**: ドメインモデル（エンティティ・値オブジェクト）とビジネスルール、ドメインエラーを定義する

**配置ファイル**:
- `key.go`: EncryptionKeyエンティティの構造体定義、ステータス定義
- `migration.go`: Migrationエンティティの構造体定義、ステータス定義
- `errors.go`: ドメイン固有のエラー定義

**命名規則**:
- ファイル名はリソース名の単数形（snake_case）

**依存関係**:
- 依存可能: 標準ライブラリのみ
- 依存禁止: `internal/`内の他のすべてのパッケージ

**例**:
```
internal/domain/
├── key.go          # EncryptionKey エンティティ
├── migration.go    # Migration エンティティ
└── errors.go       # KeyNotFoundError, KeyAlreadyExistsError 等
```

```go
// internal/domain/key.go
package domain

import "time"

type KeyStatus string

const (
    KeyStatusActive   KeyStatus = "active"
    KeyStatusDisabled KeyStatus = "disabled"
)

type EncryptionKey struct {
    ID           string
    TenantID     string
    Generation   uint
    EncryptedKey []byte
    Status       KeyStatus
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type KeyMetadata struct {
    TenantID   string
    Generation uint
    Status     KeyStatus
    CreatedAt  time.Time
}

type Key struct {
    TenantID   string
    Generation uint
    Key        []byte // 平文の鍵（Base64エンコード前）
}
```

```go
// internal/domain/errors.go
package domain

import "errors"

var (
    ErrKeyNotFound        = errors.New("key not found")
    ErrKeyAlreadyExists   = errors.New("key already exists")
    ErrKeyDisabled        = errors.New("key is disabled")
    ErrKeyAlreadyDisabled = errors.New("key is already disabled")
    ErrInvalidTenantID    = errors.New("invalid tenant ID")
    ErrInvalidGeneration  = errors.New("invalid generation")

    // マイグレーション関連エラー
    ErrMigrationFailed       = errors.New("migration failed")
    ErrMigrationFileNotFound = errors.New("migration file not found")
    ErrInvalidMigrationFile  = errors.New("invalid migration file")
)
```

```go
// internal/domain/migration.go
package domain

import "time"

type MigrationStatus string

const (
    MigrationStatusPending MigrationStatus = "pending"
    MigrationStatusApplied MigrationStatus = "applied"
)

type Migration struct {
    Version   string          // マイグレーションバージョン（例: "001", "002"）
    Name      string          // マイグレーション名（ファイル名から抽出）
    AppliedAt *time.Time      // 適用日時（未適用の場合はnil）
    FilePath  string          // マイグレーションファイルのパス
    Status    MigrationStatus // 適用状態
}
```

#### internal/usecase/ (アプリケーションロジック)

**役割**: アプリケーション固有のビジネスロジック（ユースケース）を実装する。リポジトリとKMSクライアントのインターフェースを定義する。

**配置ファイル**:
- `key_service.go`: 鍵管理のユースケース実装とリポジトリ・KMSインターフェースの定義
- `migration_service.go`: データベースマイグレーションのユースケース実装

**命名規則**:
- `{リソース名}_service.go`

**依存関係**:
- 依存可能: `domain`
- 依存禁止: `handler`, `repository`（実装）, `infra`
- **例外**: `migration_service.go`はマイグレーションSQL実行のため`gorm.DB`に直接依存する

**例**:
```
internal/usecase/
├── key_service.go
└── migration_service.go
```

```go
// internal/usecase/key_service.go
package usecase

import (
    "context"
    "crypto/rand"

    "key-management-service/internal/domain"
)

const KeySize = 32 // AES-256

// KeyRepository はデータアクセスのインターフェース
type KeyRepository interface {
    ExistsByTenantID(ctx context.Context, tenantID string) (bool, error)
    Create(ctx context.Context, key *domain.EncryptionKey) error
    FindByTenantIDAndGeneration(ctx context.Context, tenantID string, generation uint) (*domain.EncryptionKey, error)
    FindLatestActiveByTenantID(ctx context.Context, tenantID string) (*domain.EncryptionKey, error)
    FindAllByTenantID(ctx context.Context, tenantID string) ([]*domain.EncryptionKey, error)
    GetMaxGeneration(ctx context.Context, tenantID string) (uint, error)
    UpdateStatus(ctx context.Context, id string, status domain.KeyStatus) error
}

// KMSClient は暗号化/復号のインターフェース
type KMSClient interface {
    Encrypt(ctx context.Context, plaintext []byte) ([]byte, error)
    Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)
}

type KeyService struct {
    repo      KeyRepository
    kmsClient KMSClient
}

func NewKeyService(repo KeyRepository, kmsClient KMSClient) *KeyService {
    return &KeyService{
        repo:      repo,
        kmsClient: kmsClient,
    }
}

func (s *KeyService) CreateKey(ctx context.Context, tenantID string) (*domain.KeyMetadata, error) {
    // 実装...
}

func (s *KeyService) GetCurrentKey(ctx context.Context, tenantID string) (*domain.Key, error) {
    // 実装...
}

// 他のメソッド...
```

#### internal/handler/ (HTTPハンドラ)

**役割**: HTTPリクエストの受付・バリデーション・レスポンス返却、監査ログ出力を行う

**配置ファイル**:
- `key_handler.go`: 鍵管理APIのHTTPハンドラ
- `router.go`: ルーティング定義とミドルウェア適用

**命名規則**:
- `{リソース名}_handler.go`

**依存関係**:
- 依存可能: `usecase`, `domain`
- 依存禁止: `repository`, `infra`

**例**:
```
internal/handler/
├── key_handler.go
└── router.go
```

```go
// internal/handler/key_handler.go
package handler

import (
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"
    "key-management-service/internal/usecase"
)

type KeyHandler struct {
    service *usecase.KeyService
}

func NewKeyHandler(service *usecase.KeyService) *KeyHandler {
    return &KeyHandler{service: service}
}

func (h *KeyHandler) CreateKey(w http.ResponseWriter, r *http.Request) {
    tenantID := chi.URLParam(r, "tenant_id")
    // バリデーション、サービス呼び出し、レスポンス返却...
}

// 他のハンドラメソッド...
```

#### internal/repository/ (データアクセス実装)

**役割**: usecaseパッケージで定義されたリポジトリインターフェースの具体的な実装を配置する

**配置ファイル**:
- `key_repository.go`: KeyRepositoryインターフェースのgorm実装
- `migration_repository.go`: MigrationRepositoryインターフェースのgorm実装（schema_migrationsテーブル操作）

**命名規則**:
- `{リソース名}_repository.go`

**依存関係**:
- 依存可能: `domain`, `infra`
- 依存禁止: `handler`, `usecase`

**例**:
```
internal/repository/
├── key_repository.go
└── migration_repository.go
```

```go
// internal/repository/key_repository.go
package repository

import (
    "context"

    "gorm.io/gorm"
    "key-management-service/internal/domain"
)

type KeyRepository struct {
    db *gorm.DB
}

func NewKeyRepository(db *gorm.DB) *KeyRepository {
    return &KeyRepository{db: db}
}

func (r *KeyRepository) ExistsByTenantID(ctx context.Context, tenantID string) (bool, error) {
    var count int64
    err := r.db.WithContext(ctx).
        Model(&EncryptionKeyModel{}).
        Where("tenant_id = ?", tenantID).
        Count(&count).Error
    return count > 0, err
}

// 他のメソッド実装...
```

#### internal/infra/ (外部サービス接続)

**役割**: データベース接続、Cloud KMSクライアント、OpenTelemetryトレーサーなど技術的な基盤コードを配置する

**配置ファイル**:
- `database.go`: gormによるCloud SQL接続の初期化・管理
- `kms.go`: Cloud KMSクライアントの初期化・暗号化/復号実装
- `logger.go`: トレース情報付きslogハンドラ（TraceHandler）の実装
- `tracer.go`: OpenTelemetryトレーサープロバイダーの初期化

**依存関係**:
- 依存可能: 標準ライブラリ、外部ライブラリ、`config`
- 依存禁止: `domain`, `usecase`, `handler`

**例**:
```
internal/infra/
├── database.go
├── kms.go
├── logger.go
└── tracer.go
```

#### internal/middleware/ (HTTPミドルウェア)

**役割**: HTTPミドルウェア（ロギング・トレーシング等）を配置する

**配置ファイル**:
- `logging.go`: 監査ログ出力ミドルウェア
- `tracing.go`: OpenTelemetryトレーシングミドルウェア

**例**:
```
internal/middleware/
├── logging.go
└── tracing.go
```

### pkg/ (公開可能パッケージ)

**役割**: 他プロジェクトでも再利用可能な汎用コードを配置する

**使用方針**:
- 本当に汎用的なコードのみ配置する
- 不要なら作成しない。迷ったら`internal/`に置く

**例**:
```
pkg/
└── httputil/
    └── response.go     # JSONレスポンス生成ヘルパー
```

### config/ (設定)

**役割**: アプリケーション設定の読み込み・構造体定義

**配置ファイル**:
- `config.go`: 設定構造体と環境変数からの読み込みロジック

**例**:
```go
// config/config.go
package config

import "os"

type Config struct {
    Port              string
    DatabaseURL       string
    KMSKeyName        string
    OtelEnabled       bool
    OtelEndpoint      string
    OtelServiceName   string
    OtelSamplingRate  float64
    GoogleCloudProject string
    LogLevel          string
}

func Load() *Config {
    return &Config{
        Port:               getEnv("PORT", "8080"),
        DatabaseURL:        os.Getenv("DATABASE_URL"),
        KMSKeyName:         os.Getenv("KMS_KEY_NAME"),
        OtelEnabled:        os.Getenv("OTEL_ENABLED") == "true",
        OtelEndpoint:       os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
        OtelServiceName:    getEnv("OTEL_SERVICE_NAME", "key-management-service"),
        GoogleCloudProject: os.Getenv("GOOGLE_CLOUD_PROJECT"),
        LogLevel:           getEnv("LOG_LEVEL", "INFO"),
    }
}

func getEnv(key, defaultVal string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return defaultVal
}
```

### migrations/ (DBマイグレーション)

**役割**: データベーススキーマのマイグレーションファイルを配置する

**命名規則**:
- `{連番}_{説明}.sql`

**例**:
```
migrations/
└── 001_create_encryption_keys.sql
```

```sql
-- migrations/001_create_encryption_keys.sql
CREATE TABLE encryption_keys (
    id CHAR(36) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    generation INT UNSIGNED NOT NULL,
    encrypted_key BLOB NOT NULL,
    status ENUM('active', 'disabled') NOT NULL DEFAULT 'active',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uk_tenant_generation (tenant_id, generation),
    INDEX idx_tenant_id (tenant_id),
    INDEX idx_tenant_status (tenant_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### api/ (API定義)

**役割**: OpenAPI定義ファイルを配置する

**配置ファイル**:
- `openapi.yaml`: REST APIのOpenAPI 3.0定義

**例**:
```
api/
└── openapi.yaml
```

### docs/ (ドキュメント)

**役割**: プロジェクトのドキュメントを配置する

**配置ファイル**:
- `product-requirements.md`: プロダクト要求定義書
- `functional-design.md`: 機能設計書
- `architecture.md`: アーキテクチャ設計書
- `repository-structure.md`: 本ドキュメント
- `development-guidelines.md`: 開発ガイドライン

## ファイル配置規則

### ソースファイル

| ファイル種別 | 配置先 | 命名規則 | 例 |
|---|---|---|---|
| APIサーバーエントリポイント | `cmd/server/` | `main.go` | `cmd/server/main.go` |
| CLIエントリポイント | `cmd/keyctl/` | `main.go` | `cmd/keyctl/main.go` |
| CLIサブコマンド | `cmd/keyctl/` | `{コマンド名}.go` | `migrate.go` |
| ドメインモデル | `internal/domain/` | `{リソース}.go` | `key.go`, `migration.go` |
| ドメインエラー | `internal/domain/` | `errors.go` | `errors.go` |
| ユースケース | `internal/usecase/` | `{リソース}_service.go` | `key_service.go`, `migration_service.go` |
| HTTPハンドラ | `internal/handler/` | `{リソース}_handler.go` | `key_handler.go` |
| ルーター | `internal/handler/` | `router.go` | `router.go` |
| リポジトリ実装 | `internal/repository/` | `{リソース}_repository.go` | `key_repository.go`, `migration_repository.go` |
| ミドルウェア | `internal/middleware/` | `{機能名}.go` | `logging.go`, `tracing.go` |
| 基盤コード | `internal/infra/` | `{技術名}.go` | `database.go`, `kms.go`, `logger.go`, `tracer.go` |
| 設定 | `config/` | `config.go` | `config.go` |
| マイグレーション | `migrations/` | `{連番}_{説明}.sql` | `001_create_encryption_keys.sql` |
| API定義 | `api/` | `openapi.yaml` | `openapi.yaml` |

### テストファイル

| テスト種別 | 配置先 | 命名規則 | 例 |
|---|---|---|---|
| ユニットテスト | 対象ファイルと同じディレクトリ | `{対象}_test.go` | `key_service_test.go` |
| 統合テスト | 対象ファイルと同じディレクトリ | `{対象}_test.go` (build tagで分離) | `//go:build integration` |

### 設定ファイル

| ファイル種別 | 配置先 | 命名規則 |
|---|---|---|
| Go依存関係 | プロジェクトルート | `go.mod`, `go.sum` |
| ビルド定義 | プロジェクトルート | `Makefile` |
| コンテナ定義 | プロジェクトルート | `Dockerfile` |
| Git除外設定 | プロジェクトルート | `.gitignore` |

## 命名規則

### ディレクトリ名

- **小文字** を使用（Go標準）
- 短く、意味が明確な名前をつける
- 複数形は使わない（Goの慣習: `handler` not `handlers`）

### ファイル名

- **snake_case** を使用
- 1ファイルに1つの主要な型を定義することを目安にする
- 接尾辞でレイヤーを明示する（`_handler`, `_service`, `_repository`）

### コード内の命名慣習

| 対象 | 規則 | 例 |
|---|---|---|
| エクスポートされる型・関数 | PascalCase | `KeyService`, `NewKeyService` |
| 非公開の型・関数 | camelCase | `validateTenantID`, `dbClient` |
| インターフェース | 動詞+er / 役割名 | `KeyRepository`, `KMSClient` |
| コンストラクタ | `New` + 型名 | `NewKeyService`, `NewKeyHandler` |
| パッケージ名 | 小文字、短く、1単語 | `handler`, `domain`, `usecase` |

## 依存関係のルール

### レイヤー間の依存方向

```
cmd → internal/*, config/, pkg/

handler → usecase → domain
                  ↗
repository ------/
          ↘
           infra → (外部ライブラリのみ)
```

**許可される依存**:
- `cmd` → `internal/*`, `config/`, `pkg/`
- `handler` → `usecase`, `domain`
- `usecase` → `domain`
- `repository` → `domain`, `infra`
- `middleware` → `domain`（必要な場合のみ）
- `infra` → 標準ライブラリ、外部ライブラリ

**禁止される依存**:
- `domain` → 他のinternalパッケージ
- `usecase` → `handler`, `repository`（実装）, `infra`
- `handler` → `repository`, `infra`
- 循環依存は一切禁止

### 依存性逆転（インターフェース定義）

リポジトリとKMSクライアントのインターフェースは**使う側（usecase）**に定義し、実装は`repository`および`infra`パッケージに置く。これにより`usecase`は具体的な実装に依存しない。

```
usecase (interface定義) ← repository (interface実装)
usecase (interface定義) ← infra/kms (interface実装)
```

## スケーリング戦略

### 機能の追加

新しい機能を追加する際の配置方針:

1. **小規模機能**: 既存ディレクトリにファイルを追加
   - 例: 新しいAPIエンドポイント → `handler/`に新しいハンドラメソッドを追加

2. **中規模機能**: レイヤー内にサブディレクトリを作成
   - 例: 監査ログ機能 → `internal/audit/`として分離

3. **大規模機能**: ドメイン単位で分割
   - 本プロジェクトは単一ドメイン（鍵管理）のため、現状の構造を維持

### ファイルサイズの管理

**ファイル分割の目安**:
- 1ファイル: 300行以下を推奨
- 300-500行: 分割を検討
- 500行以上: 責務ごとに分割する

## アンチパターン

| アンチパターン | 理由 | 代替案 |
|---|---|---|
| `models/`パッケージ | 曖昧で肥大化する | `domain/`に配置 |
| `utils/`, `helpers/` | 無関係なコードの寄せ集めになる | 適切なパッケージに機能を配置 |
| `common/`, `shared/`の濫用 | 依存の方向が不明確になる | 必要最小限に留める |
| 実装が1つしかないインターフェース | 不要な抽象化 | テスト容易性が必要な場合のみ作成 |
| パッケージの循環参照 | コンパイルエラーになる | 依存方向を一方向に保つ |
| `handler`から直接DB操作 | レイヤーの責務違反 | `usecase`→`repository`経由で操作 |
| ドメインロジックを`handler`に記述 | テストが困難になる | `usecase`または`domain`に移動 |

## 除外設定

### .gitignore

```
# バイナリ
/bin/
*.exe

# 依存
/vendor/

# IDE
.idea/
.vscode/

# 環境変数
.env
.env.local
.env.*.local

# OS
.DS_Store
Thumbs.db

# テストカバレッジ
coverage.out
coverage.html

# ログ
*.log

# ビルド成果物
/dist/

# 一時ファイル
tmp/
*.tmp
```

## チェックリスト

- [x] 各ディレクトリの役割が明確に定義されている
- [x] レイヤー構造がディレクトリに反映されている
- [x] 命名規則が一貫している
- [x] テストコードの配置方針が決まっている
- [x] 依存関係のルールが明確である
- [x] 循環依存がない
- [x] スケーリング戦略が考慮されている
- [x] 共有コードの配置ルールが定義されている
- [x] 設定ファイルの管理方法が決まっている
- [x] ドキュメントの配置場所が明確である

# 設計書

## アーキテクチャ概要

レイヤードアーキテクチャを採用し、責務を明確に分離する。

```
┌─────────────────────────────────────────────────────────────┐
│                      CLIレイヤー (keyctl)                    │
│  - コマンドライン引数のパース                                  │
│  - 入力値のバリデーション                                      │
│  - REST APIの呼び出し                                         │
├─────────────────────────────────────────────────────────────┤
│                    APIレイヤー (REST API Server)              │
│  - HTTPリクエストの受付とルーティング                          │
│  - リクエストのバリデーション                                   │
│  - 監査ログ出力                                               │
├─────────────────────────────────────────────────────────────┤
│                   サービスレイヤー (KeyService)               │
│  - ビジネスロジック（鍵生成、ローテーション等）                  │
│  - Cloud KMSとの連携（暗号化/復号）                            │
├─────────────────────────────────────────────────────────────┤
│                   データレイヤー (KeyRepository)              │
│  - Cloud SQLへのCRUD操作                                      │
│  - gormによるトランザクション管理                               │
└─────────────────────────────────────────────────────────────┘
```

## コンポーネント設計

### 1. cmd/server (APIサーバーエントリポイント)

**責務**:
- 設定読み込み
- DB接続初期化
- KMSクライアント初期化
- DIによる依存注入
- HTTPサーバー起動とGraceful shutdown

**実装の要点**:
- main.goは薄く保ち、ロジックを持たせない
- シグナルハンドリングでGraceful shutdown

### 2. cmd/keyctl (CLIエントリポイント)

**責務**:
- cobraによるコマンドライン引数のパース
- グローバルフラグの管理（--api-url, --output, --timeout）
- サブコマンドの登録と実行
- 終了コードの設定

**実装の要点**:
- 各コマンドは独立した関数として実装
- HTTPクライアントを使用してAPIを呼び出す

### 3. internal/domain (ドメインモデル)

**責務**:
- EncryptionKey エンティティの定義
- KeyStatus（active/disabled）の定義
- KeyMetadata, Key 構造体の定義
- ドメインエラーの定義

**実装の要点**:
- 外部依存なし（標準ライブラリのみ）
- ビジネスルールをここにカプセル化

### 4. internal/usecase (サービスレイヤー)

**責務**:
- KeyService: 鍵管理のビジネスロジック
- KeyRepository インターフェースの定義
- KMSClient インターフェースの定義

**実装の要点**:
- 具体的な実装に依存しない（インターフェースに依存）
- AES-256鍵の生成ロジック

### 5. internal/handler (HTTPハンドラ)

**責務**:
- KeyHandler: 各APIエンドポイントのハンドラ
- router: chiによるルーティング設定
- リクエストバリデーション
- レスポンス生成
- 監査ログ出力

**実装の要点**:
- tenant_idの形式検証（正規表現 `^[a-zA-Z0-9_-]+$`、1-64文字）
- generationの検証（正の整数）
- エラーレスポンスの統一フォーマット

### 6. internal/repository (データアクセス)

**責務**:
- KeyRepository実装: gormを使用したCRUD操作
- EncryptionKeyModelの定義（gormタグ付き）

**実装の要点**:
- gormのプリペアドステートメント使用（SQLインジェクション対策）
- UUIDの自動生成

### 7. internal/infra (インフラ層)

**責務**:
- database.go: gormによるDB接続管理
- kms.go: Cloud KMSクライアント実装

**実装の要点**:
- 環境変数からの設定読み込み
- 接続プール設定

### 8. internal/middleware (ミドルウェア)

**責務**:
- logging.go: 監査ログ出力ミドルウェア

**実装の要点**:
- slogによる構造化ロギング
- 鍵の平文をログに出力しない

### 9. config (設定)

**責務**:
- 環境変数からの設定読み込み
- Config構造体の定義

### 10. pkg/httputil (HTTPユーティリティ)

**責務**:
- JSONレスポンス生成ヘルパー
- エラーレスポンス生成

## データフロー

### 鍵の生成
```
1. Client → POST /v1/tenants/{tenant_id}/keys
2. KeyHandler: tenant_idバリデーション
3. KeyService.CreateKey(tenantID)
   a. repo.ExistsByTenantID() で既存チェック
   b. crypto/rand でAES-256鍵を生成
   c. kmsClient.Encrypt() でKEKによる暗号化
   d. repo.Create() でDBに保存
4. 監査ログ出力
5. 201 Created + KeyMetadata を返却
```

### 鍵の取得（現在有効な鍵）
```
1. Client → GET /v1/tenants/{tenant_id}/keys/current
2. KeyHandler: tenant_idバリデーション
3. KeyService.GetCurrentKey(tenantID)
   a. repo.FindLatestActiveByTenantID() で最新有効鍵を取得
   b. kmsClient.Decrypt() でDEKを復号
4. 監査ログ出力
5. 200 OK + Key を返却
```

### 鍵の無効化
```
1. Client → DELETE /v1/tenants/{tenant_id}/keys/{generation}
2. KeyHandler: パラメータバリデーション
3. KeyService.DisableKey(tenantID, generation)
   a. repo.FindByTenantIDAndGeneration() で鍵を取得
   b. ステータス検証（既に無効化されていないか）
   c. repo.UpdateStatus() でステータスをdisabledに更新
4. 監査ログ出力
5. 202 Accepted を返却
```

## エラーハンドリング戦略

### カスタムエラー（internal/domain/errors.go）

```go
var (
    ErrKeyNotFound        = errors.New("key not found")
    ErrKeyAlreadyExists   = errors.New("key already exists")
    ErrKeyDisabled        = errors.New("key is disabled")
    ErrKeyAlreadyDisabled = errors.New("key is already disabled")
    ErrInvalidTenantID    = errors.New("invalid tenant ID")
    ErrInvalidGeneration  = errors.New("invalid generation")
)
```

### エラーハンドリングパターン

- ドメインエラー → 適切なHTTPステータスコードにマッピング
- 予期しないエラー → 500 Internal Server Error + ログ出力
- エラーメッセージは小文字で始め、句読点をつけない（Go慣習）

## テスト戦略

### ユニットテスト
- KeyService: モックリポジトリ・モックKMSを使用
- KeyHandler: httptest使用
- CLI: コマンド引数パースのテスト

### 統合テスト
- testcontainers-goでMySQLコンテナを使用
- Cloud KMSはGoogle Cloud開発環境で検証

## 依存ライブラリ

```go
// go.mod
module key-management-service

go 1.23

require (
    github.com/go-chi/chi/v5 v5.1.0
    github.com/google/uuid v1.6.0
    github.com/spf13/cobra v1.8.1
    gorm.io/driver/mysql v1.5.7
    gorm.io/gorm v1.25.12
    cloud.google.com/go/kms v1.20.5
)
```

## ディレクトリ構造

```
key-management-service/
├── cmd/
│   ├── server/
│   │   └── main.go
│   └── keyctl/
│       └── main.go
├── internal/
│   ├── domain/
│   │   ├── key.go
│   │   └── errors.go
│   ├── usecase/
│   │   └── key_service.go
│   ├── handler/
│   │   ├── key_handler.go
│   │   └── router.go
│   ├── repository/
│   │   └── key_repository.go
│   ├── infra/
│   │   ├── database.go
│   │   └── kms.go
│   └── middleware/
│       └── logging.go
├── pkg/
│   └── httputil/
│       └── response.go
├── config/
│   └── config.go
├── migrations/
│   └── 001_create_encryption_keys.sql
├── api/
│   └── openapi.yaml
├── .gitignore
├── go.mod
├── go.sum
├── Makefile
└── Dockerfile
```

## 実装の順序

1. プロジェクト初期化（go.mod, Makefile, .gitignore）
2. ドメイン層（domain/key.go, domain/errors.go）
3. 設定層（config/config.go）
4. インフラ層（infra/database.go, infra/kms.go）
5. リポジトリ層（repository/key_repository.go）
6. サービス層（usecase/key_service.go）
7. ハンドラ層（handler/key_handler.go, handler/router.go, middleware/logging.go）
8. HTTPユーティリティ（pkg/httputil/response.go）
9. APIサーバー（cmd/server/main.go）
10. マイグレーション（migrations/001_create_encryption_keys.sql）
11. CLI（cmd/keyctl/main.go）
12. ユニットテスト
13. Dockerfile

## セキュリティ考慮事項

- DEKはCloud KMSで暗号化した状態でのみ保存
- 鍵の平文をログに出力しない
- tenant_idはバリデーションで不正な文字を排除
- gormのプリペアドステートメントでSQLインジェクション対策
- 機密情報は環境変数から読み込み

## パフォーマンス考慮事項

- gormの接続プール設定（MaxOpenConns: 10, MaxIdleConns: 5）
- Cloud KMS API呼び出しはレイテンシがあるため非同期処理は検討しない（シンプルさ優先）
- tenant_id, generationへのインデックス設定済み

## 将来の拡張性

- APIバージョニング: URLパスに `/v1/` を含めている
- 環境変数による設定外部化
- OpenTelemetry対応の基盤は用意（Post-MVPで有効化）

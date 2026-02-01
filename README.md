# Key Management Service

マルチテナント対応の暗号鍵管理マイクロサービスです。エンベロープ暗号化パターンを採用し、Cloud KMSでKEK（鍵暗号化鍵）を管理、DEK（データ暗号化鍵）をCloud SQLに安全に保存します。

## 機能概要

- **鍵の生成**: テナントごとにAES-256暗号鍵を生成
- **鍵の取得**: 現在有効な鍵または特定世代の鍵を取得
- **鍵のローテーション**: 新しい世代の鍵を生成（既存の鍵は保持）
- **鍵の無効化**: 特定世代の鍵を論理削除
- **鍵一覧の取得**: テナントの全世代の鍵メタデータを取得

## 技術スタック

| 分類 | 技術 |
|------|------|
| 言語 | Go 1.25 |
| Webフレームワーク | chi |
| ORM | gorm |
| データベース | Cloud SQL (MySQL 8.4) |
| 暗号化 | Cloud KMS |
| トレーシング | OpenTelemetry / Cloud Trace |
| ロギング | slog (構造化ログ) |
| デプロイ | Cloud Run |

## 前提条件

- Go 1.25以上
- Docker (コンテナビルド用)
- Google Cloud SDK (`gcloud` CLI)
- Cloud SQL (MySQL 8.4) インスタンス
- Cloud KMS キーリング・暗号鍵

## インストール

### リポジトリのクローン

```bash
git clone <repository-url>
cd key-management/key-management-service
```

### 依存関係のインストール

```bash
go mod download
```

## ビルド

### ローカルビルド

```bash
make build
```

これにより `bin/` ディレクトリに以下のバイナリが生成されます：
- `bin/server` - REST APIサーバー
- `bin/keyctl` - CLIツール

### 個別ビルド

```bash
# APIサーバーのみ
go build -o bin/server ./cmd/server

# CLIツールのみ
go build -o bin/keyctl ./cmd/keyctl
```

### Dockerビルド

```bash
docker build -t key-management-service .
```

## 環境変数

### 必須

| 変数名 | 説明 | 例 |
|--------|------|-----|
| DATABASE_URL | Cloud SQL接続文字列 | `user:password@tcp(localhost:3306)/keydb?parseTime=true` |
| KMS_KEY_NAME | Cloud KMS暗号鍵リソース名 | `projects/my-project/locations/asia-northeast1/keyRings/my-keyring/cryptoKeys/my-key` |
| GOOGLE_CLOUD_PROJECT | GCPプロジェクトID | `my-project-id` |

### オプション

| 変数名 | デフォルト | 説明 |
|--------|-----------|------|
| PORT | 8080 | APIサーバーポート |
| LOG_LEVEL | INFO | ログレベル (DEBUG/INFO/WARN/ERROR) |
| OTEL_ENABLED | false | OpenTelemetryの有効化 |
| OTEL_SERVICE_NAME | key-management-service | サービス名 |
| OTEL_SAMPLING_RATE | 1.0 | サンプリング率 (0.0-1.0) |

### ローカル開発

`.env.example` をコピーして `.env` を作成し、環境変数を設定します：

```bash
cp .env.example .env
# .env を編集
```

## 実行

### APIサーバーの起動

```bash
# ローカルビルド後
./bin/server

# または直接実行
go run ./cmd/server
```

### Docker実行

```bash
docker run -p 8080:8080 \
  -e DATABASE_URL="..." \
  -e KMS_KEY_NAME="..." \
  -e GOOGLE_CLOUD_PROJECT="..." \
  key-management-service
```

## データベースマイグレーション

```bash
# マイグレーションの実行
./bin/keyctl migrate up

# マイグレーション状態の確認
./bin/keyctl migrate status
```

## CLI (keyctl) の使用方法

```bash
# 鍵の生成
keyctl create --tenant tenant-001

# 現在有効な鍵の取得
keyctl get --tenant tenant-001

# 特定世代の鍵の取得
keyctl get --tenant tenant-001 --generation 2

# 鍵のローテーション
keyctl rotate --tenant tenant-001

# 鍵一覧の取得
keyctl list --tenant tenant-001

# 鍵の無効化
keyctl disable --tenant tenant-001 --generation 1

# バージョン確認
keyctl version
```

### グローバルオプション

```bash
--api-url string   APIエンドポイントURL (環境変数 KEYCTL_API_URL でも設定可)
--output string    出力形式: text, json (デフォルト: text)
--timeout duration タイムアウト時間 (デフォルト: 30s)
```

## API エンドポイント

| メソッド | パス | 説明 |
|----------|------|------|
| POST | `/v1/tenants/{tenant_id}/keys` | 鍵の生成 |
| GET | `/v1/tenants/{tenant_id}/keys` | 鍵一覧の取得 |
| GET | `/v1/tenants/{tenant_id}/keys/current` | 現在有効な鍵の取得 |
| GET | `/v1/tenants/{tenant_id}/keys/{generation}` | 特定世代の鍵の取得 |
| DELETE | `/v1/tenants/{tenant_id}/keys/{generation}` | 鍵の無効化 |
| POST | `/v1/tenants/{tenant_id}/keys/rotate` | 鍵のローテーション |

## 開発

### コードフォーマット

```bash
make fmt
```

### Lint

```bash
make lint
```

### テスト

```bash
make test
```

### CI (フォーマット + Lint + テスト + ビルド)

```bash
make ci
```

### クリーンアップ

```bash
make clean
```

## ディレクトリ構造

```
key-management-service/
├── cmd/
│   ├── server/          # APIサーバーエントリポイント
│   └── keyctl/          # CLIツールエントリポイント
├── internal/
│   ├── domain/          # ドメインモデル
│   ├── usecase/         # ビジネスロジック
│   ├── handler/         # HTTPハンドラ
│   ├── repository/      # データアクセス
│   ├── infra/           # 外部サービス接続
│   └── middleware/      # HTTPミドルウェア
├── config/              # 設定
├── migrations/          # DBマイグレーション
└── pkg/                 # 公開パッケージ
```

## ドキュメント

詳細なドキュメントは `docs/` ディレクトリを参照してください：

- [プロダクト要求定義書](docs/product-requirements.md)
- [機能設計書](docs/functional-design.md)
- [アーキテクチャ設計書](docs/architecture.md)
- [リポジトリ構造定義書](docs/repository-structure.md)
- [開発ガイドライン](docs/development-guidelines.md)

## ライセンス

[ライセンスを記載]

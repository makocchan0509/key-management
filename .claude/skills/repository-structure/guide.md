# Go サーバーアプリケーション リポジトリ構造ガイド

このガイドは、Go でサーバーアプリケーションを構築する際のディレクトリ構成・ファイル命名規則・依存関係ルールをまとめたものです。

## プロジェクト構造例

```
project-root/
├── cmd/
│   └── server/
│       └── main.go              # エントリポイント
├── internal/
│   ├── domain/                  # ドメインモデル・ビジネスルール
│   │   ├── user.go
│   │   └── order.go
│   ├── usecase/                 # アプリケーションロジック
│   │   ├── user_service.go
│   │   └── order_service.go
│   ├── handler/                 # HTTPハンドラ (プレゼンテーション層)
│   │   ├── user_handler.go
│   │   └── order_handler.go
│   ├── repository/              # データアクセス実装
│   │   ├── user_repository.go
│   │   └── order_repository.go
│   ├── infra/                   # 外部サービス接続・技術的基盤
│   │   ├── database.go
│   │   └── redis.go
│   └── middleware/              # HTTPミドルウェア
│       ├── auth.go
│       └── logging.go
├── pkg/                         # 外部公開可能な共通パッケージ
│   └── httputil/
│       └── response.go
├── config/
│   └── config.go
├── migrations/
│   └── 001_create_users.sql
├── go.mod
├── go.sum
└── Makefile
```

## ディレクトリ詳細例

### cmd/ (エントリポイント)

**役割**: アプリケーションのエントリポイントを配置する。各サブディレクトリが1つの実行バイナリに対応する。

**配置ファイル**:
- `main.go`: アプリケーション起動処理のみ。DI・サーバー起動・シグナルハンドリングを行う

**命名規則**:
- サブディレクトリ名はバイナリ名と一致させる
- 例: `cmd/server/`, `cmd/worker/`, `cmd/migrate/`

**依存関係**:
- 依存可能: `internal/`, `pkg/`, `config/`
- `cmd/` 内のコードは薄く保ち、ロジックを持たせない

### internal/ (プロジェクト内部コード)

Go コンパイラが外部パッケージからの import を禁止するディレクトリ。プロジェクト固有のコードはすべてここに配置する。

#### internal/domain/

**役割**: ドメインモデル (エンティティ・値オブジェクト) とビジネスルールを定義する

**配置ファイル**:
- `{リソース名}.go`: エンティティの構造体定義・バリデーション・ドメインロジック

**命名規則**:
- ファイル名はリソース名の単数形 (snake_case)
- 例: `user.go`, `order.go`, `order_item.go`

**依存関係**:
- 依存可能: 標準ライブラリのみ
- 依存禁止: `internal/` 内の他のすべてのパッケージ

**例**:
```go
// internal/domain/user.go
package domain

import "errors"

type User struct {
    ID    string
    Name  string
    Email string
}

func (u *User) Validate() error {
    if u.Name == "" {
        return errors.New("name is required")
    }
    return nil
}
```

#### internal/usecase/

**役割**: アプリケーション固有のビジネスロジック (ユースケース) を実装する

**配置ファイル**:
- `{リソース名}_service.go`: ユースケースの実装とリポジトリインターフェースの定義

**命名規則**:
- `{リソース名}_service.go`
- 例: `user_service.go`, `order_service.go`

**依存関係**:
- 依存可能: `domain`
- 依存禁止: `handler`, `repository` (実装), `infra`

**インターフェース定義**: 使う側にインターフェースを定義する (Go の慣習)

```go
// internal/usecase/user_service.go
package usecase

import (
    "context"
    "project/internal/domain"
)

type UserRepository interface {
    FindByID(ctx context.Context, id string) (*domain.User, error)
    Save(ctx context.Context, user *domain.User) error
}

type UserService struct {
    repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
    return &UserService{repo: repo}
}
```

#### internal/handler/

**役割**: HTTPリクエストの受付・バリデーション・レスポンス返却を行う

**配置ファイル**:
- `{リソース名}_handler.go`: HTTPハンドラ関数
- `router.go`: ルーティング定義 (必要に応じて)

**命名規則**:
- `{リソース名}_handler.go`
- 例: `user_handler.go`, `order_handler.go`

**依存関係**:
- 依存可能: `usecase`, `domain`
- 依存禁止: `repository`, `infra`

#### internal/repository/

**役割**: データ永続化の具体的な実装を配置する

**配置ファイル**:
- `{リソース名}_repository.go`: リポジトリインターフェースの実装

**命名規則**:
- `{リソース名}_repository.go`
- 例: `user_repository.go`, `order_repository.go`

**依存関係**:
- 依存可能: `domain`, `infra`
- 依存禁止: `handler`, `usecase`

#### internal/infra/

**役割**: データベース接続、外部サービスクライアントなど技術的な基盤コードを配置する

**配置ファイル**:
- `database.go`: DB接続の初期化・管理
- `redis.go`: Redis接続の初期化・管理
- その他外部サービスとの接続

**依存関係**:
- 依存可能: 標準ライブラリ、外部ライブラリ
- 依存禁止: `domain`, `usecase`, `handler`

#### internal/middleware/

**役割**: HTTPミドルウェア (認証・ロギング・CORS等) を配置する

**配置ファイル**:
- 機能名をそのままファイル名にする
- 例: `auth.go`, `logging.go`, `cors.go`

### pkg/ (公開可能パッケージ)

**役割**: 他プロジェクトでも再利用可能な汎用コードを配置する

**使用方針**:
- 本当に汎用的なコードのみ配置する
- 不要なら作成しない。迷ったら `internal/` に置く

### config/ (設定)

**役割**: アプリケーション設定の読み込み・構造体定義

**配置ファイル**:
- `config.go`: 設定構造体と読み込みロジック

### migrations/ (DBマイグレーション)

**役割**: データベーススキーマのマイグレーションファイルを配置する

**命名規則**:
- `{連番}_{説明}.sql`
- 例: `001_create_users.sql`, `002_add_orders_table.sql`

## ファイル配置規則例

### ソースファイル

| ファイル種別 | 配置先 | 命名規則 | 例 |
|---|---|---|---|
| エントリポイント | `cmd/{app}/` | `main.go` | `cmd/server/main.go` |
| ドメインモデル | `internal/domain/` | `{リソース}.go` | `user.go` |
| ユースケース | `internal/usecase/` | `{リソース}_service.go` | `user_service.go` |
| HTTPハンドラ | `internal/handler/` | `{リソース}_handler.go` | `user_handler.go` |
| リポジトリ実装 | `internal/repository/` | `{リソース}_repository.go` | `user_repository.go` |
| ミドルウェア | `internal/middleware/` | `{機能名}.go` | `auth.go` |
| 基盤コード | `internal/infra/` | `{技術名}.go` | `database.go` |

### テストファイル

| テスト種別 | 配置先 | 命名規則 | 例 |
|---|---|---|---|
| ユニットテスト | 対象ファイルと同じディレクトリ | `{対象}_test.go` | `user_service_test.go` |
| 統合テスト | 対象ファイルと同じディレクトリ | `{対象}_test.go` (build tag で分離) | `//go:build integration` |

Go の慣習として、テストファイルはテスト対象と同じディレクトリに配置する。

## 命名規則例

### ディレクトリ名

- **snake_case または単純な小文字** を使用 (Go 標準)
- 短く、意味が明確な名前をつける
- 複数形は使わない (Go の慣習: `handler` not `handlers`)

### ファイル名

- **snake_case** を使用
- 1ファイルに1つの主要な型を定義することを目安にする
- 接尾辞でレイヤーを明示する (`_handler`, `_service`, `_repository`)

### Go の命名慣習

| 対象 | 規則 | 例 |
|---|---|---|
| エクスポートされる型・関数 | PascalCase | `UserService`, `NewUserService` |
| 非公開の型・関数 | camelCase | `validateEmail`, `dbClient` |
| インターフェース | 動詞 + er / 役割名 | `Reader`, `UserRepository` |
| コンストラクタ | `New` + 型名 | `NewUserService` |
| パッケージ名 | 小文字、短く、1単語 | `handler`, `domain`, `usecase` |

## 依存関係のルール

### レイヤー間の依存方向

```
handler → usecase → domain
                  ↗
repository ------/
infra → (外部ライブラリのみ)
```

**許可される依存**:
- `handler` → `usecase`, `domain`
- `usecase` → `domain`
- `repository` → `domain`, `infra`
- `middleware` → `domain` (必要な場合のみ)

**禁止される依存**:
- `domain` → 他の internal パッケージ
- `usecase` → `handler`, `repository` (実装), `infra`
- `handler` → `repository`, `infra`
- 循環依存は一切禁止

### インターフェースによる依存性逆転

リポジトリのインターフェースは **使う側 (usecase)** に定義し、実装は `repository` パッケージに置く。これにより `usecase` は具体的なデータアクセス実装に依存しない。

```
usecase (interface定義) ← repository (interface実装)
```

## スケーリング戦略

### 小〜中規模 (推奨初期構成)

上記のフラットなレイヤー構成をそのまま使用する。

### 大規模 (ドメイン分割)

リソースが増えた場合、`internal/` 以下をドメイン単位で分割する:

```
internal/
├── user/
│   ├── domain.go
│   ├── service.go
│   ├── handler.go
│   └── repository.go
├── order/
│   ├── domain.go
│   ├── service.go
│   ├── handler.go
│   └── repository.go
└── shared/
    └── middleware/
```

### ファイルサイズの目安

- 1ファイル: 300行以下を推奨
- 300-500行: 分割を検討
- 500行以上: 責務ごとに分割する

## 避けるべきアンチパターン

| アンチパターン | 理由 | 代替案 |
|---|---|---|
| `models/` パッケージ | 曖昧で肥大化する | `domain/` に配置 |
| `utils/`, `helpers/` | 無関係なコードの寄せ集めになる | 適切なパッケージに機能を配置 |
| `common/`, `shared/` の濫用 | 依存の方向が不明確になる | 必要最小限に留める |
| 実装が1つしかないインターフェース | 不要な抽象化 | テスト容易性が必要な場合のみ作成 |
| パッケージの循環参照 | コンパイルエラーになる | 依存方向を一方向に保つ |

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

# OS
.DS_Store

# テストカバレッジ
coverage.out
coverage.html

# ログ
*.log
```

## ドキュメント配置

### ドキュメントの種類と配置先

**docs/ ディレクトリ**:
- `product-requirements.md`: PRD
- `functional-design.md`: 機能設計書
- `architecture.md`: アーキテクチャ設計書
- `repository-structure.md`: 本ドキュメント
- `development-guidelines.md`: 開発ガイドライン
- `glossary.md`: 用語集

## チェックリスト

- [ ] 各ディレクトリの役割が明確に定義されている
- [ ] レイヤー構造がディレクトリに反映されている
- [ ] 命名規則が一貫している
- [ ] テストコードの配置方針が決まっている
- [ ] 依存関係のルールが明確である
- [ ] 循環依存がない
- [ ] スケーリング戦略が考慮されている
- [ ] 共有コードの配置ルールが定義されている
- [ ] 設定ファイルの管理方法が決まっている
- [ ] ドキュメントの配置場所が明確である
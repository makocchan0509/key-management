# 設計書

## アーキテクチャ概要

マイグレーション機能は、既存のレイヤードアーキテクチャに従って実装します。

```
CLI (keyctl migrate) → UseCase → Repository → Infra (Database)
                                    ↓
                              Domain (Migration)
```

## コンポーネント設計

### 1. internal/domain/migration.go

**責務**:
- マイグレーションドメインモデルの定義
- マイグレーションステータス（pending/applied）の定義
- マイグレーションエラーの定義

**実装の要点**:
- `Migration`構造体: バージョン、名前、適用日時、ファイルパスを保持
- `MigrationStatus`型: pending/appliedのenum定義

### 2. internal/repository/migration_repository.go

**責務**:
- `schema_migrations`テーブルへのアクセス
- 適用済みマイグレーション履歴の管理

**実装の要点**:
- `FindAllApplied()`: 適用済みマイグレーション一覧を取得
- `RecordMigration()`: マイグレーション適用履歴を記録
- `IsMigrationApplied()`: マイグレーションが適用済みか確認
- gormを使用したデータアクセス

### 3. internal/usecase/migration_service.go

**責務**:
- マイグレーション実行のビジネスロジック
- ファイルシステムからのマイグレーションファイル読み込み
- マイグレーションの順序制御

**実装の要点**:
- `ApplyMigrations()`: 未適用マイグレーションを番号順に実行
- `GetMigrationStatus()`: 現在のマイグレーション状況を取得
- `migrations/`ディレクトリから.sqlファイルをスキャン
- トランザクション内でSQL実行とレコード登録を実行

### 4. cmd/keyctl/migrate.go

**責務**:
- `migrate`サブコマンドの実装
- `up`, `status`サブコマンドのハンドリング

**実装の要点**:
- Cobraを使用したCLI実装
- 環境変数からDB接続情報を取得
- エラーハンドリングとユーザーフレンドリーな出力

## データフロー

### マイグレーション実行（keyctl migrate up）
```
1. CLIコマンド実行
2. 環境変数からDB接続情報を取得
3. MigrationServiceのApplyMigrations()を呼び出し
4. migrations/ディレクトリから.sqlファイルをスキャン
5. 適用済みマイグレーション履歴を取得
6. 未適用マイグレーションをフィルタリング
7. 各マイグレーションに対して:
   a. トランザクション開始
   b. SQLファイルの内容を実行
   c. schema_migrationsに履歴を記録
   d. トランザクションコミット
8. 結果を標準出力に表示
```

### マイグレーションステータス確認（keyctl migrate status）
```
1. CLIコマンド実行
2. 環境変数からDB接続情報を取得
3. MigrationServiceのGetMigrationStatus()を呼び出し
4. migrations/ディレクトリから全マイグレーションファイルをスキャン
5. 適用済みマイグレーション履歴を取得
6. マイグレーションごとにステータス（applied/pending）を判定
7. テーブル形式で標準出力に表示
```

## データベース設計

### schema_migrationsテーブル

```sql
CREATE TABLE schema_migrations (
    version VARCHAR(14) NOT NULL,
    applied_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

- `version`: マイグレーションバージョン（例: "001", "002"）
- `applied_at`: 適用日時

## エラーハンドリング戦略

### カスタムエラークラス

`internal/domain/errors.go`に追加:
- `ErrMigrationFailed`: マイグレーション実行時のエラー
- `ErrMigrationFileNotFound`: マイグレーションファイルが見つからない
- `ErrInvalidMigrationFile`: マイグレーションファイルのフォーマットが不正

### エラーハンドリングパターン

- SQLエラーはトランザクションをロールバックし、適用を中断
- ファイルシステムエラーは適切なエラーメッセージを出力
- 既に適用済みのマイグレーションは静かにスキップ

## テスト戦略

### ユニットテスト

- `internal/usecase/migration_service_test.go`
  - ApplyMigrations()の正常系・異常系
  - GetMigrationStatus()の正常系
  - mockMigrationRepositoryを使用

### 統合テスト（スコープ外）

今回は実装しませんが、将来的には以下が考えられます:
- 実際のデータベースに対するマイグレーション適用テスト
- CLIコマンドのend-to-endテスト

## 依存ライブラリ

新しい依存ライブラリは不要です。既存のライブラリを使用:
- `gorm.io/gorm`: データベースアクセス
- `github.com/spf13/cobra`: CLI実装
- `os`, `path/filepath`, `sort`: マイグレーションファイルスキャン

## ディレクトリ構造

```
key-management-service/
├── cmd/
│   └── keyctl/
│       ├── main.go (既存)
│       └── migrate.go (新規: migrate サブコマンド)
├── internal/
│   ├── domain/
│   │   ├── migration.go (新規: ドメインモデル)
│   │   └── errors.go (更新: マイグレーションエラー追加)
│   ├── repository/
│   │   └── migration_repository.go (新規: マイグレーション履歴管理)
│   └── usecase/
│       ├── migration_service.go (新規: ビジネスロジック)
│       └── migration_service_test.go (新規: ユニットテスト)
└── migrations/
    ├── 000_create_schema_migrations.sql (新規: schema_migrationsテーブル作成)
    └── 001_create_encryption_keys.sql (既存)
```

## 実装の順序

1. `migrations/000_create_schema_migrations.sql`を作成
2. `internal/domain/migration.go`を実装
3. `internal/domain/errors.go`にマイグレーションエラーを追加
4. `internal/repository/migration_repository.go`を実装
5. `internal/usecase/migration_service.go`を実装
6. `internal/usecase/migration_service_test.go`を実装
7. `cmd/keyctl/migrate.go`を実装
8. テスト実行と動作確認

## セキュリティ考慮事項

- SQLインジェクション対策: マイグレーションファイルは信頼できるソースからのみ読み込む
- パスインジェクション対策: マイグレーションディレクトリ外のファイルを読み込まないよう検証
- 環境変数からのDB接続情報は安全に扱う（ログに出力しない）

## パフォーマンス考慮事項

- マイグレーションファイルのスキャンはO(n)だが、通常ファイル数は少ないため問題なし
- 各マイグレーションは個別のトランザクション内で実行（失敗時の影響を最小化）
- 大規模なマイグレーションの場合、タイムアウト設定が必要（将来的な拡張）

## 将来の拡張性

以下の機能は将来的に追加可能:
- マイグレーションのロールバック（down migration）
- マイグレーションファイルの自動生成（`keyctl migrate create`）
- ドライラン機能（`keyctl migrate up --dry-run`）
- マイグレーションのバージョン指定実行（`keyctl migrate up --to=003`）

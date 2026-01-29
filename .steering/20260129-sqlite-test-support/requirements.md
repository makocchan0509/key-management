# 要求内容

## 概要

key_repository.goのユニットテストにおけるカバレッジを高めるため、テスト実行時にのみSQLiteを使用できるようにします。

## 背景

現在、`internal/repository/key_repository.go`にはユニットテストが存在せず、テストカバレッジが0%です。リポジトリ層のロジック（SQL構築、エラーハンドリング、ドメインモデル変換）の品質を保証するため、インメモリSQLiteデータベースを使用したユニットテストが必要です。

既に`migration_service_test.go`でSQLite（gorm.io/driver/sqlite）を使用したテストパターンが確立されているため、同じアプローチを`key_repository_test.go`に適用します。

## 実装対象の機能

### 1. key_repository_test.goの作成
- インメモリSQLiteデータベースを使用したユニットテストを実装
- 全メソッド（ExistsByTenantID, Create, FindByTenantIDAndGeneration, FindLatestActiveByTenantID, FindAllByTenantID, GetMaxGeneration, UpdateStatus）のテストを実装
- 正常系・異常系の両方をカバー

### 2. テストヘルパー関数の実装
- setupTestDB(): テスト用のインメモリSQLiteデータベースを初期化
- テーブル作成スクリプトの実行
- テストデータのセットアップ

## 受け入れ条件

### key_repository_test.go
- [ ] ExistsByTenantIDのテストが実装されている（存在する/存在しない）
- [ ] Createのテストが実装されている（正常系、UUID生成、タイムスタンプ反映）
- [ ] FindByTenantIDAndGenerationのテストが実装されている（存在する/存在しない）
- [ ] FindLatestActiveByTenantIDのテストが実装されている（最新有効鍵、無効鍵は除外、鍵なし）
- [ ] FindAllByTenantIDのテストが実装されている（複数鍵、世代順ソート）
- [ ] GetMaxGenerationのテストが実装されている（鍵あり、鍵なし）
- [ ] UpdateStatusのテストが実装されている（ステータス更新）
- [ ] テストがCGO_ENABLED=1で実行可能
- [ ] テストカバレッジが80%以上

## 成功指標

- `key_repository.go`のテストカバレッジが80%以上
- 全テストがパスする
- migration_service_test.goと同様のテストパターンが適用されている

## スコープ外

以下はこのフェーズでは実装しません:

- MySQL統合テストの実装（インメモリSQLiteのみ）
- handler層やusecase層のテスト修正
- テーブル駆動テストへのリファクタリング

## 参照ドキュメント

- `docs/development-guidelines.md` - 開発ガイドライン
- `internal/usecase/migration_service_test.go` - SQLiteテストのリファレンス実装
- `internal/repository/key_repository.go` - テスト対象コード

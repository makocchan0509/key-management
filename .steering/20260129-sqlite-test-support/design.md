# 設計書

## アーキテクチャ概要

既存の`migration_service_test.go`で確立されたSQLiteテストパターンを`key_repository_test.go`に適用します。

```
Test → setupTestDB() → SQLite in-memory DB → KeyRepository
```

## コンポーネント設計

### 1. setupTestDB()

**責務**:
- インメモリSQLiteデータベースを初期化
- encryption_keysテーブルを作成
- テスト用のgorm.DB接続を返す

**実装の要点**:
- `gorm.io/driver/sqlite`を使用
- `:memory:`でインメモリDB作成
- 既存の`migrations/001_create_encryption_keys.sql`をベースにテーブル作成（SQLite用に調整）
- SQLiteではENUM型が非対応のため、TEXT型を使用

### 2. テストケース

**責務**:
- KeyRepositoryの各メソッドをテスト
- 正常系・異常系の両方をカバー

**実装の要点**:
- 各テストで独立したDBインスタンスを使用（t.Cleanup()でクリーンアップ）
- テストデータは各テストケース内で準備
- migration_service_test.goと同様のパターンを適用

## データフロー

### テスト実行フロー
```
1. setupTestDB()でインメモリSQLiteを初期化
2. encryption_keysテーブルを作成
3. テストデータを挿入
4. KeyRepositoryのメソッドを実行
5. 結果を検証
```

## エラーハンドリング戦略

### テストエラーハンドリング

- セットアップエラーは`t.Fatalf()`で即座に失敗
- アサーションエラーは`t.Errorf()`で記録し、他のテストを継続

## テスト戦略

### ユニットテスト

各メソッドのテストケース:

1. **ExistsByTenantID**
   - テナントに鍵が存在する場合: true
   - テナントに鍵が存在しない場合: false

2. **Create**
   - 正常系: 鍵が作成される
   - UUID自動生成: IDが空の場合にUUIDが生成される
   - タイムスタンプ反映: CreatedAt/UpdatedAtがドメインエンティティに反映される

3. **FindByTenantIDAndGeneration**
   - 鍵が存在する場合: ドメインエンティティを返す
   - 鍵が存在しない場合: nilを返す

4. **FindLatestActiveByTenantID**
   - 最新有効鍵を返す
   - 無効鍵は除外される
   - 鍵がない場合: nilを返す

5. **FindAllByTenantID**
   - 複数鍵を世代順に返す
   - 鍵がない場合: 空配列を返す

6. **GetMaxGeneration**
   - 鍵がある場合: 最大世代番号を返す
   - 鍵がない場合: 0を返す

7. **UpdateStatus**
   - ステータスが更新される

## 依存ライブラリ

既に追加済み:
- `gorm.io/driver/sqlite` (migration_service_test.goで追加済み)
- `gorm.io/gorm`

## ディレクトリ構造

```
key-management-service/
└── internal/
    └── repository/
        ├── key_repository.go (既存)
        ├── key_repository_test.go (新規)
        └── migration_repository.go (既存)
```

## 実装の順序

1. `key_repository_test.go`を作成
2. setupTestDB()ヘルパー関数を実装
3. 各メソッドのテストケースを実装:
   - ExistsByTenantID
   - Create
   - FindByTenantIDAndGeneration
   - FindLatestActiveByTenantID
   - FindAllByTenantID
   - GetMaxGeneration
   - UpdateStatus
4. テスト実行と動作確認

## セキュリティ考慮事項

- テストコードであり、セキュリティリスクは低い
- インメモリDBのため、データ永続化のリスクなし

## パフォーマンス考慮事項

- インメモリSQLiteのため高速
- 各テストで独立したDBインスタンスを使用するため、テスト間の干渉なし

## 将来の拡張性

- 将来的にMySQL統合テストを追加する場合は、testcontainersライブラリの使用を検討
- テーブル駆動テストへのリファクタリングも検討可能

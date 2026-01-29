# タスクリスト

## 🚨 タスク完全完了の原則

**このファイルの全タスクが完了するまで作業を継続すること**

### 必須ルール
- **全てのタスクを`[x]`にすること**
- 「時間の都合により別タスクとして実施予定」は禁止
- 「実装が複雑すぎるため後回し」は禁止
- 未完了タスク（`[ ]`）を残したまま作業を終了しない

### 実装可能なタスクのみを計画
- 計画段階で「実装可能なタスク」のみをリストアップ
- 「将来やるかもしれないタスク」は含めない
- 「検討中のタスク」は含めない

### タスクスキップが許可される唯一のケース
以下の技術的理由に該当する場合のみスキップ可能:
- 実装方針の変更により、機能自体が不要になった
- アーキテクチャ変更により、別の実装方法に置き換わった
- 依存関係の変更により、タスクが実行不可能になった

スキップ時は必ず理由を明記:
```markdown
- [x] ~~タスク名~~（実装方針変更により不要: 具体的な技術的理由）
```

### タスクが大きすぎる場合
- タスクを小さなサブタスクに分割
- 分割したサブタスクをこのファイルに追加
- サブタスクを1つずつ完了させる

---

## フェーズ1: テストファイルとヘルパー関数の作成

- [x] internal/repository/key_repository_test.go を作成
  - [x] パッケージとインポートを定義
  - [x] setupTestDB() ヘルパー関数を実装
    - [x] インメモリSQLite接続を作成
    - [x] encryption_keysテーブルを作成（SQLite用にENUM→TEXT変換）

## フェーズ2: ExistsByTenantIDのテスト実装

- [x] TestKeyRepository_ExistsByTenantID を実装
  - [x] テナントに鍵が存在する場合のテスト
  - [x] テナントに鍵が存在しない場合のテスト

## フェーズ3: Createのテスト実装

- [x] TestKeyRepository_Create を実装
  - [x] 正常系: 鍵が作成されることを確認
  - [x] UUID自動生成: IDが空の場合にUUIDが生成されることを確認
  - [x] タイムスタンプ反映: CreatedAt/UpdatedAtがドメインエンティティに反映されることを確認

## フェーズ4: FindByTenantIDAndGenerationのテスト実装

- [x] TestKeyRepository_FindByTenantIDAndGeneration を実装
  - [x] 鍵が存在する場合のテスト
  - [x] 鍵が存在しない場合のテスト

## フェーズ5: FindLatestActiveByTenantIDのテスト実装

- [x] TestKeyRepository_FindLatestActiveByTenantID を実装
  - [x] 最新有効鍵を返すテスト
  - [x] 無効鍵は除外されることを確認
  - [x] 鍵がない場合のテスト

## フェーズ6: FindAllByTenantIDのテスト実装

- [x] TestKeyRepository_FindAllByTenantID を実装
  - [x] 複数鍵を世代順に返すテスト
  - [x] 鍵がない場合のテスト

## フェーズ7: GetMaxGenerationのテスト実装

- [x] TestKeyRepository_GetMaxGeneration を実装
  - [x] 鍵がある場合のテスト
  - [x] 鍵がない場合のテスト

## フェーズ8: UpdateStatusのテスト実装

- [x] TestKeyRepository_UpdateStatus を実装
  - [x] ステータスが更新されることを確認

## フェーズ9: 品質チェックと修正

- [x] すべてのテストが通ることを確認
  - [x] `CGO_ENABLED=1 go test ./internal/repository/...`
- [x] テストカバレッジを確認
  - [x] `CGO_ENABLED=1 go test -cover ./internal/repository/...`
  - [x] カバレッジが80%以上であることを確認
- [x] フォーマットチェック
  - [x] `gofmt -l .`
- [x] Lintチェック
  - [x] `go vet ./...`
- [x] ビルドが成功することを確認
  - [x] `go build ./cmd/...`

---

## 実装後の振り返り

### 実装完了日
2026-01-29

### 計画と実績の差分

**計画と異なった点**:
- 特になし（計画通りに完了）

**新たに必要になったタスク**:
- 特になし（migration_service_test.goで既にsqlite依存関係が追加されていたため、追加の依存関係は不要だった）

**技術的理由でスキップしたタスク**:
- なし（全タスク完了）

**⚠️ 注意**: 「時間の都合」「難しい」などの理由でスキップしたタスクはここに記載しないこと。全タスク完了が原則。

### 学んだこと

**技術的な学び**:
- SQLiteのENUM型非対応問題とTEXT型への変換方法
- gorm.io/driver/sqliteを使用したインメモリDBテストパターン
- key_repository.goの各メソッドのロジック（SQL構築、エラーハンドリング、ドメインモデル変換）の動作確認
- テストデータのセットアップとクリーンアップパターン

**プロセス上の改善点**:
- migration_service_test.goを参照することで、既存パターンとの整合性を保ちながら実装できた
- 各メソッドごとにテスト関数を分けることで、エラー発生時の原因特定が容易
- setupTestDB()ヘルパー関数により、テストごとに独立したDB環境を簡単に作成できた

### 次回への改善提案
- テーブル駆動テストパターンを採用すると、テストケースの追加が容易になる
- setupTestDB()を共通ヘルパーとして`internal/repository/testutil/`パッケージに切り出すことで、コード重複を削減できる
- エラーケース（UNIQUE制約違反、DB接続エラー等）のテストも追加すると、より堅牢性が向上する

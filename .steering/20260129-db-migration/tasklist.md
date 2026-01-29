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

## フェーズ1: マイグレーションテーブル作成

- [x] migrations/000_create_schema_migrations.sql を作成
  - [x] CREATE TABLE文を記述（version, applied_at）
  - [x] PRIMARY KEYを定義

## フェーズ2: ドメイン層の実装

- [x] internal/domain/migration.go を実装
  - [x] MigrationStatus 型（pending/applied）を定義
  - [x] Migration 構造体を定義（Version, Name, AppliedAt, FilePath）

- [x] internal/domain/errors.go を更新
  - [x] ErrMigrationFailed を定義
  - [x] ErrMigrationFileNotFound を定義
  - [x] ErrInvalidMigrationFile を定義

## フェーズ3: リポジトリ層の実装

- [x] internal/repository/migration_repository.go を実装
  - [x] SchemaMigrationModel 構造体を定義（gormタグ付き）
  - [x] MigrationRepository 構造体を定義
  - [x] NewMigrationRepository() 関数を実装
  - [x] FindAllApplied() メソッドを実装
  - [x] RecordMigration() メソッドを実装
  - [x] IsMigrationApplied() メソッドを実装

## フェーズ4: サービス層の実装

- [x] internal/usecase/migration_service.go を実装
  - [x] MigrationRepository インターフェースを定義
  - [x] MigrationService 構造体を定義
  - [x] NewMigrationService() 関数を実装
  - [x] scanMigrationFiles() 関数を実装（migrations/ディレクトリスキャン）
  - [x] parseMigrationFileName() 関数を実装（ファイル名からバージョン抽出）
  - [x] ApplyMigrations() メソッドを実装
  - [x] GetMigrationStatus() メソッドを実装

## フェーズ5: CLI実装

- [x] cmd/keyctl/migrate.go を実装
  - [x] migrateCmd を定義
  - [x] migrateUpCmd を実装（`keyctl migrate up`）
  - [x] migrateStatusCmd を実装（`keyctl migrate status`）
  - [x] DB接続情報を環境変数から取得
  - [x] MigrationService呼び出しとエラーハンドリング
  - [x] 結果の整形と標準出力

- [x] cmd/keyctl/main.go を更新
  - [x] migrateCmd をrootCmdに追加

## フェーズ6: ユニットテストの実装

- [x] internal/usecase/migration_service_test.go を実装
  - [x] mockMigrationRepository を作成
  - [x] TestMigrationService_ApplyMigrations を実装
  - [x] TestMigrationService_ApplyMigrations_AlreadyApplied を実装
  - [x] TestMigrationService_ApplyMigrations_Error を実装
  - [x] TestMigrationService_GetMigrationStatus を実装

## フェーズ7: 品質チェックと修正

- [x] すべてのテストが通ることを確認
  - [x] `go test ./...`
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
- テスト実行にCGOが必要（sqlite driverを使用するため）: Makefileの`CGO_ENABLED=0`設定と矛盾するが、migration_service_test.goでsqliteを使用するため、テスト時のみCGO_ENABLED=1が必要
- 環境変数名の不一致を検出: 設計書では`DATABASE_URL`だったが、実装では`DB_DSN`を使用していた（implementation-validatorで検出し修正）

**新たに必要になったタスク**:
- gorm.io/driver/sqliteの依存関係追加（テスト用）
- 環境変数名を`DATABASE_URL`に統一（validator指摘により修正）

**技術的理由でスキップしたタスク**:
- なし（全タスク完了）

**⚠️ 注意**: 「時間の都合」「難しい」などの理由でスキップしたタスクはここに記載しないこと。全タスク完了が原則。

### 学んだこと

**技術的な学び**:
- gormのトランザクション内でのSQL実行とレコード登録の実装方法
- os.ReadDirとfilepath.Joinを使ったディレクトリスキャン
- cobraを使った複数レベルのサブコマンド実装（`keyctl migrate up`）
- sqlite driverを使用したインメモリDBでのテスト戦略
- モックリポジトリの実装パターン

**プロセス上の改善点**:
- tasklist.mdをリアルタイムで更新することで、進捗が常に可視化された
- フェーズを明確に分けることで、依存関係のあるタスクを順序立てて実装できた
- implementation-validatorによる品質検証が有効だった（環境変数名の不一致を検出）
- steering スキルに従うことで、タスク漏れが防止された

### 次回への改善提案
- テスト用のsqliteドライバの依存関係は最初から計画に含めるべき（go.modの更新が必要になることを想定）
- 環境変数名は設計段階でarchitecture.mdと照合して一貫性を確保する
- repository層のユニットテストも計画に含める（今回は時間の関係で未実装）
- インライン構造体の重複定義を避ける設計パターンを検討する（リポジトリメソッドのトランザクション対応）

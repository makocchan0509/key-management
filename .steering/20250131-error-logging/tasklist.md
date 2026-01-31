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

## フェーズ1: infrastructureレイヤーのエラーログ実装

### 1-1. kms.goのエラーログ実装

- [x] `internal/infra/kms.go`を読み込む
- [x] `Encrypt()`メソッドにエラーログを追加
  - [x] エラー発生時にslog.ErrorContextを呼び出す
  - [x] ログに"operation": "kms_encrypt"を含める
  - [x] ログに"key_name"を含める
  - [x] ログに"error"を含める
  - [x] plaintextをログに出力していないことを確認
- [x] `Decrypt()`メソッドにエラーログを追加
  - [x] エラー発生時にslog.ErrorContextを呼び出す
  - [x] ログに"operation": "kms_decrypt"を含める
  - [x] ログに"key_name"を含める
  - [x] ログに"error"を含める
  - [x] ciphertextをログに出力していないことを確認

### 1-2. database.goのエラーログ実装

- [x] `internal/infra/database.go`を読み込む
- [x] `NewDB()`関数の1つ目のエラー箇所にログを追加
  - [x] gorm.Open()のエラー時にslog.Errorを呼び出す
  - [x] ログに"operation": "db_init"を含める
  - [x] ログに"error"を含める
  - [x] DSNをログに出力していないことを確認
- [x] `NewDB()`関数の2つ目のエラー箇所にログを追加
  - [x] db.DB()のエラー時にslog.Errorを呼び出す
  - [x] ログに"operation": "db_init"を含める
  - [x] ログに"error"を含める

## フェーズ2: repositoryレイヤーのエラーログ実装

### 2-1. key_repository.goのエラーログ実装

- [x] `internal/repository/key_repository.go`を読み込む
- [x] `ExistsByTenantID()`メソッドにエラーログを追加
  - [x] エラー発生時にslog.ErrorContextを呼び出す
  - [x] ログに"operation": "exists_by_tenant_id"を含める
  - [x] ログに"tenant_id"を含める
  - [x] ログに"error"を含める
- [x] `Create()`メソッドにエラーログを追加
  - [x] エラー発生時にslog.ErrorContextを呼び出す
  - [x] ログに"operation": "create"を含める
  - [x] ログに"tenant_id"を含める
  - [x] ログに"generation"を含める
  - [x] ログに"error"を含める
- [x] `FindByTenantIDAndGeneration()`メソッドにエラーログを追加
  - [x] gorm.ErrRecordNotFoundの場合はログ出力せずnilを返す
  - [x] それ以外のエラー時にslog.ErrorContextを呼び出す
  - [x] ログに"operation": "find_by_tenant_id_and_generation"を含める
  - [x] ログに"tenant_id"を含める
  - [x] ログに"generation"を含める
  - [x] ログに"error"を含める
- [x] `FindLatestActiveByTenantID()`メソッドにエラーログを追加
  - [x] gorm.ErrRecordNotFoundの場合はログ出力せずnilを返す
  - [x] それ以外のエラー時にslog.ErrorContextを呼び出す
  - [x] ログに"operation": "find_latest_active_by_tenant_id"を含める
  - [x] ログに"tenant_id"を含める
  - [x] ログに"error"を含める
- [x] `FindAllByTenantID()`メソッドにエラーログを追加
  - [x] エラー発生時にslog.ErrorContextを呼び出す
  - [x] ログに"operation": "find_all_by_tenant_id"を含める
  - [x] ログに"tenant_id"を含める
  - [x] ログに"error"を含める
- [x] `GetMaxGeneration()`メソッドにエラーログを追加
  - [x] エラー発生時にslog.ErrorContextを呼び出す
  - [x] ログに"operation": "get_max_generation"を含める
  - [x] ログに"tenant_id"を含める
  - [x] ログに"error"を含める
- [x] `UpdateStatus()`メソッドにエラーログを追加
  - [x] エラー発生時にslog.ErrorContextを呼び出す
  - [x] ログに"operation": "update_status"を含める
  - [x] ログに"id"を含める
  - [x] ログに"status"を含める
  - [x] ログに"error"を含める

### 2-2. migration_repository.goのエラーログ実装

- [x] `internal/repository/migration_repository.go`を読み込む
- [x] `IsMigrationApplied()`メソッドにエラーログを追加
  - [x] エラー時にslog.ErrorContextを呼び出す
  - [x] ログに"operation": "is_migration_applied"を含める
  - [x] ログに"version"を含める
  - [x] ログに"error"を含める
- [x] `RecordMigration()`メソッドにエラーログを追加
  - [x] エラー発生時にslog.ErrorContextを呼び出す
  - [x] ログに"operation": "record_migration"を含める
  - [x] ログに"version"を含める
  - [x] ログに"error"を含める
- [x] `FindAllApplied()`メソッドにエラーログを追加
  - [x] エラー発生時にslog.ErrorContextを呼び出す
  - [x] ログに"operation": "find_all_applied"を含める
  - [x] ログに"error"を含める

## フェーズ3: usecaseレイヤーのエラーログ実装

### 3-1. key_service.goのエラーログ実装

- [x] `internal/usecase/key_service.go`を読み込む
- [x] `CreateKey()`メソッドにエラーログを追加
  - [x] repo.ExistsByTenantID()のエラー時にslog.ErrorContextを呼び出す
  - [x] existsがtrueの場合にslog.WarnContextを呼び出す（ドメインエラー）
  - [x] kmsClient.Encrypt()のエラー時にslog.ErrorContextを呼び出す
  - [x] repo.Create()のエラー時にslog.ErrorContextを呼び出す
  - [x] 全てのログに"operation": "create_key"を含める
  - [x] 全てのログに"tenant_id"を含める
  - [x] plainKeyをログに出力していないことを確認
- [x] `GetCurrentKey()`メソッドにエラーログを追加
  - [x] repo.FindLatestActiveByTenantID()のエラー時にslog.ErrorContextを呼び出す
  - [x] keyがnilの場合にslog.WarnContextを呼び出す（ドメインエラー）
  - [x] kmsClient.Decrypt()のエラー時にslog.ErrorContextを呼び出す
  - [x] 全てのログに"operation": "get_current_key"を含める
  - [x] 全てのログに"tenant_id"を含める
  - [x] plainKeyをログに出力していないことを確認
- [x] `GetKeyByGeneration()`メソッドにエラーログを追加
  - [x] repo.FindByTenantIDAndGeneration()のエラー時にslog.ErrorContextを呼び出す
  - [x] keyがnilの場合にslog.WarnContextを呼び出す（ドメインエラー）
  - [x] key.Status == disabledの場合にslog.WarnContextを呼び出す（ドメインエラー）
  - [x] kmsClient.Decrypt()のエラー時にslog.ErrorContextを呼び出す
  - [x] 全てのログに"operation": "get_key_by_generation"を含める
  - [x] 全てのログに"tenant_id"と"generation"を含める
  - [x] plainKeyをログに出力していないことを確認
- [x] `RotateKey()`メソッドにエラーログを追加
  - [x] repo.GetMaxGeneration()のエラー時にslog.ErrorContextを呼び出す
  - [x] maxGen == 0の場合にslog.WarnContextを呼び出す（ドメインエラー）
  - [x] kmsClient.Encrypt()のエラー時にslog.ErrorContextを呼び出す
  - [x] repo.Create()のエラー時にslog.ErrorContextを呼び出す
  - [x] 全てのログに"operation": "rotate_key"を含める
  - [x] 全てのログに"tenant_id"を含める
  - [x] plainKeyをログに出力していないことを確認
- [x] `ListKeys()`メソッドにエラーログを追加
  - [x] repo.FindAllByTenantID()のエラー時にslog.ErrorContextを呼び出す
  - [x] ログに"operation": "list_keys"を含める
  - [x] ログに"tenant_id"を含める
- [x] `DisableKey()`メソッドにエラーログを追加
  - [x] repo.FindByTenantIDAndGeneration()のエラー時にslog.ErrorContextを呼び出す
  - [x] keyがnilの場合にslog.WarnContextを呼び出す（ドメインエラー）
  - [x] key.Status == disabledの場合にslog.WarnContextを呼び出す（ドメインエラー）
  - [x] repo.UpdateStatus()のエラー時にslog.ErrorContextを呼び出す
  - [x] 全てのログに"operation": "disable_key"を含める
  - [x] 全てのログに"tenant_id"と"generation"を含める

### 3-2. migration_service.goのエラーログ実装

- [x] `internal/usecase/migration_service.go`を読み込む
- [x] `ApplyMigrations()`メソッドにエラーログを追加
  - [x] scanMigrationFiles()のエラー時にslog.Errorを呼び出す
  - [x] repo.IsMigrationApplied()のエラー時にslog.ErrorContextを呼び出す
  - [x] migration実行のエラー時にslog.ErrorContextを呼び出す
  - [x] 全てのログに"operation": "apply_migrations"を含める
  - [x] マイグレーション固有のログに"version"を含める
- [x] `applyMigration()`メソッドにエラーログを追加
  - [x] os.ReadFile()のエラー時にslog.ErrorContextを呼び出す
  - [x] tx.Exec()のエラー時にslog.ErrorContextを呼び出す
  - [x] tx.Create()のエラー時にslog.ErrorContextを呼び出す
  - [x] 全てのログに"operation": "apply_migration"を含める
  - [x] 全てのログに"version"を含める
- [x] `GetMigrationStatus()`メソッドにエラーログを追加
  - [x] repo.FindAllApplied()のエラー時にslog.ErrorContextを呼び出す
  - [x] ログに"operation": "get_migration_status"を含める

## フェーズ4: 品質チェックと修正

- [x] すべてのテストが通ることを確認
  - [x] `cd key-management-service && go test ./...`を実行
  - [x] テスト結果を確認し、エラーがないことを確認
- [x] リントエラーがないことを確認
  - [x] `cd key-management-service && golangci-lint run`を実行
  - [x] リント結果を確認し、エラーがないことを確認
- [x] セキュリティチェック
  - [x] `grep -r "plainKey" key-management-service/internal/ | grep slog`を実行
  - [x] 平文鍵がログに出力されていないことを確認
  - [x] `grep -r "plaintext" key-management-service/internal/ | grep slog`を実行
  - [x] 平文データがログに出力されていないことを確認（KMS関連のコンテキスト情報は除く）
- [x] ビルドが成功することを確認
  - [x] `cd key-management-service && go build ./cmd/...`を実行
  - [x] ビルド結果を確認し、エラーがないことを確認

## フェーズ5: ドキュメント更新

- [x] このファイル（tasklist.md）の「実装後の振り返り」セクションを更新
  - [x] 実装完了日を記録
  - [x] 計画と実績の差分を記録
  - [x] 学んだことを記録
  - [x] 次回への改善提案を記録

---

## 実装後の振り返り

### 実装完了日
2026-01-31

### 計画と実績の差分

**計画と異なった点**:
- migration_repository.goのメソッド名がタスクリストと異なっていた（IsApplied → IsMigrationApplied、GetAppliedMigrations → FindAllApplied）
  - 実際のコードに合わせてログを実装した
- migration_service.goでは、ApplyMigrations()内でscanMigrationFiles()のエラーをログ出力する必要があった
  - タスクリストではos.ReadDir()と記載されていたが、実際はscanMigrationFiles()メソッド経由でReadDirが呼ばれる構造だった

**新たに必要になったタスク**:
- 未使用のimport文の削除（migration_repository.goのerrorsパッケージ）
  - テスト実行時にコンパイルエラーが発生したため、修正が必要だった

**技術的理由でスキップしたタスク**:
- なし（全タスク完了）

### 学んだこと

**技術的な学び**:
- Go言語のslogパッケージを使った構造化ログの実装方法
  - slog.ErrorContext()とslog.WarnContext()の使い分け
  - システムエラー（インフラ障害）にはslog.Error、ドメインエラー（ビジネスルール違反）にはslog.Warnを使用
- エラーログにコンテキスト情報（tenant_id、generation、operation等）を含めることで、デバッグが容易になる
- gorm.ErrRecordNotFoundは通常フローのため、ログ出力不要という設計判断
- セキュリティ要件：平文鍵やDSNをログに出力しないための実装パターン

**プロセス上の改善点**:
- tasklist.mdを読み込んで実装対象を確認し、実際のコードと照らし合わせながら進めることで、見落としを防げた
- 各メソッドの実装完了時に即座にtasklist.mdを更新することで、進捗が明確になった
- テスト→lint→セキュリティチェック→ビルドの順で品質確認を行うことで、問題を段階的に検出できた

### 次回への改善提案
- タスクリスト作成時に、実際のソースコードのメソッド名を正確に確認する
- 複雑なメソッド（特にプライベートメソッド経由で処理が行われる場合）は、事前にコードを読んでタスクリストを詳細化する
- import文の追加と同時に、未使用のimportが残らないように注意する（golangci-lintで検出されるが、事前に気づければより効率的）

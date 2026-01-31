# 要求内容

## 概要

ソースコード内のinfra、usecase、repositoryレイヤーにおいて、エラー発生時にログにエラー内容を出力する機能を実装する。

## 背景

現在のコードでは、エラーが発生した際に上位レイヤーにエラーを伝播させるのみで、ログにエラー内容が記録されていない。これにより以下の問題が発生している:

1. **デバッグの困難さ**: エラーが発生した箇所とコンテキストが特定しにくい
2. **運用監視の不足**: Cloud Loggingでエラーを追跡できない
3. **監査証跡の不足**: どの操作でエラーが発生したか記録が残らない

開発ガイドラインには「全操作の監査ログ出力」が明記されているが、現状ではhandlerレイヤーのみでのログ出力に留まっており、下位レイヤー（infra、usecase、repository）でのエラーログが不足している。

## 実装対象の機能

### 1. infrastructureレイヤーでのエラーログ出力

**対象ファイル**:
- `internal/infra/kms.go`: Cloud KMS操作時のエラー
- `internal/infra/database.go`: データベース接続時のエラー

**機能**:
- Cloud KMS暗号化/復号エラー時にslog.Errorでログ出力
- データベース接続エラー時にslog.Errorでログ出力
- エラーコンテキスト（操作名、パラメータ）を構造化ログとして記録

### 2. usecaseレイヤーでのエラーログ出力

**対象ファイル**:
- `internal/usecase/key_service.go`: ビジネスロジック実行時のエラー
- `internal/usecase/migration_service.go`: マイグレーション実行時のエラー

**機能**:
- リポジトリ操作エラー時にslog.Errorでログ出力
- KMS操作エラー時にslog.Errorでログ出力
- ビジネスロジックエラー時にslog.Warnでログ出力（ドメインエラーは警告レベル）
- エラーコンテキスト（tenant_id、generation等）を構造化ログとして記録

### 3. repositoryレイヤーでのエラーログ出力

**対象ファイル**:
- `internal/repository/key_repository.go`: データアクセス時のエラー
- `internal/repository/migration_repository.go`: マイグレーション管理時のエラー

**機能**:
- データベースクエリエラー時にslog.Errorでログ出力
- エラーコンテキスト（クエリ対象、条件）を構造化ログとして記録

## 受け入れ条件

### infrastructureレイヤー
- [ ] `infra/kms.go`の`Encrypt()`メソッドでエラー時にslog.Errorでログ出力
- [ ] `infra/kms.go`の`Decrypt()`メソッドでエラー時にslog.Errorでログ出力
- [ ] `infra/database.go`の`NewDB()`関数でエラー時にslog.Errorでログ出力
- [ ] エラーログに操作名（"encrypt", "decrypt", "db_init"）が含まれる
- [ ] 鍵の平文データがログに出力されていない（セキュリティ要件）

### usecaseレイヤー
- [ ] `usecase/key_service.go`の全メソッドでエラー時にslog.Error/Warnでログ出力
- [ ] `usecase/migration_service.go`の全メソッドでエラー時にslog.Error/Warnでログ出力
- [ ] ドメインエラー（ErrKeyNotFound等）はslog.Warnレベル
- [ ] システムエラー（DB接続エラー等）はslog.Errorレベル
- [ ] エラーログにtenant_id、generation等のコンテキスト情報が含まれる
- [ ] 鍵の平文データがログに出力されていない（セキュリティ要件）

### repositoryレイヤー
- [ ] `repository/key_repository.go`の全メソッドでエラー時にslog.Errorでログ出力
- [ ] `repository/migration_repository.go`の全メソッドでエラー時にslog.Errorでログ出力
- [ ] gorm.ErrRecordNotFoundは通常フローのためログ出力不要（nilを返すのみ）
- [ ] エラーログにクエリ対象（tenant_id等）が含まれる

### 全レイヤー共通
- [ ] ログフォーマットがCloud Loggingの構造化ログ形式に準拠
- [ ] エラーログに"error"フィールドが含まれる
- [ ] 既存のテストが全て通る（エラーログ追加による影響がない）

## 成功指標

- 全てのエラー発生箇所でログが出力され、Cloud Loggingで追跡可能になる
- エラーログに十分なコンテキスト情報が含まれ、デバッグが容易になる
- セキュリティ要件（鍵の平文を出力しない）が守られている

## スコープ外

以下はこのフェーズでは実装しません:

- handlerレイヤーでのエラーログ（既にmiddleware/logging.goで実装済み）
- エラーログの集約・分析機能
- アラート設定
- カスタムログフォーマッター実装
- ログレベルの動的変更機能

## 参照ドキュメント

- `docs/development-guidelines.md` - エラーハンドリング規約、ログ出力規約
- `docs/architecture.md` - ログ出力要件、セキュリティ要件
- `docs/repository-structure.md` - レイヤー構造と責務

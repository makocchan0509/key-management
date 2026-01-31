# 設計: .envファイルサポート

## アーキテクチャ概要

### 変更箇所

1. **依存関係追加**: `github.com/joho/godotenv`を`go.mod`に追加
2. **main.go修正**: アプリケーション起動時に`.env`を読み込む
3. **gitignore更新**: `.env`を除外リストに追加
4. **サンプルファイル追加**: `.env.example`を作成

### 設計方針

**原則**:
- 既存の`config.Load()`の動作を変更しない
- `.env`読み込みはmain.go内で`config.Load()`の前に実行
- ファイルが存在しない場合はエラーを出さず、静かに無視する
- 環境変数が既に設定されている場合は`.env`の値で上書きしない（godotenvのデフォルト動作）

## 詳細設計

### 1. main.goの修正

**変更箇所**: `cmd/server/main.go`

**変更内容**:

```go
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv" // 追加

	"key-management-service/config"
	"key-management-service/internal/handler"
	"key-management-service/internal/infra"
	"key-management-service/internal/repository"
	"key-management-service/internal/usecase"
)

func main() {
	ctx := context.Background()

	// .envファイルを読み込む（存在しない場合は無視）
	// 既存の環境変数は上書きしない
	_ = godotenv.Load()

	// 設定読み込み
	cfg := config.Load()

	// 以下、既存のコードそのまま...
```

**設計のポイント**:
- `godotenv.Load()`を`config.Load()`の前に実行
- エラーを無視（`_ = godotenv.Load()`）して、ファイルが存在しない場合も続行
- `godotenv.Load()`はデフォルトで既存の環境変数を上書きしない動作

### 2. .gitignoreの更新

**変更箇所**: `key-management-service/.gitignore`

**追加内容**:

```gitignore
# 環境変数ファイル（機密情報を含むため）
.env
```

**設計のポイント**:
- `.env`ファイルには機密情報（DATABASE_URL, KMS_KEY_NAMEなど）が含まれる
- Gitにコミットされないようにする

### 3. .env.exampleの作成

**新規ファイル**: `key-management-service/.env.example`

**内容**:

```
# データベース接続
# 例: mysql://user:password@localhost:3306/keymanagement?parseTime=true
DATABASE_URL=

# Google Cloud設定
# 例: my-gcp-project
GOOGLE_CLOUD_PROJECT=

# KMS鍵名
# 例: projects/my-project/locations/global/keyRings/my-keyring/cryptoKeys/my-key
KMS_KEY_NAME=

# サーバー設定（オプション、デフォルト: 8080）
PORT=8080

# ログレベル（オプション、デフォルト: INFO）
# 選択肢: DEBUG, INFO, WARN, ERROR
LOG_LEVEL=INFO
```

**設計のポイント**:
- 実際の値は含めず、フォーマットと例のみを示す
- 必須項目と任意項目を明記
- コメントでデフォルト値を説明

### 4. go.modの更新

**変更内容**:

```bash
go get github.com/joho/godotenv
```

これにより`go.mod`と`go.sum`が自動的に更新される。

**使用するバージョン**:
- 最新の安定版を使用（執筆時点: v1.5.1）

## 実装パターン

### godotenvの動作仕様

`godotenv.Load()`の動作:

1. カレントディレクトリから`.env`ファイルを探す
2. ファイルが存在する場合、`KEY=VALUE`形式で読み込む
3. 既に環境変数が設定されている場合は上書きしない（重要）
4. ファイルが存在しない場合はエラーを返すが、`_ =`で無視可能

```go
// パターン1: ファイルが存在しなくてもエラーにしない
_ = godotenv.Load()

// パターン2: エラーハンドリングする場合
if err := godotenv.Load(); err != nil {
    // ファイルが存在しない場合はログだけ出して続行
    slog.Debug("no .env file found, using environment variables")
}
```

**今回の採用**: パターン1（完全に無視）
- 理由: 本番環境では`.env`が存在しないため、エラーメッセージすら不要
- シンプルで明確

### 優先順位

環境変数の優先順位（高い順）:

1. **既存の環境変数** - 実行前にexportされた値
2. **`/env`ファイル** - godotenv.Load()で読み込まれる値
3. **デフォルト値** - config.goのgetEnv()で定義される値

この順序により、本番環境では既存の環境変数が優先され、`.env`による上書きは発生しない。

## ファイル配置

```
key-management-service/
├── .env                # gitignoreに追加（ローカル開発用、コミットしない）
├── .env.example        # サンプルファイル（コミットする）
├── .gitignore          # .envを追加
├── cmd/
│   └── server/
│       └── main.go     # godotenv.Load()を追加
├── config/
│   └── config.go       # 変更なし
└── go.mod              # godotenvを追加
```

## 動作フロー

### ローカル開発環境の場合

```
1. アプリケーション起動
2. godotenv.Load()が`.env`を読み込む
3. 環境変数に`.env`の内容が設定される
4. config.Load()が環境変数から設定を読み込む
5. アプリケーションが正常に起動
```

### 本番環境の場合（Cloud Run）

```
1. アプリケーション起動
2. godotenv.Load()が`.env`を探すが見つからない
3. エラーを無視して続行
4. config.Load()が環境変数から設定を読み込む（Cloud Runで設定済み）
5. アプリケーションが正常に起動
```

### 環境変数が既に設定されている場合

```
1. export DATABASE_URL="production-db-url"
2. アプリケーション起動
3. godotenv.Load()が`.env`を読み込む
   - `.env`にDATABASE_URL=local-db-urlがあっても上書きしない
4. config.Load()がexportされた値（production-db-url）を読み込む
5. アプリケーションが正常に起動
```

## セキュリティ考慮事項

### 機密情報の保護

1. **`.env`を`.gitignore`に追加**
   - DATABASE_URL、KMS_KEY_NAMEなどの機密情報を含む
   - 誤ってコミットしないようにする

2. **`.env.example`には実際の値を含めない**
   - フォーマットと説明のみ
   - 開発者は`.env.example`をコピーして`.env`を作成

3. **本番環境では使用しない**
   - Cloud Runの環境変数機能を使用
   - `.env`ファイルはデプロイしない

## テスト方針

### 既存テストへの影響

- `godotenv.Load()`は`main.go`内でのみ実行
- `config.Load()`の動作は変更しない
- 既存の単体テストは影響を受けない

### 動作確認

以下の3パターンで動作確認を実施:

1. **`.env`ファイルが存在する場合**
   - `.env`から環境変数が読み込まれることを確認
   - アプリケーションが正常に起動することを確認

2. **`.env`ファイルが存在しない場合**
   - エラーが発生しないことを確認
   - 既存の環境変数から読み込まれることを確認

3. **環境変数が既に設定されている場合**
   - `.env`の値で上書きされないことを確認
   - 既存の環境変数が優先されることを確認

## 非機能要件への対応

### パフォーマンス

- `.env`読み込みはアプリケーション起動時の1回のみ
- ファイル読み込みによる起動時間への影響は微小（数ミリ秒程度）

### 保守性

- `github.com/joho/godotenv`は広く使われている標準的なライブラリ
- メンテナンスが活発（最終更新: 2024年）
- シンプルなAPIで理解しやすい

## 実装後の使い方

### 開発者向け手順

1. **`.env.example`をコピー**
   ```bash
   cp .env.example .env
   ```

2. **`.env`に実際の値を設定**
   ```bash
   # .env
   DATABASE_URL=root:password@tcp(localhost:3306)/keymanagement?parseTime=true
   GOOGLE_CLOUD_PROJECT=my-local-project
   KMS_KEY_NAME=projects/my-local-project/locations/global/keyRings/test/cryptoKeys/test-key
   PORT=8080
   LOG_LEVEL=DEBUG
   ```

3. **アプリケーションを起動**
   ```bash
   go run cmd/server/main.go
   ```

環境変数が自動的に`.env`から読み込まれる。

## 変更の影響範囲

### 変更されるファイル

- `cmd/server/main.go` - godotenv.Load()を追加
- `.gitignore` - .envを追加
- `go.mod`, `go.sum` - godotenvの依存関係を追加

### 新規作成されるファイル

- `.env.example` - サンプルファイル

### 変更されないファイル

- `config/config.go` - 変更なし
- その他のビジネスロジック - 変更なし
- 既存のテストコード - 変更なし

## リスクと対策

### リスク1: .envファイルがGitにコミットされる

**対策**:
- `.gitignore`に`.env`を追加
- `.env.example`をサンプルとして提供

### リスク2: 本番環境で.envが誤って使用される

**対策**:
- Cloud Runには`.env`ファイルをデプロイしない
- Cloud Runの環境変数設定を優先

### リスク3: 環境変数の優先順位の誤解

**対策**:
- ドキュメントに優先順位を明記
- godotenvのデフォルト動作（既存の環境変数を上書きしない）を活用

## まとめ

この設計により、以下を実現:

1. ✅ ローカル開発が簡単になる（`.env`に環境変数を記述するだけ）
2. ✅ 本番環境での動作に影響しない（`.env`が存在しなくても動作）
3. ✅ セキュリティが保たれる（`.env`はGitにコミットされない）
4. ✅ 既存のコードへの影響が最小限（`main.go`に1行追加するだけ）
5. ✅ 後方互換性が保たれる（既存の環境変数設定方法も引き続き使用可能）

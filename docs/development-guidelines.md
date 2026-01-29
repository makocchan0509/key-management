# 開発ガイドライン (Development Guidelines)

## コーディング規約

### 型定義・構造体

**原則**:
- 構造体のフィールドにはタグを付与し、責務を明確にする
- インターフェースは小さく保つ（1-3メソッド）
- インターフェースは使う側（usecase）で定義する

**例**:
```go
// ✅ 良い例: フィールドにタグを付与し、責務を明確にする
type EncryptionKey struct {
    ID           string    `gorm:"type:char(36);primaryKey"`
    TenantID     string    `gorm:"type:varchar(64);not null;uniqueIndex:uk_tenant_generation"`
    Generation   uint      `gorm:"not null;uniqueIndex:uk_tenant_generation"`
    EncryptedKey []byte    `gorm:"type:blob;not null"`
    Status       string    `gorm:"type:enum('active','disabled');not null;default:'active'"`
    CreatedAt    time.Time `gorm:"type:datetime(6);not null;autoCreateTime"`
    UpdatedAt    time.Time `gorm:"type:datetime(6);not null;autoUpdateTime"`
}

// ❌ 悪い例: フィールド名が曖昧、タグなし
type Key struct {
    A string
    B string
    C []byte
}
```

### インターフェースの設計

```go
// ✅ 良い例: 小さなインターフェース（usecaseで定義）
// internal/usecase/key_service.go
type KeyRepository interface {
    ExistsByTenantID(ctx context.Context, tenantID string) (bool, error)
    Create(ctx context.Context, key *domain.EncryptionKey) error
    FindByTenantIDAndGeneration(ctx context.Context, tenantID string, generation uint) (*domain.EncryptionKey, error)
}

type KMSClient interface {
    Encrypt(ctx context.Context, plaintext []byte) ([]byte, error)
    Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)
}

// ❌ 悪い例: 巨大なインターフェース
type KeyManager interface {
    Create(ctx context.Context, tenantID string) error
    Get(ctx context.Context, tenantID string) ([]byte, error)
    Rotate(ctx context.Context, tenantID string) error
    List(ctx context.Context, tenantID string) ([]*Key, error)
    Disable(ctx context.Context, tenantID string, gen uint) error
    Encrypt(ctx context.Context, data []byte) ([]byte, error)
    Decrypt(ctx context.Context, data []byte) ([]byte, error)
    // ... 多すぎる
}
```

### 命名規則

#### 変数・関数

**原則**:
- 変数: camelCase（ローカル）/ PascalCase（エクスポート）、名詞または名詞句
- 関数: 動詞で始める
- 定数: PascalCase（エクスポート）/ camelCase（非公開）
- Boolean変数: is, has, can, should を使用。関数名では不要

**例**:
```go
// ✅ 良い例
tenantID := "tenant-001"
var encryptedKeys []*EncryptionKey
isActive := true

func FindByTenantID(ctx context.Context, tenantID string) (*EncryptionKey, error) { }
func validateTenantID(tenantID string) error { }

// Boolean関数はValid()のような形式
func (k *EncryptionKey) Valid() bool { }

// ❌ 悪い例
tid := "tenant-001"       // 省略しすぎ
var keys []*EncryptionKey // 曖昧
active := true            // Boolean であることが不明確

func DoIt(ctx context.Context, id string) (*EncryptionKey, error) { } // 何をする関数か不明
```

#### 型・クラス・インターフェース

**原則**:
- 構造体: PascalCase、名詞
- インターフェース: 動詞+er（1メソッドの場合）、または役割名

**例**:
```go
// 構造体
type KeyService struct { }
type KeyRepository struct { }

// インターフェース（1メソッド）
type Encrypter interface {
    Encrypt(ctx context.Context, plaintext []byte) ([]byte, error)
}

// インターフェース（複数メソッド）
type KeyRepository interface { }
type KMSClient interface { }
```

#### その他の命名規則

| 対象 | 規則 | 例 |
|---|---|---|
| パッケージ名 | 小文字、短く、1単語 | `handler`, `domain`, `usecase` |
| レシーバ名 | 型名の先頭1-2文字 | `func (s *KeyService)`, `func (r *KeyRepository)` |
| 頭字語 | 全て大文字または全て小文字 | `tenantID`, `apiURL`, `HTTPHandler` |
| エクスポートされる型 | PascalCase | `KeyService`, `EncryptionKey` |
| 非公開の型・関数 | camelCase | `validateInput`, `dbClient` |

### コードフォーマット

**インデント**: タブ（Goの標準）

**行の長さ**: 最大 120 文字

**フォーマッター**: `gofmt` / `goimports`

### 関数設計

**原則**:
- 単一責務: 1つの関数は1つの責務
- 関数の長さ: 50行以内を目標（100行以上はリファクタリング対象）
- パラメータ数: 4個を超える場合は構造体でまとめる
- context.Context は常に第1引数

**例**:
```go
// ✅ 良い例: パラメータが多い場合は構造体を使う
type CreateKeyInput struct {
    TenantID string
}

func (s *KeyService) CreateKey(ctx context.Context, input CreateKeyInput) (*domain.KeyMetadata, error) {
    // 実装
}

// ❌ 悪い例: パラメータが多すぎる
func (s *KeyService) CreateKey(
    ctx context.Context,
    tenantID string,
    keyType string,
    algorithm string,
    expiresAt *time.Time,
) (*domain.KeyMetadata, error) {
    // 実装
}

// ❌ 悪い例: context を構造体のフィールドに保持する
type KeyService struct {
    ctx context.Context // しない
}
```

### エラーハンドリング

**原則**:
- 予期されるエラー: ドメインエラーとして定義し、適切に処理
- 予期しないエラー: コンテキストを付与して上位に伝播
- エラーを無視しない

**カスタムエラーの定義**:
```go
// internal/domain/errors.go
package domain

import "errors"

var (
    ErrKeyNotFound        = errors.New("key not found")
    ErrKeyAlreadyExists   = errors.New("key already exists")
    ErrKeyDisabled        = errors.New("key is disabled")
    ErrKeyAlreadyDisabled = errors.New("key is already disabled")
    ErrInvalidTenantID    = errors.New("invalid tenant ID")
    ErrInvalidGeneration  = errors.New("invalid generation")
)
```

**エラーハンドリングパターン**:
```go
// ✅ 良い例: エラーを即座にチェックし、コンテキストを付与
func (s *KeyService) GetCurrentKey(ctx context.Context, tenantID string) (*domain.Key, error) {
    key, err := s.repo.FindLatestActiveByTenantID(ctx, tenantID)
    if err != nil {
        return nil, fmt.Errorf("finding current key for tenant %s: %w", tenantID, err)
    }
    if key == nil {
        return nil, domain.ErrKeyNotFound
    }

    plainKey, err := s.kmsClient.Decrypt(ctx, key.EncryptedKey)
    if err != nil {
        return nil, fmt.Errorf("decrypting key: %w", err)
    }

    return &domain.Key{
        TenantID:   key.TenantID,
        Generation: key.Generation,
        Key:        plainKey,
    }, nil
}

// ❌ 悪い例: エラーを無視する
func (s *KeyService) GetCurrentKey(ctx context.Context, tenantID string) *domain.Key {
    key, _ := s.repo.FindLatestActiveByTenantID(ctx, tenantID) // エラー無視
    plainKey, _ := s.kmsClient.Decrypt(ctx, key.EncryptedKey)  // エラー無視
    return &domain.Key{Key: plainKey}
}
```

**エラーメッセージ**:
```go
// ✅ 良い例: 小文字で始め、句読点をつけない（Goの慣習）
return fmt.Errorf("finding key by tenant %s: %w", tenantID, err)

// ❌ 悪い例: 大文字で始め、句読点をつける
return fmt.Errorf("Failed to find key by tenant %s.", tenantID)
```

### コメント規約

**パッケージコメント**:
```go
// Package usecase はアプリケーションのユースケースを実装する。
// ドメインモデルとリポジトリインターフェースを組み合わせて
// ビジネスロジックを提供する。
package usecase
```

**エクスポートされるシンボルのコメント**:
```go
// KeyService は暗号鍵に関するビジネスロジックを提供する。
type KeyService struct {
    repo      KeyRepository
    kmsClient KMSClient
}

// CreateKey は指定されたテナントに対して新しい暗号鍵を生成する。
// 既に鍵が存在する場合は ErrKeyAlreadyExists を返す。
func (s *KeyService) CreateKey(ctx context.Context, tenantID string) (*domain.KeyMetadata, error) {
    // 実装
}
```

**インラインコメント**:
```go
// ✅ 良い例: なぜそうするかを説明
// Cloud KMSのAPI呼び出しは遅延する可能性があるため、タイムアウトを設定
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

// ❌ 悪い例: 何をしているか（コードを見れば分かる）
// コンテキストにタイムアウトを設定する
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
```

### セキュリティ

**入力検証**: ハンドラ層（境界）で検証する

```go
// internal/handler/key_handler.go
func (h *KeyHandler) CreateKey(w http.ResponseWriter, r *http.Request) {
    tenantID := chi.URLParam(r, "tenant_id")

    // 入力検証
    if err := validateTenantID(tenantID); err != nil {
        respondError(w, http.StatusBadRequest, "INVALID_TENANT_ID", err.Error())
        return
    }

    // サービス呼び出し
    metadata, err := h.service.CreateKey(r.Context(), tenantID)
    // ...
}

func validateTenantID(tenantID string) error {
    if tenantID == "" {
        return errors.New("tenant_id is required")
    }
    if len(tenantID) > 64 {
        return errors.New("tenant_id must be 64 characters or less")
    }
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, tenantID)
    if !matched {
        return errors.New("tenant_id contains invalid characters")
    }
    return nil
}
```

**機密情報**: 環境変数から読み込み、ハードコードしない

```go
// ✅ 良い例
kmsKeyName := os.Getenv("KMS_KEY_NAME")
if kmsKeyName == "" {
    log.Fatal("KMS_KEY_NAME is not set")
}

// ❌ 悪い例
const kmsKeyName = "projects/my-project/locations/asia-northeast1/keyRings/my-keyring/cryptoKeys/my-key"
```

**インジェクション対策**: gormのプリペアドステートメントを使用

```go
// ✅ 良い例: gormのメソッドを使用（自動的にプレースホルダが使われる）
err := r.db.WithContext(ctx).
    Where("tenant_id = ? AND generation = ?", tenantID, generation).
    First(&key).Error

// ❌ 悪い例: 文字列結合
err := r.db.WithContext(ctx).
    Raw("SELECT * FROM encryption_keys WHERE tenant_id = '" + tenantID + "'").
    Scan(&key).Error
```

**鍵の平文をログに出力しない**:
```go
// ✅ 良い例: tenant_id、generationのみ記録
slog.Info("key retrieved",
    "operation", "GET_CURRENT_KEY",
    "tenant_id", tenantID,
    "generation", key.Generation,
)

// ❌ 悪い例: 鍵データを出力
slog.Info("key retrieved",
    "key", plainKey, // 絶対にしない
)
```

### パフォーマンス

**スライス・マップの事前確保**:
```go
// ✅ 良い例: 容量を事前確保
keys := make([]*domain.EncryptionKey, 0, len(tenantIDs))
for _, tenantID := range tenantIDs {
    key, err := repo.FindLatestActiveByTenantID(ctx, tenantID)
    if err != nil {
        return nil, err
    }
    keys = append(keys, key)
}

// ✅ 良い例: マップの事前確保
keyMap := make(map[string]*domain.EncryptionKey, len(keys))
for _, k := range keys {
    keyMap[k.TenantID] = k
}
```

**N+1問題の回避**:
```go
// ❌ 悪い例: N+1クエリ
for _, tenantID := range tenantIDs {
    key, _ := repo.FindByTenantID(ctx, tenantID) // N回クエリが走る
    results = append(results, key)
}

// ✅ 良い例: 一括取得
keys, err := repo.FindByTenantIDs(ctx, tenantIDs) // 1回のクエリ
keyMap := make(map[string]*domain.EncryptionKey, len(keys))
for _, k := range keys {
    keyMap[k.TenantID] = k
}
```

---

## Git 運用ルール

### ブランチ戦略（GitHub Flow）

**GitHub Flowとは**:
シンプルで軽量なブランチモデル。mainブランチを常にデプロイ可能な状態に保ち、すべての変更はfeatureブランチからPRを通じてマージします。

**ブランチ種別**:
- `main`: 本番環境にデプロイ可能な状態（常に安定）
- `feature/[機能名]`: 新機能開発
- `fix/[修正内容]`: バグ修正
- `refactor/[対象]`: リファクタリング

**フロー**:
```
main ─────────────────────────────────────────→
  │                                      ↑
  └─ feature/add-key-rotation ──────────→┘ (PR & merge)
  │                              ↑
  └─ fix/invalid-tenant-validation ──→┘ (PR & merge)
```

**運用ルール**:
1. `main`は常にデプロイ可能な状態を維持
2. 作業はすべて`main`から分岐したブランチで行う
3. 変更が完了したらPRを作成し、レビューを受ける
4. CIがパスし、レビューで承認されたら`main`にマージ
5. マージ後、必要に応じてデプロイ

**マージ方針**:
- feature/fix → main: squash merge（コミット履歴をクリーンに保つ）
- リリース時: タグを付与（例: `v1.0.0`）

### コミットメッセージ規約

**フォーマット** (Conventional Commits):
```
<type>(<scope>): <subject>

<body>

<footer>
```

**Type 一覧**:

| Type | 説明 |
|---|---|
| feat | 新機能 |
| fix | バグ修正 |
| docs | ドキュメント |
| style | フォーマット (動作に影響なし) |
| refactor | リファクタリング |
| perf | パフォーマンス改善 |
| test | テスト追加・修正 |
| build | ビルドシステム |
| ci | CI/CD 設定 |
| chore | その他 |

**例**:
```
feat(key): 鍵のローテーション機能を追加

テナントの暗号鍵を新しい世代に更新する機能を実装しました。

- POST /v1/tenants/{tenant_id}/keys/rotate エンドポイント追加
- KeyService.RotateKey メソッド実装
- 監査ログに ROTATE_KEY 操作を記録

Closes #42
```

### プルリクエストプロセス

**作成前のチェック**:
- [ ] 全てのテストがパス
- [ ] Lint / 静的解析エラーがない
- [ ] 競合が解決されている

**PR テンプレート**:
```markdown
## 変更の種類
- [ ] 新機能 (feat)
- [ ] バグ修正 (fix)
- [ ] リファクタリング (refactor)
- [ ] ドキュメント (docs)
- [ ] その他 (chore)

## 変更内容
### 何を変更したか
[簡潔な説明]

### なぜ変更したか
[背景・理由]

### どのように変更したか
- [変更点1]
- [変更点2]

## テスト
- [ ] ユニットテスト追加
- [ ] 統合テスト追加
- [ ] 手動テスト実施

## 関連 Issue
Closes #[番号]
```

**レビュープロセス**:
1. セルフレビュー
2. 自動テスト実行
3. レビュアーアサイン
4. レビューフィードバック対応
5. 承認後マージ

---

## テスト戦略

### テストピラミッド

```
       /\
      /E2E\       少 (遅い、高コスト)
     /------\
    / 統合   \     中
   /----------\
  / ユニット   \   多 (速い、低コスト)
 /--------------\
```

**目標比率**:
- ユニットテスト: 70%
- 統合テスト: 20%
- E2E テスト: 10%

### テストの種類

#### ユニットテスト

**対象**: 個別の関数・メソッド

**カバレッジ目標**:

| 対象 | 目標 |
|---|---|
| 全体 | 80% 以上 |
| ビジネスロジック層 (usecase) | 90% 以上 |

**配置**: 対象ファイルと同じディレクトリに `_test.go` ファイルを配置

#### 統合テスト・E2Eテスト

Cloud KMSはエミュレータが提供されていないため、統合テスト以降はGoogle Cloud環境にデプロイして検証する。

**対象**:
- API + Service + Repository の結合テスト
- KMS暗号化/復号の動作確認
- CLI → API → DB の一連のフロー
- トレーシングスパンの検証

### テスト命名規則

**パターン**: `Test[対象]_[条件]_[期待結果]`

```go
// ✅ 良い例
func TestKeyService_CreateKey_ReturnsKeyMetadata(t *testing.T) { }
func TestKeyService_CreateKey_ExistingKey_ReturnsError(t *testing.T) { }
func TestKeyService_GetCurrentKey_NotFound_ReturnsError(t *testing.T) { }

// ❌ 悪い例
func TestCreate(t *testing.T) { }
func Test1(t *testing.T) { }
func TestKeyServiceWorks(t *testing.T) { }
```

### テストの実装

**テーブル駆動テスト**:
```go
func TestKeyService_CreateKey(t *testing.T) {
    tests := []struct {
        name      string
        tenantID  string
        setupMock func(*mockKeyRepository)
        wantErr   error
    }{
        {
            name:     "new tenant",
            tenantID: "tenant-001",
            setupMock: func(m *mockKeyRepository) {
                m.existsResult = false
            },
            wantErr: nil,
        },
        {
            name:     "existing tenant",
            tenantID: "tenant-001",
            setupMock: func(m *mockKeyRepository) {
                m.existsResult = true
            },
            wantErr: domain.ErrKeyAlreadyExists,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            repo := &mockKeyRepository{}
            tt.setupMock(repo)
            kms := &mockKMSClient{}
            svc := usecase.NewKeyService(repo, kms)

            // Act
            _, err := svc.CreateKey(context.Background(), tt.tenantID)

            // Assert
            if !errors.Is(err, tt.wantErr) {
                t.Errorf("want error %v, got %v", tt.wantErr, err)
            }
        })
    }
}
```

### モック・スタブの使用

**原則**:
- 外部依存（Cloud KMS、Cloud SQL）はモック化
- ビジネスロジックは実際の実装を使用

```go
// インターフェースに基づくモック
type mockKeyRepository struct {
    existsResult bool
    existsErr    error
    key          *domain.EncryptionKey
    findErr      error
}

func (m *mockKeyRepository) ExistsByTenantID(ctx context.Context, tenantID string) (bool, error) {
    return m.existsResult, m.existsErr
}

func (m *mockKeyRepository) FindLatestActiveByTenantID(ctx context.Context, tenantID string) (*domain.EncryptionKey, error) {
    return m.key, m.findErr
}
```

---

## コードレビュー基準

### レビューポイント

**機能性**:
- [ ] 要件を満たしているか
- [ ] エッジケースが考慮されているか
- [ ] エラーハンドリングが適切か

**可読性**:
- [ ] 命名が明確か
- [ ] コメントが適切か
- [ ] 複雑なロジックが説明されているか

**保守性**:
- [ ] 重複コードがないか
- [ ] 責務が明確に分離されているか
- [ ] 変更の影響範囲が限定的か

**パフォーマンス**:
- [ ] N+1 問題がないか
- [ ] 不要な計算がないか
- [ ] メモリリークの可能性がないか

**セキュリティ**:
- [ ] 入力検証が適切か
- [ ] 機密情報がハードコードされていないか
- [ ] 鍵の平文がログに出力されていないか

### レビューコメントの優先度

- `[必須]`: 修正必須（マージをブロック）
- `[推奨]`: 修正推奨（可能であれば対応）
- `[提案]`: 検討してほしい（任意）
- `[質問]`: 理解のための質問

---

## 品質自動化

### 自動化カテゴリ

| カテゴリ | ツール | 用途 |
|---|---|---|
| フォーマット | gofmt / goimports | コードスタイルの自動整形 |
| Lint | golangci-lint | コーディング規約の自動検出 |
| 静的解析 | go vet | 論理エラーの検出 |
| テスト | go test | ユニット・統合テストの実行 |
| ビルド | go build | コンパイル確認 |

### CI/CD パイプライン

```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'
      - name: Format check
        run: goimports -l . | grep . && exit 1 || true
      - name: Vet
        run: go vet ./...
      - name: Lint
        run: golangci-lint run
      - name: Test
        run: go test -race -coverprofile=coverage.out ./...
      - name: Build
        run: go build ./cmd/...
```

### Makefile

```makefile
.PHONY: fmt lint test build

fmt:
	goimports -w .

lint:
	go vet ./...
	golangci-lint run

test:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

build:
	go build ./cmd/...

ci: fmt lint test build
```

---

## 開発環境セットアップ

### 必要なツール

| ツール | バージョン | インストール方法 |
|---|---|---|
| Go | 1.23.x | https://go.dev/dl/ |
| golangci-lint | v1.61.x | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |
| goimports | latest | `go install golang.org/x/tools/cmd/goimports@latest` |
| Docker | 27.x | https://www.docker.com/products/docker-desktop |

### セットアップ手順

```bash
# 1. リポジトリのクローン
git clone <repository-url>
cd key-management-service

# 2. 依存関係のインストール
go mod download

# 3. 環境変数の設定
cp .env.example .env
# .env ファイルを編集

# 4. ローカルでのビルド確認
make build

# 5. テストの実行
make test
```

### 環境変数

| 変数名 | 必須 | 説明 |
|---|---|---|
| KMS_KEY_NAME | 必須 | Cloud KMSの暗号鍵リソース名 |
| DATABASE_URL | 必須 | Cloud SQL接続文字列 |
| GOOGLE_CLOUD_PROJECT | 必須 | GCPプロジェクトID |
| PORT | 任意 | APIサーバーポート（デフォルト: 8080） |
| LOG_LEVEL | 任意 | ログレベル（デフォルト: INFO） |
| OTEL_ENABLED | 任意 | OpenTelemetryの有効化（デフォルト: false） |

---

## チェックリスト

### コーディング規約
- [ ] 命名規則に従っている（頭字語、レシーバ名など）
- [ ] 関数が単一の責務を持っている
- [ ] エラーハンドリングが実装されている（エラーを無視していない）
- [ ] エラーメッセージが小文字で始まっている
- [ ] マジックナンバーがない
- [ ] セキュリティ（入力検証、機密情報管理）が適切
- [ ] 鍵の平文がログに出力されていない

### 開発プロセス
- [ ] ブランチ戦略に従っている
- [ ] コミットメッセージがConventional Commitsに従っている
- [ ] PRテンプレートを使用している
- [ ] テストが追加されている
- [ ] コードレビューを受けている
- [ ] CIがパスしている

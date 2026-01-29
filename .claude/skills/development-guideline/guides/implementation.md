# 実装ガイド (Implementation Guide) - Go

## 型定義・構造体

### 構造体の設計

```go
// ✅ 良い例: フィールドにタグを付与し、責務を明確にする
type User struct {
    ID        string    `json:"id" db:"id"`
    Name      string    `json:"name" db:"name"`
    Email     string    `json:"email" db:"email"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ❌ 悪い例: フィールド名が曖昧、タグなし
type User struct {
    A string
    B string
    C string
}
```

### インターフェースの設計

```go
// ✅ 良い例: 小さなインターフェース (Go の慣習)
type Reader interface {
    Read(ctx context.Context, id string) (*User, error)
}

type Writer interface {
    Write(ctx context.Context, user *User) error
}

// 必要に応じて合成する
type ReadWriter interface {
    Reader
    Writer
}

// ❌ 悪い例: 巨大なインターフェース
type UserManager interface {
    Create(ctx context.Context, user *User) error
    Read(ctx context.Context, id string) (*User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context) ([]*User, error)
    Search(ctx context.Context, query string) ([]*User, error)
    Export(ctx context.Context, format string) ([]byte, error)
}
```

**原則**:
- インターフェースは使う側で定義する
- インターフェースは小さく保つ (1-3メソッド)
- 実装が1つしかない場合、テスト容易性が不要ならインターフェースを作らない

## 命名規則

### 変数・関数

```go
// 変数: camelCase (ローカル) / PascalCase (エクスポート)
userID := "123"
var taskList []*Task
isCompleted := true

// 関数: 動詞 + 対象
func FindUserByID(ctx context.Context, id string) (*User, error) { }
func validateEmail(email string) error { }
func calculateTotal(items []CartItem) int { }

// Boolean: is, has, can, should は変数名に使う。関数名では不要
var isValid bool
func Valid(email string) bool { } // isValid() ではなく Valid()
```

### 構造体・インターフェース

```go
// 構造体: PascalCase、名詞
type TaskService struct { }
type UserRepository struct { }

// インターフェース: 動詞+er が基本 (1メソッドの場合)
type Reader interface { }
type Validator interface { }

// 複数メソッドの場合は役割名
type UserRepository interface { }
type TaskStore interface { }
```

### 定数・パッケージ

```go
// 定数: PascalCase (エクスポート) / camelCase (非公開)
const MaxRetryCount = 3
const defaultTimeout = 5 * time.Second

// iota による列挙
type TaskStatus int

const (
    TaskStatusPending TaskStatus = iota
    TaskStatusInProgress
    TaskStatusCompleted
)

// パッケージ名: 小文字、短く、1単語
// ✅ handler, domain, usecase, auth
// ❌ httpHandlers, domainModels, user_service
```

### レシーバ名

```go
// ✅ 良い例: 型名の先頭1-2文字
func (s *UserService) FindByID(ctx context.Context, id string) (*User, error) { }
func (r *UserRepository) Save(ctx context.Context, user *User) error { }

// ❌ 悪い例: self, this, me
func (self *UserService) FindByID(ctx context.Context, id string) (*User, error) { }
func (this *UserRepository) Save(ctx context.Context, user *User) error { }
```

### 頭字語

```go
// ✅ 良い例: 頭字語は全て大文字または全て小文字
var userID string   // "Id" ではなく "ID"
var httpClient *http.Client
type HTTPHandler struct { }
var apiURL string

// ❌ 悪い例
var userId string
type HttpHandler struct { }
var apiUrl string
```

## 関数設計

### 単一責務の原則

```go
// ✅ 良い例: 単一の責務
func calculateTotal(items []CartItem) int {
    total := 0
    for _, item := range items {
        total += item.Price * item.Quantity
    }
    return total
}

func formatPrice(amount int) string {
    return fmt.Sprintf("¥%d", amount)
}

// ❌ 悪い例: 複数の責務
func calculateAndFormatPrice(items []CartItem) string {
    total := 0
    for _, item := range items {
        total += item.Price * item.Quantity
    }
    return fmt.Sprintf("¥%d", total)
}
```

### 関数の長さ

- 目標: 20行以内
- 推奨: 50行以内
- 100行以上: リファクタリングを検討

### パラメータの設計

```go
// ✅ 良い例: パラメータが多い場合はオプション構造体を使う
type CreateTaskInput struct {
    Title       string
    Description string
    Priority    TaskPriority
    DueDate     *time.Time
}

func (s *TaskService) Create(ctx context.Context, input CreateTaskInput) (*Task, error) {
    // 実装
}

// ❌ 悪い例: パラメータが多すぎる
func (s *TaskService) Create(
    ctx context.Context,
    title string,
    description string,
    priority string,
    dueDate *time.Time,
    tags []string,
    assignee string,
) (*Task, error) {
    // 実装
}
```

### context.Context

```go
// ✅ 良い例: 第1引数は常に context.Context
func (s *UserService) FindByID(ctx context.Context, id string) (*User, error) { }

// ❌ 悪い例: context を構造体のフィールドに保持する
type UserService struct {
    ctx context.Context // しない
}
```

## エラーハンドリング

### カスタムエラー

```go
// エラー変数 (センチネルエラー)
var (
    ErrNotFound     = errors.New("not found")
    ErrUnauthorized = errors.New("unauthorized")
    ErrForbidden    = errors.New("forbidden")
)

// 詳細情報を持つエラー型
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
}

// ドメイン固有エラー
type NotFoundError struct {
    Resource string
    ID       string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}
```

### エラーハンドリングパターン

```go
// ✅ 良い例: エラーを即座にチェックし、コンテキストを付与
func (s *TaskService) FindByID(ctx context.Context, id string) (*Task, error) {
    task, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("finding task %s: %w", id, err)
    }
    if task == nil {
        return nil, &NotFoundError{Resource: "Task", ID: id}
    }
    return task, nil
}

// ❌ 悪い例: エラーを無視する
func (s *TaskService) FindByID(ctx context.Context, id string) *Task {
    task, _ := s.repo.FindByID(ctx, id) // エラー無視
    return task
}
```

### エラーの判定

```go
// errors.Is でセンチネルエラーを判定
if errors.Is(err, ErrNotFound) {
    // 404レスポンスを返す
}

// errors.As で型付きエラーを判定
var validErr *ValidationError
if errors.As(err, &validErr) {
    // バリデーションエラーのフィールドを使う
    log.Printf("field: %s, message: %s", validErr.Field, validErr.Message)
}
```

### エラーメッセージ

```go
// ✅ 良い例: 小文字で始め、句読点をつけない (Go の慣習)
return fmt.Errorf("finding user by id %s: %w", id, err)

// ❌ 悪い例: 大文字で始め、句読点をつける
return fmt.Errorf("Failed to find user by ID %s.", id)
```

## 並行処理

### goroutine の基本

```go
// ✅ 良い例: errgroup で goroutine を管理
func (s *Service) FetchAll(ctx context.Context, ids []string) ([]*User, error) {
    g, ctx := errgroup.WithContext(ctx)
    users := make([]*User, len(ids))

    for i, id := range ids {
        g.Go(func() error {
            user, err := s.repo.FindByID(ctx, id)
            if err != nil {
                return fmt.Errorf("fetching user %s: %w", id, err)
            }
            users[i] = user
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return nil, err
    }
    return users, nil
}

// ❌ 悪い例: goroutine を放置する (リーク)
func (s *Service) FetchAll(ctx context.Context, ids []string) []*User {
    for _, id := range ids {
        go s.repo.FindByID(ctx, id) // 結果を受け取らない
    }
    return nil
}
```

### チャネルの使用

```go
// ✅ 良い例: チャネルの方向を明示する
func produce(ctx context.Context) <-chan int {
    ch := make(chan int)
    go func() {
        defer close(ch)
        for i := 0; ; i++ {
            select {
            case <-ctx.Done():
                return
            case ch <- i:
            }
        }
    }()
    return ch
}
```

## コメント規約

### パッケージコメント

```go
// ✅ 良い例: パッケージの役割を説明
// Package usecase はアプリケーションのユースケースを実装する。
// ドメインモデルとリポジトリインターフェースを組み合わせて
// ビジネスロジックを提供する。
package usecase
```

### エクスポートされるシンボルのコメント

```go
// ✅ 良い例: シンボル名で始める (Go の慣習)
// UserService はユーザーに関するビジネスロジックを提供する。
type UserService struct {
    repo UserRepository
}

// FindByID は指定されたIDのユーザーを取得する。
// ユーザーが存在しない場合は ErrNotFound を返す。
func (s *UserService) FindByID(ctx context.Context, id string) (*User, error) {
    // 実装
}

// ❌ 悪い例: シンボル名で始めない
// ユーザーを取得する関数
func (s *UserService) FindByID(ctx context.Context, id string) (*User, error) { }
```

### インラインコメント

```go
// ✅ 良い例: なぜそうするかを説明
// タイムアウトを長めに設定: 外部APIの応答が遅い場合がある
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

// ❌ 悪い例: 何をしているか (コードを見れば分かる)
// コンテキストにタイムアウトを設定する
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
```

## セキュリティ

### 入力検証

```go
// ✅ 良い例: 境界で検証する
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    var input CreateUserInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }

    if err := input.Validate(); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    user, err := h.service.Create(r.Context(), input)
    // ...
}

func (i *CreateUserInput) Validate() error {
    if i.Name == "" {
        return &ValidationError{Field: "name", Message: "name is required"}
    }
    if len(i.Name) > 200 {
        return &ValidationError{Field: "name", Message: "name must be 200 characters or less"}
    }
    return nil
}
```

### SQLインジェクション対策

```go
// ✅ 良い例: プレースホルダを使用
row := db.QueryRowContext(ctx, "SELECT * FROM users WHERE id = $1", id)

// ❌ 悪い例: 文字列結合
row := db.QueryRowContext(ctx, "SELECT * FROM users WHERE id = '"+id+"'")
```

### 機密情報の管理

```go
// ✅ 良い例: 環境変数から読み込み
apiKey := os.Getenv("API_KEY")
if apiKey == "" {
    log.Fatal("API_KEY is not set")
}

// ❌ 悪い例: ハードコード
const apiKey = "sk-1234567890abcdef"
```

## パフォーマンス

### スライスの事前確保

```go
// ✅ 良い例: 容量を事前確保
users := make([]*User, 0, len(ids))
for _, id := range ids {
    user, err := repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }
    users = append(users, user)
}

// ❌ 悪い例: 容量指定なし (再アロケーション発生)
var users []*User
for _, id := range ids {
    user, _ := repo.FindByID(ctx, id)
    users = append(users, user)
}
```

### マップの事前確保

```go
// ✅ 良い例
userMap := make(map[string]*User, len(users))
for _, u := range users {
    userMap[u.ID] = u
}
```

### N+1 問題・高コストアルゴリズムの回避

ループ内で繰り返しI/O (DB クエリ、API コール等) を発行する **N+1 問題** をはじめ、処理時間が著しく増大するアルゴリズムは原則として避ける。

```go
// ❌ 悪い例: N+1 クエリ (注文ごとにユーザーを取得)
func (s *OrderService) ListWithUser(ctx context.Context) ([]*OrderWithUser, error) {
    orders, err := s.orderRepo.FindAll(ctx)
    if err != nil {
        return nil, err
    }
    result := make([]*OrderWithUser, 0, len(orders))
    for _, o := range orders {
        user, err := s.userRepo.FindByID(ctx, o.UserID) // N回クエリが走る
        if err != nil {
            return nil, err
        }
        result = append(result, &OrderWithUser{Order: o, User: user})
    }
    return result, nil
}

// ✅ 良い例: 一括取得してマップで結合
func (s *OrderService) ListWithUser(ctx context.Context) ([]*OrderWithUser, error) {
    orders, err := s.orderRepo.FindAll(ctx)
    if err != nil {
        return nil, err
    }

    userIDs := make([]string, 0, len(orders))
    for _, o := range orders {
        userIDs = append(userIDs, o.UserID)
    }

    users, err := s.userRepo.FindByIDs(ctx, userIDs) // 1回のクエリ
    if err != nil {
        return nil, err
    }
    userMap := make(map[string]*User, len(users))
    for _, u := range users {
        userMap[u.ID] = u
    }

    result := make([]*OrderWithUser, 0, len(orders))
    for _, o := range orders {
        result = append(result, &OrderWithUser{Order: o, User: userMap[o.UserID]})
    }
    return result, nil
}
```

**代表的な注意パターン**:

| パターン | 問題 | 対策 |
|---|---|---|
| ループ内 DB クエリ (N+1) | データ量に比例してクエリ数が増加 | `IN` 句等で一括取得しマップで結合 |
| ループ内 API コール | レイテンシが N 倍に増加 | バッチ API の利用、または並行処理 |
| ネストループによる O(n²) 処理 | データ量の二乗に比例して遅延 | マップによる O(n) への改善 |
| 大量データの全件取得 | メモリ圧迫・転送遅延 | ページネーション、ストリーム処理 |

**やむを得ず高コストな実装が必要な場合**:
- コードレビュー時に理由・影響範囲を明示し、レビュアーと合意の上で採用する
- コメントに採用理由と許容条件 (想定データ量等) を記載する
- 将来の改善可能性がある場合は TODO を残す

```go
// NOTE: 外部APIがバッチ取得をサポートしていないため、ループ内で個別取得している。
// 想定データ量: 最大20件。これを超える場合は並行処理への変更を検討する。
// レビュー合意: 2025-01-15 @reviewer-name
// TODO: バッチAPIが提供されたら一括取得に変更する (Issue #456)
```

### 不要なアロケーション回避

```go
// ✅ 良い例: strings.Builder で文字列を結合
var b strings.Builder
for _, s := range items {
    b.WriteString(s)
}
result := b.String()

// ❌ 悪い例: += で文字列を結合 (毎回アロケーション)
result := ""
for _, s := range items {
    result += s
}
```

## テストコード

### テストの構造

```go
func TestUserService_FindByID(t *testing.T) {
    tests := []struct {
        name    string
        id      string
        want    *User
        wantErr error
    }{
        {
            name: "existing user",
            id:   "user-1",
            want: &User{ID: "user-1", Name: "Alice"},
        },
        {
            name:    "non-existing user",
            id:      "unknown",
            wantErr: ErrNotFound,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            repo := &mockUserRepository{
                users: map[string]*User{
                    "user-1": {ID: "user-1", Name: "Alice"},
                },
            }
            svc := NewUserService(repo)

            // Act
            got, err := svc.FindByID(context.Background(), tt.id)

            // Assert
            if tt.wantErr != nil {
                if !errors.Is(err, tt.wantErr) {
                    t.Errorf("want error %v, got %v", tt.wantErr, err)
                }
                return
            }
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if got.ID != tt.want.ID {
                t.Errorf("want ID %s, got %s", tt.want.ID, got.ID)
            }
        })
    }
}
```

### テストヘルパー

```go
// t.Helper() を使ってテストヘルパーを明示する
func setupTestDB(t *testing.T) *sql.DB {
    t.Helper()
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatalf("failed to open db: %v", err)
    }
    t.Cleanup(func() { db.Close() })
    return db
}
```

### モックの作成

```go
// インターフェースに基づくモック
type mockUserRepository struct {
    users map[string]*User
}

func (m *mockUserRepository) FindByID(ctx context.Context, id string) (*User, error) {
    user, ok := m.users[id]
    if !ok {
        return nil, ErrNotFound
    }
    return user, nil
}

func (m *mockUserRepository) Save(ctx context.Context, user *User) error {
    m.users[user.ID] = user
    return nil
}
```

## リファクタリング

### マジックナンバーの排除

```go
// ✅ 良い例: 定数を定義
const (
    maxRetryCount = 3
    retryDelay    = 1 * time.Second
)

for i := 0; i < maxRetryCount; i++ {
    if err := doSomething(); err == nil {
        return nil
    }
    time.Sleep(retryDelay)
}

// ❌ 悪い例: マジックナンバー
for i := 0; i < 3; i++ {
    if err := doSomething(); err == nil {
        return nil
    }
    time.Sleep(1 * time.Second)
}
```

### 早期リターン

```go
// ✅ 良い例: 早期リターンでネストを浅く
func (s *UserService) Create(ctx context.Context, input CreateUserInput) (*User, error) {
    if err := input.Validate(); err != nil {
        return nil, err
    }

    exists, err := s.repo.ExistsByEmail(ctx, input.Email)
    if err != nil {
        return nil, fmt.Errorf("checking email: %w", err)
    }
    if exists {
        return nil, &ValidationError{Field: "email", Message: "already in use"}
    }

    user := &User{
        ID:    uuid.New().String(),
        Name:  input.Name,
        Email: input.Email,
    }
    if err := s.repo.Save(ctx, user); err != nil {
        return nil, fmt.Errorf("saving user: %w", err)
    }
    return user, nil
}

// ❌ 悪い例: 深いネスト
func (s *UserService) Create(ctx context.Context, input CreateUserInput) (*User, error) {
    if err := input.Validate(); err == nil {
        exists, err := s.repo.ExistsByEmail(ctx, input.Email)
        if err == nil {
            if !exists {
                user := &User{ID: uuid.New().String(), Name: input.Name, Email: input.Email}
                if err := s.repo.Save(ctx, user); err == nil {
                    return user, nil
                } else {
                    return nil, err
                }
            } else {
                return nil, errors.New("email already in use")
            }
        } else {
            return nil, err
        }
    } else {
        return nil, err
    }
}
```

## 品質自動化 (Go)

### ツール構成

| カテゴリ | ツール | 用途 |
|---|---|---|
| フォーマット | `gofmt` / `goimports` | コードスタイルの自動整形 |
| 静的解析 | `go vet` | 論理エラー・疑わしいコードの検出 |
| Lint | `golangci-lint` | 複数 Linter の統合実行 |
| テスト | `go test` | ユニット・統合テストの実行 |
| カバレッジ | `go test -coverprofile` | テストカバレッジの計測 |
| ビルド | `go build` | コンパイル確認 |

### カバレッジ計測

```bash
# カバレッジ計測
go test -coverprofile=coverage.out ./...

# カバレッジレポートの表示
go tool cover -func=coverage.out

# HTML レポートの生成
go tool cover -html=coverage.out -o coverage.html
```

### CI/CD (GitHub Actions)

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
      - run: goimports -l . | grep . && exit 1 || true
      - run: go vet ./...
      - run: golangci-lint run
      - run: go test -race -coverprofile=coverage.out ./...
      - run: go build ./...
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

## チェックリスト

実装完了前に確認:

### コード品質
- [ ] 命名が Go の慣習に従っている (頭字語、レシーバ名など)
- [ ] 関数が単一の責務を持っている
- [ ] マジックナンバーがない
- [ ] 早期リターンでネストが浅い

### エラーハンドリング
- [ ] すべてのエラーをチェックしている
- [ ] エラーに `%w` でコンテキストを付与している
- [ ] エラーメッセージが小文字で始まっている

### 並行処理
- [ ] goroutine がリークしない
- [ ] context によるキャンセルが適切に伝播する
- [ ] 共有リソースへのアクセスが保護されている

### パフォーマンス
- [ ] スライス・マップの容量を事前確保している
- [ ] 不要なアロケーションを避けている
- [ ] defer の使い忘れがない (Close, Unlock 等)

### テスト
- [ ] テーブル駆動テストを使用している
- [ ] t.Helper() をヘルパー関数に付与している
- [ ] エッジケースがカバーされている

### ツール
- [ ] `go vet` でエラーがない
- [ ] `go lint` (staticcheck 等) でエラーがない
- [ ] `gofmt` / `goimports` でフォーマット済み

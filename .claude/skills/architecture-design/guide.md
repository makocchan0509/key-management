# アーキテクチャ設計ガイド

## 基本原則

### 1. 技術選定にはバージョンと理由を明記すること

**悪い例**:
```
- Go
- chi
- opentelemetry-go
- gorm
- mysql
```

**良い例**:
```
- Go 1.25.5
  - （理由を記載）

- chi 5.2.3
  - （理由を記載）

- opentelemetry-go 1.39.0
  - (理由を記載)

- gorm v1.31.0
  - (理由を記載)

- mysql 8.4
  - (理由を記載)
```

### 2. レイヤー分離の原則

各レイヤーの責務を明確にし、依存関係を一方向に保ちます:

```
API → Service → Data (OK)
API ← Service (NG)
API → Data (NG)
```

### 3. 測定可能な要件

すべてのパフォーマンス要件は測定可能な形で記述します。

## レイヤードアーキテクチャの設計

### 各レイヤーの責務

#### ドメインモデル・インタフェース:
goによる実装例
```go
package domain

import (
	"context"
	"gorm.io/gorm"
)

// Task は GORM モデル
type Task struct {
	gorm.Model        // ID, CreatedAt, UpdatedAt, DeletedAt を自動付与
	Title             string `gorm:"size:255;not null"`
	EstimatedPriority int    `gorm:"index"`
}

// TaskRepository インターフェース
type TaskRepository interface {
	Save(ctx context.Context, task *Task) error
	FindByID(ctx context.Context, id uint) (*Task, error)
}

// GORM を使った実装
type gormTaskRepository struct {
	db *gorm.DB
}

func NewGORMTaskRepository(db *gorm.DB) TaskRepository {
	return &gormTaskRepository{db: db}
}

func (r *gormTaskRepository) Save(ctx context.Context, task *Task) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *gormTaskRepository) FindByID(ctx context.Context, id uint) (*Task, error) {
	var task Task
	err := r.db.WithContext(ctx).First(&task, id).Error
	return &task, err
}
```

#### サービスレイヤー:
goによる実装例
```go
package service

import (
    "context"
    "your-app/domain"
)

type taskService struct {
    repo domain.TaskRepository // インターフェースに依存させる
}

func NewTaskService(r domain.TaskRepository) domain.TaskService {
    return &taskService{repo: r}
}

func (s *taskService) Create(ctx context.Context, title string) (*domain.Task, error) {
    // ビジネスロジック: 優先度の自動推定（例）
    priority := len(title) % 5 

    task := &domain.Task{
        Title:             title,
        EstimatedPriority: priority,
    }

    if err := s.repo.Save(ctx, task); err != nil {
        return nil, err
    }
    return task, nil
}
```

#### CLIレイヤー:
goによる実装例
```go
package ui

import (
	"context"
	"fmt"
	"your-app/domain"
)

type CLIHandler struct {
	service domain.TaskService
}

func (h *CLIHandler) HandleAddTask(title string) {
	// CLI固有のバリデーション
	if title == "" {
		fmt.Println("Error: title is required")
		return
	}

	task, err := h.service.Create(context.Background(), title)
	if err != nil {
		fmt.Printf("Failed: %v\n", err)
		return
	}
	fmt.Printf("Successfully created task (ID: %d, Priority: %d)\n", task.ID, task.EstimatedPriority)
}
```

#### APIレイヤー:
goによる実装例
```go
package ui

import (
	"encoding/json"
	"net/http"
	"your-app/domain"
)

type HTTPHandler struct {
	service domain.TaskService
}

type CreateTaskRequest struct {
	Title string `json:"title"`
}

func (h *HTTPHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// サービスレイヤーを呼び出し
	task, err := h.service.Create(r.Context(), req.Title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}
```

#### APIレイヤー GRPCパターン:
goによる実装例
```go
package ui

import (
	"context"
	"your-app/domain"
	pb "your-app/proto/task" // 生成されたコード
)

type GRPCHandler struct {
	pb.UnimplementedTaskServiceServer
	service domain.TaskService
}

func (h *GRPCHandler) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.TaskResponse, error) {
	// サービスレイヤーを呼び出し
	task, err := h.service.Create(ctx, req.Title)
	if err != nil {
		return nil, err
	}

	return &pb.TaskResponse{
		Id:       uint64(task.ID),
		Title:    task.Title,
		Priority: int32(task.EstimatedPriority),
	}, nil
}
```

## パフォーマンス要件の設定

### 具体的な数値目標
以下は例
```
コマンド実行時間: 1000ms以内(平均的なPC環境で)
└─ 測定方法: console.timeでCLI起動から結果表示まで計測
└─ 測定環境: CPU Core i5相当、メモリ8GB、SSD

API応答時間: 1000ms以内
└─ 測定方法: テストコードの中でAPI呼び出し前後のタイムスタンプから算出する
└─ 測定環境: CPU Core i5相当、メモリ8GB、SSD
```

## セキュリティ設計

### 以下の観点で具体的な実装方法を記載すること

1. **入力検証**
- CLI、APIそれぞれでクライアントの入力値、リクエストパラメーターを検証する

2. **秘匿情報の暗号化**
- データベース上に秘匿性の高い情報を保存する場合は暗号化することを検討する。
- 暗号化を行う場合、暗号鍵の管理方法を明確にする。

3. **OWASP Top 10に基づいたWebアプリケーションセキュリティ対策**
- OWASP Top 10を参考に、セキュリティ対策を行う。[サイト](https://owasp.org/Top10/2025/)を参照すること。

## チェックリスト

- [ ] すべての技術選定に理由が記載されている
- [ ] アーキテクチャが明確に定義されている
- [ ] パフォーマンス要件が測定可能である
- [ ] セキュリティ考慮事項が記載されている
- [ ] テスト戦略が定義されている
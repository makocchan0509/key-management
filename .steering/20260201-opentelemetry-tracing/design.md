# 設計: OpenTelemetryトレーシング

## アーキテクチャ概要

### 変更箇所

1. **config/config.go**: OpenTelemetry関連の設定を追加
2. **internal/infra/tracer.go**: トレーサープロバイダーの初期化（新規作成）
3. **internal/infra/logger.go**: トレース情報付きロガー（新規作成）
4. **internal/infra/database.go**: gormのOpenTelemetryプラグイン追加
5. **internal/middleware/tracing.go**: HTTPミドルウェア（新規作成）
6. **internal/usecase/key_service.go**: サービス層でのスパン生成
7. **internal/handler/router.go**: トレーシングミドルウェア適用
8. **cmd/server/main.go**: トレーサー初期化・シャットダウン・ロガー設定
9. **go.mod**: OpenTelemetry依存関係追加

### 設計方針

**原則**:
- OTEL_ENABLED=false（デフォルト）の場合は既存の動作に影響しない
- functional-design.mdに記載のパターンに準拠
- 各レイヤーで適切なスパンを生成
- エラー時はspan.RecordError()で記録

## 詳細設計

### 1. config/config.go

**追加フィールド**:

```go
type Config struct {
    // 既存フィールド
    Port               string
    DatabaseURL        string
    KMSKeyName         string
    GoogleCloudProject string
    LogLevel           string
    // 追加フィールド
    OtelEnabled       bool
    OtelEndpoint      string
    OtelServiceName   string
    OtelSamplingRate  float64
}
```

**Load関数の修正**:

```go
func Load() *Config {
    return &Config{
        Port:               getEnv("PORT", "8080"),
        DatabaseURL:        os.Getenv("DATABASE_URL"),
        KMSKeyName:         os.Getenv("KMS_KEY_NAME"),
        GoogleCloudProject: os.Getenv("GOOGLE_CLOUD_PROJECT"),
        LogLevel:           getEnv("LOG_LEVEL", "INFO"),
        OtelEnabled:        os.Getenv("OTEL_ENABLED") == "true",
        OtelEndpoint:       os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
        OtelServiceName:    getEnv("OTEL_SERVICE_NAME", "key-management-service"),
        OtelSamplingRate:   getEnvFloat("OTEL_SAMPLING_RATE", 1.0),
    }
}

func getEnvFloat(key string, defaultVal float64) float64 {
    if val := os.Getenv(key); val != "" {
        if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 1 {
            return f
        }
    }
    return defaultVal
}
```

### 2. internal/infra/tracer.go（新規作成）

**ファイル内容**:

```go
package infra

import (
    "context"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
    "key-management-service/config"
)

// InitTracer はトレーサープロバイダーを初期化する。
// OTEL_ENABLED=false の場合は nil を返す（トレーシング無効）。
func InitTracer(ctx context.Context, cfg *config.Config) (*sdktrace.TracerProvider, error) {
    if !cfg.OtelEnabled {
        return nil, nil
    }

    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint(cfg.OtelEndpoint),
        otlptracegrpc.WithInsecure(), // ローカル開発用。本番ではTLS設定を推奨
    )
    if err != nil {
        return nil, err
    }

    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceName(cfg.OtelServiceName),
        ),
    )
    if err != nil {
        return nil, err
    }

    // サンプリング率を設定
    sampler := sdktrace.TraceIDRatioBased(cfg.OtelSamplingRate)

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
        sdktrace.WithSampler(sdktrace.ParentBased(sampler)),
    )

    otel.SetTracerProvider(tp)

    // W3C TraceContext伝搬を設定
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))

    return tp, nil
}
```

### 3. internal/infra/logger.go（新規作成）

**ファイル内容**:

slogのカスタムハンドラを作成し、コンテキストからトレース情報を抽出してログに付与:

```go
package infra

import (
    "context"
    "log/slog"

    "go.opentelemetry.io/otel/trace"
    "key-management-service/config"
)

// TraceHandler はトレース情報をログに付与するslogハンドラ。
type TraceHandler struct {
    handler      slog.Handler
    projectID    string
    otelEnabled  bool
}

// NewTraceHandler はトレース情報付きのslogハンドラを生成する。
func NewTraceHandler(handler slog.Handler, cfg *config.Config) *TraceHandler {
    return &TraceHandler{
        handler:     handler,
        projectID:   cfg.GoogleCloudProject,
        otelEnabled: cfg.OtelEnabled,
    }
}

// Enabled はハンドラがログを処理するかどうかを返す。
func (h *TraceHandler) Enabled(ctx context.Context, level slog.Level) bool {
    return h.handler.Enabled(ctx, level)
}

// Handle はログレコードを処理し、トレース情報を付与する。
func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
    if h.otelEnabled {
        span := trace.SpanFromContext(ctx)
        if span.SpanContext().IsValid() {
            spanCtx := span.SpanContext()
            traceID := spanCtx.TraceID().String()
            spanID := spanCtx.SpanID().String()
            sampled := spanCtx.IsSampled()

            // 基本トレース情報を追加
            r.AddAttrs(
                slog.String("trace", traceID),
                slog.String("spanId", spanID),
                slog.Bool("traceSampled", sampled),
            )

            // Google Cloud Logging連携用フィールドを追加
            if h.projectID != "" {
                r.AddAttrs(
                    slog.String("logging.googleapis.com/trace",
                        "projects/"+h.projectID+"/traces/"+traceID),
                    slog.String("logging.googleapis.com/spanId", spanID),
                )
            }
        }
    }

    return h.handler.Handle(ctx, r)
}

// WithAttrs は属性を追加した新しいハンドラを返す。
func (h *TraceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return &TraceHandler{
        handler:     h.handler.WithAttrs(attrs),
        projectID:   h.projectID,
        otelEnabled: h.otelEnabled,
    }
}

// WithGroup はグループを追加した新しいハンドラを返す。
func (h *TraceHandler) WithGroup(name string) slog.Handler {
    return &TraceHandler{
        handler:     h.handler.WithGroup(name),
        projectID:   h.projectID,
        otelEnabled: h.otelEnabled,
    }
}

// SetupLogger はトレース情報付きのグローバルロガーを設定する。
func SetupLogger(cfg *config.Config, level slog.Level) {
    jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
    traceHandler := NewTraceHandler(jsonHandler, cfg)
    slog.SetDefault(slog.New(traceHandler))
}
```

**出力例**（OTEL_ENABLED=true時）:

```json
{
  "time": "2026-02-01T10:30:00.000Z",
  "level": "INFO",
  "msg": "Key operation completed",
  "operation": "CREATE_KEY",
  "tenant_id": "tenant-001",
  "trace": "0af7651916cd43dd8448eb211c80319c",
  "spanId": "b7ad6b7169203331",
  "traceSampled": true,
  "logging.googleapis.com/trace": "projects/my-project/traces/0af7651916cd43dd8448eb211c80319c",
  "logging.googleapis.com/spanId": "b7ad6b7169203331"
}
```

### 4. internal/infra/database.go

**修正内容**:

gormのOpenTelemetryプラグインを条件付きで適用（セクション番号調整: 旧3→新4）:

```go
package infra

import (
    "log/slog"
    "time"

    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
    "gorm.io/plugin/opentelemetry/tracing"
    "key-management-service/config"
)

// NewDB はgormによるデータベース接続を初期化する。
func NewDB(dsn string, cfg *config.Config) (*gorm.DB, error) {
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Silent),
    })
    if err != nil {
        slog.Error("failed to open database connection",
            "operation", "db_init",
            "error", err,
        )
        return nil, err
    }

    // OpenTelemetryプラグインを適用（OTEL_ENABLED=trueの場合のみ）
    if cfg.OtelEnabled {
        if err := db.Use(tracing.NewPlugin()); err != nil {
            slog.Error("failed to apply OpenTelemetry plugin",
                "operation", "db_init",
                "error", err,
            )
            return nil, err
        }
    }

    sqlDB, err := db.DB()
    if err != nil {
        slog.Error("failed to get underlying sql.DB",
            "operation", "db_init",
            "error", err,
        )
        return nil, err
    }

    // 接続プール設定
    sqlDB.SetMaxOpenConns(10)
    sqlDB.SetMaxIdleConns(5)
    sqlDB.SetConnMaxLifetime(30 * time.Minute)

    return db, nil
}
```

### 4. internal/middleware/tracing.go（新規作成）

**ファイル内容**:

```go
package middleware

import (
    "net/http"

    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Tracing はOpenTelemetryによるHTTPトレーシングミドルウェアを返す。
func Tracing(serviceName string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return otelhttp.NewHandler(next, serviceName,
            otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
                return r.Method + " " + r.URL.Path
            }),
        )
    }
}
```

### 5. internal/usecase/key_service.go

**修正内容**:

各メソッドにスパン生成を追加:

```go
package usecase

import (
    "context"
    "crypto/rand"
    "fmt"
    "log/slog"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"

    "key-management-service/internal/domain"
)

var tracer = otel.Tracer("key-management-service")

// CreateKey は指定されたテナントに対して新しい暗号鍵を生成する。
func (s *KeyService) CreateKey(ctx context.Context, tenantID string) (*domain.KeyMetadata, error) {
    ctx, span := tracer.Start(ctx, "KeyService.CreateKey",
        trace.WithAttributes(
            attribute.String("tenant.id", tenantID),
        ),
    )
    defer span.End()

    // 既存チェック
    exists, err := s.repo.ExistsByTenantID(ctx, tenantID)
    if err != nil {
        span.RecordError(err)
        slog.ErrorContext(ctx, "failed to check existing key", ...)
        return nil, fmt.Errorf("checking existing key: %w", err)
    }
    // 以下、既存ロジック...

    span.SetAttributes(attribute.Int("key.generation", 1))
    return metadata, nil
}
```

**パターン**:
- メソッド冒頭で`tracer.Start()`
- `defer span.End()`でスパン終了を保証
- エラー時は`span.RecordError(err)`
- 成功時は追加属性を`span.SetAttributes()`

### 6. internal/handler/router.go

**修正内容**:

トレーシングミドルウェアをルーターに適用:

```go
package handler

import (
    "net/http"

    "github.com/go-chi/chi/v5"
    chimiddleware "github.com/go-chi/chi/v5/middleware"
    "key-management-service/config"
    "key-management-service/internal/middleware"
)

// NewRouter はルーターを生成する。
func NewRouter(h *KeyHandler, cfg *config.Config) http.Handler {
    r := chi.NewRouter()

    // ミドルウェア
    r.Use(chimiddleware.Logger)
    r.Use(chimiddleware.Recoverer)
    r.Use(chimiddleware.RequestID)

    // トレーシングミドルウェア（OTEL_ENABLED=trueの場合のみ）
    if cfg.OtelEnabled {
        r.Use(middleware.Tracing(cfg.OtelServiceName))
    }

    // ルート定義
    r.Route("/v1/tenants/{tenant_id}/keys", func(r chi.Router) {
        r.Post("/", h.CreateKey)
        r.Get("/", h.ListKeys)
        r.Get("/current", h.GetCurrentKey)
        r.Get("/{generation}", h.GetKeyByGeneration)
        r.Delete("/{generation}", h.DisableKey)
        r.Post("/rotate", h.RotateKey)
    })

    return r
}
```

### 8. cmd/server/main.go

**修正内容**:

トレーサー初期化、ロガー設定、シャットダウンを追加:

```go
func main() {
    ctx := context.Background()

    // .envファイルを読み込む
    _ = godotenv.Load()

    // 設定読み込み
    cfg := config.Load()

    // ログレベル設定
    var logLevel slog.Level
    switch cfg.LogLevel {
    case "DEBUG":
        logLevel = slog.LevelDebug
    case "WARN":
        logLevel = slog.LevelWarn
    case "ERROR":
        logLevel = slog.LevelError
    default:
        logLevel = slog.LevelInfo
    }

    // トレーサー初期化（ロガー設定の前に実行）
    tp, err := infra.InitTracer(ctx, cfg)
    if err != nil {
        slog.Error("failed to init tracer", "error", err)
        os.Exit(1)
    }
    if tp != nil {
        defer func() {
            if err := tp.Shutdown(ctx); err != nil {
                slog.Error("failed to shutdown tracer", "error", err)
            }
        }()
    }

    // トレース情報付きロガーを設定
    infra.SetupLogger(cfg, logLevel)

    // DB初期化（cfgを渡す）
    db, err := infra.NewDB(cfg.DatabaseURL, cfg)
    if err != nil {
        slog.Error("failed to init database", "error", err)
        os.Exit(1)
    }

    // ...

    // ルーター生成（cfgを渡す）
    router := handler.NewRouter(h, cfg)

    // ...
}
```

### 8. go.mod

**追加依存関係**:

```bash
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/sdk/trace
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
go get go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
go get gorm.io/plugin/opentelemetry/tracing
```

## ファイル変更サマリー

| ファイル | 操作 | 変更内容 |
|---------|------|---------|
| config/config.go | 修正 | OpenTelemetry設定フィールドを追加 |
| internal/infra/tracer.go | 新規 | トレーサープロバイダー初期化 |
| internal/infra/logger.go | 新規 | トレース情報付きslogハンドラ |
| internal/infra/database.go | 修正 | gormのOTelプラグイン適用、NewDB関数のシグネチャ変更 |
| internal/middleware/tracing.go | 新規 | HTTPトレーシングミドルウェア |
| internal/usecase/key_service.go | 修正 | 各メソッドにスパン生成を追加 |
| internal/handler/router.go | 修正 | トレーシングミドルウェア適用、NewRouter関数のシグネチャ変更 |
| cmd/server/main.go | 修正 | トレーサー初期化・ロガー設定・シャットダウン、関数呼び出しの修正 |
| go.mod, go.sum | 修正 | OpenTelemetry依存関係追加 |
| .env.example | 修正 | OpenTelemetry環境変数のサンプル追加 |

## 関数シグネチャの変更

以下の関数はシグネチャが変更される:

1. `infra.NewDB(dsn string)` → `infra.NewDB(dsn string, cfg *config.Config)`
2. `handler.NewRouter(h *KeyHandler)` → `handler.NewRouter(h *KeyHandler, cfg *config.Config)`

これにより、main.goでの呼び出し箇所も修正が必要。

## テスト影響

### 既存テストへの影響

- `NewDB`と`NewRouter`のシグネチャ変更により、テストコードの修正が必要
- テスト用に`OtelEnabled: false`の設定を渡す

### 新規テスト

- 今回はユニットテストの新規追加は行わない（統合テストはCloud環境で実施）

## セキュリティ考慮事項

- トレーシングデータにはテナントIDが含まれるが、機密情報（鍵データ）は含めない
- Cloud Traceへの送信はTLS暗号化（本番環境）

## 後方互換性

- OTEL_ENABLED=false（デフォルト）の場合、既存の動作と完全に互換
- 環境変数を設定しなければ、トレーシングは無効

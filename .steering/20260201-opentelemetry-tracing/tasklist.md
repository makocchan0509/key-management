# タスクリスト: OpenTelemetryトレーシング

## 概要

OpenTelemetryによる分散トレーシングを実装し、APIリクエストの全体的なレイテンシ分析を可能にする。

## フェーズ1: 依存関係の追加

### 1.1 OpenTelemetry関連パッケージのインストール

- [x] `go get go.opentelemetry.io/otel`
- [x] `go get go.opentelemetry.io/otel/sdk/trace`
- [x] `go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc`
- [x] `go get go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`
- [x] `go get gorm.io/plugin/opentelemetry/tracing`
- [x] `go mod tidy`で依存関係を整理

## フェーズ2: 設定の追加

### 2.1 config/config.goの修正

- [x] Config構造体にOpenTelemetry関連フィールドを追加
  - OtelEnabled (bool)
  - OtelEndpoint (string)
  - OtelServiceName (string)
  - OtelSamplingRate (float64)
- [x] Load()関数を修正して環境変数から読み込む
- [x] getEnvFloat()ヘルパー関数を追加
- [x] strconvパッケージをimportに追加

### 2.2 .env.exampleの修正
- [x] OpenTelemetry関連の環境変数を追加
  - OTEL_ENABLED
  - OTEL_EXPORTER_OTLP_ENDPOINT
  - OTEL_SERVICE_NAME
  - OTEL_SAMPLING_RATE

## フェーズ3: インフラ層の実装

### 3.1 internal/infra/tracer.goの新規作成

- [x] InitTracer関数を実装
  - cfg.OtelEnabledがfalseの場合はnilを返す
  - OTLPエクスポーター（gRPC）を設定
  - リソース属性（サービス名）を設定
  - サンプリング率を設定
  - W3C TraceContext伝搬を設定
  - TracerProviderを返す

### 3.2 internal/infra/logger.goの新規作成

- [x] TraceHandler構造体を定義（slog.Handlerを内包）
- [x] NewTraceHandler関数を実装
- [x] Enabled, Handle, WithAttrs, WithGroupメソッドを実装
- [x] Handle内でコンテキストからSpanを取得
- [x] trace, spanId, traceSampledフィールドをログに追加
- [x] logging.googleapis.com/trace, logging.googleapis.com/spanIdをログに追加
- [x] SetupLogger関数を実装（グローバルロガーを設定）
- [x] go.opentelemetry.io/otel/traceパッケージをimportに追加

### 3.3 internal/infra/database.goの修正

- [x] NewDB関数のシグネチャを変更（cfg *config.Configを追加）
- [x] gorm OpenTelemetryプラグインのimportを追加
- [x] cfg.OtelEnabled=trueの場合、tracing.NewPlugin()を適用
- [x] config パッケージをimportに追加

## フェーズ4: ミドルウェアの実装

### 4.1 internal/middleware/tracing.goの新規作成

- [x] Tracing関数を実装
  - serviceNameを引数に取る
  - otelhttp.NewHandlerでHTTPハンドラをラップ
  - スパン名フォーマッター（METHOD + PATH）を設定

## フェーズ5: ハンドラー層の修正

### 5.1 internal/handler/router.goの修正

- [x] NewRouter関数のシグネチャを変更（cfg *config.Configを追加）
- [x] config パッケージをimportに追加
- [x] internal/middleware パッケージをimportに追加
- [x] cfg.OtelEnabled=trueの場合、トレーシングミドルウェアを適用

## フェーズ6: サービス層のトレーシング

### 6.1 internal/usecase/key_service.goの修正

- [x] OpenTelemetryパッケージをimportに追加
  - go.opentelemetry.io/otel
  - go.opentelemetry.io/otel/attribute
  - go.opentelemetry.io/otel/trace
- [x] パッケージレベルでtracerを定義
- [x] CreateKeyメソッドにスパン生成を追加
- [x] GetCurrentKeyメソッドにスパン生成を追加
- [x] GetKeyByGenerationメソッドにスパン生成を追加
- [x] RotateKeyメソッドにスパン生成を追加
- [x] ListKeysメソッドにスパン生成を追加
- [x] DisableKeyメソッドにスパン生成を追加

## フェーズ7: エントリポイントの修正

### 7.1 cmd/server/main.goの修正

- [x] infra.InitTracer()の呼び出しを追加（config.Load()の後、ログ設定の前）
- [x] TracerProviderのShutdownをdeferで設定
- [x] 既存のslog.SetDefault()をinfra.SetupLogger()に置き換え
- [x] infra.NewDB()の呼び出しをNewDB(dsn, cfg)に変更
- [x] handler.NewRouter()の呼び出しをNewRouter(h, cfg)に変更

## フェーズ8: テストの修正

### 8.1 既存テストの修正

- [x] internal/handler/router_test.goが存在する場合、NewRouterの呼び出しを修正（テストではNewRouterを直接使用していないため修正不要）
- [x] internal/repository/key_repository_test.goが存在する場合、NewDBの呼び出しを修正（テストではNewDBを直接使用していないため修正不要）
- [x] テスト用にOtelEnabled: falseの設定を渡す（テストでは該当関数を使用していないため修正不要）

## フェーズ9: 動作確認

### 9.1 テストの実行

- [x] `go test ./...`が成功することを確認

### 9.2 Lintチェック

- [x] `golangci-lint run`が成功することを確認

### 9.3 ビルド確認

- [x] `go build ./cmd/...`が成功することを確認

## 実装後の振り返り

### 実装完了日

2026-02-01

### 計画と実績の差分

1. **CLIツール（cmd/keyctl/migrate.go）の修正が追加で必要だった**
   - 計画にはCLIツールの修正が含まれていなかったが、NewDBのシグネチャ変更に伴い修正が必要になった
   - CLIではトレーシングを使用しないため、`OtelEnabled: false`の設定を渡す形で対応

2. **テストファイルの修正は不要だった**
   - 計画ではNewRouterやNewDBを使用するテストの修正を想定していたが、既存テストではこれらの関数を直接使用していなかったため修正不要だった

### 学んだこと

1. **slogのカスタムハンドラによるトレース情報付与**
   - slog.Handlerインターフェースを実装することで、コンテキストからSpanを取得し、ログに自動的にトレース情報を付与できる
   - Google Cloud Logging連携には`logging.googleapis.com/trace`形式のフィールドが必要

2. **OpenTelemetryの初期化順序**
   - TracerProviderはロガー設定より前に初期化する必要がある
   - シャットダウンはdeferで確実に実行する

3. **gormのOpenTelemetryプラグイン**
   - `gorm.io/plugin/opentelemetry/tracing`を使うことで、DB操作のスパンを自動生成できる
   - `db.Use(tracing.NewPlugin())`で簡単に適用可能

4. **関数シグネチャ変更の影響範囲**
   - NewDBやNewRouterのシグネチャを変更すると、呼び出し元（main.go、CLIツール）も修正が必要
   - 事前に影響範囲を洗い出すことが重要

### 次回への改善提案

1. **TLS設定の環境変数化**
   - 現在はOTLPエクスポーターのTLSがハードコードで無効化されている
   - 本番環境向けに`OTEL_INSECURE`環境変数を追加し、TLS設定を制御できるようにする

2. **Handler層でのスパン生成追加**
   - 現在はService層でのみスパンを生成しているが、Handler層でもスパンを生成するとより詳細なトレースが可能
   - スペックのスパン構成図に記載の`[child] API Handler`レイヤーを実装

3. **トレーシング関連のユニットテスト追加**
   - tracer.go、logger.go、tracing.goのテストがない
   - モックSpanを使用したテストを追加することでカバレッジ向上

4. **タスクリストにCLIツールの修正を含める**
   - 関数シグネチャ変更時は、CLIツールへの影響も事前にタスクに含める

### 実装しなかったタスク

なし（全タスク完了）

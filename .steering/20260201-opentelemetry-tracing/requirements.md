# 要求定義: OpenTelemetryトレーシング

## 概要

OpenTelemetryを使用してアプリケーションパフォーマンスのモニタリングを実現する。APIリクエストの受付から応答まで、関数ごとにスパンを生成し、Cloud Traceにエクスポートする。

## 背景

functional-design.mdおよびarchitecture.mdに記載されている通り、分散トレーシングは以下の目的で導入する:

- APIリクエストの全体的なレイテンシ分析
- 各レイヤー（Handler, Service, Repository, KMS）での処理時間の可視化
- エラー発生箇所の特定
- Cloud Traceとの連携によるAPMモニタリング

## 要求内容

### 1. 環境変数による制御

以下の環境変数でトレーシングを制御する:

| 変数名 | 必須 | 説明 | デフォルト値 |
|--------|------|------|-------------|
| OTEL_ENABLED | 任意 | OpenTelemetryの有効化 | false |
| OTEL_EXPORTER_OTLP_ENDPOINT | 任意 | OTLPエクスポート先（OTEL_ENABLED=true時に必須） | なし |
| OTEL_SERVICE_NAME | 任意 | サービス名 | key-management-service |
| OTEL_SAMPLING_RATE | 任意 | サンプリング率 0.0〜1.0 | 1.0 |
| GOOGLE_CLOUD_PROJECT | 必須 | GCPプロジェクトID（既存） | なし |

### 2. トレーサープロバイダーの初期化

**ファイル**: `internal/infra/tracer.go`（新規作成）

- OTEL_ENABLED=trueの場合のみトレーシングを有効化
- OTEL_ENABLED=false（デフォルト）の場合はトレーシング無効
- OTLPエクスポーター（gRPC）を使用
- W3C TraceContext伝搬を設定

### 3. HTTPミドルウェアによるトレーシング

**ファイル**: `internal/middleware/tracing.go`（新規作成）

- otelhttp.NewHandlerを使用してHTTPリクエストのスパンを自動生成
- スパン名: `{HTTP_METHOD} {URL_PATH}`
- 属性: http.method, http.route, http.status_code

### 4. サービス層でのスパン生成

**ファイル**: `internal/usecase/key_service.go`（修正）

各メソッドでスパンを生成:
- CreateKey
- GetCurrentKey
- GetKeyByGeneration
- RotateKey
- ListKeys
- DisableKey

属性:
- tenant.id
- key.generation（該当する場合）

エラー時はspan.RecordError()でエラーを記録。

### 5. gormのトレーシング

**ファイル**: `internal/infra/database.go`（修正）

- OTEL_ENABLED=trueの場合、gormのOpenTelemetryプラグインを適用
- 属性: db.system, db.operation

### 6. main.goの修正

**ファイル**: `cmd/server/main.go`（修正）

- トレーサープロバイダーの初期化を追加
- shutdown時にtp.Shutdown()を呼び出す
- トレーシングミドルウェアをルーターに適用

### 7. config.goの修正

**ファイル**: `config/config.go`（修正）

OpenTelemetry関連の設定を追加:
- OtelEnabled (bool)
- OtelEndpoint (string)
- OtelServiceName (string)
- OtelSamplingRate (float64)

### 8. ログとトレースのリンク

**ファイル**: `internal/infra/logger.go`（新規作成）

ログ出力時にトレースID、スパンID、サンプリング状態を自動的に付与する:

**出力フィールド**:
- `trace`: トレースID（例: "0af7651916cd43dd8448eb211c80319c"）
- `spanId`: スパンID（例: "b7ad6b7169203331"）
- `traceSampled`: サンプリング状態（true/false）

**Google Cloud Logging連携**:
Cloud Loggingでトレースとログをリンクさせるため、以下のフィールドも出力:
- `logging.googleapis.com/trace`: "projects/{project_id}/traces/{trace_id}"形式
- `logging.googleapis.com/spanId`: スパンID

**実装方式**:
- slogのカスタムハンドラを作成
- コンテキストからOpenTelemetryのSpanを取得
- ログ出力時に自動的にトレース情報を付与
- トレーシングが無効（OTEL_ENABLED=false）の場合はフィールドを出力しない

**ログ出力例**:
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

## スパン構成

```
[root] HTTP Request
  ├── [child] API Handler (CreateKey/GetKey/etc.)
  │     ├── [child] KeyService.CreateKey
  │     │     ├── [child] KMSClient.Encrypt/Decrypt
  │     │     └── [child] gorm DB操作
  │     └── [child] AuditLog.Write（将来対応）
  └── [child] HTTP Response
```

## 非機能要件

### パフォーマンス

- トレーシングがパフォーマンスに与える影響を最小限にする
- サンプリング率を調整可能にする（本番環境では低いレートを推奨）

### 後方互換性

- OTEL_ENABLED=false（デフォルト）の場合、既存の動作に影響しない
- 既存のテストがすべてパスすること

### 保守性

- OpenTelemetry公式ライブラリを使用
- 設計書（functional-design.md）に記載のパターンに準拠

## 成功基準

1. OTEL_ENABLED=trueの場合、スパンが生成されること
2. OTEL_ENABLED=trueの場合、ログにtrace、spanId、traceSampledが出力されること
3. OTEL_ENABLED=false（デフォルト）の場合、トレーシングが無効であること
4. 既存のテストがすべてパスすること
5. golangci-lintがパスすること
6. ビルドが成功すること

## 実装範囲外

- CLIツール（keyctl）のトレーシング（将来対応）
- Cloud Traceへのエクスポート設定（環境依存、手動設定）

## 参考情報

- functional-design.md: 分散トレーシング設計セクション
- architecture.md: 分散トレーシング技術スタック
- repository-structure.md: internal/infra/tracer.go, internal/middleware/tracing.go

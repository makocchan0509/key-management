package infra

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"

	"key-management-service/config"
)

// TraceHandler はトレース情報をログに付与するslogハンドラ。
type TraceHandler struct {
	handler     slog.Handler
	projectID   string
	otelEnabled bool
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

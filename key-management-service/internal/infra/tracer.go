// Package infra は外部サービスとの接続を提供する。
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
		// otlptracegrpc.WithInsecure(), // ローカル開発用。本番ではTLS設定を推奨
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

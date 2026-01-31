// Package main はAPIサーバーのエントリポイント。
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

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
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})))

	// DB初期化
	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL is not set")
		os.Exit(1)
	}
	db, err := infra.NewDB(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to init database", "error", err)
		os.Exit(1)
	}

	// KMSクライアント初期化
	kmsClient, err := infra.NewKMSClient(ctx)
	if err != nil {
		slog.Error("failed to init KMS client", "error", err)
		os.Exit(1)
	}
	defer func() {
		if closeErr := kmsClient.Close(); closeErr != nil {
			slog.Error("failed to close KMS client", "error", closeErr)
		}
	}()

	// DI
	repo := repository.NewKeyRepository(db)
	service := usecase.NewKeyService(repo, kmsClient)
	h := handler.NewKeyHandler(service)
	router := handler.NewRouter(h)

	// サーバー起動
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		<-sigCh

		slog.Info("shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
	}()

	slog.Info("starting server", "port", cfg.Port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}

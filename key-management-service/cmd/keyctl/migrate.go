package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"key-management-service/internal/domain"
	"key-management-service/internal/infra"
	"key-management-service/internal/repository"
	"key-management-service/internal/usecase"

	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage database migrations",
	Long:  "Manage database migrations for the key management service",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply pending migrations",
	Long:  "Apply all pending migrations to the database",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// DB接続情報を環境変数から取得
		dsn := os.Getenv("DATABASE_URL")
		if dsn == "" {
			return fmt.Errorf("DATABASE_URL environment variable is required")
		}

		// データベース接続
		db, err := infra.NewDB(dsn)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}

		// migrationsディレクトリのパスを取得（実行ファイルの位置から相対パス）
		migrationsDir := os.Getenv("MIGRATIONS_DIR")
		if migrationsDir == "" {
			// デフォルト: ./migrations
			migrationsDir = "./migrations"
		}

		// 絶対パスに変換
		absPath, err := filepath.Abs(migrationsDir)
		if err != nil {
			return fmt.Errorf("failed to resolve migrations directory: %w", err)
		}

		// MigrationServiceを初期化
		migrationRepo := repository.NewMigrationRepository(db)
		migrationService := usecase.NewMigrationService(migrationRepo, db, absPath)

		// マイグレーション実行
		appliedCount, err := migrationService.ApplyMigrations(ctx)
		if err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}

		if appliedCount == 0 {
			fmt.Println("No pending migrations.")
		} else {
			fmt.Printf("Applied %d migration(s) successfully.\n", appliedCount)
		}

		return nil
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	Long:  "Show the status of all migrations (applied/pending)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// DB接続情報を環境変数から取得
		dsn := os.Getenv("DATABASE_URL")
		if dsn == "" {
			return fmt.Errorf("DATABASE_URL environment variable is required")
		}

		// データベース接続
		db, err := infra.NewDB(dsn)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}

		// migrationsディレクトリのパスを取得
		migrationsDir := os.Getenv("MIGRATIONS_DIR")
		if migrationsDir == "" {
			migrationsDir = "./migrations"
		}

		// 絶対パスに変換
		absPath, err := filepath.Abs(migrationsDir)
		if err != nil {
			return fmt.Errorf("failed to resolve migrations directory: %w", err)
		}

		// MigrationServiceを初期化
		migrationRepo := repository.NewMigrationRepository(db)
		migrationService := usecase.NewMigrationService(migrationRepo, db, absPath)

		// マイグレーションステータスを取得
		migrations, err := migrationService.GetMigrationStatus(ctx)
		if err != nil {
			return fmt.Errorf("failed to get migration status: %w", err)
		}

		// テーブル形式で出力
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "VERSION\tNAME\tSTATUS\tAPPLIED AT")
		fmt.Fprintln(w, "-------\t----\t------\t----------")

		for _, migration := range migrations {
			appliedAt := "-"
			if migration.AppliedAt != nil {
				appliedAt = migration.AppliedAt.Format("2006-01-02 15:04:05")
			}

			status := "pending"
			if migration.Status == domain.MigrationStatusApplied {
				status = "applied"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", migration.Version, migration.Name, status, appliedAt)
		}

		if err := w.Flush(); err != nil {
			return fmt.Errorf("failed to flush output: %w", err)
		}

		return nil
	},
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
}

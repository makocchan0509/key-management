package domain

import "time"

// MigrationStatus はマイグレーションの適用状態を表す
type MigrationStatus string

const (
	MigrationStatusPending MigrationStatus = "pending"
	MigrationStatusApplied MigrationStatus = "applied"
)

// Migration はデータベースマイグレーションを表すドメインモデル
type Migration struct {
	Version   string          // マイグレーションバージョン（例: "001", "002"）
	Name      string          // マイグレーション名（ファイル名から抽出）
	AppliedAt *time.Time      // 適用日時（未適用の場合はnil）
	FilePath  string          // マイグレーションファイルのパス
	Status    MigrationStatus // 適用状態
}

// Package domain はドメインモデルとビジネスルールを定義する。
package domain

import "time"

// KeyStatus は暗号鍵のステータスを表す。
type KeyStatus string

const (
	// KeyStatusActive は有効な鍵を表す。
	KeyStatusActive KeyStatus = "active"
	// KeyStatusDisabled は無効化された鍵を表す。
	KeyStatusDisabled KeyStatus = "disabled"
)

// EncryptionKey は暗号鍵エンティティを表す。
type EncryptionKey struct {
	ID           string
	TenantID     string
	Generation   uint
	EncryptedKey []byte
	Status       KeyStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// KeyMetadata は暗号鍵のメタデータを表す（平文鍵を含まない）。
type KeyMetadata struct {
	TenantID   string
	Generation uint
	Status     KeyStatus
	CreatedAt  time.Time
}

// Key は復号済みの暗号鍵を表す。
type Key struct {
	TenantID   string
	Generation uint
	Key        []byte // 平文の鍵（Base64エンコード前）
}

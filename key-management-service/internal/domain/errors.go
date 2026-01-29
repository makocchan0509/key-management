package domain

import "errors"

var (
	// ErrKeyNotFound は指定されたテナント・世代の鍵が存在しない場合のエラー。
	ErrKeyNotFound = errors.New("key not found")

	// ErrKeyAlreadyExists は指定されたテナントに既に鍵が存在する場合のエラー。
	ErrKeyAlreadyExists = errors.New("key already exists")

	// ErrKeyDisabled は指定された鍵が無効化されている場合のエラー。
	ErrKeyDisabled = errors.New("key is disabled")

	// ErrKeyAlreadyDisabled は指定された鍵が既に無効化されている場合のエラー。
	ErrKeyAlreadyDisabled = errors.New("key is already disabled")

	// ErrInvalidTenantID はテナントIDの形式が不正な場合のエラー。
	ErrInvalidTenantID = errors.New("invalid tenant ID")

	// ErrInvalidGeneration は世代番号が不正な場合のエラー。
	ErrInvalidGeneration = errors.New("invalid generation")

	// ErrMigrationFailed はマイグレーション実行時のエラー。
	ErrMigrationFailed = errors.New("migration failed")

	// ErrMigrationFileNotFound はマイグレーションファイルが見つからない場合のエラー。
	ErrMigrationFileNotFound = errors.New("migration file not found")

	// ErrInvalidMigrationFile はマイグレーションファイルのフォーマットが不正な場合のエラー。
	ErrInvalidMigrationFile = errors.New("invalid migration file")
)

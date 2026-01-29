// Package config はアプリケーション設定の読み込みを提供する。
package config

import "os"

// Config はアプリケーション設定を表す。
type Config struct {
	Port               string
	DatabaseURL        string
	KMSKeyName         string
	GoogleCloudProject string
	LogLevel           string
}

// Load は環境変数から設定を読み込む。
func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		KMSKeyName:         os.Getenv("KMS_KEY_NAME"),
		GoogleCloudProject: os.Getenv("GOOGLE_CLOUD_PROJECT"),
		LogLevel:           getEnv("LOG_LEVEL", "INFO"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

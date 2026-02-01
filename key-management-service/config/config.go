// Package config はアプリケーション設定の読み込みを提供する。
package config

import (
	"os"
	"strconv"
)

// Config はアプリケーション設定を表す。
type Config struct {
	Port               string
	DatabaseURL        string
	KMSKeyName         string
	GoogleCloudProject string
	LogLevel           string
	OtelEnabled        bool
	OtelEndpoint       string
	OtelServiceName    string
	OtelSamplingRate   float64
}

// Load は環境変数から設定を読み込む。
func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		KMSKeyName:         os.Getenv("KMS_KEY_NAME"),
		GoogleCloudProject: os.Getenv("GOOGLE_CLOUD_PROJECT"),
		LogLevel:           getEnv("LOG_LEVEL", "INFO"),
		OtelEnabled:        os.Getenv("OTEL_ENABLED") == "true",
		OtelEndpoint:       os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		OtelServiceName:    getEnv("OTEL_SERVICE_NAME", "key-management-service"),
		OtelSamplingRate:   getEnvFloat("OTEL_SAMPLING_RATE", 1.0),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 1 {
			return f
		}
	}
	return defaultVal
}

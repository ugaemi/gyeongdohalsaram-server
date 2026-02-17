package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port        int
	LogLevel    string
	LogFormat   string
	DatabaseURL string

	// Game Center auth
	GCBundleIDs          []string
	GCTimestampTolerance time.Duration
}

func Load() *Config {
	return &Config{
		Port:                 getEnvInt("PORT", 8080),
		LogLevel:             getEnv("LOG_LEVEL", "info"),
		LogFormat:            getEnv("LOG_FORMAT", "text"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/gyeongdohalsaram?sslmode=disable"),
		GCBundleIDs:          getEnvStringSlice("GC_BUNDLE_IDS"),
		GCTimestampTolerance: time.Duration(getEnvInt("GC_TIMESTAMP_TOLERANCE", 300)) * time.Second,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvStringSlice(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			result = append(result, s)
		}
	}
	return result
}

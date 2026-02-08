package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port      int
	LogLevel  string
	LogFormat string
}

func Load() *Config {
	return &Config{
		Port:      getEnvInt("PORT", 8080),
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "text"),
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

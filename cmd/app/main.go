package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	LogLevel        string
	ShutdownTimeout time.Duration

	DBMaxConns        int32
	DBMinConns        int32
	DBConnectTimeout  time.Duration
	DBQueryTimeout    time.Duration
	DBMaxConnLifetime time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:        os.Getenv("HTTP_ADDR"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		ShutdownTimeout: getDuration("SHUTDOWN_TIMEOUT", 10*time.Second),

		DBMaxConns:        int32(getInt("DATABASE_MAX_CONNS", 10)),
		DBMinConns:        int32(getInt("DATABASE_MIN_CONNS", 2)),
		DBConnectTimeout:  getDuration("DATABASE_CONNECT_TIMEOUT", 5*time.Second),
		DBQueryTimeout:    getDuration("DATABASE_QUERY_TIMEOUT", 3*time.Second),
		DBMaxConnLifetime: getDuration("DATABASE_MAX_CONN_LIFETIME", 30*time.Minute),
	}

	if cfg.HTTPAddr == "" {
		return nil, fmt.Errorf("HTTP_ADDR is required")
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

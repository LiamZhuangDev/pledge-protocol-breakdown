package config

import (
	"os"
	"time"
)

const (
	defaultEnv        = "local"
	defaultPort       = "8080"
	defaultAPIVersion = "1"
	defaultChainID    = "97"
	defaultSyncEvery  = 2 * time.Minute
)

type Config struct {
	Env          string
	Port         string
	APIVersion   string
	ChainID      string
	SyncInterval time.Duration
}

func Load() Config {
	return Config{
		Env:          readEnv("PLEDGE_ENV", defaultEnv),
		Port:         readEnv("PLEDGE_API_PORT", defaultPort),
		APIVersion:   readEnv("PLEDGE_API_VERSION", defaultAPIVersion),
		ChainID:      readEnv("PLEDGE_CHAIN_ID", defaultChainID),
		SyncInterval: readDurationEnv("PLEDGE_SYNC_INTERVAL", defaultSyncEvery),
	}
}

func readEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func readDurationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}

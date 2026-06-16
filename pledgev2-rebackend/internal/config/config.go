package config

import (
	"os"
	"time"
)

const (
	defaultEnv         = "local"
	defaultPort        = "8080"
	defaultAPIVersion  = "1"
	defaultChainID     = "97"
	defaultSyncEvery   = 2 * time.Minute
	defaultAdminUser   = "admin"
	defaultAdminPass   = "password"
	defaultTokenTTL    = time.Hour
	defaultTokenSecret = "local-development-secret"
	defaultPriceSymbol = "PLGR"
)

type Config struct {
	Env           string
	Port          string
	APIVersion    string
	ChainID       string
	SyncInterval  time.Duration
	AdminUsername string
	AdminPassword string
	TokenSecret   string
	TokenTTL      time.Duration
	PriceSymbol   string
}

func Load() Config {
	return Config{
		Env:           readEnv("PLEDGE_ENV", defaultEnv),
		Port:          readEnv("PLEDGE_API_PORT", defaultPort),
		APIVersion:    readEnv("PLEDGE_API_VERSION", defaultAPIVersion),
		ChainID:       readEnv("PLEDGE_CHAIN_ID", defaultChainID),
		SyncInterval:  readDurationEnv("PLEDGE_SYNC_INTERVAL", defaultSyncEvery),
		AdminUsername: readEnv("PLEDGE_ADMIN_USERNAME", defaultAdminUser),
		AdminPassword: readEnv("PLEDGE_ADMIN_PASSWORD", defaultAdminPass),
		TokenSecret:   readEnv("PLEDGE_TOKEN_SECRET", defaultTokenSecret),
		TokenTTL:      readDurationEnv("PLEDGE_TOKEN_TTL", defaultTokenTTL),
		PriceSymbol:   readEnv("PLEDGE_PRICE_SYMBOL", defaultPriceSymbol),
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

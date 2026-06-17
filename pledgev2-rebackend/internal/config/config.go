package config

import (
	"fmt"
	"os"
	"strings"
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
	defaultStoreDriver = "memory"
	defaultRedisAddr   = "127.0.0.1:6379"
	defaultPriceTTL    = 30 * time.Second
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
	StoreDriver   string
	MySQLDSN      string
	RedisAddress  string
	RedisPassword string
	RedisDB       int
	PriceCacheTTL time.Duration
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
		StoreDriver:   strings.ToLower(readEnv("PLEDGE_STORE", defaultStoreDriver)),
		MySQLDSN:      readEnv("PLEDGE_MYSQL_DSN", ""),
		RedisAddress:  readEnv("PLEDGE_REDIS_ADDR", defaultRedisAddr),
		RedisPassword: readEnv("PLEDGE_REDIS_PASSWORD", ""),
		RedisDB:       readIntEnv("PLEDGE_REDIS_DB", 0),
		PriceCacheTTL: readDurationEnv("PLEDGE_PRICE_CACHE_TTL", defaultPriceTTL),
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

func readIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil {
		return fallback
	}
	return parsed
}

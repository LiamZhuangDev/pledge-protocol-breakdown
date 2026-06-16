package config

import "os"

const (
	defaultEnv        = "local"
	defaultPort       = "8080"
	defaultAPIVersion = "1"
)

type Config struct {
	Env        string
	Port       string
	APIVersion string
}

func Load() Config {
	return Config{
		Env:        readEnv("PLEDGE_ENV", defaultEnv),
		Port:       readEnv("PLEDGE_API_PORT", defaultPort),
		APIVersion: readEnv("PLEDGE_API_VERSION", defaultAPIVersion),
	}
}

func readEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

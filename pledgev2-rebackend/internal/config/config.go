package config

import "os"

const (
	defaultEnv  = "local"
	defaultPort = "8080"
)

type Config struct {
	Env  string
	Port string
}

func Load() Config {
	return Config{
		Env:  readEnv("PLEDGE_ENV", defaultEnv),
		Port: readEnv("PLEDGE_API_PORT", defaultPort),
	}
}

func readEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

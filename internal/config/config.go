package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port          string
	RedisURL      string
	RedisPassword string
	RedisDB       int
	JWTSecret     string
	GinMode       string
	CORSOrigins   []string
}

func Load() *Config {
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	corsOrigins := strings.Split(getEnv("CORS_ORIGINS", "http://localhost:3000"), ",")

	return &Config{
		Port:          getEnv("PORT", "8080"),
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       redisDB,
		JWTSecret:     getEnv("JWT_SECRET", "your-super-secret-key-change-this-in-production"),
		GinMode:       getEnv("GIN_MODE", "debug"),
		CORSOrigins:   corsOrigins,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

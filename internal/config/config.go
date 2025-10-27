package config

import (
	"os"
)

type Config struct {
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	RedisHost     string
	RedisPort     string
	SessionSecret string
	GinMode       string
	OpenAIAPIKey  string
}

func Load() *Config {
	return &Config{
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "3306"),
		DBUser:        getEnv("DB_USER", "taskuser"),
		DBPassword:    getEnv("DB_PASSWORD", "taskpassword"),
		DBName:        getEnv("DB_NAME", "task_management"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		SessionSecret: getEnv("SESSION_SECRET", "default-secret-key-change-me"),
		GinMode:       getEnv("GIN_MODE", "debug"),
		OpenAIAPIKey:  getEnv("OPENAI_API_KEY", ""),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

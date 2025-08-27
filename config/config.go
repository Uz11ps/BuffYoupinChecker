package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
	MarketAPIKey  string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	Port          string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Функция для получения значения с fallback
	getEnvWithDefault := func(key, defaultValue string) string {
		if value := os.Getenv(key); value != "" {
			return value
		}
		return defaultValue
	}

	return &Config{
		TelegramToken: getEnvWithDefault("TELEGRAM_BOT_TOKEN", ""),
		MarketAPIKey:  getEnvWithDefault("MARKET_API_KEY", ""),
		DBHost:        getEnvWithDefault("DB_HOST", "localhost"),
		DBPort:        getEnvWithDefault("DB_PORT", "5432"),
		DBUser:        getEnvWithDefault("DB_USER", "postgres"),
		DBPassword:    getEnvWithDefault("DB_PASSWORD", ""),
		DBName:        getEnvWithDefault("DB_NAME", "skin_analyzer"),
		Port:          getEnvWithDefault("PORT", "8080"),
	}
}
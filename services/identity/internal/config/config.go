package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
    AppEnv      string
    AppPort     int
    DatabaseURL string
}

func Load() (Config, error) {
    // Optional: load .env for local development.
    // Existing environment variables take precedence.
    _ = godotenv.Load()

    cfg := Config{
        AppEnv:      getEnv("APP_ENV", "development"),
        DatabaseURL: os.Getenv("DATABASE_URL"),
    }

    port, err := strconv.Atoi(getEnv("APP_PORT", "8081"))
    if err != nil {
        return Config{}, fmt.Errorf("invalid APP_PORT")
    }

    cfg.AppPort = port

    if cfg.DatabaseURL == "" {
        return Config{}, fmt.Errorf("DATABASE_URL is required")
    }

    return cfg, nil
}

func getEnv(key, fallback string) string {
    value := os.Getenv(key)
    if value == "" {
        return fallback
    }

    return value
}

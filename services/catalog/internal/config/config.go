package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort           int
	DatabaseURL       string
	IdentityBaseURL   string
	R2Endpoint        string
	R2Bucket          string
	R2AccessKeyID     string
	R2SecretAccessKey string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return Config{}, errors.New("could not load .env")
	}

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8082"
	}

	appPort, err := strconv.Atoi(port)
	if err != nil || appPort < 1 || appPort > 65535 {
		return Config{}, errors.New("APP_PORT must be between 1 and 65535")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	identityBaseURL := os.Getenv("IDENTITY_BASE_URL")
	if identityBaseURL == "" {
		identityBaseURL = "http://localhost:8081"
	}

	return Config{
		AppPort:           appPort,
		DatabaseURL:       databaseURL,
		IdentityBaseURL:   identityBaseURL,
		R2Endpoint:        os.Getenv("R2_ENDPOINT"),
		R2Bucket:          os.Getenv("R2_BUCKET"),
		R2AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		R2SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
	}, nil
}

package main

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type config struct {
	databaseURL  string
	appPort      int
	identityURL  string
	catalogURL   string
	catalogToken string
}

func loadConfig() (config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return config{}, errors.New("could not load .env")
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return config{}, errors.New("DATABASE_URL is required")
	}

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8083"
	}
	appPort, err := strconv.Atoi(port)
	if err != nil || appPort < 1 || appPort > 65535 {
		return config{}, errors.New("APP_PORT must be between 1 and 65535")
	}

	return config{
		databaseURL:  databaseURL,
		appPort:      appPort,
		identityURL:  os.Getenv("IDENTITY_BASE_URL"),
		catalogURL:   os.Getenv("CATALOG_BASE_URL"),
		catalogToken: strings.TrimSpace(os.Getenv("BOOKING_CATALOG_TOKEN")),
	}, nil
}

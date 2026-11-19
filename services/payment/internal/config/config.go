package config

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort           int
	DatabaseURL       string
	IdentityBaseURL   string
	BookingBaseURL    string
	BookingToken      string
	MidtransServerKey string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return Config{}, errors.New("could not load .env")
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8084"
	}
	appPort, err := strconv.Atoi(port)
	if err != nil || appPort < 1 || appPort > 65535 {
		return Config{}, errors.New("APP_PORT must be between 1 and 65535")
	}
	identityURL := os.Getenv("IDENTITY_BASE_URL")
	if identityURL == "" {
		identityURL = "http://localhost:8081"
	}
	bookingURL := os.Getenv("BOOKING_BASE_URL")
	if bookingURL == "" {
		bookingURL = "http://localhost:8083"
	}
	return Config{AppPort: appPort, DatabaseURL: databaseURL,
		IdentityBaseURL: identityURL, BookingBaseURL: bookingURL,
		BookingToken:      strings.TrimSpace(os.Getenv("PAYMENT_BOOKING_TOKEN")),
		MidtransServerKey: strings.TrimSpace(os.Getenv("MIDTRANS_SERVER_KEY"))}, nil
}

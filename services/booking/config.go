package main

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

type config struct {
	databaseURL  string
	appPort      int
	identityURL  string
	catalogURL   string
	catalogToken string
	paymentToken string
	demoPayments bool
	rabbitURL    string
	paymentURL   string
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

	rabbitURL := strings.TrimSpace(os.Getenv("RABBITMQ_URL"))
	paymentURL := strings.TrimSpace(os.Getenv("PAYMENT_BASE_URL"))
	if paymentURL == "" {
		paymentURL = "http://localhost:8084"
	}
	if rabbitURL != "" {
		parsed, parseError := url.Parse(rabbitURL)
		_, amqpError := amqp.ParseURI(rabbitURL)
		if parseError != nil || amqpError != nil || parsed.Hostname() == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return config{}, errors.New("RABBITMQ_URL must be a valid amqp or amqps connection URL")
		}
		if strings.TrimSpace(os.Getenv("PAYMENT_BOOKING_TOKEN")) == "" {
			return config{}, errors.New("PAYMENT_BOOKING_TOKEN is required for the payment consumer")
		}
		if os.Getenv("BOOKING_DEMO_PAYMENTS") == "true" {
			return config{}, errors.New("disable BOOKING_DEMO_PAYMENTS before enabling the payment consumer")
		}
	}

	return config{
		databaseURL:  databaseURL,
		rabbitURL:    rabbitURL,
		paymentURL:   paymentURL,
		appPort:      appPort,
		identityURL:  os.Getenv("IDENTITY_BASE_URL"),
		catalogURL:   os.Getenv("CATALOG_BASE_URL"),
		catalogToken: strings.TrimSpace(os.Getenv("BOOKING_CATALOG_TOKEN")),
		paymentToken: strings.TrimSpace(os.Getenv("PAYMENT_BOOKING_TOKEN")),
		demoPayments: os.Getenv("BOOKING_DEMO_PAYMENTS") == "true",
	}, nil
}

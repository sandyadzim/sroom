package config

import (
	"errors"
	"net"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Config struct {
	AppPort           int
	DatabaseURL       string
	IdentityBaseURL   string
	BookingBaseURL    string
	BookingToken      string
	MidtransServerKey string
	RabbitMQURL       string
	SMTPAddress       string
	MailFrom          string
	WebBaseURL        string
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
	rabbitURL := strings.TrimSpace(os.Getenv("RABBITMQ_URL"))
	if rabbitURL != "" {
		parsed, parseError := url.Parse(rabbitURL)
		_, amqpError := amqp.ParseURI(rabbitURL)
		if parseError != nil || amqpError != nil || parsed.Hostname() == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return Config{}, errors.New("RABBITMQ_URL must be a valid amqp or amqps connection URL")
		}
	}
	smtpAddress := strings.TrimSpace(os.Getenv("SMTP_ADDR"))
	if smtpAddress != "" {
		host, port, err := net.SplitHostPort(smtpAddress)
		number, portError := strconv.Atoi(port)
		if err != nil || host == "" || portError != nil || number < 1 || number > 65535 {
			return Config{}, errors.New("SMTP_ADDR must be host:port")
		}
	}
	mailFrom := os.Getenv("MAIL_FROM")
	if mailFrom == "" {
		mailFrom = "SROOM <noreply@sroom.test>"
	}
	if _, err := mail.ParseAddress(mailFrom); err != nil || strings.ContainsAny(mailFrom, "\r\n") {
		return Config{}, errors.New("MAIL_FROM must be a single email address")
	}
	webBaseURL := strings.TrimSpace(os.Getenv("WEB_BASE_URL"))
	if webBaseURL == "" {
		webBaseURL = "http://localhost:5173"
	}
	web, err := url.Parse(webBaseURL)
	if err != nil || (web.Scheme != "http" && web.Scheme != "https") || web.Hostname() == "" || web.User != nil || web.RawQuery != "" || web.ForceQuery || web.Fragment != "" || (web.Path != "" && web.Path != "/") {
		return Config{}, errors.New("WEB_BASE_URL must be an HTTP origin")
	}
	return Config{AppPort: appPort, DatabaseURL: databaseURL,
		SMTPAddress: smtpAddress, MailFrom: mailFrom, WebBaseURL: strings.TrimRight(webBaseURL, "/"),
		IdentityBaseURL: identityURL, BookingBaseURL: bookingURL,
		BookingToken:      strings.TrimSpace(os.Getenv("PAYMENT_BOOKING_TOKEN")),
		MidtransServerKey: strings.TrimSpace(os.Getenv("MIDTRANS_SERVER_KEY")), RabbitMQURL: rabbitURL}, nil
}

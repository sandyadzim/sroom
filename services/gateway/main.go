package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil)).With("service", "gateway")
	if err := run(logger); err != nil {
		logger.Error("gateway stopped", "operation", "gateway.run", "reason", err.Error())
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	port := envOrDefault("APP_PORT", "8080")
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return errors.New("invalid APP_PORT")
	}
	identityURL, err := upstreamURL("IDENTITY_URL", "http://localhost:8081")
	if err != nil {
		return err
	}
	catalogURL, err := upstreamURL("CATALOG_URL", "http://localhost:8082")
	if err != nil {
		return err
	}
	bookingURL, err := upstreamURL("BOOKING_URL", "http://localhost:8083")
	if err != nil {
		return err
	}

	gin.SetMode(gin.ReleaseMode)
	server := &http.Server{
		Addr:              net.JoinHostPort("127.0.0.1", strconv.Itoa(portNumber)),
		Handler:           newRouter(identityURL, catalogURL, bookingURL, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      40 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return errors.New("could not listen on APP_PORT")
	}

	stopping, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serveResult := make(chan error, 1)
	go func() {
		serveResult <- server.Serve(listener)
	}()

	logger.Info("server started", "operation", "api.listen", "port", portNumber)
	select {
	case err := <-serveResult:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return errors.New("HTTP server failed")
		}
	case <-stopping.Done():
		deadline, cancel := context.WithTimeout(context.Background(), server.WriteTimeout)
		defer cancel()
		if err := server.Shutdown(deadline); err != nil {
			return errors.New("HTTP server shutdown timed out")
		}
	}
	return nil
}

func upstreamURL(name, fallback string) (*url.URL, error) {
	base, err := url.Parse(envOrDefault(name, fallback))
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Hostname() == "" ||
		base.User != nil || (base.Path != "" && base.Path != "/") || base.ForceQuery ||
		base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("invalid %s: expected an HTTP origin", name)
	}
	if port := base.Port(); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return nil, fmt.Errorf("invalid %s: port out of range", name)
		}
	}
	return base, nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

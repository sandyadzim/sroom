package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"payment/internal/config"
	"payment/internal/database"
	"payment/internal/handler"
	"payment/internal/router"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil)).With("service", "payment")
	if err := run(log); err != nil {
		log.Error("payment stopped", "operation", "payment.run", "reason", err.Error())
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	pool, err := db.DB()
	if err != nil {
		return errors.New("could not get PostgreSQL pool")
	}
	defer func() {
		if err := pool.Close(); err != nil {
			log.Error("could not close PostgreSQL pool", "operation", "database.close")
		}
	}()

	healthHandler := handler.NewHealthHandler(pool.PingContext)
	engine, err := router.New(healthHandler)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.AppPort),
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	stopping, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return errors.New("could not listen on APP_PORT")
	}
	serverResult := make(chan error, 1)
	go func() {
		serverResult <- server.Serve(listener)
	}()
	log.Info("server started", "operation", "http.listen", "port", cfg.AppPort)

	select {
	case err := <-serverResult:
		if !errors.Is(err, http.ErrServerClosed) {
			return errors.New("HTTP server failed")
		}
	case <-stopping.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			return errors.New("HTTP server shutdown failed")
		}
	}
	return nil
}

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

	"booking/internal/catalog"
	"booking/internal/checkout"
	"booking/internal/identity"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil)).With("service", "booking")
	if err := run(log); err != nil {
		log.Error("booking stopped", "operation", "booking.run", "reason", err.Error())
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	db, err := gorm.Open(postgres.Open(cfg.databaseURL), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               logger.Discard,
	})
	if err != nil {
		return errors.New("could not open PostgreSQL with the supplied DATABASE_URL")
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

	router, err := newRouter(pool.PingContext)
	if err != nil {
		return err
	}
	identityClient, err := identity.NewClient(cfg.identityURL)
	if err != nil {
		return err
	}
	var catalogClient *catalog.Client
	if cfg.catalogToken != "" {
		catalogClient, err = catalog.NewClient(cfg.catalogURL, cfg.catalogToken)
		if err != nil {
			return err
		}
	}
	checkoutService, err := checkout.NewService(catalogClient, checkout.NewRepository(db))
	if err != nil {
		return err
	}
	registerCheckout(router, quoteHandler{identity: identityClient, checkout: checkoutService, now: time.Now})
	registerPaymentContext(router, checkoutService, cfg.paymentToken)
	registerDemoConfirmation(router, checkoutService, cfg.paymentToken, cfg.demoPayments, time.Now)
	server := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.appPort),
		Handler:           router,
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
	log.Info("server started", "operation", "http.listen", "port", cfg.appPort)
	stopDemoExpiry := startDemoExpiry(stopping, checkoutService, cfg, log)
	defer stopDemoExpiry()

	select {
	case err := <-serverResult:
		if !errors.Is(err, http.ErrServerClosed) {
			return errors.New("HTTP server failed")
		}
	case <-stopping.Done():
		deadline, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(deadline); err != nil {
			return errors.New("HTTP server shutdown failed")
		}
	}
	return nil
}

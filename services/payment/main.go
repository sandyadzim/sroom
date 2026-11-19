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

	"payment/internal/booking"
	"payment/internal/config"
	"payment/internal/database"
	"payment/internal/handler"
	"payment/internal/identity"
	"payment/internal/mailer"
	"payment/internal/messaging"
	"payment/internal/midtrans"
	"payment/internal/repository"
	"payment/internal/router"
	"payment/internal/service"
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
	identityClient, err := identity.NewClient(cfg.IdentityBaseURL)
	if err != nil {
		return err
	}
	bookingClient, err := booking.NewClient(cfg.BookingBaseURL, cfg.BookingToken)
	if err != nil {
		return err
	}
	paymentRepo := repository.NewPaymentRepository(db)
	midtransClient := midtrans.NewClient(cfg.MidtransServerKey)
	paymentService := service.NewPaymentService(paymentRepo, bookingClient,
		midtransClient, cfg.BookingToken != "" && cfg.MidtransServerKey != "")
	invoiceService, err := service.NewInvoiceService(paymentRepo, bookingClient)
	if err != nil {
		return err
	}
	router.RegisterPayment(engine, identityClient, handler.NewPaymentHandler(paymentService))
	router.RegisterPaymentReads(engine, identityClient, handler.NewReadPaymentHandler(paymentService))
	router.RegisterInvoices(engine, identityClient, handler.NewInvoiceHandler(invoiceService))
	if cfg.BookingToken != "" {
		router.RegisterConfirmation(engine, handler.NewConfirmationHandler(paymentService, cfg.BookingToken))
	}
	refreshService := service.NewRefreshPaymentService(paymentService, midtransClient, cfg.MidtransServerKey != "")
	router.RegisterPaymentRefresh(engine, identityClient, handler.NewRefreshPaymentHandler(refreshService))
	webhookService := service.NewWebhookService(paymentRepo, midtransClient, cfg.MidtransServerKey)
	router.RegisterWebhook(engine, handler.NewWebhookHandler(webhookService))
	var emailWorker *service.EmailService
	if cfg.SMTPAddress != "" {
		sender, err := mailer.NewSMTP(cfg.SMTPAddress, cfg.MailFrom)
		if err != nil {
			return errors.New("could not configure Mailpit SMTP")
		}
		emailWorker, err = service.NewEmailService(paymentRepo, sender, cfg.WebBaseURL, log)
		if err != nil {
			return err
		}
	}
	server := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.AppPort),
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
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
	if emailWorker != nil {
		emailContext, cancelEmail := context.WithCancel(stopping)
		emailStopped := make(chan struct{})
		go func() { defer close(emailStopped); emailWorker.Run(emailContext) }()
		defer func() { cancelEmail(); <-emailStopped }()
	}
	if cfg.RabbitMQURL != "" {
		relayContext, cancelRelay := context.WithCancel(stopping)
		relayStopped := make(chan struct{})
		relay := service.NewOutboxRelay(paymentRepo, messaging.NewRabbitPublisher(cfg.RabbitMQURL), log)
		go func() {
			defer close(relayStopped)
			relay.Run(relayContext)
		}()
		defer func() {
			cancelRelay()
			<-relayStopped
		}()
		if cfg.BookingToken != "" {
			consumerContext, cancelConsumer := context.WithCancel(stopping)
			consumerStopped := make(chan struct{})
			consumer := messaging.NewInvoiceConsumer(cfg.RabbitMQURL, invoiceService, log)
			go func() { defer close(consumerStopped); consumer.Run(consumerContext) }()
			defer func() { cancelConsumer(); <-consumerStopped }()
		}
	}

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

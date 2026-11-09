package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v3"

	"identity/internal/config"
	"identity/internal/database"
	"identity/internal/identity/handler"
	"identity/internal/identity/middleware"
	"identity/internal/identity/repository"
	"identity/internal/identity/response"
	"identity/internal/identity/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil)).With("service", "identity")

	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "operation", "api.start", "reason", "invalid_config")
		os.Exit(1)
	}

	// 2. Connect to PostgreSQL
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "operation", "database.connect", "reason", "connection_failed")
		os.Exit(1)
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("database pool unavailable", "operation", "database.pool", "reason", "pool_unavailable")
		os.Exit(1)
	}
	defer sqlDB.Close()

	logger.Info("database connected", "operation", "database.connect", "result", "success")

	// 3. Init dependency
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	registerService := service.NewRegisterService(userRepo)
	loginService := service.NewLoginService(userRepo, sessionRepo)
	authService := service.NewAuthService(sessionRepo)
	meService := service.NewMeService(userRepo)
	logoutService := service.NewLogoutService(sessionRepo)

	registerHandler := handler.NewRegisterHandler(registerService)
	loginHandler := handler.NewLoginHandler(loginService)
	meHandler := handler.NewMeHandler(meService)
	logoutHandler := handler.NewLogoutHandler(logoutService)

	authMiddleware := middleware.NewAuthMiddleware(authService)

	// 4. Initialize Fiber
	app := fiber.New(fiber.Config{ErrorHandler: response.ErrorHandler})

	// 5. Route Group
	api := app.Group("/api")
	v1 := api.Group("/v1")
	identityGroup := v1.Group("/identity")

	// Public routes
	identityGroup.Post("/register", middleware.RegisterRateLimiter(), registerHandler.Register)
	identityGroup.Post("/login", middleware.LoginRateLimiter(), loginHandler.Login)

	// Protected routes
	identityGroup.Get("/me", authMiddleware.RequireAuth, meHandler.Me)
	identityGroup.Post("/logout", authMiddleware.RequireAuth, logoutHandler.Logout)

	// 6. Start HTTP server
	addr := fmt.Sprintf(":%d", cfg.AppPort)

	logger.Info("server starting", "operation", "api.listen", "port", cfg.AppPort)

	if err := app.Listen(addr); err != nil {
		logger.Error("server stopped", "operation", "api.listen", "reason", "listen_failed")
		os.Exit(1)
	}
}

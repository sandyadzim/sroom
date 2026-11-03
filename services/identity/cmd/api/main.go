package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v3"

	"identity/internal/config"
	"identity/internal/database"
	"identity/internal/identity/handler"
	"identity/internal/identity/repository"
	"identity/internal/identity/service"
)

func main() {
    // 1. Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("failed to load config: ", err)
    }

    // 2. Connect to PostgreSQL
    db, err := database.Connect(cfg.DatabaseURL)
    if err != nil {
        log.Fatal("failed to connect database: ", err)
    }

    sqlDB, err := db.DB()
    if err != nil {
        log.Fatal("failed to access database pool: ", err)
    }
    defer sqlDB.Close()

    log.Println("database connected successfully")

    // 3. Init dependency
    userRepo := repository.NewUserRepository(db)

    registerService := service.NewRegisterService(userRepo)

    registerHandler := handler.NewRegisterHandler(registerService)

    // 4. Initialize Fiber
    app := fiber.New()

    // 5. Route Group
    api := app.Group("/api")
    v1 := api.Group("/v1")
    identityGroup := v1.Group("/identity")

    identityGroup.Post("/register", registerHandler.Register)

    // 6. Start HTTP server
    addr := fmt.Sprintf(":%d", cfg.AppPort)

    log.Printf("server listening on %s", addr)

    if err := app.Listen(addr); err != nil {
        log.Fatal(err)
    }
}

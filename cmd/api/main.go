package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/config"
	_ "github.com/abdulshakoor02/goCrmBackend/docs"
	"github.com/abdulshakoor02/goCrmBackend/internal/adapters/handler"
	"github.com/abdulshakoor02/goCrmBackend/internal/adapters/storage"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/services"
	"github.com/abdulshakoor02/goCrmBackend/pkg/database"
	"github.com/abdulshakoor02/goCrmBackend/pkg/logger"
	"github.com/abdulshakoor02/goCrmBackend/pkg/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
)

// @title CRM Backend API
// @version 1.0
// @description Multi-tenant CRM Backend API with Hexagonal Architecture.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:3000
// @BasePath /api/v1
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
func main() {
	// 1. Load Config
	cfg := config.LoadConfig()

	// 2. Init Logger
	logger.InitLogger(cfg)

	// 3. Connect Database
	mongoClient, err := database.ConnectMongoDB(cfg)
	if err != nil {
		slog.Error("Failed to connect to MongoDB", "error", err)
		panic(err)
	}
	defer mongoClient.Disconnect(context.Background())

	db := mongoClient.Database(cfg.DBName)

	// 4. Init Repositories
	tenantRepo := storage.NewMongoTenantRepository(db)
	userRepo := storage.NewMongoUserRepository(db)

	// 5. Init Services
	tenantService := services.NewTenantService(tenantRepo, userRepo)
	userService := services.NewUserService(userRepo)
	authService := services.NewAuthService(userRepo, cfg)

	// 6. Init Handlers
	tenantHandler := handler.NewTenantHandler(tenantService)
	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(authService)

	// 7. Setup Fiber
	app := fiber.New(fiber.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			slog.Error("Fiber Error", "status", code, "error", err.Error())
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	// Middleware
	app.Use(recover.New())
	app.Use(cors.New()) // Default CORS
	app.Use(middleware.RequestLogger())

	// Swagger
	app.Get("/swagger/*", swagger.HandlerDefault)

	// 8. Routes
	api := app.Group("/api/v1")

	// Public Routes
	api.Post("/tenants", tenantHandler.RegisterTenant)
	api.Post("/auth/login", authHandler.Login)
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Protected Routes
	protected := api.Group("/", middleware.Protected(cfg.JWTSecret))

	// Tenant Management (Protected)
	protected.Get("/tenants/:id", tenantHandler.GetTenant)
	protected.Put("/tenants/:id", tenantHandler.UpdateTenant)
	protected.Post("/tenants/list", tenantHandler.ListTenants)

	// User Management (Protected)
	protected.Post("/users", userHandler.CreateUser)
	protected.Get("/users/:id", userHandler.GetUser)
	protected.Put("/users/:id", userHandler.UpdateUser)
	protected.Post("/users/list", userHandler.ListUsers)

	// 9. Start Server
	slog.Info("Starting server", "port", cfg.ServerPort)
	if err := app.Listen(":" + cfg.ServerPort); err != nil {
		slog.Error("Server failed to start", "error", err)
	}
}

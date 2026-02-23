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

	casbinlib "github.com/casbin/casbin/v2"
	mongodbadapter "github.com/casbin/mongodb-adapter/v3"
	mongooptions "go.mongodb.org/mongo-driver/mongo/options"
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

	if err := database.RunMigrations(context.Background(), db); err != nil {
		slog.Error("Failed to run migrations", "error", err)
	}

	// 4. Init Repositories
	tenantRepo := storage.NewMongoTenantRepository(db)
	userRepo := storage.NewMongoUserRepository(db)
	leadRepo := storage.NewMongoLeadRepository(db)
	leadCategoryRepo := storage.NewMongoLeadCategoryRepository(db)
	leadSourceRepo := storage.NewMongoLeadSourceRepository(db)
	leadCommentRepo := storage.NewMongoLeadCommentRepository(db)
	leadAppointmentRepo := storage.NewMongoLeadAppointmentRepository(db)

	// 5. Init Services
	tenantService := services.NewTenantService(tenantRepo, userRepo)
	userService := services.NewUserService(userRepo)
	authService := services.NewAuthService(userRepo, cfg)
	leadService := services.NewLeadService(leadRepo)
	leadCategoryService := services.NewLeadCategoryService(leadCategoryRepo)
	leadSourceService := services.NewLeadSourceService(leadSourceRepo)
	leadCommentService := services.NewLeadCommentService(leadCommentRepo, leadRepo)
	leadAppointmentService := services.NewLeadAppointmentService(leadAppointmentRepo, leadRepo)

	// 6. Init Casbin with MongoDB Adapter
	mongoClientOption := mongooptions.Client().ApplyURI(cfg.MongoURI)
	casbinAdapter, err := mongodbadapter.NewAdapterWithClientOption(mongoClientOption, cfg.DBName)
	if err != nil {
		slog.Error("Failed to initialize Casbin MongoDB adapter", "error", err)
		panic(err)
	}

	enforcer, err := casbinlib.NewEnforcer("rbac_model.conf", casbinAdapter)
	if err != nil {
		slog.Error("Failed to initialize Casbin enforcer", "error", err)
		panic(err)
	}

	// Load policies from MongoDB
	if err := enforcer.LoadPolicy(); err != nil {
		slog.Error("Failed to load Casbin policies", "error", err)
		panic(err)
	}

	hasPolicies, _ := enforcer.HasPolicy("admin", "/api/v1/users", "POST")
	if !hasPolicies {
		slog.Info("Seeding default RBAC policies...")
		// Admin permissions
		enforcer.AddPolicy("admin", "/api/v1/tenants/:id", "GET")
		enforcer.AddPolicy("admin", "/api/v1/tenants/:id", "PUT")
		enforcer.AddPolicy("admin", "/api/v1/tenants/list", "POST")
		enforcer.AddPolicy("admin", "/api/v1/users", "POST")
		enforcer.AddPolicy("admin", "/api/v1/users/:id", "GET")
		enforcer.AddPolicy("admin", "/api/v1/users/:id", "PUT")
		enforcer.AddPolicy("admin", "/api/v1/users/list", "POST")

		// Lead permissions (Admin & Users)
		enforcer.AddPolicy("admin", "/api/v1/leads", "POST")
		enforcer.AddPolicy("admin", "/api/v1/leads/:id", "GET")
		enforcer.AddPolicy("admin", "/api/v1/leads/:id", "PUT")
		enforcer.AddPolicy("admin", "/api/v1/leads/list", "POST")

		// Lead Category permissions (Admin only write, Admin/User read)
		enforcer.AddPolicy("admin", "/api/v1/lead-categories", "POST")
		enforcer.AddPolicy("admin", "/api/v1/lead-categories/:id", "GET")
		enforcer.AddPolicy("admin", "/api/v1/lead-categories/:id", "PUT")
		enforcer.AddPolicy("admin", "/api/v1/lead-categories/:id", "DELETE")
		enforcer.AddPolicy("admin", "/api/v1/lead-categories/list", "POST")

		// Lead Source permissions (Admin only write, Admin/User read)
		enforcer.AddPolicy("admin", "/api/v1/lead-sources", "POST")
		enforcer.AddPolicy("admin", "/api/v1/lead-sources/:id", "GET")
		enforcer.AddPolicy("admin", "/api/v1/lead-sources/:id", "PUT")
		enforcer.AddPolicy("admin", "/api/v1/lead-sources/:id", "DELETE")
		enforcer.AddPolicy("admin", "/api/v1/lead-sources/list", "POST")

		// Lead Comment permissions (Admin and User full layout)
		enforcer.AddPolicy("admin", "/api/v1/leads/:lead_id/comments", "POST")
		enforcer.AddPolicy("admin", "/api/v1/leads/:lead_id/comments/:id", "GET")
		enforcer.AddPolicy("admin", "/api/v1/leads/:lead_id/comments/:id", "PUT")
		enforcer.AddPolicy("admin", "/api/v1/leads/:lead_id/comments/:id", "DELETE")
		enforcer.AddPolicy("admin", "/api/v1/leads/:lead_id/comments/list", "POST")

		// Lead Appointment permissions
		enforcer.AddPolicy("admin", "/api/v1/leads/:lead_id/appointments", "POST")
		enforcer.AddPolicy("admin", "/api/v1/leads/:lead_id/appointments/:id", "GET")
		enforcer.AddPolicy("admin", "/api/v1/leads/:lead_id/appointments/:id", "PUT")
		enforcer.AddPolicy("admin", "/api/v1/leads/:lead_id/appointments/:id", "DELETE")
		enforcer.AddPolicy("admin", "/api/v1/leads/:lead_id/appointments/list", "POST")

		// Regular user permissions
		enforcer.AddPolicy("user", "/api/v1/tenants/:id", "GET")
		enforcer.AddPolicy("user", "/api/v1/users/:id", "GET")
		enforcer.AddPolicy("user", "/api/v1/leads", "POST")
		enforcer.AddPolicy("user", "/api/v1/leads/:id", "GET")
		enforcer.AddPolicy("user", "/api/v1/leads/:id", "PUT")
		enforcer.AddPolicy("user", "/api/v1/leads/list", "POST")

		enforcer.AddPolicy("user", "/api/v1/lead-categories/:id", "GET")
		enforcer.AddPolicy("user", "/api/v1/lead-categories/list", "POST")

		enforcer.AddPolicy("user", "/api/v1/lead-sources/:id", "GET")
		enforcer.AddPolicy("user", "/api/v1/lead-sources/list", "POST")

		enforcer.AddPolicy("user", "/api/v1/leads/:lead_id/comments", "POST")
		enforcer.AddPolicy("user", "/api/v1/leads/:lead_id/comments/:id", "GET")
		enforcer.AddPolicy("user", "/api/v1/leads/:lead_id/comments/:id", "PUT")
		enforcer.AddPolicy("user", "/api/v1/leads/:lead_id/comments/:id", "DELETE")
		enforcer.AddPolicy("user", "/api/v1/leads/:lead_id/comments/list", "POST")

		enforcer.AddPolicy("user", "/api/v1/leads/:lead_id/appointments", "POST")
		enforcer.AddPolicy("user", "/api/v1/leads/:lead_id/appointments/:id", "GET")
		enforcer.AddPolicy("user", "/api/v1/leads/:lead_id/appointments/:id", "PUT")
		enforcer.AddPolicy("user", "/api/v1/leads/:lead_id/appointments/:id", "DELETE")
		enforcer.AddPolicy("user", "/api/v1/leads/:lead_id/appointments/list", "POST")

		// Role inheritance: superadmin inherits admin
		enforcer.AddGroupingPolicy("superadmin", "admin")

		// Save to MongoDB
		if err := enforcer.SavePolicy(); err != nil {
			slog.Error("Failed to save default policies", "error", err)
			panic(err)
		}
		slog.Info("Default RBAC policies seeded to MongoDB")
	}

	// Seed permission endpoints for Admin if missing
	hasPermissionPolicies, _ := enforcer.HasPolicy("admin", "/api/v1/permissions", "GET")
	if !hasPermissionPolicies {
		slog.Info("Seeding permission-management RBAC policies...")
		enforcer.AddPolicy("admin", "/api/v1/permissions", "POST")
		enforcer.AddPolicy("admin", "/api/v1/permissions", "DELETE")
		enforcer.AddPolicy("admin", "/api/v1/permissions", "GET")
		enforcer.AddPolicy("admin", "/api/v1/permissions/roles/inherit", "POST")
		enforcer.AddPolicy("admin", "/api/v1/permissions/roles/inherit", "GET")
		if err := enforcer.SavePolicy(); err != nil {
			slog.Error("Failed to save permission endpoint policies", "error", err)
			panic(err)
		}
		slog.Info("Permission management policies seeded to MongoDB")
	}

	slog.Info("Casbin RBAC enforcer initialized with MongoDB adapter")

	// 7. Init Handlers
	tenantHandler := handler.NewTenantHandler(tenantService)
	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(authService)
	permissionService := services.NewPermissionService(enforcer)
	permissionHandler := handler.NewPermissionHandler(permissionService)
	leadHandler := handler.NewLeadHandler(leadService)
	leadCategoryHandler := handler.NewLeadCategoryHandler(leadCategoryService)
	leadSourceHandler := handler.NewLeadSourceHandler(leadSourceService)
	leadCommentHandler := handler.NewLeadCommentHandler(leadCommentService)
	leadAppointmentHandler := handler.NewLeadAppointmentHandler(leadAppointmentService)

	// 8. Init Casbin Middleware
	authz := middleware.NewCasbinMiddleware(enforcer)

	// 9. Setup Fiber
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

	// Global Middleware
	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(middleware.RequestLogger())

	// Swagger
	app.Get("/swagger/*", swagger.HandlerDefault)

	// 10. Routes
	api := app.Group("/api/v1")

	// Public Routes
	api.Post("/tenants", tenantHandler.RegisterTenant)
	api.Post("/auth/login", authHandler.Login)
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Protected Routes (JWT Auth + RBAC)
	protected := api.Group("/", middleware.Protected(cfg.JWTSecret))

	// Permissions endpoint (protected but no RBAC check - any authenticated user can see their permissions)
	protected.Get("/permissions/my-permissions", permissionHandler.GetPermissions)

	// RBAC Management endpoints (Protected + RBAC) -> only admins can manage this
	protected.Post("/permissions", authz.RoutePermission(), permissionHandler.AddPermission)
	protected.Delete("/permissions", authz.RoutePermission(), permissionHandler.RemovePermission)
	protected.Get("/permissions", authz.RoutePermission(), permissionHandler.GetAllPermissions)
	protected.Post("/permissions/roles/inherit", authz.RoutePermission(), permissionHandler.AssignRoleInheritance)
	protected.Get("/permissions/roles/inherit", authz.RoutePermission(), permissionHandler.GetRoleInheritances)

	// Tenant Management (Protected + RBAC)
	protected.Get("/tenants/:id", authz.RoutePermission(), tenantHandler.GetTenant)
	protected.Put("/tenants/:id", authz.RoutePermission(), tenantHandler.UpdateTenant)
	protected.Post("/tenants/list", authz.RoutePermission(), tenantHandler.ListTenants)

	// User Management (Protected + RBAC)
	protected.Post("/users", authz.RoutePermission(), userHandler.CreateUser)
	protected.Get("/users/:id", authz.RoutePermission(), userHandler.GetUser)
	protected.Put("/users/:id", authz.RoutePermission(), userHandler.UpdateUser)
	protected.Post("/users/list", authz.RoutePermission(), userHandler.ListUsers)

	// Lead Management (Protected + RBAC)
	protected.Post("/leads", authz.RoutePermission(), leadHandler.CreateLead)
	protected.Get("/leads/:id", authz.RoutePermission(), leadHandler.GetLead)
	protected.Put("/leads/:id", authz.RoutePermission(), leadHandler.UpdateLead)
	protected.Post("/leads/list", authz.RoutePermission(), leadHandler.ListLeads)

	// Lead Categories (Protected + RBAC)
	protected.Post("/lead-categories", authz.RoutePermission(), leadCategoryHandler.CreateLeadCategory)
	protected.Get("/lead-categories/:id", authz.RoutePermission(), leadCategoryHandler.GetLeadCategory)
	protected.Put("/lead-categories/:id", authz.RoutePermission(), leadCategoryHandler.UpdateLeadCategory)
	protected.Delete("/lead-categories/:id", authz.RoutePermission(), leadCategoryHandler.DeleteLeadCategory)
	protected.Post("/lead-categories/list", authz.RoutePermission(), leadCategoryHandler.ListLeadCategories)

	// Lead Sources (Protected + RBAC)
	protected.Post("/lead-sources", authz.RoutePermission(), leadSourceHandler.CreateLeadSource)
	protected.Get("/lead-sources/:id", authz.RoutePermission(), leadSourceHandler.GetLeadSource)
	protected.Put("/lead-sources/:id", authz.RoutePermission(), leadSourceHandler.UpdateLeadSource)
	protected.Delete("/lead-sources/:id", authz.RoutePermission(), leadSourceHandler.DeleteLeadSource)
	protected.Post("/lead-sources/list", authz.RoutePermission(), leadSourceHandler.ListLeadSources)

	// Lead Comments (Protected + RBAC)
	protected.Post("/leads/:lead_id/comments", authz.RoutePermission(), leadCommentHandler.CreateLeadComment)
	protected.Get("/leads/:lead_id/comments/:id", authz.RoutePermission(), leadCommentHandler.GetLeadComment)
	protected.Put("/leads/:lead_id/comments/:id", authz.RoutePermission(), leadCommentHandler.UpdateLeadComment)
	protected.Delete("/leads/:lead_id/comments/:id", authz.RoutePermission(), leadCommentHandler.DeleteLeadComment)
	protected.Post("/leads/:lead_id/comments/list", authz.RoutePermission(), leadCommentHandler.ListLeadComments)

	// Lead Appointments
	protected.Post("/leads/:lead_id/appointments", authz.RoutePermission(), leadAppointmentHandler.CreateLeadAppointment)
	protected.Get("/leads/:lead_id/appointments/:id", authz.RoutePermission(), leadAppointmentHandler.GetLeadAppointment)
	protected.Put("/leads/:lead_id/appointments/:id", authz.RoutePermission(), leadAppointmentHandler.UpdateLeadAppointment)
	protected.Delete("/leads/:lead_id/appointments/:id", authz.RoutePermission(), leadAppointmentHandler.DeleteLeadAppointment)
	protected.Post("/leads/:lead_id/appointments/list", authz.RoutePermission(), leadAppointmentHandler.ListLeadAppointments)

	// 11. Start Server
	slog.Info("Starting server", "port", cfg.ServerPort)
	if err := app.Listen(":" + cfg.ServerPort); err != nil {
		slog.Error("Server failed to start", "error", err)
	}
}

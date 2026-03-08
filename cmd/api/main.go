package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/config"
	"github.com/abdulshakoor02/goCrmBackend/docs"
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
	cfg := config.LoadConfig()

	logger.InitLogger(cfg)

	docs.SwaggerInfo.Host = "localhost:" + cfg.ServerPort

	mongoClient, err := database.ConnectMongoDB(cfg)
	if err != nil {
		slog.Error("Failed to connect to MongoDB", "error", err)
		panic(err)
	}
	defer mongoClient.Disconnect(context.Background())

	db := mongoClient.Database(cfg.DBName)

	if err := database.RunMigrations(context.Background(), db, cfg); err != nil {
		slog.Error("Failed to run migrations", "error", err)
	}

	tenantRepo := storage.NewMongoTenantRepository(db)
	userRepo := storage.NewMongoUserRepository(db)
	leadRepo := storage.NewMongoLeadRepository(db)
	leadCategoryRepo := storage.NewMongoLeadCategoryRepository(db)
	leadSourceRepo := storage.NewMongoLeadSourceRepository(db)
	leadCommentRepo := storage.NewMongoLeadCommentRepository(db)
	leadAppointmentRepo := storage.NewMongoLeadAppointmentRepository(db)
	qualificationRepo := storage.NewMongoQualificationRepository(db)
	countryRepo := storage.NewMongoCountryRepository(db)
	productRepo := storage.NewMongoProductRepository(db)
	invoiceRepo := storage.NewMongoInvoiceRepository(db)
	receiptRepo := storage.NewMongoReceiptRepository(db)

	tenantService := services.NewTenantService(tenantRepo, userRepo)
	userService := services.NewUserService(userRepo)
	authService := services.NewAuthService(userRepo, tenantRepo, countryRepo, cfg)
	leadService := services.NewLeadService(leadRepo)
	leadCategoryService := services.NewLeadCategoryService(leadCategoryRepo)
	leadSourceService := services.NewLeadSourceService(leadSourceRepo)
	leadCommentService := services.NewLeadCommentService(leadCommentRepo, leadRepo)
	leadAppointmentService := services.NewLeadAppointmentService(leadAppointmentRepo, leadRepo)
	qualificationService := services.NewQualificationService(qualificationRepo)
	countryService := services.NewCountryService(countryRepo)
	productService := services.NewProductService(productRepo, tenantRepo)
	invoiceService := services.NewInvoiceService(invoiceRepo, productRepo, tenantRepo, leadRepo, receiptRepo)
	receiptService := services.NewReceiptService(receiptRepo, invoiceRepo, tenantRepo, leadRepo)

	permissionRuleRepo := storage.NewMongoPermissionRuleRepository(db)
	rolePermissionRepo := storage.NewMongoRolePermissionRepository(db)
	permissionService := services.NewPermissionService(permissionRuleRepo, rolePermissionRepo)

	permissionHandler := handler.NewPermissionHandler(permissionService)

	authz := middleware.NewAuthMiddleware(rolePermissionRepo)

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

	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(middleware.RequestLogger())

	app.Get("/swagger/*", swagger.HandlerDefault)

	api := app.Group("/api/v1")

	api.Post("/auth/login", handler.NewAuthHandler(authService).Login)
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	qualificationHandler := handler.NewQualificationHandler(qualificationService)
	api.Post("/qualifications", qualificationHandler.CreateQualification)
	api.Get("/qualifications/:id", qualificationHandler.GetQualification)
	api.Put("/qualifications/:id", qualificationHandler.UpdateQualification)
	api.Delete("/qualifications/:id", qualificationHandler.DeleteQualification)
	api.Post("/qualifications/list", qualificationHandler.ListQualifications)

	countryHandler := handler.NewCountryHandler(countryService)
	api.Post("/countries", countryHandler.CreateCountry)
	api.Get("/countries/:id", countryHandler.GetCountry)
	api.Put("/countries/:id", countryHandler.UpdateCountry)
	api.Delete("/countries/:id", countryHandler.DeleteCountry)
	api.Post("/countries/list", countryHandler.ListCountries)

	protected := api.Group("/", middleware.Protected(cfg.JWTSecret), middleware.NewFilterContextMiddleware(permissionService, rolePermissionRepo))

	protected.Get("/permissions/my-permissions", permissionHandler.GetPermissions)

	protected.Post("/permissions", authz, permissionHandler.AddPermission)
	protected.Delete("/permissions", authz, permissionHandler.RemovePermission)
	protected.Get("/permissions", authz, permissionHandler.GetAllPermissions)
	protected.Post("/permissions/roles/inherit", authz, permissionHandler.AssignRoleInheritance)
	protected.Get("/permissions/roles/inherit", authz, permissionHandler.GetRoleInheritances)

	protected.Get("/permissions/available-rules", authz, permissionHandler.GetAvailableRules)
	protected.Post("/permissions/rules", authz, permissionHandler.CreatePermissionRule)
	protected.Put("/permissions/rules/:id", authz, permissionHandler.UpdatePermissionRule)
	protected.Delete("/permissions/rules/:id", authz, permissionHandler.DeletePermissionRule)

	protected.Get("/permissions/roles", authz, permissionHandler.GetAllRoles)
	protected.Get("/permissions/roles/:role", authz, permissionHandler.GetRolePermissions)
	protected.Post("/permissions/roles/:role/bulk", authz, permissionHandler.BulkUpdateRolePermissions)

	tenantHandler := handler.NewTenantHandler(tenantService)
	protected.Post("/tenants", authz, tenantHandler.RegisterTenant)
	protected.Get("/tenants/:id", authz, tenantHandler.GetTenant)
	protected.Put("/tenants/:id", authz, tenantHandler.UpdateTenant)
	protected.Post("/tenants/list", authz, tenantHandler.ListTenants)
	protected.Get("/user/tenant", authz, tenantHandler.GetUserTenant)

	userHandler := handler.NewUserHandler(userService)
	protected.Post("/users", authz, userHandler.CreateUser)
	protected.Get("/users/:id", authz, userHandler.GetUser)
	protected.Put("/users/:id", authz, userHandler.UpdateUser)
	protected.Post("/users/list", authz, userHandler.ListUsers)

	leadHandler := handler.NewLeadHandler(leadService)
	protected.Post("/leads", authz, leadHandler.CreateLead)
	protected.Get("/leads/:id", authz, leadHandler.GetLead)
	protected.Put("/leads/:id", authz, leadHandler.UpdateLead)
	protected.Put("/leads/:id/status", authz, leadHandler.UpdateLeadStatus)
	protected.Post("/leads/list", authz, leadHandler.ListLeads)

	leadCategoryHandler := handler.NewLeadCategoryHandler(leadCategoryService)
	protected.Post("/lead-categories", authz, leadCategoryHandler.CreateLeadCategory)
	protected.Get("/lead-categories/:id", authz, leadCategoryHandler.GetLeadCategory)
	protected.Put("/lead-categories/:id", authz, leadCategoryHandler.UpdateLeadCategory)
	protected.Delete("/lead-categories/:id", authz, leadCategoryHandler.DeleteLeadCategory)
	protected.Post("/lead-categories/list", authz, leadCategoryHandler.ListLeadCategories)

	leadSourceHandler := handler.NewLeadSourceHandler(leadSourceService)
	protected.Post("/lead-sources", authz, leadSourceHandler.CreateLeadSource)
	protected.Get("/lead-sources/:id", authz, leadSourceHandler.GetLeadSource)
	protected.Put("/lead-sources/:id", authz, leadSourceHandler.UpdateLeadSource)
	protected.Delete("/lead-sources/:id", authz, leadSourceHandler.DeleteLeadSource)
	protected.Post("/lead-sources/list", authz, leadSourceHandler.ListLeadSources)

	leadCommentHandler := handler.NewLeadCommentHandler(leadCommentService)
	protected.Post("/leads/:lead_id/comments", authz, leadCommentHandler.CreateLeadComment)
	protected.Get("/leads/:lead_id/comments/:id", authz, leadCommentHandler.GetLeadComment)
	protected.Put("/leads/:lead_id/comments/:id", authz, leadCommentHandler.UpdateLeadComment)
	protected.Delete("/leads/:lead_id/comments/:id", authz, leadCommentHandler.DeleteLeadComment)
	protected.Post("/leads/:lead_id/comments/list", authz, leadCommentHandler.ListLeadComments)

	leadAppointmentHandler := handler.NewLeadAppointmentHandler(leadAppointmentService)
	protected.Post("/leads/:lead_id/appointments", authz, leadAppointmentHandler.CreateLeadAppointment)
	protected.Get("/leads/:lead_id/appointments/:id", authz, leadAppointmentHandler.GetLeadAppointment)
	protected.Put("/leads/:lead_id/appointments/:id", authz, leadAppointmentHandler.UpdateLeadAppointment)
	protected.Delete("/leads/:lead_id/appointments/:id", authz, leadAppointmentHandler.DeleteLeadAppointment)
	protected.Post("/leads/:lead_id/appointments/list", authz, leadAppointmentHandler.ListLeadAppointments)

	productHandler := handler.NewProductHandler(productService)
	protected.Post("/products", authz, productHandler.CreateProduct)
	protected.Get("/products/:id", authz, productHandler.GetProduct)
	protected.Put("/products/:id", authz, productHandler.UpdateProduct)
	protected.Delete("/products/:id", authz, productHandler.DeleteProduct)
	protected.Post("/products/list", authz, productHandler.ListProducts)

	invoiceHandler := handler.NewInvoiceHandler(invoiceService)
	protected.Post("/leads/:lead_id/invoices", authz, invoiceHandler.CreateInvoice)
	protected.Get("/invoices/:id", authz, invoiceHandler.GetInvoice)
	protected.Put("/invoices/:id", authz, invoiceHandler.UpdateInvoice)
	protected.Put("/invoices/:id/due-date", authz, invoiceHandler.UpdateDueDate)
	protected.Post("/invoices/list", authz, invoiceHandler.ListInvoices)
	protected.Get("/leads/:lead_id/invoices", authz, invoiceHandler.GetInvoicesByLeadID)

	receiptHandler := handler.NewReceiptHandler(receiptService)
	protected.Post("/invoices/:invoice_id/receipts", authz, receiptHandler.CreateReceipt)
	protected.Get("/receipts/:id", authz, receiptHandler.GetReceipt)
	protected.Put("/receipts/:id", authz, receiptHandler.UpdateReceipt)
	protected.Delete("/receipts/:id", authz, receiptHandler.DeleteReceipt)
	protected.Post("/invoices/:invoice_id/receipts/list", authz, receiptHandler.ListReceiptsByInvoiceID)

	slog.Info("Starting server", "port", cfg.ServerPort)
	if err := app.Listen(":" + cfg.ServerPort); err != nil {
		slog.Error("Server failed to start", "error", err)
	}
}

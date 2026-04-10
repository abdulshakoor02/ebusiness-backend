package database

import (
	"context"
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/config"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// RunMigrations runs application boot time database setup such as ensuring indexes
func RunMigrations(ctx context.Context, db *mongo.Database, cfg *config.Config) error {
	slog.Info("Running database migrations...")

	if err := ensureUserIndexes(ctx, db.Collection("users")); err != nil {
		slog.Error("Failed to apply user indexes", "error", err)
		return err
	}

	if err := ensureLeadIndexes(ctx, db.Collection("leads")); err != nil {
		slog.Error("Failed to apply lead indexes", "error", err)
		return err
	}

	if err := seedPermissionRules(ctx, db.Collection("permission_rules")); err != nil {
		slog.Error("Failed to seed permission rules", "error", err)
		return err
	}

	if err := migrateImportPermissionPath(ctx, db.Collection("permission_rules")); err != nil {
		slog.Warn("Failed to migrate import permission path", "error", err)
	}

	if err := seedRolePermissions(ctx, db.Collection("role_permissions"), db.Collection("permission_rules")); err != nil {
		slog.Error("Failed to seed role permissions", "error", err)
		return err
	}

	if err := seedRoleInheritances(ctx, db.Collection("role_inheritances")); err != nil {
		slog.Error("Failed to seed role inheritances", "error", err)
		return err
	}

	if err := createRolePermissionsWithRulesView(ctx, db); err != nil {
		slog.Warn("Failed to create role_permissions_with_rules view (may already exist)", "error", err)
	}

	if err := seedServiceProviderTenant(ctx, db, cfg); err != nil {
		slog.Error("Failed to seed service provider tenant", "error", err)
		return err
	}

	if err := ensureQualificationIndexes(ctx, db.Collection("qualifications")); err != nil {
		slog.Error("Failed to apply qualification indexes", "error", err)
		return err
	}

	if err := ensureCountryIndexes(ctx, db.Collection("countries")); err != nil {
		slog.Error("Failed to apply country indexes", "error", err)
		return err
	}

	if err := seedQualifications(ctx, db.Collection("qualifications")); err != nil {
		slog.Error("Failed to seed qualifications", "error", err)
		return err
	}

	if err := seedCountries(ctx, db.Collection("countries")); err != nil {
		slog.Error("Failed to seed countries", "error", err)
		return err
	}

	slog.Info("Database migrations completed successfully")
	return nil
}

func ensureUserIndexes(ctx context.Context, collection *mongo.Collection) error {
	// We want email and mobile to be unique
	// Note: We use sparse on mobile just in case old records don't have it,
	// though it's best to handle data explicitly if this is a production application
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "email", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetName("unique_tenant_email"),
		},
		{
			Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "mobile", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetName("unique_tenant_mobile").
				SetSparse(true), // sparse allows multiple docs without a mobile field (e.g. during dev transitions)
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func ensureLeadIndexes(ctx context.Context, collection *mongo.Collection) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "email", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetName("unique_tenant_email"),
		},
		{
			Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "phone", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetName("unique_tenant_phone").
				SetSparse(true), // allows partial updates with missing phone numbers
		},
		// Compound index for admin queries: tenant-wide list + status filter + sort by created_at
		{
			Keys: bson.D{
				{Key: "tenant_id", Value: 1},
				{Key: "status", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("tenant_status_created"),
		},
		// Compound index for user queries: assigned_to scoped list + status filter + sort by created_at
		{
			Keys: bson.D{
				{Key: "tenant_id", Value: 1},
				{Key: "assigned_to", Value: 1},
				{Key: "status", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("tenant_assigned_status_created"),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// seedPermissionRules seeds the database with system permission rules (incremental - adds new rules if they don't exist)
func seedPermissionRules(ctx context.Context, collection *mongo.Collection) error {
	slog.Info("Seeding system permission rules (incremental)...")

	// Define all system permission rules (existing + new)
	tenantView := domain.NewPermissionRule("tenants", "Tenant Management", "view", "View Tenant Details", "/api/v1/tenants/:id", "GET", "View tenant information", true)
	tenantView.RequiresRole = "superadmin"

	tenantUpdate := domain.NewPermissionRule("tenants", "Tenant Management", "update", "Update Tenant", "/api/v1/tenants/:id", "PUT", "Update tenant information", true)
	tenantUpdate.RequiresRole = "superadmin"

	tenantCreate := domain.NewPermissionRule("tenants", "Tenant Management", "create", "Create Tenant", "/api/v1/tenants", "POST", "Create new tenant", true)
	tenantCreate.RequiresRole = "superadmin"

	tenantList := domain.NewPermissionRule("tenants", "Tenant Management", "list", "List Tenants", "/api/v1/tenants/list", "POST", "List all tenants", true)
	tenantList.RequiresRole = "superadmin"

	// User Tenant View (for admin and user roles to view their own tenant)
	userTenantView := domain.NewPermissionRule("user-tenants", "User Tenant View", "view", "View Own Tenant", "/api/v1/user/tenant", "GET", "View current user's tenant", true)

	// User Tenant Update (for admin role to update their own tenant)
	userTenantUpdate := domain.NewPermissionRule("user-tenants", "User Tenant Management", "update", "Update Own Tenant", "/api/v1/user/tenant", "PUT", "Update current user's tenant information", true)

	rules := []domain.PermissionRule{
		// Tenant Management
		*tenantView,
		*tenantUpdate,
		*tenantCreate,
		*tenantList,

		// User Tenant View
		*userTenantView,
		*userTenantUpdate,

		// User Management
		*domain.NewPermissionRule("users", "User Management", "create", "Create User", "/api/v1/users", "POST", "Create a new user", true),
		*domain.NewPermissionRule("users", "User Management", "view", "View User", "/api/v1/users/:id", "GET", "View user details", true),
		*domain.NewPermissionRule("users", "User Management", "update", "Update User", "/api/v1/users/:id", "PUT", "Update user information", true),
		*domain.NewPermissionRule("users", "User Management", "list", "List Users", "/api/v1/users/list", "POST", "List all users", true),

		// Lead Management
		*domain.NewPermissionRule("leads", "Lead Management", "create", "Create Lead", "/api/v1/leads", "POST", "Create a new lead", true),
		*domain.NewPermissionRule("leads", "Lead Management", "view", "View Lead", "/api/v1/leads/:id", "GET", "View lead details", true),
		*domain.NewPermissionRule("leads", "Lead Management", "update", "Update Lead", "/api/v1/leads/:id", "PUT", "Update lead information", true),
		*domain.NewPermissionRule("leads", "Lead Management", "list", "List Leads", "/api/v1/leads/list", "POST", "List all leads", true),
		*domain.NewPermissionRule("leads", "Lead Management", "import", "Import Leads", "/api/v1/leads/import*", "POST", "Import leads from Excel/CSV file", true),

		// Lead Management - Own Scope (for user role)
		*domain.NewPermissionRule("leads", "Lead Management", "list_own", "List Own Leads", "/api/v1/leads/list", "POST", "List leads assigned to self", true, "self", "assigned_to"),
		*domain.NewPermissionRule("leads", "Lead Management", "view_own", "View Own Lead", "/api/v1/leads/:id", "GET", "View leads assigned to self", true, "self", "assigned_to"),
		*domain.NewPermissionRule("leads", "Lead Management", "update_own", "Update Own Lead", "/api/v1/leads/:id", "PUT", "Update leads assigned to self", true, "self", "assigned_to"),

		// Lead Category Management
		*domain.NewPermissionRule("lead-categories", "Lead Categories", "create", "Create Category", "/api/v1/lead-categories", "POST", "Create a lead category", true),
		*domain.NewPermissionRule("lead-categories", "Lead Categories", "view", "View Category", "/api/v1/lead-categories/:id", "GET", "View category details", true),
		*domain.NewPermissionRule("lead-categories", "Lead Categories", "update", "Update Category", "/api/v1/lead-categories/:id", "PUT", "Update category", true),
		*domain.NewPermissionRule("lead-categories", "Lead Categories", "delete", "Delete Category", "/api/v1/lead-categories/:id", "DELETE", "Delete category", true),
		*domain.NewPermissionRule("lead-categories", "Lead Categories", "list", "List Categories", "/api/v1/lead-categories/list", "POST", "List all categories", true),

		// Lead Source Management
		*domain.NewPermissionRule("lead-sources", "Lead Sources", "create", "Create Source", "/api/v1/lead-sources", "POST", "Create a lead source", true),
		*domain.NewPermissionRule("lead-sources", "Lead Sources", "view", "View Source", "/api/v1/lead-sources/:id", "GET", "View source details", true),
		*domain.NewPermissionRule("lead-sources", "Lead Sources", "update", "Update Source", "/api/v1/lead-sources/:id", "PUT", "Update source", true),
		*domain.NewPermissionRule("lead-sources", "Lead Sources", "delete", "Delete Source", "/api/v1/lead-sources/:id", "DELETE", "Delete source", true),
		*domain.NewPermissionRule("lead-sources", "Lead Sources", "list", "List Sources", "/api/v1/lead-sources/list", "POST", "List all sources", true),

		// Lead Comment Management
		*domain.NewPermissionRule("lead-comments", "Lead Comments", "create", "Create Comment", "/api/v1/leads/:lead_id/comments", "POST", "Add comment to lead", true),
		*domain.NewPermissionRule("lead-comments", "Lead Comments", "view", "View Comment", "/api/v1/leads/:lead_id/comments/:id", "GET", "View comment details", true),
		*domain.NewPermissionRule("lead-comments", "Lead Comments", "update", "Update Comment", "/api/v1/leads/:lead_id/comments/:id", "PUT", "Update comment", true),
		*domain.NewPermissionRule("lead-comments", "Lead Comments", "delete", "Delete Comment", "/api/v1/leads/:lead_id/comments/:id", "DELETE", "Delete comment", true),
		*domain.NewPermissionRule("lead-comments", "Lead Comments", "list", "List Comments", "/api/v1/leads/:lead_id/comments/list", "POST", "List all comments", true),

		// Lead Comment Management - Own Scope (for user role)
		*domain.NewPermissionRule("lead-comments", "Lead Comments", "list_own", "List Own Comments", "/api/v1/leads/:lead_id/comments/list", "POST", "List comments by self", true, "self", "created_by"),

		// Lead Appointment Management
		*domain.NewPermissionRule("lead-appointments", "Lead Appointments", "create", "Create Appointment", "/api/v1/leads/:lead_id/appointments", "POST", "Schedule appointment", true),
		*domain.NewPermissionRule("lead-appointments", "Lead Appointments", "view", "View Appointment", "/api/v1/leads/:lead_id/appointments/:id", "GET", "View appointment details", true),
		*domain.NewPermissionRule("lead-appointments", "Lead Appointments", "update", "Update Appointment", "/api/v1/leads/:lead_id/appointments/:id", "PUT", "Update appointment", true),
		*domain.NewPermissionRule("lead-appointments", "Lead Appointments", "delete", "Delete Appointment", "/api/v1/leads/:lead_id/appointments/:id", "DELETE", "Cancel appointment", true),
		*domain.NewPermissionRule("lead-appointments", "Lead Appointments", "list", "List Appointments", "/api/v1/leads/:lead_id/appointments/list", "POST", "List all appointments", true),

		// Lead Appointment Management - Own Scope (for user role)
		*domain.NewPermissionRule("lead-appointments", "Lead Appointments", "list_own", "List Own Appointments", "/api/v1/leads/:lead_id/appointments/list", "POST", "List appointments by self", true, "self", "created_by"),

		// Lead Follow-Up Management
		*domain.NewPermissionRule("lead-follow-ups", "Lead Follow-Ups", "create", "Create Follow-Up", "/api/v1/leads/:lead_id/follow-ups", "POST", "Create follow-up", true),
		*domain.NewPermissionRule("lead-follow-ups", "Lead Follow-Ups", "view", "View Follow-Up", "/api/v1/leads/:lead_id/follow-ups/:id", "GET", "View follow-up details", true),
		*domain.NewPermissionRule("lead-follow-ups", "Lead Follow-Ups", "update", "Update Follow-Up", "/api/v1/leads/:lead_id/follow-ups/:id", "PUT", "Update follow-up", true),
		*domain.NewPermissionRule("lead-follow-ups", "Lead Follow-Ups", "delete", "Delete Follow-Up", "/api/v1/leads/:lead_id/follow-ups/:id", "DELETE", "Delete follow-up", true),
		*domain.NewPermissionRule("lead-follow-ups", "Lead Follow-Ups", "list", "List Follow-Ups", "/api/v1/leads/:lead_id/follow-ups/list", "POST", "List all follow-ups", true),

		// Lead Follow-Up Management - Own Scope (for user role)
		*domain.NewPermissionRule("lead-follow-ups", "Lead Follow-Ups", "list_own", "List Own Follow-Ups", "/api/v1/leads/:lead_id/follow-ups/list", "POST", "List follow-ups by self", true, "self", "created_by"),

		// Permission Management
		*domain.NewPermissionRule("permissions", "Permission Management", "manage", "Manage Permissions", "/api/v1/permissions", "*", "Full permission management access", true),
		*domain.NewPermissionRule("permissions", "Permission Management", "view", "View Permissions", "/api/v1/permissions", "GET", "View all permissions", true),
		*domain.NewPermissionRule("permissions", "Permission Management", "create", "Create Permission", "/api/v1/permissions", "POST", "Create new permission", true),
		*domain.NewPermissionRule("permissions", "Permission Management", "delete", "Delete Permission", "/api/v1/permissions", "DELETE", "Remove permission", true),
		*domain.NewPermissionRule("permissions", "Permission Management", "manage-rules", "Manage Rules", "/api/v1/permissions/rules", "*", "Manage permission rules", true),
		*domain.NewPermissionRule("permissions", "Permission Management", "view-rules", "View Rules", "/api/v1/permissions/rules", "GET", "View permission rules", true),
		*domain.NewPermissionRule("permissions", "Permission Management", "inherit-roles", "Manage Role Inheritance", "/api/v1/permissions/roles/inherit", "*", "Manage role inheritance", true),
		*domain.NewPermissionRule("permissions", "Permission Management", "view-roles", "View Roles", "/api/v1/permissions/roles", "GET", "View all roles and their permissions", true),
		*domain.NewPermissionRule("permissions", "Permission Management", "view-role-permissions", "View Role Permissions", "/api/v1/permissions/roles/:role", "GET", "View permissions for a specific role", true),
		*domain.NewPermissionRule("permissions", "Permission Management", "bulk-update-roles", "Bulk Update Roles", "/api/v1/permissions/roles/:role/bulk", "POST", "Bulk update role permissions", true),

		// Permission Rules Management endpoints for admin
		*domain.NewPermissionRule("permissions", "Permission Management", "available-rules", "View Available Rules", "/api/v1/permissions/available-rules", "GET", "View available permission rules", true),

		// Product Management
		*domain.NewPermissionRule("products", "Product Management", "create", "Create Product", "/api/v1/products", "POST", "Create a new product", true),
		*domain.NewPermissionRule("products", "Product Management", "view", "View Product", "/api/v1/products/:id", "GET", "View product details", true),
		*domain.NewPermissionRule("products", "Product Management", "update", "Update Product", "/api/v1/products/:id", "PUT", "Update product information", true),
		*domain.NewPermissionRule("products", "Product Management", "delete", "Delete Product", "/api/v1/products/:id", "DELETE", "Delete product", true),
		*domain.NewPermissionRule("products", "Product Management", "list", "List Products", "/api/v1/products/list", "POST", "List all products", true),

		// Invoice Management
		*domain.NewPermissionRule("invoices", "Invoice Management", "create", "Create Invoice", "/api/v1/leads/:lead_id/invoices", "POST", "Create a new invoice", true),
		*domain.NewPermissionRule("invoices", "Invoice Management", "view", "View Invoice", "/api/v1/invoices/:id", "GET", "View invoice details", true),
		*domain.NewPermissionRule("invoices", "Invoice Management", "update", "Update Invoice", "/api/v1/invoices/:id", "PUT", "Update invoice", true),
		*domain.NewPermissionRule("invoices", "Invoice Management", "update-due-date", "Update Due Date", "/api/v1/invoices/:id/due-date", "PUT", "Update invoice due date", true),
		*domain.NewPermissionRule("invoices", "Invoice Management", "list", "List Invoices", "/api/v1/invoices/list", "POST", "List all invoices", true),
		*domain.NewPermissionRule("invoices", "Invoice Management", "view-by-lead", "View Invoices By Lead", "/api/v1/leads/:lead_id/invoices", "GET", "View invoices for a lead", true),

		// Receipt Management
		*domain.NewPermissionRule("receipts", "Receipt Management", "create", "Create Receipt", "/api/v1/invoices/:invoice_id/receipts", "POST", "Create a new receipt", true),
		*domain.NewPermissionRule("receipts", "Receipt Management", "view", "View Receipt", "/api/v1/receipts/:id", "GET", "View receipt details", true),
		*domain.NewPermissionRule("receipts", "Receipt Management", "update", "Update Receipt", "/api/v1/receipts/:id", "PUT", "Update receipt", true),
		*domain.NewPermissionRule("receipts", "Receipt Management", "delete", "Delete Receipt", "/api/v1/receipts/:id", "DELETE", "Delete receipt", true),
		*domain.NewPermissionRule("receipts", "Receipt Management", "list", "List Receipts", "/api/v1/invoices/:invoice_id/receipts/list", "POST", "List all receipts for an invoice", true),

		// Charts
		*domain.NewPermissionRule("charts", "Charts", "view", "View Monthly Chart Summary", "/api/v1/charts/monthly-summary", "GET", "View monthly chart data for appointments and comments", true),

		// AI
		*domain.NewPermissionRule("ai", "AI Services", "chat", "Use AI Chat", "/api/v1/ai/chat", "POST", "Allow using the AI Chat with tool calling", true),
	}

	// Use upsert to incrementally add new rules (won't duplicate existing ones)
	upsertCount := 0
	for _, rule := range rules {
		filter := bson.M{"resource": rule.Resource, "action": rule.Action}
		update := bson.M{"$setOnInsert": rule}
		opts := options.Update().SetUpsert(true)
		result, err := collection.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			slog.Warn("Failed to upsert permission rule", "resource", rule.Resource, "action", rule.Action, "error", err)
			continue
		}
		if result.UpsertedCount > 0 {
			upsertCount++
		}
	}

	slog.Info("System permission rules seeded/updated successfully", "new_rules", upsertCount, "total_defined", len(rules))
	return nil
}

func migrateImportPermissionPath(ctx context.Context, collection *mongo.Collection) error {
	filter := bson.M{"resource": "leads", "action": "import", "path": "/api/v1/leads/import"}
	update := bson.M{"$set": bson.M{"path": "/api/v1/leads/import*"}}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.ModifiedCount > 0 {
		slog.Info("Migrated import permission path from /api/v1/leads/import to /api/v1/leads/import*")
	}

	return nil
}

func seedRolePermissions(ctx context.Context, rolePermsCollection, permRulesCollection *mongo.Collection) error {
	slog.Info("Seeding role permissions (incremental)...")

	// Build lookup by resource+action (unique per rule, unlike path+method which can have duplicates)
	type ruleKey struct {
		resource string
		action   string
	}
	ruleIDMap := make(map[ruleKey]primitive.ObjectID)

	cursor, err := permRulesCollection.Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var rule domain.PermissionRule
		if err := cursor.Decode(&rule); err != nil {
			return err
		}
		ruleIDMap[ruleKey{resource: rule.Resource, action: rule.Action}] = rule.ID
	}

	permissions := []struct {
		role     string
		resource string
		action   string
	}{
		// Superadmin exclusively
		{role: "superadmin", resource: "tenants", action: "view"},
		{role: "superadmin", resource: "tenants", action: "update"},
		{role: "superadmin", resource: "tenants", action: "create"},
		{role: "superadmin", resource: "tenants", action: "list"},

		// Admin permissions (unscoped — admin sees all data)
		{role: "admin", resource: "users", action: "create"},
		{role: "admin", resource: "users", action: "view"},
		{role: "admin", resource: "users", action: "update"},
		{role: "admin", resource: "users", action: "list"},
		{role: "admin", resource: "leads", action: "create"},
		{role: "admin", resource: "leads", action: "view"},
		{role: "admin", resource: "leads", action: "update"},
		{role: "admin", resource: "leads", action: "list"},
		{role: "admin", resource: "leads", action: "import"},
		{role: "admin", resource: "lead-categories", action: "create"},
		{role: "admin", resource: "lead-categories", action: "view"},
		{role: "admin", resource: "lead-categories", action: "update"},
		{role: "admin", resource: "lead-categories", action: "delete"},
		{role: "admin", resource: "lead-categories", action: "list"},
		{role: "admin", resource: "lead-sources", action: "create"},
		{role: "admin", resource: "lead-sources", action: "view"},
		{role: "admin", resource: "lead-sources", action: "update"},
		{role: "admin", resource: "lead-sources", action: "delete"},
		{role: "admin", resource: "lead-sources", action: "list"},
		{role: "admin", resource: "lead-comments", action: "create"},
		{role: "admin", resource: "lead-comments", action: "view"},
		{role: "admin", resource: "lead-comments", action: "update"},
		{role: "admin", resource: "lead-comments", action: "delete"},
		{role: "admin", resource: "lead-comments", action: "list"},
		{role: "admin", resource: "lead-appointments", action: "create"},
		{role: "admin", resource: "lead-appointments", action: "view"},
		{role: "admin", resource: "lead-appointments", action: "update"},
		{role: "admin", resource: "lead-appointments", action: "delete"},
		{role: "admin", resource: "lead-appointments", action: "list"},

		// Product Management (admin only)
		{role: "admin", resource: "products", action: "create"},
		{role: "admin", resource: "products", action: "view"},
		{role: "admin", resource: "products", action: "update"},
		{role: "admin", resource: "products", action: "delete"},
		{role: "admin", resource: "products", action: "list"},

		// Invoice Management (admin only - creating invoices)
		{role: "admin", resource: "invoices", action: "create"},
		{role: "admin", resource: "invoices", action: "view"},
		{role: "admin", resource: "invoices", action: "update"},
		{role: "admin", resource: "invoices", action: "update-due-date"},
		{role: "admin", resource: "invoices", action: "list"},
		{role: "admin", resource: "invoices", action: "view-by-lead"},

		// Receipt Management (admin only - creating receipts)
		{role: "admin", resource: "receipts", action: "create"},
		{role: "admin", resource: "receipts", action: "view"},
		{role: "admin", resource: "receipts", action: "update"},
		{role: "admin", resource: "receipts", action: "delete"},
		{role: "admin", resource: "receipts", action: "list"},

		// User Tenant View (admin and user roles)
		{role: "admin", resource: "user-tenants", action: "view"},
		{role: "admin", resource: "user-tenants", action: "update"},
		{role: "admin", resource: "tenants", action: "view"},

		// User permissions (scoped — user sees only own data where applicable)
		{role: "user", resource: "tenants", action: "view"},
		{role: "user", resource: "user-tenants", action: "view"},
		{role: "user", resource: "users", action: "view"},
		{role: "user", resource: "leads", action: "create"},
		{role: "user", resource: "leads", action: "view_own"},
		{role: "user", resource: "leads", action: "update_own"},
		{role: "user", resource: "leads", action: "list_own"},
		{role: "user", resource: "leads", action: "import"},
		{role: "user", resource: "lead-categories", action: "view"},
		{role: "user", resource: "lead-categories", action: "list"},
		{role: "user", resource: "lead-sources", action: "view"},
		{role: "user", resource: "lead-sources", action: "list"},
		{role: "user", resource: "lead-comments", action: "create"},
		{role: "user", resource: "lead-comments", action: "view"},
		{role: "user", resource: "lead-comments", action: "update"},
		{role: "user", resource: "lead-comments", action: "delete"},
		{role: "user", resource: "lead-comments", action: "list_own"},
		{role: "user", resource: "lead-appointments", action: "create"},
		{role: "user", resource: "lead-appointments", action: "view"},
		{role: "user", resource: "lead-appointments", action: "update"},
		{role: "user", resource: "lead-appointments", action: "delete"},
		{role: "user", resource: "lead-appointments", action: "list_own"},

		// Product Management (user - read only)
		{role: "user", resource: "products", action: "view"},
		{role: "user", resource: "products", action: "list"},

		// Invoice Management (user - read only)
		{role: "user", resource: "invoices", action: "view"},
		{role: "user", resource: "invoices", action: "list"},
		{role: "user", resource: "invoices", action: "view-by-lead"},

		// Receipt Management (user - read only)
		{role: "user", resource: "receipts", action: "view"},
		{role: "user", resource: "receipts", action: "list"},

		// Permission management (admin only)
		{role: "admin", resource: "permissions", action: "view"},
		{role: "admin", resource: "permissions", action: "create"},
		{role: "admin", resource: "permissions", action: "delete"},
		{role: "admin", resource: "permissions", action: "manage"},
		{role: "admin", resource: "permissions", action: "manage-rules"},
		{role: "admin", resource: "permissions", action: "view-rules"},
		{role: "admin", resource: "permissions", action: "inherit-roles"},
		{role: "admin", resource: "permissions", action: "view-roles"},
		{role: "admin", resource: "permissions", action: "view-role-permissions"},
		{role: "admin", resource: "permissions", action: "bulk-update-roles"},
		{role: "admin", resource: "permissions", action: "available-rules"},

		// Charts (admin and user)
		{role: "admin", resource: "charts", action: "view"},
		{role: "user", resource: "charts", action: "view"},

		// AI
		{role: "admin", resource: "ai", action: "chat"},
	}

	// Insert incrementally - check if role permission already exists before inserting
	insertCount := 0
	for _, p := range permissions {
		ruleID, ok := ruleIDMap[ruleKey{resource: p.resource, action: p.action}]
		if !ok {
			slog.Warn("Permission rule not found for seeding", "resource", p.resource, "action", p.action)
			continue
		}

		// Check if this role permission already exists
		existingCount, err := rolePermsCollection.CountDocuments(ctx, bson.M{
			"role":               p.role,
			"permission_rule_id": ruleID,
		})
		if err != nil {
			slog.Warn("Failed to check existing role permission", "error", err)
			continue
		}

		if existingCount > 0 {
			continue // Skip - already exists
		}

		// Insert new role permission
		_, err = rolePermsCollection.InsertOne(ctx, domain.NewRolePermission(p.role, ruleID))
		if err != nil {
			slog.Warn("Failed to insert role permission", "role", p.role, "resource", p.resource, "action", p.action, "error", err)
			continue
		}
		insertCount++
	}

	slog.Info("Role permissions seeded successfully", "new_permissions", insertCount)
	return nil
}

func seedRoleInheritances(ctx context.Context, collection *mongo.Collection) error {
	count, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}
	if count > 0 {
		slog.Info("Role inheritances already seeded, skipping")
		return nil
	}

	inheritances := []*domain.RoleInheritance{
		{ChildRole: "superadmin", ParentRole: "admin"},
	}

	docs := make([]interface{}, len(inheritances))
	for i, iObj := range inheritances {
		docs[i] = iObj
	}

	_, err = collection.InsertMany(ctx, docs)
	if err != nil {
		return err
	}

	slog.Info("Role inheritances seeded successfully", "count", len(inheritances))
	return nil
}

func createRolePermissionsWithRulesView(ctx context.Context, db *mongo.Database) error {
	viewName := "role_permissions_with_rules"

	pipeline := mongo.Pipeline{
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "permission_rules"},
			{Key: "localField", Value: "permission_rule_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "permission_rule"},
		}}},
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$permission_rule"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
		{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 1},
			{Key: "role", Value: 1},
			{Key: "created_at", Value: 1},
			{Key: "path", Value: "$permission_rule.path"},
			{Key: "method", Value: "$permission_rule.method"},
			{Key: "resource", Value: "$permission_rule.resource"},
			{Key: "action", Value: "$permission_rule.action"},
			{Key: "scope_type", Value: "$permission_rule.scope_type"},
			{Key: "filter_field", Value: "$permission_rule.filter_field"},
		}}},
	}

	err := db.CreateView(ctx, viewName, "role_permissions", pipeline)
	if err != nil {
		slog.Debug("View creation info", "error", err)
		return err
	}

	slog.Info("Created role_permissions_with_rules view")
	return nil
}

func seedServiceProviderTenant(ctx context.Context, db *mongo.Database, cfg *config.Config) error {
	tenantsCollection := db.Collection("tenants")
	usersCollection := db.Collection("users")

	// 1. Check if the service provider tenant exists
	var tenant domain.Tenant
	err := tenantsCollection.FindOne(ctx, bson.M{"name": cfg.ServiceTenantName}).Decode(&tenant)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Create tenant
			slog.Info("Service provider tenant not found. Creating...", "name", cfg.ServiceTenantName)
			tenant = *domain.NewTenant(cfg.ServiceTenantName, cfg.SuperAdminEmail)
			_, err = tenantsCollection.InsertOne(ctx, tenant)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// 2. Check if the superadmin user exists
	count, err := usersCollection.CountDocuments(ctx, bson.M{"email": cfg.SuperAdminEmail})
	if err != nil {
		return err
	}

	if count == 0 {
		slog.Info("Super admin user not found. Creating...", "email", cfg.SuperAdminEmail)

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(cfg.SuperAdminPassword), 14)
		if err != nil {
			return err
		}

		user := domain.NewUser(
			tenant.ID,
			cfg.SuperAdminName,
			cfg.SuperAdminEmail,
			"", // Mobile
			string(hashedPassword),
			"superadmin", // explicitly superadmin role
		)

		_, err = usersCollection.InsertOne(ctx, user)
		if err != nil {
			return err
		}

		slog.Info("Super admin user created successfully", "tenant_id", tenant.ID.Hex(), "user_id", user.ID.Hex())
	}

	return nil
}

func ensureQualificationIndexes(ctx context.Context, collection *mongo.Collection) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("unique_name"),
		},
	}
	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func ensureCountryIndexes(ctx context.Context, collection *mongo.Collection) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "iso2", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("unique_iso2"),
		},
		{
			Keys:    bson.D{{Key: "iso3", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("unique_iso3"),
		},
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("unique_name"),
		},
	}
	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func seedQualifications(ctx context.Context, collection *mongo.Collection) error {
	count, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}
	if count > 0 {
		slog.Info("Qualifications already seeded, skipping")
		return nil
	}

	qualifications := []*domain.Qualification{
		domain.NewQualification("High School"),
		domain.NewQualification("Associate's Degree"),
		domain.NewQualification("Bachelor's Degree"),
		domain.NewQualification("Master's Degree"),
		domain.NewQualification("Doctorate (PhD)"),
		domain.NewQualification("Post-Doctorate"),
		domain.NewQualification("Professional Certification"),
		domain.NewQualification("Diploma"),
		domain.NewQualification("Vocational Training"),
		domain.NewQualification("Other"),
	}

	docs := make([]interface{}, len(qualifications))
	for i, q := range qualifications {
		docs[i] = q
	}

	_, err = collection.InsertMany(ctx, docs)
	if err != nil {
		return err
	}

	slog.Info("Qualifications seeded successfully", "count", len(qualifications))
	return nil
}

func seedCountries(ctx context.Context, collection *mongo.Collection) error {
	count, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}
	if count > 0 {
		slog.Info("Countries already seeded, skipping")
		return nil
	}

	countries := []*domain.Country{
		domain.NewCountry("Afghanistan", "AF", "AFG", "+93", "AFN", "Afghan Afghani"),
		domain.NewCountry("Albania", "AL", "ALB", "+355", "ALL", "Albanian Lek"),
		domain.NewCountry("Algeria", "DZ", "DZA", "+213", "DZD", "Algerian Dinar"),
		domain.NewCountry("Andorra", "AD", "AND", "+376", "EUR", "Euro"),
		domain.NewCountry("Angola", "AO", "AGO", "+244", "AOA", "Angolan Kwanza"),
		domain.NewCountry("Argentina", "AR", "ARG", "+54", "ARS", "Argentine Peso"),
		domain.NewCountry("Armenia", "AM", "ARM", "+374", "AMD", "Armenian Dram"),
		domain.NewCountry("Australia", "AU", "AUS", "+61", "AUD", "Australian Dollar"),
		domain.NewCountry("Austria", "AT", "AUT", "+43", "EUR", "Euro"),
		domain.NewCountry("Azerbaijan", "AZ", "AZE", "+994", "AZN", "Azerbaijani Manat"),
		domain.NewCountry("Bahamas", "BS", "BHS", "+1", "BSD", "Bahamian Dollar"),
		domain.NewCountry("Bahrain", "BH", "BHR", "+973", "BHD", "Bahraini Dinar"),
		domain.NewCountry("Bangladesh", "BD", "BGD", "+880", "BDT", "Bangladeshi Taka"),
		domain.NewCountry("Belarus", "BY", "BLR", "+375", "BYN", "Belarusian Ruble"),
		domain.NewCountry("Belgium", "BE", "BEL", "+32", "EUR", "Euro"),
		domain.NewCountry("Belize", "BZ", "BLZ", "+501", "BZD", "Belize Dollar"),
		domain.NewCountry("Benin", "BJ", "BEN", "+229", "XOF", "West African CFA Franc"),
		domain.NewCountry("Bhutan", "BT", "BTN", "+975", "BTN", "Bhutanese Ngultrum"),
		domain.NewCountry("Bolivia", "BO", "BOL", "+591", "BOB", "Bolivian Boliviano"),
		domain.NewCountry("Bosnia and Herzegovina", "BA", "BIH", "+387", "BAM", "Bosnia-Herzegovina Convertible Mark"),
		domain.NewCountry("Botswana", "BW", "BWA", "+267", "BWP", "Botswanan Pula"),
		domain.NewCountry("Brazil", "BR", "BRA", "+55", "BRL", "Brazilian Real"),
		domain.NewCountry("Brunei", "BN", "BRN", "+673", "BND", "Brunei Dollar"),
		domain.NewCountry("Bulgaria", "BG", "BGR", "+359", "BGN", "Bulgarian Lev"),
		domain.NewCountry("Burkina Faso", "BF", "BFA", "+226", "XOF", "West African CFA Franc"),
		domain.NewCountry("Burundi", "BI", "BDI", "+257", "BIF", "Burundian Franc"),
		domain.NewCountry("Cambodia", "KH", "KHM", "+855", "KHR", "Cambodian Riel"),
		domain.NewCountry("Cameroon", "CM", "CMR", "+237", "XAF", "Central African CFA Franc"),
		domain.NewCountry("Canada", "CA", "CAN", "+1", "CAD", "Canadian Dollar"),
		domain.NewCountry("Cape Verde", "CV", "CPV", "+238", "CVE", "Cape Verdean Escudo"),
		domain.NewCountry("Central African Republic", "CF", "CAF", "+236", "XAF", "Central African CFA Franc"),
		domain.NewCountry("Chad", "TD", "TCD", "+235", "XAF", "Central African CFA Franc"),
		domain.NewCountry("Chile", "CL", "CHL", "+56", "CLP", "Chilean Peso"),
		domain.NewCountry("China", "CN", "CHN", "+86", "CNY", "Chinese Yuan"),
		domain.NewCountry("Colombia", "CO", "COL", "+57", "COP", "Colombian Peso"),
		domain.NewCountry("Congo", "CG", "COG", "+242", "XAF", "Central African CFA Franc"),
		domain.NewCountry("Costa Rica", "CR", "CRI", "+506", "CRC", "Costa Rican Colón"),
		domain.NewCountry("Croatia", "HR", "HRV", "+385", "EUR", "Euro"),
		domain.NewCountry("Cuba", "CU", "CUB", "+53", "CUP", "Cuban Peso"),
		domain.NewCountry("Cyprus", "CY", "CYP", "+357", "EUR", "Euro"),
		domain.NewCountry("Czech Republic", "CZ", "CZE", "+420", "CZK", "Czech Republic Koruna"),
		domain.NewCountry("Denmark", "DK", "DNK", "+45", "DKK", "Danish Krone"),
		domain.NewCountry("Djibouti", "DJ", "DJI", "+253", "DJF", "Djiboutian Franc"),
		domain.NewCountry("Dominican Republic", "DO", "DOM", "+1", "DOP", "Dominican Peso"),
		domain.NewCountry("Ecuador", "EC", "ECU", "+593", "USD", "United States Dollar"),
		domain.NewCountry("Egypt", "EG", "EGY", "+20", "EGP", "Egyptian Pound"),
		domain.NewCountry("El Salvador", "SV", "SLV", "+503", "USD", "United States Dollar"),
		domain.NewCountry("Eritrea", "ER", "ERI", "+291", "ERN", "Eritrean Nakfa"),
		domain.NewCountry("Estonia", "EE", "EST", "+372", "EUR", "Euro"),
		domain.NewCountry("Ethiopia", "ET", "ETH", "+251", "ETB", "Ethiopian Birr"),
		domain.NewCountry("Fiji", "FJ", "FJI", "+679", "FJD", "Fijian Dollar"),
		domain.NewCountry("Finland", "FI", "FIN", "+358", "EUR", "Euro"),
		domain.NewCountry("France", "FR", "FRA", "+33", "EUR", "Euro"),
		domain.NewCountry("Gabon", "GA", "GAB", "+241", "XAF", "Central African CFA Franc"),
		domain.NewCountry("Gambia", "GM", "GMB", "+220", "GMD", "Gambian Dalasi"),
		domain.NewCountry("Georgia", "GE", "GEO", "+995", "GEL", "Georgian Lari"),
		domain.NewCountry("Germany", "DE", "DEU", "+49", "EUR", "Euro"),
		domain.NewCountry("Ghana", "GH", "GHA", "+233", "GHS", "Ghanaian Cedi"),
		domain.NewCountry("Greece", "GR", "GRC", "+30", "EUR", "Euro"),
		domain.NewCountry("Guatemala", "GT", "GTM", "+502", "GTQ", "Guatemalan Quetzal"),
		domain.NewCountry("Guinea", "GN", "GIN", "+224", "GNF", "Guinean Franc"),
		domain.NewCountry("Guyana", "GY", "GUY", "+592", "GYD", "Guyanaese Dollar"),
		domain.NewCountry("Haiti", "HT", "HTI", "+509", "HTG", "Haitian Gourde"),
		domain.NewCountry("Honduras", "HN", "HND", "+504", "HNL", "Honduran Lempira"),
		domain.NewCountry("Hungary", "HU", "HUN", "+36", "HUF", "Hungarian Forint"),
		domain.NewCountry("Iceland", "IS", "ISL", "+354", "ISK", "Icelandic Króna"),
		domain.NewCountry("India", "IN", "IND", "+91", "INR", "Indian Rupee"),
		domain.NewCountry("Indonesia", "ID", "IDN", "+62", "IDR", "Indonesian Rupiah"),
		domain.NewCountry("Iran", "IR", "IRN", "+98", "IRR", "Iranian Rial"),
		domain.NewCountry("Iraq", "IQ", "IRQ", "+964", "IQD", "Iraqi Dinar"),
		domain.NewCountry("Ireland", "IE", "IRL", "+353", "EUR", "Euro"),
		domain.NewCountry("Israel", "IL", "ISR", "+972", "ILS", "Israeli New Sheqel"),
		domain.NewCountry("Italy", "IT", "ITA", "+39", "EUR", "Euro"),
		domain.NewCountry("Jamaica", "JM", "JAM", "+1", "JMD", "Jamaican Dollar"),
		domain.NewCountry("Japan", "JP", "JPN", "+81", "JPY", "Japanese Yen"),
		domain.NewCountry("Jordan", "JO", "JOR", "+962", "JOD", "Jordanian Dinar"),
		domain.NewCountry("Kazakhstan", "KZ", "KAZ", "+7", "KZT", "Kazakhstani Tenge"),
		domain.NewCountry("Kenya", "KE", "KEN", "+254", "KES", "Kenyan Shilling"),
		domain.NewCountry("Kuwait", "KW", "KWT", "+965", "KWD", "Kuwaiti Dinar"),
		domain.NewCountry("Kyrgyzstan", "KG", "KGZ", "+996", "KGS", "Kyrgystani Som"),
		domain.NewCountry("Laos", "LA", "LAO", "+856", "LAK", "Laotian Kip"),
		domain.NewCountry("Latvia", "LV", "LVA", "+371", "EUR", "Euro"),
		domain.NewCountry("Lebanon", "LB", "LBN", "+961", "LBP", "Lebanese Pound"),
		domain.NewCountry("Lesotho", "LS", "LSO", "+266", "LSL", "Lesotho Loti"),
		domain.NewCountry("Liberia", "LR", "LBR", "+231", "LRD", "Liberian Dollar"),
		domain.NewCountry("Libya", "LY", "LBY", "+218", "LYD", "Libyan Dinar"),
		domain.NewCountry("Lithuania", "LT", "LTU", "+370", "EUR", "Euro"),
		domain.NewCountry("Luxembourg", "LU", "LUX", "+352", "EUR", "Euro"),
		domain.NewCountry("Madagascar", "MG", "MDG", "+261", "MGA", "Malagasy Ariary"),
		domain.NewCountry("Malawi", "MW", "MWI", "+265", "MWK", "Malawian Kwacha"),
		domain.NewCountry("Malaysia", "MY", "MYS", "+60", "MYR", "Malaysian Ringgit"),
		domain.NewCountry("Maldives", "MV", "MDV", "+960", "MVR", "Maldivian Rufiyaa"),
		domain.NewCountry("Mali", "ML", "MLI", "+223", "XOF", "West African CFA Franc"),
		domain.NewCountry("Malta", "MT", "MLT", "+356", "EUR", "Euro"),
		domain.NewCountry("Mauritania", "MR", "MRT", "+222", "MRU", "Mauritanian Ouguiya"),
		domain.NewCountry("Mauritius", "MU", "MUS", "+230", "MUR", "Mauritian Rupee"),
		domain.NewCountry("Mexico", "MX", "MEX", "+52", "MXN", "Mexican Peso"),
		domain.NewCountry("Moldova", "MD", "MDA", "+373", "MDL", "Moldovan Leu"),
		domain.NewCountry("Monaco", "MC", "MCO", "+377", "EUR", "Euro"),
		domain.NewCountry("Mongolia", "MN", "MNG", "+976", "MNT", "Mongolian Tugrik"),
		domain.NewCountry("Montenegro", "ME", "MNE", "+382", "EUR", "Euro"),
		domain.NewCountry("Morocco", "MA", "MAR", "+212", "MAD", "Moroccan Dirham"),
		domain.NewCountry("Mozambique", "MZ", "MOZ", "+258", "MZN", "Mozambican Metical"),
		domain.NewCountry("Myanmar", "MM", "MMR", "+95", "MMK", "Myanma Kyat"),
		domain.NewCountry("Namibia", "NA", "NAM", "+264", "NAD", "Namibian Dollar"),
		domain.NewCountry("Nepal", "NP", "NPL", "+977", "NPR", "Nepalese Rupee"),
		domain.NewCountry("Netherlands", "NL", "NLD", "+31", "EUR", "Euro"),
		domain.NewCountry("New Zealand", "NZ", "NZL", "+64", "NZD", "New Zealand Dollar"),
		domain.NewCountry("Nicaragua", "NI", "NIC", "+505", "NIO", "Nicaraguan Córdoba"),
		domain.NewCountry("Niger", "NE", "NER", "+227", "XOF", "West African CFA Franc"),
		domain.NewCountry("Nigeria", "NG", "NGA", "+234", "NGN", "Nigerian Naira"),
		domain.NewCountry("North Korea", "KP", "PRK", "+850", "KPW", "North Korean Won"),
		domain.NewCountry("North Macedonia", "MK", "MKD", "+389", "MKD", "Macedonian Denar"),
		domain.NewCountry("Norway", "NO", "NOR", "+47", "NOK", "Norwegian Krone"),
		domain.NewCountry("Oman", "OM", "OMN", "+968", "OMR", "Omani Rial"),
		domain.NewCountry("Pakistan", "PK", "PAK", "+92", "PKR", "Pakistani Rupee"),
		domain.NewCountry("Panama", "PA", "PAN", "+507", "PAB", "Panamanian Balboa"),
		domain.NewCountry("Papua New Guinea", "PG", "PNG", "+675", "PGK", "Papua New Guinean Kina"),
		domain.NewCountry("Paraguay", "PY", "PRY", "+595", "PYG", "Paraguayan Guarani"),
		domain.NewCountry("Peru", "PE", "PER", "+51", "PEN", "Peruvian Nuevo Sol"),
		domain.NewCountry("Philippines", "PH", "PHL", "+63", "PHP", "Philippine Peso"),
		domain.NewCountry("Poland", "PL", "POL", "+48", "PLN", "Polish Zloty"),
		domain.NewCountry("Portugal", "PT", "PRT", "+351", "EUR", "Euro"),
		domain.NewCountry("Qatar", "QA", "QAT", "+974", "QAR", "Qatari Rial"),
		domain.NewCountry("Romania", "RO", "ROU", "+40", "RON", "Romanian Leu"),
		domain.NewCountry("Russia", "RU", "RUS", "+7", "RUB", "Russian Ruble"),
		domain.NewCountry("Rwanda", "RW", "RWA", "+250", "RWF", "Rwandan Franc"),
		domain.NewCountry("Saudi Arabia", "SA", "SAU", "+966", "SAR", "Saudi Riyal"),
		domain.NewCountry("Senegal", "SN", "SEN", "+221", "XOF", "West African CFA Franc"),
		domain.NewCountry("Serbia", "RS", "SRB", "+381", "RSD", "Serbian Dinar"),
		domain.NewCountry("Sierra Leone", "SL", "SLE", "+232", "SLL", "Sierra Leonean Leone"),
		domain.NewCountry("Singapore", "SG", "SGP", "+65", "SGD", "Singapore Dollar"),
		domain.NewCountry("Slovakia", "SK", "SVK", "+421", "EUR", "Euro"),
		domain.NewCountry("Slovenia", "SI", "SVN", "+386", "EUR", "Euro"),
		domain.NewCountry("Somalia", "SO", "SOM", "+252", "SOS", "Somali Shilling"),
		domain.NewCountry("South Africa", "ZA", "ZAF", "+27", "ZAR", "South African Rand"),
		domain.NewCountry("South Korea", "KR", "KOR", "+82", "KRW", "South Korean Won"),
		domain.NewCountry("South Sudan", "SS", "SSD", "+211", "SSP", "South Sudanese Pound"),
		domain.NewCountry("Spain", "ES", "ESP", "+34", "EUR", "Euro"),
		domain.NewCountry("Sri Lanka", "LK", "LKA", "+94", "LKR", "Sri Lankan Rupee"),
		domain.NewCountry("Sudan", "SD", "SDN", "+249", "SDG", "Sudanese Pound"),
		domain.NewCountry("Suriname", "SR", "SUR", "+597", "SRD", "Surinamese Dollar"),
		domain.NewCountry("Sweden", "SE", "SWE", "+46", "SEK", "Swedish Krona"),
		domain.NewCountry("Switzerland", "CH", "CHE", "+41", "CHF", "Swiss Franc"),
		domain.NewCountry("Syria", "SY", "SYR", "+963", "SYP", "Syrian Pound"),
		domain.NewCountry("Taiwan", "TW", "TWN", "+886", "TWD", "New Taiwan Dollar"),
		domain.NewCountry("Tajikistan", "TJ", "TJK", "+992", "TJS", "Tajikistani Somoni"),
		domain.NewCountry("Tanzania", "TZ", "TZA", "+255", "TZS", "Tanzanian Shilling"),
		domain.NewCountry("Thailand", "TH", "THA", "+66", "THB", "Thai Baht"),
		domain.NewCountry("Togo", "TG", "TGO", "+228", "XOF", "West African CFA Franc"),
		domain.NewCountry("Trinidad and Tobago", "TT", "TTO", "+1", "TTD", "Trinidad and Tobago Dollar"),
		domain.NewCountry("Tunisia", "TN", "TUN", "+216", "TND", "Tunisian Dinar"),
		domain.NewCountry("Turkey", "TR", "TUR", "+90", "TRY", "Turkish Lira"),
		domain.NewCountry("Turkmenistan", "TM", "TKM", "+993", "TMT", "Turkmenistani Manat"),
		domain.NewCountry("Uganda", "UG", "UGA", "+256", "UGX", "Ugandan Shilling"),
		domain.NewCountry("Ukraine", "UA", "UKR", "+380", "UAH", "Ukrainian Hryvnia"),
		domain.NewCountry("United Arab Emirates", "AE", "ARE", "+971", "AED", "United Arab Emirates Dirham"),
		domain.NewCountry("United Kingdom", "GB", "GBR", "+44", "GBP", "British Pound Sterling"),
		domain.NewCountry("United States", "US", "USA", "+1", "USD", "United States Dollar"),
		domain.NewCountry("Uruguay", "UY", "URY", "+598", "UYU", "Uruguayan Peso"),
		domain.NewCountry("Uzbekistan", "UZ", "UZB", "+998", "UZS", "Uzbekistan Som"),
		domain.NewCountry("Venezuela", "VE", "VEN", "+58", "VES", "Venezuelan Bolívar"),
		domain.NewCountry("Vietnam", "VN", "VNM", "+84", "VND", "Vietnamese Dong"),
		domain.NewCountry("Yemen", "YE", "YEM", "+967", "YER", "Yemeni Rial"),
		domain.NewCountry("Zambia", "ZM", "ZMB", "+260", "ZMW", "Zambian Kwacha"),
		domain.NewCountry("Zimbabwe", "ZW", "ZWE", "+263", "ZWL", "Zimbabwean Dollar"),
	}

	docs := make([]interface{}, len(countries))
	for i, c := range countries {
		docs[i] = c
	}

	_, err = collection.InsertMany(ctx, docs)
	if err != nil {
		return err
	}

	slog.Info("Countries seeded successfully", "count", len(countries))
	return nil
}

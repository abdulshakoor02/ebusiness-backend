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
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// seedPermissionRules seeds the database with system permission rules
func seedPermissionRules(ctx context.Context, collection *mongo.Collection) error {
	// Check if already seeded
	count, err := collection.CountDocuments(ctx, bson.M{"is_system": true})
	if err != nil {
		return err
	}

	if count > 0 {
		slog.Info("Permission rules already seeded, skipping...")
		return nil
	}

	slog.Info("Seeding system permission rules...")

	// Define all system permission rules

	tenantView := domain.NewPermissionRule("tenants", "Tenant Management", "view", "View Tenant Details", "/api/v1/tenants/:id", "GET", "View tenant information", true)
	tenantView.RequiresRole = "superadmin"

	tenantUpdate := domain.NewPermissionRule("tenants", "Tenant Management", "update", "Update Tenant", "/api/v1/tenants/:id", "PUT", "Update tenant information", true)
	tenantUpdate.RequiresRole = "superadmin"

	tenantCreate := domain.NewPermissionRule("tenants", "Tenant Management", "create", "Create Tenant", "/api/v1/tenants", "POST", "Create new tenant", true)
	tenantCreate.RequiresRole = "superadmin"

	tenantList := domain.NewPermissionRule("tenants", "Tenant Management", "list", "List Tenants", "/api/v1/tenants/list", "POST", "List all tenants", true)
	tenantList.RequiresRole = "superadmin"

	rules := []domain.PermissionRule{
		// Tenant Management
		*tenantView,
		*tenantUpdate,
		*tenantCreate,
		*tenantList,

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
	}

	docs := make([]interface{}, len(rules))
	for i, rule := range rules {
		docs[i] = rule
	}

	_, err = collection.InsertMany(ctx, docs)
	if err != nil {
		return err
	}

	slog.Info("System permission rules seeded successfully", "count", len(rules))
	return nil
}

func seedRolePermissions(ctx context.Context, rolePermsCollection, permRulesCollection *mongo.Collection) error {
	count, err := rolePermsCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}
	if count > 0 {
		slog.Info("Role permissions already seeded, skipping")
		return nil
	}

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

		// User permissions (scoped — user sees only own data where applicable)
		{role: "user", resource: "tenants", action: "view"},
		{role: "user", resource: "users", action: "view"},
		{role: "user", resource: "leads", action: "create"},
		{role: "user", resource: "leads", action: "view_own"},
		{role: "user", resource: "leads", action: "update_own"},
		{role: "user", resource: "leads", action: "list_own"},
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
	}

	var docs []interface{}
	for _, p := range permissions {
		ruleID, ok := ruleIDMap[ruleKey{resource: p.resource, action: p.action}]
		if !ok {
			slog.Warn("Permission rule not found for seeding", "resource", p.resource, "action", p.action)
			continue
		}
		docs = append(docs, domain.NewRolePermission(p.role, ruleID))
	}

	if len(docs) == 0 {
		slog.Warn("No role permissions to seed")
		return nil
	}

	_, err = rolePermsCollection.InsertMany(ctx, docs)
	if err != nil {
		return err
	}

	slog.Info("Role permissions seeded successfully", "count", len(docs))
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

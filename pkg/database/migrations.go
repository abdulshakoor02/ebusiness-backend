package database

import (
	"context"
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// RunMigrations runs application boot time database setup such as ensuring indexes
func RunMigrations(ctx context.Context, db *mongo.Database) error {
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
	rules := []domain.PermissionRule{
		// Tenant Management
		*domain.NewPermissionRule("tenants", "Tenant Management", "view", "View Tenant Details", "/api/v1/tenants/:id", "GET", "View tenant information", true),
		*domain.NewPermissionRule("tenants", "Tenant Management", "update", "Update Tenant", "/api/v1/tenants/:id", "PUT", "Update tenant information", true),
		*domain.NewPermissionRule("tenants", "Tenant Management", "list", "List Tenants", "/api/v1/tenants/list", "POST", "List all tenants", true),

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

		// Lead Appointment Management
		*domain.NewPermissionRule("lead-appointments", "Lead Appointments", "create", "Create Appointment", "/api/v1/leads/:lead_id/appointments", "POST", "Schedule appointment", true),
		*domain.NewPermissionRule("lead-appointments", "Lead Appointments", "view", "View Appointment", "/api/v1/leads/:lead_id/appointments/:id", "GET", "View appointment details", true),
		*domain.NewPermissionRule("lead-appointments", "Lead Appointments", "update", "Update Appointment", "/api/v1/leads/:lead_id/appointments/:id", "PUT", "Update appointment", true),
		*domain.NewPermissionRule("lead-appointments", "Lead Appointments", "delete", "Delete Appointment", "/api/v1/leads/:lead_id/appointments/:id", "DELETE", "Cancel appointment", true),
		*domain.NewPermissionRule("lead-appointments", "Lead Appointments", "list", "List Appointments", "/api/v1/leads/:lead_id/appointments/list", "POST", "List all appointments", true),

		// Permission Management
		*domain.NewPermissionRule("permissions", "Permission Management", "manage", "Manage Permissions", "/api/v1/permissions", "*", "Full permission management access", true),
		*domain.NewPermissionRule("permissions", "Permission Management", "view", "View Permissions", "/api/v1/permissions", "GET", "View all permissions", true),
		*domain.NewPermissionRule("permissions", "Permission Management", "create", "Create Permission", "/api/v1/permissions", "POST", "Create new permission", true),
		*domain.NewPermissionRule("permissions", "Permission Management", "delete", "Delete Permission", "/api/v1/permissions", "DELETE", "Remove permission", true),
		*domain.NewPermissionRule("permissions", "Permission Management", "manage-rules", "Manage Rules", "/api/v1/permissions/rules", "*", "Manage permission rules", true),
		*domain.NewPermissionRule("permissions", "Permission Management", "inherit-roles", "Manage Role Inheritance", "/api/v1/permissions/roles/inherit", "*", "Manage role inheritance", true),
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

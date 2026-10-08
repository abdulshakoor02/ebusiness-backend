package database

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ensureDashboardIndexes supports bounded time-series, tenant fences, and the
// personal organizer/creator/author lookups without changing existing indexes.
func ensureDashboardIndexes(ctx context.Context, db *mongo.Database) error {
	indexes := map[string][]mongo.IndexModel{
		"tenants": {dashboardIndex("dashboard_created", "created_at")},
		"users":   {dashboardIndex("dashboard_tenant_created", "tenant_id", "created_at"), dashboardIndex("dashboard_created", "created_at")},
		"leads": {
			dashboardIndex("dashboard_tenant_created", "tenant_id", "created_at"),
			dashboardIndex("dashboard_assigned_created", "tenant_id", "assigned_to", "created_at"),
			dashboardIndex("dashboard_created", "created_at"),
		},
		"lead_appointments": {
			dashboardIndex("dashboard_tenant_start", "tenant_id", "start_time"),
			dashboardIndex("dashboard_organizer_start", "tenant_id", "organizer_id", "start_time"),
		},
		"lead_follow_ups": {
			dashboardIndex("dashboard_tenant_status_start", "tenant_id", "status", "start_time"),
			dashboardIndex("dashboard_creator_status_start", "tenant_id", "creator_id", "status", "start_time"),
			dashboardIndex("dashboard_creator_start", "tenant_id", "creator_id", "start_time"),
		},
		"lead_comments": {
			dashboardIndex("dashboard_tenant_created", "tenant_id", "created_at"),
			dashboardIndex("dashboard_author_created", "tenant_id", "author_id", "created_at"),
		},
		"invoices": {dashboardIndex("dashboard_tenant_created", "tenant_id", "created_at")},
		"receipts": {dashboardIndex("dashboard_tenant_payment", "tenant_id", "payment_date")},
	}
	for collection, models := range indexes {
		if _, err := db.Collection(collection).Indexes().CreateMany(ctx, models); err != nil {
			return err
		}
	}
	return nil
}

func dashboardIndex(name string, fields ...string) mongo.IndexModel {
	keys := make(bson.D, 0, len(fields))
	for _, field := range fields {
		keys = append(keys, bson.E{Key: field, Value: 1})
	}
	return mongo.IndexModel{Keys: keys, Options: options.Index().SetName(name)}
}

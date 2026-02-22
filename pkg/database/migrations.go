package database

import (
	"context"
	"log/slog"

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

	slog.Info("Database migrations completed successfully")
	return nil
}

func ensureUserIndexes(ctx context.Context, collection *mongo.Collection) error {
	// We want email and mobile to be unique
	// Note: We use sparse on mobile just in case old records don't have it,
	// though it's best to handle data explicitly if this is a production application
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "email", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetName("unique_email"),
		},
		{
			Keys: bson.D{{Key: "mobile", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetName("unique_mobile").
				SetSparse(true), // sparse allows multiple docs without a mobile field (e.g. during dev transitions)
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

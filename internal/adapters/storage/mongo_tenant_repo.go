package storage

import (
	"context"
	"errors"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoTenantRepository struct {
	collection *mongo.Collection
}

func NewMongoTenantRepository(db *mongo.Database) *MongoTenantRepository {
	return &MongoTenantRepository{
		collection: db.Collection("tenants"),
	}
}

func (r *MongoTenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	_, err := r.collection.InsertOne(ctx, tenant)
	return err
}

func (r *MongoTenantRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Tenant, error) {
	var tenant domain.Tenant
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&tenant)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("tenant not found")
		}
		return nil, err
	}
	return &tenant, nil
}

func (r *MongoTenantRepository) List(ctx context.Context, filter interface{}, offset, limit int64) ([]*domain.Tenant, int64, error) {
	// Parse filter
	query := bson.M{}
	if f, ok := filter.(map[string]interface{}); ok {
		for k, v := range f {
			// Basic equality for now, can be expanded for regex search
			query[k] = v
		}
	}

	// Count total
	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	// Find with pagination
	findOptions := options.Find()
	findOptions.SetSkip(offset)
	findOptions.SetLimit(limit)
	findOptions.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var tenants []*domain.Tenant
	if err = cursor.All(ctx, &tenants); err != nil {
		return nil, 0, err
	}

	return tenants, total, nil
}

func (r *MongoTenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	update := bson.M{
		"$set": bson.M{
			"name":       tenant.Name,
			"email":      tenant.Email,
			"logo_url":   tenant.LogoURL,
			"address":    tenant.Address,
			"country_id": tenant.CountryID,
			"tax":        tenant.Tax,
			"updated_at": time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": tenant.ID}, update)
	return err
}

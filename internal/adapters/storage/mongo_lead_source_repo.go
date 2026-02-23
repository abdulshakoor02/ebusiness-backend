package storage

import (
	"context"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoLeadSourceRepository struct {
	collection *mongo.Collection
}

func NewMongoLeadSourceRepository(db *mongo.Database) *MongoLeadSourceRepository {
	return &MongoLeadSourceRepository{
		collection: db.Collection("lead_sources"),
	}
}

func (r *MongoLeadSourceRepository) Create(ctx context.Context, source *domain.LeadSource) error {
	_, err := r.collection.InsertOne(ctx, source)
	return err
}

func (r *MongoLeadSourceRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.LeadSource, error) {
	filter := bson.M{"_id": id}

	if tenantID, ok := getTenantIDFromContext(ctx); ok {
		filter["tenant_id"] = tenantID
	}

	var source domain.LeadSource
	err := r.collection.FindOne(ctx, filter).Decode(&source)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, err // Returning actual error to be wrapped/handled by caller
		}
		return nil, err
	}
	return &source, nil
}

func (r *MongoLeadSourceRepository) List(ctx context.Context, filter interface{}, offset, limit int64) ([]*domain.LeadSource, int64, error) {
	bsonFilter := bson.M{}
	if filter != nil {
		if f, ok := filter.(map[string]interface{}); ok {
			for k, v := range f {
				bsonFilter[k] = v
			}
		}
	}

	if tenantID, ok := getTenantIDFromContext(ctx); ok {
		bsonFilter["tenant_id"] = tenantID
	}

	total, err := r.collection.CountDocuments(ctx, bsonFilter)
	if err != nil {
		return nil, 0, err
	}

	findOptions := options.Find().
		SetSkip(offset).
		SetLimit(limit).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, bsonFilter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var sources []*domain.LeadSource
	if err = cursor.All(ctx, &sources); err != nil {
		return nil, 0, err
	}

	return sources, total, nil
}

func (r *MongoLeadSourceRepository) Update(ctx context.Context, source *domain.LeadSource) error {
	filter := bson.M{"_id": source.ID}

	if tenantID, ok := getTenantIDFromContext(ctx); ok {
		filter["tenant_id"] = tenantID
	}

	update := bson.M{
		"$set": bson.M{
			"name":        source.Name,
			"description": source.Description,
			"updated_at":  time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoLeadSourceRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}

	if tenantID, ok := getTenantIDFromContext(ctx); ok {
		filter["tenant_id"] = tenantID
	}

	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

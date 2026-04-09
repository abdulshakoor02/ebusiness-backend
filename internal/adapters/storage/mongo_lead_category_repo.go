package storage

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/pkg/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoLeadCategoryRepository struct {
	collection *mongo.Collection
}

func NewMongoLeadCategoryRepository(db *mongo.Database) *MongoLeadCategoryRepository {
	return &MongoLeadCategoryRepository{
		collection: db.Collection("lead_categories"),
	}
}

func (r *MongoLeadCategoryRepository) Create(ctx context.Context, category *domain.LeadCategory) error {
	_, err := r.collection.InsertOne(ctx, category)
	return err
}

func (r *MongoLeadCategoryRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.LeadCategory, error) {
	filter := bson.M{"_id": id}

	scopeFilter := middleware.GetScopeFilter(ctx)
	if !scopeFilter.IsSystemAdmin {
		if tenantID, ok := getTenantIDFromContext(ctx); ok {
			filter["tenant_id"] = tenantID
		}
	}

	var category domain.LeadCategory
	err := r.collection.FindOne(ctx, filter).Decode(&category)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("lead category not found")
		}
		return nil, err
	}
	return &category, nil
}

func (r *MongoLeadCategoryRepository) FindByName(ctx context.Context, tenantID primitive.ObjectID, name string) (*domain.LeadCategory, error) {
	filter := bson.M{"tenant_id": tenantID, "name": bson.M{"$regex": "^" + name + "$", "$options": "i"}}

	var category domain.LeadCategory
	err := r.collection.FindOne(ctx, filter).Decode(&category)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("lead category not found")
		}
		return nil, err
	}
	return &category, nil
}

func (r *MongoLeadCategoryRepository) List(ctx context.Context, filter interface{}, offset, limit int64) ([]*domain.LeadCategory, int64, error) {
	scopeFilter := middleware.GetScopeFilter(ctx)
	query := bson.M{}

	if !scopeFilter.IsSystemAdmin {
		tenantID, ok := getTenantIDFromContext(ctx)
		if !ok {
			slog.Warn("List lead categories called without tenant context")
			return nil, 0, errors.New("tenant context required")
		}
		query["tenant_id"] = tenantID
	}

	if f, ok := filter.(map[string]interface{}); ok {
		for k, v := range f {
			query[k] = v
		}
	}

	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	findOptions := options.Find()
	findOptions.SetSkip(offset)
	findOptions.SetLimit(limit)
	findOptions.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var categories []*domain.LeadCategory
	if err = cursor.All(ctx, &categories); err != nil {
		return nil, 0, err
	}

	return categories, total, nil
}

func (r *MongoLeadCategoryRepository) Update(ctx context.Context, category *domain.LeadCategory) error {
	filter := bson.M{"_id": category.ID}

	scopeFilter := middleware.GetScopeFilter(ctx)
	if !scopeFilter.IsSystemAdmin {
		if tenantID, ok := getTenantIDFromContext(ctx); ok {
			filter["tenant_id"] = tenantID
		}
	}

	update := bson.M{
		"$set": bson.M{
			"name":        category.Name,
			"description": category.Description,
			"updated_at":  time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoLeadCategoryRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}

	scopeFilter := middleware.GetScopeFilter(ctx)
	if !scopeFilter.IsSystemAdmin {
		if tenantID, ok := getTenantIDFromContext(ctx); ok {
			filter["tenant_id"] = tenantID
		}
	}

	res, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return errors.New("lead category not found or unauthorized")
	}

	return nil
}

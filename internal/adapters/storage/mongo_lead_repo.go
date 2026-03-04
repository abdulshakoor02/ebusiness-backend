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

type MongoLeadRepository struct {
	collection *mongo.Collection
}

func NewMongoLeadRepository(db *mongo.Database) *MongoLeadRepository {
	return &MongoLeadRepository{
		collection: db.Collection("leads"),
	}
}

func (r *MongoLeadRepository) Create(ctx context.Context, lead *domain.Lead) error {
	_, err := r.collection.InsertOne(ctx, lead)
	return err
}

func (r *MongoLeadRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Lead, error) {
	filter := bson.M{"_id": id}

	if tenantID, ok := getTenantIDFromContext(ctx); ok {
		filter["tenant_id"] = tenantID
	}

	var lead domain.Lead
	err := r.collection.FindOne(ctx, filter).Decode(&lead)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("lead not found")
		}
		return nil, err
	}
	return &lead, nil
}

func (r *MongoLeadRepository) List(ctx context.Context, filter interface{}, offset, limit int64) ([]*domain.Lead, int64, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		slog.Warn("List leads called without tenant context")
		return nil, 0, errors.New("tenant context required")
	}

	query := bson.M{"tenant_id": tenantID}

	if f, ok := filter.(map[string]interface{}); ok {
		for k, v := range f {
			query[k] = v
		}
	}

	scopeFilter := middleware.GetScopeFilter(ctx)
	slog.Debug("Lead List - Scope filter", "scope_type", scopeFilter.ScopeType, "self_user_id", scopeFilter.SelfUserID, "filter_field", scopeFilter.FilterField)
	if scopeFilter.ScopeType == "self" && scopeFilter.SelfUserID != "" && scopeFilter.FilterField != "" && !scopeFilter.IsSystemAdmin {
		userOID, err := primitive.ObjectIDFromHex(scopeFilter.SelfUserID)
		if err == nil {
			query[scopeFilter.FilterField] = userOID
			slog.Debug("Lead List - Added scope filter", "filter_field", scopeFilter.FilterField, "user_id", scopeFilter.SelfUserID)
		} else {
			slog.Warn("Lead List - Invalid user ID for scope filter", "user_id", scopeFilter.SelfUserID, "error", err)
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

	var leads []*domain.Lead
	if err = cursor.All(ctx, &leads); err != nil {
		return nil, 0, err
	}

	return leads, total, nil
}

func (r *MongoLeadRepository) Update(ctx context.Context, lead *domain.Lead) error {
	filter := bson.M{"_id": lead.ID}

	if tenantID, ok := getTenantIDFromContext(ctx); ok {
		filter["tenant_id"] = tenantID
	}

	update := bson.M{
		"$set": bson.M{
			"first_name":  lead.FirstName,
			"last_name":   lead.LastName,
			"company":     lead.Company,
			"title":       lead.Title,
			"email":       lead.Email,
			"phone":       lead.Phone,
			"status":      lead.Status,
			"source_id":   lead.SourceID,
			"category_id": lead.CategoryID,
			"assigned_to": lead.AssignedTo,
			"updated_at":  time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

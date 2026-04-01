package storage

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/abdulshakoor02/goCrmBackend/pkg/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoLeadFollowUpRepository struct {
	collection *mongo.Collection
}

func NewMongoLeadFollowUpRepository(db *mongo.Database) *MongoLeadFollowUpRepository {
	return &MongoLeadFollowUpRepository{
		collection: db.Collection("lead_follow_ups"),
	}
}

func (r *MongoLeadFollowUpRepository) Create(ctx context.Context, followUp *domain.LeadFollowUp) error {
	_, err := r.collection.InsertOne(ctx, followUp)
	return err
}

func (r *MongoLeadFollowUpRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.LeadFollowUp, error) {
	filter := bson.M{"_id": id}

	scopeFilter := middleware.GetScopeFilter(ctx)
	if !scopeFilter.IsSystemAdmin {
		if tenantID, ok := getTenantIDFromContext(ctx); ok {
			filter["tenant_id"] = tenantID
		}
	}

	var followUp domain.LeadFollowUp
	err := r.collection.FindOne(ctx, filter).Decode(&followUp)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("lead follow-up not found")
		}
		return nil, err
	}
	return &followUp, nil
}

func (r *MongoLeadFollowUpRepository) ListByLeadID(ctx context.Context, leadID primitive.ObjectID, filter interface{}, offset, limit int64) ([]*ports.FollowUpListItem, int64, error) {
	scopeFilter := middleware.GetScopeFilter(ctx)
	query := bson.M{
		"lead_id": leadID,
	}

	if !scopeFilter.IsSystemAdmin {
		tenantID, ok := getTenantIDFromContext(ctx)
		if !ok {
			slog.Warn("List lead follow-ups called without tenant context")
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

	// Build aggregation pipeline: match → sort → skip → limit → lookup creator
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: query}},
		{{Key: "$sort", Value: bson.D{{Key: "start_time", Value: 1}}}},
		{{Key: "$skip", Value: offset}},
		{{Key: "$limit", Value: limit}},

		// Lookup creator from users collection
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "creator_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "creator"},
		}}},
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$creator"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var followUps []*ports.FollowUpListItem
	if err = cursor.All(ctx, &followUps); err != nil {
		return nil, 0, err
	}

	return followUps, total, nil
}

func (r *MongoLeadFollowUpRepository) Update(ctx context.Context, followUp *domain.LeadFollowUp) error {
	filter := bson.M{"_id": followUp.ID}

	scopeFilter := middleware.GetScopeFilter(ctx)
	if !scopeFilter.IsSystemAdmin {
		if tenantID, ok := getTenantIDFromContext(ctx); ok {
			filter["tenant_id"] = tenantID
		}
	}

	update := bson.M{
		"$set": bson.M{
			"title":       followUp.Title,
			"description": followUp.Description,
			"start_time":  followUp.StartTime,
			"end_time":    followUp.EndTime,
			"status":      followUp.Status,
			"updated_at":  time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoLeadFollowUpRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
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
		return errors.New("lead follow-up not found or unauthorized")
	}

	return nil
}

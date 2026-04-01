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
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoLeadCommentRepository struct {
	collection *mongo.Collection
}

func NewMongoLeadCommentRepository(db *mongo.Database) *MongoLeadCommentRepository {
	return &MongoLeadCommentRepository{
		collection: db.Collection("lead_comments"),
	}
}

func (r *MongoLeadCommentRepository) Create(ctx context.Context, comment *domain.LeadComment) error {
	_, err := r.collection.InsertOne(ctx, comment)
	return err
}

func (r *MongoLeadCommentRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.LeadComment, error) {
	filter := bson.M{"_id": id}

	scopeFilter := middleware.GetScopeFilter(ctx)
	if !scopeFilter.IsSystemAdmin {
		if tenantID, ok := getTenantIDFromContext(ctx); ok {
			filter["tenant_id"] = tenantID
		}
	}

	var comment domain.LeadComment
	err := r.collection.FindOne(ctx, filter).Decode(&comment)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("lead comment not found")
		}
		return nil, err
	}
	return &comment, nil
}

func (r *MongoLeadCommentRepository) GetLatestByLeadID(ctx context.Context, leadID primitive.ObjectID) (*domain.LeadComment, error) {
	filter := bson.M{"lead_id": leadID}

	scopeFilter := middleware.GetScopeFilter(ctx)
	if !scopeFilter.IsSystemAdmin {
		if tenantID, ok := getTenantIDFromContext(ctx); ok {
			filter["tenant_id"] = tenantID
		}
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "updated_at", Value: -1}})

	var comment domain.LeadComment
	err := r.collection.FindOne(ctx, filter, opts).Decode(&comment)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &comment, nil
}

func (r *MongoLeadCommentRepository) ListByLeadID(ctx context.Context, leadID primitive.ObjectID, filter interface{}, offset, limit int64) ([]*ports.CommentListItem, int64, error) {
	scopeFilter := middleware.GetScopeFilter(ctx)
	query := bson.M{
		"lead_id": leadID,
	}

	if !scopeFilter.IsSystemAdmin {
		tenantID, ok := getTenantIDFromContext(ctx)
		if !ok {
			slog.Warn("List lead comments called without tenant context")
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

	// Build aggregation pipeline: match → sort → skip → limit → lookup author
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: query}},
		{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: 1}}}},
		{{Key: "$skip", Value: offset}},
		{{Key: "$limit", Value: limit}},

		// Lookup author from users collection
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "author_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "author"},
		}}},
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$author"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var comments []*ports.CommentListItem
	if err = cursor.All(ctx, &comments); err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

func (r *MongoLeadCommentRepository) Update(ctx context.Context, comment *domain.LeadComment) error {
	filter := bson.M{"_id": comment.ID}

	scopeFilter := middleware.GetScopeFilter(ctx)
	if !scopeFilter.IsSystemAdmin {
		if tenantID, ok := getTenantIDFromContext(ctx); ok {
			filter["tenant_id"] = tenantID
		}
	}

	update := bson.M{
		"$set": bson.M{
			"content":    comment.Content,
			"updated_at": time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoLeadCommentRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
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
		return errors.New("lead comment not found or unauthorized")
	}

	return nil
}

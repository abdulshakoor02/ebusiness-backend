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

type MongoUserRepository struct {
	collection *mongo.Collection
}

func NewMongoUserRepository(db *mongo.Database) *MongoUserRepository {
	return &MongoUserRepository{
		collection: db.Collection("users"),
	}
}

// Helper to extract tenant ID from context
// This assumes middleware has placed tenantID in context
func getTenantIDFromContext(ctx context.Context) (primitive.ObjectID, bool) {
	// Key type should be defined in a shared package to avoid collisions
	// For now using string "tenant_id"
	val := ctx.Value("tenant_id")
	if val == nil {
		return primitive.NilObjectID, false
	}
	if idStr, ok := val.(string); ok {
		id, err := primitive.ObjectIDFromHex(idStr)
		if err == nil {
			return id, true
		}
	}
	if id, ok := val.(primitive.ObjectID); ok {
		return id, true
	}
	return primitive.NilObjectID, false
}

func (r *MongoUserRepository) Create(ctx context.Context, user *domain.User) error {
	// If User struct already has TenantID, we respect it (e.g. during initial setup)
	// Otherwise, we could enforce it from context
	_, err := r.collection.InsertOne(ctx, user)
	return err
}

func (r *MongoUserRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.User, error) {
	// Validation: Ensure the user belongs to the tenant in context, unless context has no tenant (e.g. system admin or login)
	filter := bson.M{"_id": id}

	scopeFilter := middleware.GetScopeFilter(ctx)
	if !scopeFilter.IsSystemAdmin {
		if tenantID, ok := getTenantIDFromContext(ctx); ok {
			filter["tenant_id"] = tenantID
		}
	}

	var user domain.User
	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (r *MongoUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	// Email should be unique per tenant? OR globally unique?
	// Usually email is unique per tenant, but for login we might need to find user by email globally
	// (if email is username).
	// If multi-tenant with shared login, email must be globally unique.
	// If isolated login (subdomains), then per tenant.
	// Assuming globally unique emails for simplicity in this implementation plan.

	filter := bson.M{"email": email}
	// Note: We might NOT want to filter by tenant_id here if this is used for Login where we don't know the tenant yet.
	// However, if specific tenant context is provided, we should respect it.

	scopeFilter := middleware.GetScopeFilter(ctx)
	if !scopeFilter.IsSystemAdmin {
		if tenantID, ok := getTenantIDFromContext(ctx); ok {
			filter["tenant_id"] = tenantID
		}
	}

	var user domain.User
	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (r *MongoUserRepository) List(ctx context.Context, filter interface{}, offset, limit int64) ([]*domain.User, int64, error) {
	scopeFilter := middleware.GetScopeFilter(ctx)
	query := bson.M{}

	if !scopeFilter.IsSystemAdmin {
		tenantID, ok := getTenantIDFromContext(ctx)
		if !ok {
			slog.Warn("List users called without tenant context")
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

	var users []*domain.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *MongoUserRepository) Update(ctx context.Context, user *domain.User) error {
	filter := bson.M{"_id": user.ID}

	scopeFilter := middleware.GetScopeFilter(ctx)
	if !scopeFilter.IsSystemAdmin {
		if tenantID, ok := getTenantIDFromContext(ctx); ok {
			filter["tenant_id"] = tenantID
		}
	}

	update := bson.M{
		"$set": bson.M{
			"name":       user.Name,
			"email":      user.Email,
			"mobile":     user.Mobile,
			"role":       user.Role,
			"updated_at": time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

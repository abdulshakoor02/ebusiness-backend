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

type MongoPermissionRuleRepository struct {
	collection *mongo.Collection
}

func NewMongoPermissionRuleRepository(db *mongo.Database) *MongoPermissionRuleRepository {
	return &MongoPermissionRuleRepository{
		collection: db.Collection("permission_rules"),
	}
}

func (r *MongoPermissionRuleRepository) Create(ctx context.Context, rule *domain.PermissionRule) error {
	_, err := r.collection.InsertOne(ctx, rule)
	return err
}

func (r *MongoPermissionRuleRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.PermissionRule, error) {
	var rule domain.PermissionRule
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&rule)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("permission rule not found")
		}
		return nil, err
	}
	return &rule, nil
}

func (r *MongoPermissionRuleRepository) GetByResourceAndAction(ctx context.Context, resource, action string) (*domain.PermissionRule, error) {
	var rule domain.PermissionRule
	err := r.collection.FindOne(ctx, bson.M{"resource": resource, "action": action}).Decode(&rule)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("permission rule not found")
		}
		return nil, err
	}
	return &rule, nil
}

func (r *MongoPermissionRuleRepository) ListAll(ctx context.Context) ([]*domain.PermissionRule, error) {
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "resource", Value: 1}, {Key: "action", Value: 1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rules []*domain.PermissionRule
	if err = cursor.All(ctx, &rules); err != nil {
		return nil, err
	}

	return rules, nil
}

func (r *MongoPermissionRuleRepository) ListByResource(ctx context.Context, resource string) ([]*domain.PermissionRule, error) {
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "action", Value: 1}})

	cursor, err := r.collection.Find(ctx, bson.M{"resource": resource}, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rules []*domain.PermissionRule
	if err = cursor.All(ctx, &rules); err != nil {
		return nil, err
	}

	return rules, nil
}

func (r *MongoPermissionRuleRepository) Update(ctx context.Context, rule *domain.PermissionRule) error {
	filter := bson.M{"_id": rule.ID}

	update := bson.M{
		"$set": bson.M{
			"resource_label": rule.ResourceLabel,
			"action_label":   rule.ActionLabel,
			"path":           rule.Path,
			"method":         rule.Method,
			"description":    rule.Description,
			"updated_at":     time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoPermissionRuleRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	// First check if it's a system rule
	var rule domain.PermissionRule
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&rule)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("permission rule not found")
		}
		return err
	}

	if rule.IsSystem {
		return errors.New("cannot delete system permission rules")
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

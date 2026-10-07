package storage

import (
	"context"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoFacebookConnectionRepository struct {
	collection *mongo.Collection
}

func NewMongoFacebookConnectionRepository(db *mongo.Database) *MongoFacebookConnectionRepository {
	return &MongoFacebookConnectionRepository{
		collection: db.Collection("facebook_connections"),
	}
}

func (r *MongoFacebookConnectionRepository) Upsert(ctx context.Context, conn *domain.FacebookConnection) error {
	filter := bson.M{"tenant_id": conn.TenantID}
	update := bson.M{"$set": conn}
	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *MongoFacebookConnectionRepository) GetByTenantID(ctx context.Context, tenantID primitive.ObjectID) (*domain.FacebookConnection, error) {
	var conn domain.FacebookConnection
	err := r.collection.FindOne(ctx, bson.M{"tenant_id": tenantID}).Decode(&conn)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &conn, nil
}

func (r *MongoFacebookConnectionRepository) ListAll(ctx context.Context) ([]*domain.FacebookConnection, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var out []*domain.FacebookConnection
	if err := cursor.All(ctx, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []*domain.FacebookConnection{}
	}
	return out, nil
}

func (r *MongoFacebookConnectionRepository) DeleteByTenantID(ctx context.Context, tenantID primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"tenant_id": tenantID})
	return err
}

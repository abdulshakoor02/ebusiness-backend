package storage

import (
	"context"
	"errors"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoProductRepository struct {
	collection *mongo.Collection
}

func NewMongoProductRepository(db *mongo.Database) *MongoProductRepository {
	return &MongoProductRepository{
		collection: db.Collection("products"),
	}
}

func (r *MongoProductRepository) Create(ctx context.Context, product *domain.Product) error {
	_, err := r.collection.InsertOne(ctx, product)
	return err
}

func (r *MongoProductRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Product, error) {
	filter := bson.M{"_id": id}

	var product domain.Product
	err := r.collection.FindOne(ctx, filter).Decode(&product)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}
	return &product, nil
}

func (r *MongoProductRepository) GetByIDAndTenant(ctx context.Context, id, tenantID primitive.ObjectID) (*domain.Product, error) {
	filter := bson.M{"_id": id, "tenant_id": tenantID}

	var product domain.Product
	err := r.collection.FindOne(ctx, filter).Decode(&product)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}
	return &product, nil
}

func (r *MongoProductRepository) List(ctx context.Context, filter interface{}, offset, limit int64) ([]*domain.Product, int64, error) {
	query := bson.M{}

	if f, ok := filter.(map[string]interface{}); ok {
		for k, v := range f {
			query[k] = v
		}
	}

	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	cursor, err := r.collection.Find(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var products []*domain.Product
	if err = cursor.All(ctx, &products); err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

func (r *MongoProductRepository) Update(ctx context.Context, product *domain.Product) error {
	filter := bson.M{"_id": product.ID}
	update := bson.M{
		"$set": bson.M{
			"name":        product.Name,
			"description": product.Description,
			"price":       product.Price,
			"updated_at":  time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoProductRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

var _ ports.ProductRepository = (*MongoProductRepository)(nil)

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

type MongoCountryRepository struct {
	collection *mongo.Collection
}

func NewMongoCountryRepository(db *mongo.Database) *MongoCountryRepository {
	return &MongoCountryRepository{
		collection: db.Collection("countries"),
	}
}

func (r *MongoCountryRepository) Create(ctx context.Context, country *domain.Country) error {
	_, err := r.collection.InsertOne(ctx, country)
	return err
}

func (r *MongoCountryRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Country, error) {
	filter := bson.M{"_id": id}

	var country domain.Country
	err := r.collection.FindOne(ctx, filter).Decode(&country)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("country not found")
		}
		return nil, err
	}
	return &country, nil
}

func (r *MongoCountryRepository) List(ctx context.Context, filter interface{}, offset, limit int64) ([]*domain.Country, int64, error) {
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

	findOptions := options.Find()
	findOptions.SetSkip(offset)
	findOptions.SetLimit(limit)
	findOptions.SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := r.collection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var countries []*domain.Country
	if err = cursor.All(ctx, &countries); err != nil {
		return nil, 0, err
	}

	return countries, total, nil
}

func (r *MongoCountryRepository) Update(ctx context.Context, country *domain.Country) error {
	filter := bson.M{"_id": country.ID}

	update := bson.M{
		"$set": bson.M{
			"name":          country.Name,
			"iso2":          country.ISO2,
			"iso3":          country.ISO3,
			"phone_code":    country.PhoneCode,
			"currency":      country.Currency,
			"currency_name": country.CurrencyName,
			"is_active":     country.IsActive,
			"updated_at":    time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoCountryRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}

	res, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return errors.New("country not found")
	}

	return nil
}

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

type MongoQualificationRepository struct {
	collection *mongo.Collection
}

func NewMongoQualificationRepository(db *mongo.Database) *MongoQualificationRepository {
	return &MongoQualificationRepository{
		collection: db.Collection("qualifications"),
	}
}

func (r *MongoQualificationRepository) Create(ctx context.Context, qualification *domain.Qualification) error {
	_, err := r.collection.InsertOne(ctx, qualification)
	return err
}

func (r *MongoQualificationRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Qualification, error) {
	filter := bson.M{"_id": id}

	var qualification domain.Qualification
	err := r.collection.FindOne(ctx, filter).Decode(&qualification)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("qualification not found")
		}
		return nil, err
	}
	return &qualification, nil
}

func (r *MongoQualificationRepository) FindByName(ctx context.Context, name string) (*domain.Qualification, error) {
	filter := bson.M{"name": bson.M{"$regex": "^" + name + "$", "$options": "i"}}

	var qualification domain.Qualification
	err := r.collection.FindOne(ctx, filter).Decode(&qualification)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("qualification not found")
		}
		return nil, err
	}
	return &qualification, nil
}

func (r *MongoQualificationRepository) List(ctx context.Context, filter interface{}, offset, limit int64) ([]*domain.Qualification, int64, error) {
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
	findOptions.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var qualifications []*domain.Qualification
	if err = cursor.All(ctx, &qualifications); err != nil {
		return nil, 0, err
	}

	return qualifications, total, nil
}

func (r *MongoQualificationRepository) Update(ctx context.Context, qualification *domain.Qualification) error {
	filter := bson.M{"_id": qualification.ID}

	update := bson.M{
		"$set": bson.M{
			"name":       qualification.Name,
			"is_active":  qualification.IsActive,
			"updated_at": time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoQualificationRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}

	res, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return errors.New("qualification not found")
	}

	return nil
}

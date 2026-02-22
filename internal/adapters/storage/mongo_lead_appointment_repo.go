package storage

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoLeadAppointmentRepository struct {
	collection *mongo.Collection
}

func NewMongoLeadAppointmentRepository(db *mongo.Database) *MongoLeadAppointmentRepository {
	return &MongoLeadAppointmentRepository{
		collection: db.Collection("lead_appointments"),
	}
}

func (r *MongoLeadAppointmentRepository) Create(ctx context.Context, appointment *domain.LeadAppointment) error {
	_, err := r.collection.InsertOne(ctx, appointment)
	return err
}

func (r *MongoLeadAppointmentRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.LeadAppointment, error) {
	filter := bson.M{"_id": id}

	if tenantID, ok := getTenantIDFromContext(ctx); ok {
		filter["tenant_id"] = tenantID
	}

	var appointment domain.LeadAppointment
	err := r.collection.FindOne(ctx, filter).Decode(&appointment)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("lead appointment not found")
		}
		return nil, err
	}
	return &appointment, nil
}

func (r *MongoLeadAppointmentRepository) ListByLeadID(ctx context.Context, leadID primitive.ObjectID, filter interface{}, offset, limit int64) ([]*domain.LeadAppointment, int64, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		slog.Warn("List lead appointments called without tenant context")
		return nil, 0, errors.New("tenant context required")
	}

	query := bson.M{
		"tenant_id": tenantID,
		"lead_id":   leadID,
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
	// Sort by closest start time ascending
	findOptions.SetSort(bson.D{{Key: "start_time", Value: 1}})

	cursor, err := r.collection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var appointments []*domain.LeadAppointment
	if err = cursor.All(ctx, &appointments); err != nil {
		return nil, 0, err
	}

	return appointments, total, nil
}

func (r *MongoLeadAppointmentRepository) Update(ctx context.Context, appointment *domain.LeadAppointment) error {
	filter := bson.M{"_id": appointment.ID}

	if tenantID, ok := getTenantIDFromContext(ctx); ok {
		filter["tenant_id"] = tenantID
	}

	update := bson.M{
		"$set": bson.M{
			"title":       appointment.Title,
			"description": appointment.Description,
			"start_time":  appointment.StartTime,
			"end_time":    appointment.EndTime,
			"status":      appointment.Status,
			"updated_at":  time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoLeadAppointmentRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}

	if tenantID, ok := getTenantIDFromContext(ctx); ok {
		filter["tenant_id"] = tenantID
	}

	res, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return errors.New("lead appointment not found or unauthorized")
	}

	return nil
}

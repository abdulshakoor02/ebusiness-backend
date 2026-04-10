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

	scopeFilter := middleware.GetScopeFilter(ctx)
	if !scopeFilter.IsSystemAdmin {
		if tenantID, ok := getTenantIDFromContext(ctx); ok {
			filter["tenant_id"] = tenantID
		}
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

func (r *MongoLeadAppointmentRepository) ListByLeadID(ctx context.Context, leadID primitive.ObjectID, filter interface{}, offset, limit int64) ([]*ports.AppointmentListItem, int64, error) {
	scopeFilter := middleware.GetScopeFilter(ctx)
	query := bson.M{
		"lead_id": leadID,
	}

	if !scopeFilter.IsSystemAdmin {
		tenantID, ok := getTenantIDFromContext(ctx)
		if !ok {
			slog.Warn("List lead appointments called without tenant context")
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

	// Build aggregation pipeline: match → sort → skip → limit → lookup organizer
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: query}},
		{{Key: "$sort", Value: bson.D{{Key: "start_time", Value: 1}}}},
		{{Key: "$skip", Value: offset}},
		{{Key: "$limit", Value: limit}},

		// Lookup organizer from users collection
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "organizer_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "organizer"},
		}}},
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$organizer"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var appointments []*ports.AppointmentListItem
	if err = cursor.All(ctx, &appointments); err != nil {
		return nil, 0, err
	}

	return appointments, total, nil
}

func (r *MongoLeadAppointmentRepository) Update(ctx context.Context, appointment *domain.LeadAppointment) error {
	filter := bson.M{"_id": appointment.ID}

	scopeFilter := middleware.GetScopeFilter(ctx)
	if !scopeFilter.IsSystemAdmin {
		if tenantID, ok := getTenantIDFromContext(ctx); ok {
			filter["tenant_id"] = tenantID
		}
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
		return errors.New("lead appointment not found or unauthorized")
	}

	return nil
}

func (r *MongoLeadAppointmentRepository) CountByDateRange(ctx context.Context, startDate, endDate time.Time) (map[string]int64, error) {
	scopeFilter := middleware.GetScopeFilter(ctx)
	match := bson.M{
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	if !scopeFilter.IsSystemAdmin {
		tenantID, ok := getTenantIDFromContext(ctx)
		if !ok {
			return nil, errors.New("tenant context required")
		}
		match["tenant_id"] = tenantID
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{
			{Key: "$group", Value: bson.D{
				{Key: "_id", Value: bson.M{
					"$dateToString": bson.M{
						"format": "%Y-%m-%d",
						"date":   "$created_at",
					},
				}},
				{Key: "count", Value: bson.M{"$sum": 1}},
			}},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make(map[string]int64)
	for cursor.Next(ctx) {
		var doc struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		result[doc.ID] = doc.Count
	}

	return result, nil
}

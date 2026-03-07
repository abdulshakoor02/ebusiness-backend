package storage

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/abdulshakoor02/goCrmBackend/pkg/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoLeadRepository struct {
	collection *mongo.Collection
}

func NewMongoLeadRepository(db *mongo.Database) *MongoLeadRepository {
	return &MongoLeadRepository{
		collection: db.Collection("leads"),
	}
}

func (r *MongoLeadRepository) Create(ctx context.Context, lead *domain.Lead) error {
	_, err := r.collection.InsertOne(ctx, lead)
	return err
}

func (r *MongoLeadRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Lead, error) {
	filter := bson.M{"_id": id}

	scopeFilter := middleware.GetScopeFilter(ctx)
	if !scopeFilter.IsSystemAdmin {
		if tenantID, ok := getTenantIDFromContext(ctx); ok {
			filter["tenant_id"] = tenantID
		}
	}

	var lead domain.Lead
	err := r.collection.FindOne(ctx, filter).Decode(&lead)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("lead not found")
		}
		return nil, err
	}
	return &lead, nil
}

func (r *MongoLeadRepository) List(ctx context.Context, filter interface{}, search string, offset, limit int64) ([]*ports.LeadListItem, int64, error) {
	scopeFilter := middleware.GetScopeFilter(ctx)
	query := bson.M{}

	if !scopeFilter.IsSystemAdmin {
		tenantID, ok := getTenantIDFromContext(ctx)
		if !ok {
			slog.Warn("List leads called without tenant context")
			return nil, 0, errors.New("tenant context required")
		}
		query["tenant_id"] = tenantID
	}

	// Known fields that store ObjectIDs — string values must be converted
	objectIDFields := map[string]bool{
		"category_id":      true,
		"source_id":        true,
		"assigned_to":      true,
		"country_id":       true,
		"qualification_id": true,
	}

	if f, ok := filter.(map[string]interface{}); ok {
		for k, v := range f {
			if objectIDFields[k] {
				if strVal, ok := v.(string); ok && strVal != "" {
					oid, err := primitive.ObjectIDFromHex(strVal)
					if err == nil {
						query[k] = oid
						continue
					}
				}
			}
			query[k] = v
		}
	}

	// Apply search on the derived search_text field
	search = strings.TrimSpace(search)
	if search != "" {
		escaped := regexp.QuoteMeta(strings.ToLower(search))
		query["search_text"] = primitive.Regex{Pattern: escaped, Options: ""}
	}

	slog.Debug("Lead List - Scope filter", "scope_type", scopeFilter.ScopeType, "self_user_id", scopeFilter.SelfUserID, "filter_field", scopeFilter.FilterField)
	if scopeFilter.ScopeType == "self" && scopeFilter.SelfUserID != "" && scopeFilter.FilterField != "" && !scopeFilter.IsSystemAdmin {
		userOID, err := primitive.ObjectIDFromHex(scopeFilter.SelfUserID)
		if err == nil {
			query[scopeFilter.FilterField] = userOID
			slog.Debug("Lead List - Added scope filter", "filter_field", scopeFilter.FilterField, "user_id", scopeFilter.SelfUserID)
		} else {
			slog.Warn("Lead List - Invalid user ID for scope filter", "user_id", scopeFilter.SelfUserID, "error", err)
		}
	}

	// Count total matching documents
	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	// Build aggregation pipeline: match → sort → skip → limit → lookups → unwinds
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: query}},
		{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: -1}}}},
		{{Key: "$skip", Value: offset}},
		{{Key: "$limit", Value: limit}},

		// Lookup category
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "lead_categories"},
			{Key: "localField", Value: "category_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "category"},
		}}},
		// Lookup source
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "lead_sources"},
			{Key: "localField", Value: "source_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "source"},
		}}},
		// Lookup assigned user
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "assigned_to"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "assigned_to_user"},
		}}},
		// Lookup country
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "countries"},
			{Key: "localField", Value: "country_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "country"},
		}}},
		// Lookup qualification
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "qualifications"},
			{Key: "localField", Value: "qualification_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "qualification"},
		}}},

		// Unwind all lookups (preserveNullAndEmptyArrays for optional fields)
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$category"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$source"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$assigned_to_user"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$country"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$qualification"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var leads []*ports.LeadListItem
	if err = cursor.All(ctx, &leads); err != nil {
		return nil, 0, err
	}

	return leads, total, nil
}

func (r *MongoLeadRepository) Update(ctx context.Context, lead *domain.Lead) error {
	filter := bson.M{"_id": lead.ID}

	scopeFilter := middleware.GetScopeFilter(ctx)
	if !scopeFilter.IsSystemAdmin {
		if tenantID, ok := getTenantIDFromContext(ctx); ok {
			filter["tenant_id"] = tenantID
		}
	}

	update := bson.M{
		"$set": bson.M{
			"first_name":       lead.FirstName,
			"last_name":        lead.LastName,
			"designation":      lead.Designation,
			"email":            lead.Email,
			"phone":            lead.Phone,
			"source_id":        lead.SourceID,
			"category_id":      lead.CategoryID,
			"assigned_to":      lead.AssignedTo,
			"country_id":       lead.CountryID,
			"qualification_id": lead.QualificationID,
			"status":           lead.Status,
			"converted_at":     lead.ConvertedAt,
			"search_text":      lead.SearchText,
			"updated_at":       time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

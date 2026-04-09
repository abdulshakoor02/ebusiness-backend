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
	"go.mongodb.org/mongo-driver/mongo/options"
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

	// Apply scope filter (e.g., assigned_to = self)
	if scopeFilter.ScopeType == "self" && scopeFilter.SelfUserID != "" && scopeFilter.FilterField != "" && !scopeFilter.IsSystemAdmin {
		userOID, err := primitive.ObjectIDFromHex(scopeFilter.SelfUserID)
		if err == nil {
			filter[scopeFilter.FilterField] = userOID
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

	// Date range filter parameters (not document fields)
	dateRangeFields := map[string]bool{
		"date_from":  true,
		"date_to":    true,
		"date_field": true,
	}

	// Variables for date range filtering
	var dateFrom, dateTo time.Time
	var dateField string = "created_at" // default to created_at

	if f, ok := filter.(map[string]interface{}); ok {
		for k, v := range f {
			// Skip date range parameters - they need special handling
			if dateRangeFields[k] {
				switch k {
				case "date_from":
					if strVal, ok := v.(string); ok && strVal != "" {
						if parsed, err := time.Parse(time.RFC3339, strVal); err == nil {
							dateFrom = parsed
						}
					}
				case "date_to":
					if strVal, ok := v.(string); ok && strVal != "" {
						if parsed, err := time.Parse(time.RFC3339, strVal); err == nil {
							dateTo = parsed
						}
					}
				case "date_field":
					if strVal, ok := v.(string); ok && strVal != "" {
						dateField = strVal
					}
				}
				continue
			}

			// Handle ObjectID fields
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

	// Apply date range filter if provided
	if !dateFrom.IsZero() || !dateTo.IsZero() {
		dateQuery := bson.M{}
		if !dateFrom.IsZero() {
			dateQuery["$gte"] = dateFrom
		}
		if !dateTo.IsZero() {
			dateQuery["$lte"] = dateTo
		}
		query[dateField] = dateQuery
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

	// Apply scope filter (e.g., assigned_to = self)
	if scopeFilter.ScopeType == "self" && scopeFilter.SelfUserID != "" && scopeFilter.FilterField != "" && !scopeFilter.IsSystemAdmin {
		userOID, err := primitive.ObjectIDFromHex(scopeFilter.SelfUserID)
		if err == nil {
			filter[scopeFilter.FilterField] = userOID
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

func (r *MongoLeadRepository) UpdateComments(ctx context.Context, leadID primitive.ObjectID, comments string) error {
	filter := bson.M{"_id": leadID}

	scopeFilter := middleware.GetScopeFilter(ctx)
	if !scopeFilter.IsSystemAdmin {
		if tenantID, ok := getTenantIDFromContext(ctx); ok {
			filter["tenant_id"] = tenantID
		}
	}

	// Apply scope filter (e.g., assigned_to = self)
	if scopeFilter.ScopeType == "self" && scopeFilter.SelfUserID != "" && scopeFilter.FilterField != "" && !scopeFilter.IsSystemAdmin {
		userOID, err := primitive.ObjectIDFromHex(scopeFilter.SelfUserID)
		if err == nil {
			filter[scopeFilter.FilterField] = userOID
		}
	}

	update := bson.M{
		"$set": bson.M{
			"comments":   comments,
			"updated_at": time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoLeadRepository) BulkInsert(ctx context.Context, leads []*domain.Lead) (int, error) {
	docs := make([]interface{}, len(leads))
	for i, lead := range leads {
		docs[i] = lead
	}

	opts := options.InsertMany().SetOrdered(false)
	result, err := r.collection.InsertMany(ctx, docs, opts)
	if err != nil {
		return 0, err
	}

	return len(result.InsertedIDs), nil
}

func (r *MongoLeadRepository) FindByEmailOrPhone(ctx context.Context, tenantID primitive.ObjectID, email, phone string) (*domain.Lead, error) {
	var filter bson.M

	if email != "" && phone != "" {
		filter = bson.M{
			"tenant_id": tenantID,
			"$or": []bson.M{
				{"email": email},
				{"phone": phone},
			},
		}
	} else if email != "" {
		filter = bson.M{"tenant_id": tenantID, "email": email}
	} else if phone != "" {
		filter = bson.M{"tenant_id": tenantID, "phone": phone}
	} else {
		return nil, errors.New("email or phone required for lookup")
	}

	var lead domain.Lead
	err := r.collection.FindOne(ctx, filter).Decode(&lead)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &lead, nil
}

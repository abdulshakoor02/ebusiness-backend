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

type MongoInvoiceRepository struct {
	collection       *mongo.Collection
	tenantCollection *mongo.Collection
}

func NewMongoInvoiceRepository(db *mongo.Database) *MongoInvoiceRepository {
	return &MongoInvoiceRepository{
		collection:       db.Collection("invoices"),
		tenantCollection: db.Collection("tenants"),
	}
}

func (r *MongoInvoiceRepository) Create(ctx context.Context, invoice *domain.Invoice) error {
	_, err := r.collection.InsertOne(ctx, invoice)
	return err
}

func (r *MongoInvoiceRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Invoice, error) {
	filter := bson.M{"_id": id}

	var invoice domain.Invoice
	err := r.collection.FindOne(ctx, filter).Decode(&invoice)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("invoice not found")
		}
		return nil, err
	}
	return &invoice, nil
}

func (r *MongoInvoiceRepository) GetByIDAndTenant(ctx context.Context, id, tenantID primitive.ObjectID) (*domain.Invoice, error) {
	filter := bson.M{"_id": id, "tenant_id": tenantID}

	var invoice domain.Invoice
	err := r.collection.FindOne(ctx, filter).Decode(&invoice)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("invoice not found")
		}
		return nil, err
	}
	return &invoice, nil
}

func (r *MongoInvoiceRepository) GetByLeadID(ctx context.Context, leadID primitive.ObjectID) ([]*domain.Invoice, error) {
	filter := bson.M{"lead_id": leadID}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var invoices []*domain.Invoice
	if err = cursor.All(ctx, &invoices); err != nil {
		return nil, err
	}

	return invoices, nil
}

func (r *MongoInvoiceRepository) List(ctx context.Context, filter interface{}, offset, limit int64) ([]*domain.Invoice, int64, error) {
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

	var invoices []*domain.Invoice
	if err = cursor.All(ctx, &invoices); err != nil {
		return nil, 0, err
	}

	return invoices, total, nil
}

func (r *MongoInvoiceRepository) Update(ctx context.Context, invoice *domain.Invoice) error {
	filter := bson.M{"_id": invoice.ID}
	update := bson.M{
		"$set": bson.M{
			"items":           invoice.Items,
			"subtotal":        invoice.Subtotal,
			"discount":        invoice.Discount,
			"tax_percentage":  invoice.TaxPercentage,
			"tax_amount":      invoice.TaxAmount,
			"total_amount":    invoice.TotalAmount,
			"paid_amount":     invoice.PaidAmount,
			"paid_amount_vat": invoice.PaidAmountVat,
			"due_date":        invoice.DueDate,
			"status":          invoice.Status,
			"updated_at":      invoice.UpdatedAt,
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoInvoiceRepository) IncrementInvoiceNumber(ctx context.Context, tenantID primitive.ObjectID) (int64, error) {
	filter := bson.M{"_id": tenantID}
	update := bson.M{
		"$inc": bson.M{"next_invoice_number": 1},
	}

	result := r.tenantCollection.FindOneAndUpdate(ctx, filter, update)
	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return 1, nil
		}
		return 0, result.Err()
	}

	var tenant domain.Tenant
	if err := r.tenantCollection.FindOne(ctx, bson.M{"_id": tenantID}).Decode(&tenant); err != nil {
		return 1, nil
	}

	return tenant.NextInvoiceNumber, nil
}

// AggregateByDateRange returns invoice counts by status and total amounts by status for a tenant within a date range.
func (r *MongoInvoiceRepository) AggregateByDateRange(ctx context.Context, tenantID primitive.ObjectID, startDate, endDate time.Time) (map[string]int64, map[string]float64, error) {
	match := bson.M{
		"tenant_id":  tenantID,
		"created_at": bson.M{"$gte": startDate, "$lte": endDate},
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$status"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "total_amount", Value: bson.D{{Key: "$sum", Value: "$total_amount"}}},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close(ctx)

	countByStatus := make(map[string]int64)
	amountByStatus := make(map[string]float64)

	for cursor.Next(ctx) {
		var result struct {
			ID          string  `bson:"_id"`
			Count       int64   `bson:"count"`
			TotalAmount float64 `bson:"total_amount"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, nil, err
		}
		countByStatus[result.ID] = result.Count
		amountByStatus[result.ID] = result.TotalAmount
	}

	return countByStatus, amountByStatus, nil
}

var _ ports.InvoiceRepository = (*MongoInvoiceRepository)(nil)

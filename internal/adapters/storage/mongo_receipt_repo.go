package storage

import (
	"context"
	"errors"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoReceiptRepository struct {
	collection       *mongo.Collection
	tenantCollection *mongo.Collection
}

func NewMongoReceiptRepository(db *mongo.Database) *MongoReceiptRepository {
	return &MongoReceiptRepository{
		collection:       db.Collection("receipts"),
		tenantCollection: db.Collection("tenants"),
	}
}

func (r *MongoReceiptRepository) Create(ctx context.Context, receipt *domain.Receipt) error {
	_, err := r.collection.InsertOne(ctx, receipt)
	return err
}

func (r *MongoReceiptRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Receipt, error) {
	filter := bson.M{"_id": id}

	var receipt domain.Receipt
	err := r.collection.FindOne(ctx, filter).Decode(&receipt)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("receipt not found")
		}
		return nil, err
	}
	return &receipt, nil
}

func (r *MongoReceiptRepository) GetByIDAndTenant(ctx context.Context, id, tenantID primitive.ObjectID) (*domain.Receipt, error) {
	filter := bson.M{"_id": id, "tenant_id": tenantID}

	var receipt domain.Receipt
	err := r.collection.FindOne(ctx, filter).Decode(&receipt)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("receipt not found")
		}
		return nil, err
	}
	return &receipt, nil
}

func (r *MongoReceiptRepository) GetByInvoiceID(ctx context.Context, invoiceID primitive.ObjectID) ([]*domain.Receipt, error) {
	filter := bson.M{"invoice_id": invoiceID}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var receipts []*domain.Receipt
	if err = cursor.All(ctx, &receipts); err != nil {
		return nil, err
	}

	return receipts, nil
}

func (r *MongoReceiptRepository) SumPaidAmountByInvoiceID(ctx context.Context, invoiceID primitive.ObjectID) (float64, float64, error) {
	filter := bson.M{"invoice_id": invoiceID}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total_amount_paid", Value: bson.D{{Key: "$sum", Value: "$amount_paid"}}},
			{Key: "total_with_vat", Value: bson.D{{Key: "$sum", Value: "$total_paid"}}},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, 0, err
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var result struct {
			TotalAmountPaid float64 `bson:"total_amount_paid"`
			TotalWithVat    float64 `bson:"total_with_vat"`
		}
		if err := cursor.Decode(&result); err != nil {
			return 0, 0, err
		}
		return result.TotalAmountPaid, result.TotalWithVat, nil
	}

	return 0, 0, nil
}

func (r *MongoReceiptRepository) IncrementReceiptNumber(ctx context.Context, tenantID primitive.ObjectID) (int64, error) {
	filter := bson.M{"_id": tenantID}
	update := bson.M{
		"$inc": bson.M{"next_receipt_number": 1},
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

	return tenant.NextReceiptNumber, nil
}

var _ ports.ReceiptRepository = (*MongoReceiptRepository)(nil)

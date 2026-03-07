package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Receipt struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID      primitive.ObjectID `bson:"tenant_id" json:"tenant_id"`
	InvoiceID     primitive.ObjectID `bson:"invoice_id" json:"invoice_id"`
	ReceiptNumber int64              `bson:"receipt_number" json:"receipt_number"`
	AmountPaid    float64            `bson:"amount_paid" json:"amount_paid"`
	TaxAmount     float64            `bson:"tax_amount" json:"tax_amount"`
	TotalPaid     float64            `bson:"total_paid" json:"total_paid"`
	PaymentDate   time.Time          `bson:"payment_date" json:"payment_date"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
}

func NewReceipt(tenantID, invoiceID primitive.ObjectID, receiptNumber, amountPaid float64, taxPercentage float64, paymentDate time.Time) *Receipt {
	taxAmount := amountPaid * (taxPercentage / 100)
	totalPaid := amountPaid + taxAmount

	return &Receipt{
		ID:            primitive.NewObjectID(),
		TenantID:      tenantID,
		InvoiceID:     invoiceID,
		ReceiptNumber: int64(receiptNumber),
		AmountPaid:    amountPaid,
		TaxAmount:     taxAmount,
		TotalPaid:     totalPaid,
		PaymentDate:   paymentDate,
		CreatedAt:     time.Now(),
	}
}

package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InvoiceItem struct {
	ProductID   primitive.ObjectID `bson:"product_id" json:"product_id"`
	ProductName string             `bson:"product_name" json:"product_name"`
	Quantity    int                `bson:"quantity" json:"quantity"`
	UnitPrice   float64            `bson:"unit_price" json:"unit_price"`
	Total       float64            `bson:"total" json:"total"`
}

type Invoice struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID      primitive.ObjectID `bson:"tenant_id" json:"tenant_id"`
	LeadID        primitive.ObjectID `bson:"lead_id" json:"lead_id"`
	InvoiceNumber int64              `bson:"invoice_number" json:"invoice_number"`
	Items         []InvoiceItem      `bson:"items" json:"items"`
	Subtotal      float64            `bson:"subtotal" json:"subtotal"`
	Discount      float64            `bson:"discount" json:"discount"`
	TaxPercentage float64            `bson:"tax_percentage" json:"tax_percentage"`
	TaxAmount     float64            `bson:"tax_amount" json:"tax_amount"`
	TotalAmount   float64            `bson:"total_amount" json:"total_amount"`
	PaidAmount    float64            `bson:"paid_amount" json:"paid_amount"`
	PaidAmountVat float64            `bson:"paid_amount_vat" json:"paid_amount_vat"`
	DueDate       *time.Time         `bson:"due_date,omitempty" json:"due_date,omitempty"`
	Status        string             `bson:"status" json:"status"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

const (
	InvoiceStatusPending = "pending"
	InvoiceStatusPartial = "partial"
	InvoiceStatusPaid    = "paid"
)

func NewInvoice(tenantID, leadID primitive.ObjectID, invoiceNumber int64, items []InvoiceItem, discount, taxPercentage float64) *Invoice {
	subtotal := calculateSubtotal(items)
	taxableAmount := subtotal - discount
	taxAmount := taxableAmount * (taxPercentage / 100)
	totalAmount := taxableAmount + taxAmount

	return &Invoice{
		ID:            primitive.NewObjectID(),
		TenantID:      tenantID,
		LeadID:        leadID,
		InvoiceNumber: invoiceNumber,
		Items:         items,
		Subtotal:      subtotal,
		Discount:      discount,
		TaxPercentage: taxPercentage,
		TaxAmount:     taxAmount,
		TotalAmount:   totalAmount,
		PaidAmount:    0,
		PaidAmountVat: 0,
		Status:        InvoiceStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func calculateSubtotal(items []InvoiceItem) float64 {
	var subtotal float64
	for _, item := range items {
		subtotal += item.Total
	}
	return subtotal
}

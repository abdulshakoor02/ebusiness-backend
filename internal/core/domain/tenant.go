package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Address struct {
	Street      string `bson:"street" json:"street"`
	AddressLine string `bson:"address_line" json:"address_line"`
	City        string `bson:"city" json:"city"`
	State       string `bson:"state" json:"state"`
	ZipCode     string `bson:"zip_code" json:"zip_code"`
	Country     string `bson:"country" json:"country"`
}

type Tenant struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name              string             `bson:"name" json:"name"`
	Email             string             `bson:"email" json:"email"`
	LogoURL           string             `bson:"logo_url" json:"logo_url"`
	StampURL          string             `bson:"stamp_url" json:"stamp_url"`
	Address           Address            `bson:"address" json:"address"`
	CountryID         primitive.ObjectID `bson:"country_id,omitempty" json:"country_id,omitempty"`
	Tax               float64            `bson:"tax" json:"tax"`
	NextInvoiceNumber int64              `bson:"next_invoice_number" json:"next_invoice_number"`
	NextReceiptNumber int64              `bson:"next_receipt_number" json:"next_receipt_number"`
	CreatedAt         time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt         time.Time          `bson:"updated_at" json:"updated_at"`
}

func NewTenant(name, email string) *Tenant {
	return &Tenant{
		ID:                primitive.NewObjectID(),
		Name:              name,
		Email:             email,
		NextInvoiceNumber: 1,
		NextReceiptNumber: 1,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Country struct {
	ID           primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Name         string              `bson:"name" json:"name"`
	ISO2         string              `bson:"iso2" json:"iso2"`
	ISO3         string              `bson:"iso3" json:"iso3"`
	PhoneCode    string              `bson:"phone_code" json:"phone_code"`
	Currency     string              `bson:"currency" json:"currency"`
	CurrencyName string              `bson:"currency_name" json:"currency_name"`
	IsActive     bool                `bson:"is_active" json:"is_active"`
	CreatedAt    time.Time           `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time           `bson:"updated_at" json:"updated_at"`
	CreatedBy    *primitive.ObjectID `bson:"created_by,omitempty" json:"created_by,omitempty"`
	UpdatedBy    *primitive.ObjectID `bson:"updated_by,omitempty" json:"updated_by,omitempty"`
}

func NewCountry(name, iso2, iso3, phoneCode, currency, currencyName string) *Country {
	return &Country{
		ID:           primitive.NewObjectID(),
		Name:         name,
		ISO2:         iso2,
		ISO3:         iso3,
		PhoneCode:    phoneCode,
		Currency:     currency,
		CurrencyName: currencyName,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

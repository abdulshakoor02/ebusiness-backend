package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Lead struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID   primitive.ObjectID `bson:"tenant_id" json:"tenant_id"`
	CategoryID primitive.ObjectID `bson:"category_id,omitempty" json:"category_id,omitempty"` // nullable/optional
	SourceID   primitive.ObjectID `bson:"source_id,omitempty" json:"source_id,omitempty"`     // nullable/optional
	AssignedTo primitive.ObjectID `bson:"assigned_to,omitempty" json:"assigned_to,omitempty"` // nullable/optional
	FirstName  string             `bson:"first_name" json:"first_name"`
	LastName   string             `bson:"last_name" json:"last_name"`
	Company    string             `bson:"company" json:"company"`
	Title      string             `bson:"title" json:"title"`
	Email      string             `bson:"email" json:"email"`
	Phone      string             `bson:"phone" json:"phone"`
	Status     string             `bson:"status" json:"status"` // e.g., "New", "Contacted", "Qualified", "Lost"
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time          `bson:"updated_at" json:"updated_at"`
}

func NewLead(tenantID primitive.ObjectID, firstName, lastName, company, title, email, phone, status string) *Lead {
	return &Lead{
		ID:        primitive.NewObjectID(),
		TenantID:  tenantID,
		FirstName: firstName,
		LastName:  lastName,
		Company:   company,
		Title:     title,
		Email:     email,
		Phone:     phone,
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

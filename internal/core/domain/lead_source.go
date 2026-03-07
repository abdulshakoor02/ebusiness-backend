package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadSource struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID    primitive.ObjectID `bson:"tenant_id" json:"tenant_id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

func NewLeadSource(tenantID primitive.ObjectID, name, description string) *LeadSource {
	return &LeadSource{
		ID:          primitive.NewObjectID(),
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

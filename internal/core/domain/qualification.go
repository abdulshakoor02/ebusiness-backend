package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Qualification struct {
	ID        primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Name      string              `bson:"name" json:"name"`
	IsActive  bool                `bson:"is_active" json:"is_active"`
	CreatedAt time.Time           `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time           `bson:"updated_at" json:"updated_at"`
	CreatedBy *primitive.ObjectID `bson:"created_by,omitempty" json:"created_by,omitempty"`
	UpdatedBy *primitive.ObjectID `bson:"updated_by,omitempty" json:"updated_by,omitempty"`
}

func NewQualification(name string) *Qualification {
	return &Qualification{
		ID:        primitive.NewObjectID(),
		Name:      name,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

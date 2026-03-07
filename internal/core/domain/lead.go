package domain

import (
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	LeadStatusLead     = "lead"
	LeadStatusActive   = "active"
	LeadStatusInactive = "inactive"
)

type Lead struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID        primitive.ObjectID `bson:"tenant_id" json:"tenant_id"`
	CategoryID      primitive.ObjectID `bson:"category_id,omitempty" json:"category_id,omitempty"`
	SourceID        primitive.ObjectID `bson:"source_id,omitempty" json:"source_id,omitempty"`
	AssignedTo      primitive.ObjectID `bson:"assigned_to,omitempty" json:"assigned_to,omitempty"`
	CountryID       primitive.ObjectID `bson:"country_id,omitempty" json:"country_id,omitempty"`
	QualificationID primitive.ObjectID `bson:"qualification_id,omitempty" json:"qualification_id,omitempty"`
	FirstName       string             `bson:"first_name" json:"first_name"`
	LastName        string             `bson:"last_name" json:"last_name"`
	Designation     string             `bson:"designation,omitempty" json:"designation,omitempty"`
	Email           string             `bson:"email" json:"email"`
	Phone           string             `bson:"phone" json:"phone"`
	Status          string             `bson:"status" json:"status"`
	SearchText      string             `bson:"search_text" json:"-"`
	ConvertedAt     *time.Time         `bson:"converted_at,omitempty" json:"converted_at,omitempty"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}

// BuildSearchText builds a lowercase concatenated string of searchable fields
// for efficient single-field regex search.
func (l *Lead) BuildSearchText() {
	l.SearchText = strings.ToLower(l.FirstName + " " + l.LastName + " " + l.Email + " " + l.Phone)
}

func NewLead(tenantID primitive.ObjectID, firstName, lastName, designation, email, phone string) *Lead {
	lead := &Lead{
		ID:          primitive.NewObjectID(),
		TenantID:    tenantID,
		FirstName:   firstName,
		LastName:    lastName,
		Designation: designation,
		Email:       email,
		Phone:       phone,
		Status:      LeadStatusLead,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	lead.BuildSearchText()
	return lead
}

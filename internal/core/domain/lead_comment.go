package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadComment struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID  primitive.ObjectID `bson:"tenant_id" json:"tenant_id"`
	LeadID    primitive.ObjectID `bson:"lead_id" json:"lead_id"`
	AuthorID  primitive.ObjectID `bson:"author_id" json:"author_id"` // user who wrote it
	Content   string             `bson:"content" json:"content"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

func NewLeadComment(tenantID, leadID, authorID primitive.ObjectID, content string) *LeadComment {
	return &LeadComment{
		ID:        primitive.NewObjectID(),
		TenantID:  tenantID,
		LeadID:    leadID,
		AuthorID:  authorID,
		Content:   content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

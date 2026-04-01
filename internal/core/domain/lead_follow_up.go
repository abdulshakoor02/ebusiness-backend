package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FollowUpStatus string

const (
	StatusActive FollowUpStatus = "active"
	StatusClosed FollowUpStatus = "closed"
)

type LeadFollowUp struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID    primitive.ObjectID `bson:"tenant_id" json:"tenant_id"`
	LeadID      primitive.ObjectID `bson:"lead_id" json:"lead_id"`
	CreatorID   primitive.ObjectID `bson:"creator_id" json:"creator_id"` // user who created it
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	StartTime   time.Time          `bson:"start_time" json:"start_time"`
	EndTime     time.Time          `bson:"end_time" json:"end_time"`
	Status      FollowUpStatus     `bson:"status" json:"status"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

func NewLeadFollowUp(tenantID, leadID, creatorID primitive.ObjectID, title, description string, startTime, endTime time.Time, status FollowUpStatus) *LeadFollowUp {
	return &LeadFollowUp{
		ID:          primitive.NewObjectID(),
		TenantID:    tenantID,
		LeadID:      leadID,
		CreatorID:   creatorID,
		Title:       title,
		Description: description,
		StartTime:   startTime,
		EndTime:     endTime,
		Status:      status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func IsValidFollowUpStatus(status FollowUpStatus) bool {
	return status == StatusActive || status == StatusClosed
}

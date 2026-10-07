package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FacebookCampaignAssignment maps one Facebook campaign to one CRM user.
// Leads arriving from that campaign are auto-assigned to the mapped user.
type FacebookCampaignAssignment struct {
	TenantID   primitive.ObjectID `bson:"tenant_id" json:"tenant_id"`
	CampaignID string             `bson:"campaign_id" json:"campaign_id"`
	AdAccount  string             `bson:"ad_account,omitempty" json:"ad_account,omitempty"`
	AssignedTo primitive.ObjectID `bson:"assigned_to" json:"assigned_to"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time          `bson:"updated_at" json:"updated_at"`
}

func NewFacebookCampaignAssignment(tenantID primitive.ObjectID, campaignID, adAccount string, assignedTo primitive.ObjectID) *FacebookCampaignAssignment {
	now := time.Now()
	return &FacebookCampaignAssignment{
		TenantID:   tenantID,
		CampaignID: campaignID,
		AdAccount:  adAccount,
		AssignedTo: assignedTo,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

package ports

import (
	"context"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FacebookCampaignAssignmentRepository persists per-campaign user mappings.
type FacebookCampaignAssignmentRepository interface {
	Upsert(ctx context.Context, a *domain.FacebookCampaignAssignment) error
	GetByCampaign(ctx context.Context, tenantID primitive.ObjectID, campaignID string) (*domain.FacebookCampaignAssignment, error)
	ListByTenant(ctx context.Context, tenantID primitive.ObjectID) ([]*domain.FacebookCampaignAssignment, error)
	Delete(ctx context.Context, tenantID primitive.ObjectID, campaignID string) error
}

// FacebookCampaignView is a campaign plus its optional assignment.
type FacebookCampaignView struct {
	CampaignID   string     `json:"campaign_id"`
	Name         string     `json:"name"`
	Status       string     `json:"status,omitempty"`
	Objective    string     `json:"objective,omitempty"`
	AdAccount    string     `json:"ad_account,omitempty"`
	AssignedTo   string     `json:"assigned_to,omitempty"`
	AssignedName string     `json:"assigned_name,omitempty"`
	AssignedAt   *time.Time `json:"assigned_at,omitempty"`
}

// AssignFacebookCampaignRequest maps a campaign to a user.
type AssignFacebookCampaignRequest struct {
	CampaignID string `json:"campaign_id"`
	AssignedTo string `json:"assigned_to"`
}

// FacebookAdsService lists campaigns and manages their user assignments.
type FacebookAdsService interface {
	ListCampaigns(ctx context.Context, tenantID primitive.ObjectID) ([]FacebookCampaignView, error)
	AssignCampaign(ctx context.Context, tenantID primitive.ObjectID, req AssignFacebookCampaignRequest) (*domain.FacebookCampaignAssignment, error)
	UnassignCampaign(ctx context.Context, tenantID primitive.ObjectID, campaignID string) error
}

// FacebookLeadIngestResult summarizes a webhook delivery.
type FacebookLeadIngestResult struct {
	Accepted   bool   `json:"accepted"`
	LeadID     string `json:"lead_id,omitempty"`
	AssignedTo string `json:"assigned_to,omitempty"`
	Created    bool   `json:"created"`
	Reason     string `json:"reason,omitempty"`
}

// FacebookLeadService ingests Lead Ads webhook deliveries.
type FacebookLeadService interface {
	VerifyWebhook(mode, token, challenge string) (string, error)
	IngestLead(ctx context.Context, payload map[string]interface{}) (*FacebookLeadIngestResult, error)
}

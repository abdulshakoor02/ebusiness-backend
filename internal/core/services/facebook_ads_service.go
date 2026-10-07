package services

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/abdulshakoor02/goCrmBackend/pkg/facebook"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FacebookAdsService struct {
	connRepo   ports.FacebookConnectionRepository
	assignRepo ports.FacebookCampaignAssignmentRepository
	userRepo   ports.UserRepository
	client     *facebook.Client
}

func NewFacebookAdsService(connRepo ports.FacebookConnectionRepository, assignRepo ports.FacebookCampaignAssignmentRepository, userRepo ports.UserRepository, client *facebook.Client) *FacebookAdsService {
	return &FacebookAdsService{
		connRepo:   connRepo,
		assignRepo: assignRepo,
		userRepo:   userRepo,
		client:     client,
	}
}

func (s *FacebookAdsService) tokenForTenant(ctx context.Context, tenantID primitive.ObjectID) (string, error) {
	conn, err := s.connRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return "", err
	}
	if conn == nil || conn.AccessToken == "" {
		return "", errors.New("facebook is not connected for this tenant")
	}
	if conn.TokenExpiresAt != nil && time.Now().After(*conn.TokenExpiresAt) {
		slog.Warn("Facebook token expired", "tenant_id", tenantID.Hex())
	}
	return conn.AccessToken, nil
}

func (s *FacebookAdsService) ListCampaigns(ctx context.Context, tenantID primitive.ObjectID) ([]ports.FacebookCampaignView, error) {
	token, err := s.tokenForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	accounts, err := s.client.GetAdAccounts(ctx, token)
	if err != nil {
		slog.Error("Failed to list ad accounts", "error", err)
		return nil, err
	}

	assignments, err := s.assignRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	byCampaign := make(map[string]*domain.FacebookCampaignAssignment, len(assignments))
	for _, a := range assignments {
		byCampaign[a.CampaignID] = a
	}

	userNames := make(map[primitive.ObjectID]string)
	nameOf := func(id primitive.ObjectID) string {
		if n, ok := userNames[id]; ok {
			return n
		}
		u, err := s.userRepo.GetByID(ctx, id)
		if err != nil || u == nil {
			userNames[id] = ""
			return ""
		}
		userNames[id] = u.Name
		return u.Name
	}

	views := make([]ports.FacebookCampaignView, 0)
	for _, acct := range accounts {
		camps, err := s.client.GetCampaigns(ctx, token, acct.ID)
		if err != nil {
			slog.Warn("Failed to list campaigns for ad account", "ad_account", acct.ID, "error", err)
			continue
		}
		for _, cp := range camps {
			v := ports.FacebookCampaignView{
				CampaignID: cp.ID,
				Name:       cp.Name,
				Status:     cp.EffectiveStatus,
				Objective:  cp.Objective,
				AdAccount:  acct.ID,
			}
			if v.Status == "" {
				v.Status = cp.Status
			}
			// Only surface currently running campaigns.
			if !strings.EqualFold(v.Status, "ACTIVE") {
				continue
			}
			if a, ok := byCampaign[cp.ID]; ok {
				v.AssignedTo = a.AssignedTo.Hex()
				v.AssignedName = nameOf(a.AssignedTo)
				assignedAt := a.UpdatedAt
				v.AssignedAt = &assignedAt
			}
			views = append(views, v)
		}
	}

	sort.Slice(views, func(i, j int) bool { return views[i].Name < views[j].Name })
	return views, nil
}

func (s *FacebookAdsService) AssignCampaign(ctx context.Context, tenantID primitive.ObjectID, req ports.AssignFacebookCampaignRequest) (*domain.FacebookCampaignAssignment, error) {
	if strings.TrimSpace(req.CampaignID) == "" {
		return nil, errors.New("campaign_id is required")
	}
	userID, err := primitive.ObjectIDFromHex(req.AssignedTo)
	if err != nil {
		return nil, errors.New("invalid assigned_to user id format")
	}

	// The assignee must belong to the same tenant.
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return nil, errors.New("assigned user not found")
	}
	if u.TenantID != tenantID {
		return nil, errors.New("assigned user must belong to the same tenant")
	}

	existing, err := s.assignRepo.GetByCampaign(ctx, tenantID, req.CampaignID)
	if err != nil {
		return nil, err
	}

	a := existing
	if a == nil {
		a = domain.NewFacebookCampaignAssignment(tenantID, req.CampaignID, "", userID)
	} else {
		a.AssignedTo = userID
		a.UpdatedAt = time.Now()
	}
	if err := s.assignRepo.Upsert(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *FacebookAdsService) UnassignCampaign(ctx context.Context, tenantID primitive.ObjectID, campaignID string) error {
	if strings.TrimSpace(campaignID) == "" {
		return errors.New("campaign_id is required")
	}
	return s.assignRepo.Delete(ctx, tenantID, campaignID)
}

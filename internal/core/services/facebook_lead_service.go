package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/abdulshakoor02/goCrmBackend/pkg/facebook"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FacebookLeadService struct {
	connRepo   ports.FacebookConnectionRepository
	assignRepo ports.FacebookCampaignAssignmentRepository
	leadRepo   ports.LeadRepository
	sourceRepo ports.LeadSourceRepository
	client     *facebook.Client
	verifyTok  string
}

func NewFacebookLeadService(connRepo ports.FacebookConnectionRepository, assignRepo ports.FacebookCampaignAssignmentRepository, leadRepo ports.LeadRepository, sourceRepo ports.LeadSourceRepository, client *facebook.Client, verifyToken string) *FacebookLeadService {
	return &FacebookLeadService{
		connRepo:   connRepo,
		assignRepo: assignRepo,
		leadRepo:   leadRepo,
		sourceRepo: sourceRepo,
		client:     client,
		verifyTok:  verifyToken,
	}
}

func (s *FacebookLeadService) VerifyWebhook(mode, token, challenge string) (string, error) {
	if s.verifyTok == "" {
		return "", errors.New("facebook webhook is not configured (FACEBOOK_WEBHOOK_VERIFY_TOKEN missing)")
	}
	if mode != "subscribe" || token != s.verifyTok || challenge == "" {
		return "", errors.New("webhook verification failed")
	}
	return challenge, nil
}

func leadgenIDsFromPayload(payload map[string]interface{}) []string {
	ids := make([]string, 0, 2)
	seen := make(map[string]bool)
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	obj, _ := payload["object"].(string)
	if obj != "" && obj != "page" {
		return ids
	}
	entries, _ := payload["entry"].([]interface{})
	for _, e := range entries {
		entry, _ := e.(map[string]interface{})
		if entry == nil {
			continue
		}
		changes, _ := entry["changes"].([]interface{})
		for _, ch := range changes {
			change, _ := ch.(map[string]interface{})
			if change == nil {
				continue
			}
			if f, _ := change["field"].(string); f != "" && f != "leadgen" {
				continue
			}
			value, _ := change["value"].(map[string]interface{})
			if value == nil {
				continue
			}
			if id, _ := value["leadgen_id"].(string); id != "" {
				add(id)
			}
		}
	}
	return ids
}
func fieldValue(fields []facebook.LeadField, names ...string) string {
	for _, want := range names {
		for _, f := range fields {
			if strings.EqualFold(strings.TrimSpace(f.Name), want) && len(f.Values) > 0 {
				return strings.TrimSpace(f.Values[0])
			}
		}
	}
	return ""
}

func splitName(full string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(full))
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func (s *FacebookLeadService) IngestLead(ctx context.Context, payload map[string]interface{}) (*ports.FacebookLeadIngestResult, error) {
	if payload == nil {
		return &ports.FacebookLeadIngestResult{Accepted: false, Reason: "empty payload"}, nil
	}

	// Resolve which tenant this delivery belongs to.
	tenantID, conn, err := s.resolveTenantForPayload(ctx, payload)
	if err != nil {
		return &ports.FacebookLeadIngestResult{Accepted: false, Reason: err.Error()}, nil
	}

	// Tenant-scoped context so downstream repos enforce isolation.
	ctx = context.WithValue(ctx, "tenant_id", tenantID.Hex())

	ids := leadgenIDsFromPayload(payload)
	if len(ids) == 0 {
		return &ports.FacebookLeadIngestResult{Accepted: false, Reason: "no leadgen_id in payload"}, nil
	}

	var last *ports.FacebookLeadIngestResult
	for _, leadgenID := range ids {
		r, err := s.ingestOne(ctx, tenantID, conn.AccessToken, leadgenID)
		if err != nil {
			slog.Error("Facebook lead ingest failed", "leadgen_id", leadgenID, "error", err)
			last = &ports.FacebookLeadIngestResult{Accepted: false, Reason: err.Error()}
			continue
		}
		last = r
	}
	if last == nil {
		return &ports.FacebookLeadIngestResult{Accepted: false, Reason: "no leads processed"}, nil
	}
	return last, nil
}

func (s *FacebookLeadService) ingestOne(ctx context.Context, tenantID primitive.ObjectID, token, leadgenID string) (*ports.FacebookLeadIngestResult, error) {
	fg, err := s.client.GetLeadgenLead(ctx, token, leadgenID)
	if err != nil {
		return &ports.FacebookLeadIngestResult{Accepted: false, Reason: "failed to fetch lead from facebook"}, nil
	}

	// Resolve the ad -> campaign (best effort; unassigned leads still stored).
	campaignID := ""
	if strings.TrimSpace(fg.AdID) != "" {
		if ad, err := s.client.GetAdCampaign(ctx, token, fg.AdID); err == nil && ad != nil {
			campaignID = strings.TrimSpace(ad.Campaign.ID)
		} else if err != nil {
			slog.Warn("Failed to resolve ad to campaign", "ad_id", fg.AdID, "error", err)
		}
	}

	// Campaign -> assigned CRM user.
	assignedTo := primitive.NilObjectID
	if campaignID != "" {
		if a, err := s.assignRepo.GetByCampaign(ctx, tenantID, campaignID); err == nil && a != nil && !a.AssignedTo.IsZero() {
			assignedTo = a.AssignedTo
		}
	}

	first := fieldValue(fg.FieldData, "first_name", "firstname", "given_name")
	last := fieldValue(fg.FieldData, "last_name", "lastname", "surname", "family_name")
	email := fieldValue(fg.FieldData, "email", "email_address")
	phone := fieldValue(fg.FieldData, "phone_number", "phone", "mobile_phone", "mobile")
	if first == "" && last == "" {
		first, last = splitName(fieldValue(fg.FieldData, "full_name", "name"))
	}
	if first == "" {
		first = "Facebook"
	}
	if last == "" {
		last = "Lead"
	}

	// Dedup on email/phone within the tenant.
	existing, err := s.leadRepo.FindByEmailOrPhone(ctx, tenantID, email, phone)
	if err != nil {
		slog.Error("Lead lookup failed", "error", err)
		return &ports.FacebookLeadIngestResult{Accepted: false, Reason: "lead lookup failed"}, nil
	}

	sourceID := s.facebookSourceID(ctx, tenantID)
	assignedToStr := ""
	if !assignedTo.IsZero() {
		assignedToStr = assignedTo.Hex()
	}

	req := ports.CreateLeadRequest{
		FirstName:   first,
		LastName:    last,
		Designation: fmt.Sprintf("Facebook Ad (campaign %s)", campaignIDOrUnknown(campaignID)),
		Email:       email,
		Phone:       phone,
		SourceID:    sourceID,
		AssignedTo:  assignedToStr,
	}

	if existing != nil {
		// Refresh the assignee from the campaign mapping on repeat deliveries.
		updated := false
		if !assignedTo.IsZero() && existing.AssignedTo != assignedTo {
			existing.AssignedTo = assignedTo
			updated = true
		}
		if email != "" && existing.Email == "" {
			existing.Email = email
			updated = true
		}
		if phone != "" && existing.Phone == "" {
			existing.Phone = phone
			updated = true
		}
		if existing.SourceID.IsZero() && sourceID != "" {
			if sid, err := primitive.ObjectIDFromHex(sourceID); err == nil {
				existing.SourceID = sid
				updated = true
			}
		}
		existing.UpdatedAt = time.Now()
		existing.BuildSearchText()
		if updated {
			if err := s.leadRepo.Update(ctx, existing); err != nil {
				return &ports.FacebookLeadIngestResult{Accepted: false, Reason: "failed to update lead"}, nil
			}
		}
		return &ports.FacebookLeadIngestResult{
			Accepted:   true,
			LeadID:     existing.ID.Hex(),
			AssignedTo: assignedToStr,
			Created:    false,
			Reason:     "duplicate — matched existing lead",
		}, nil
	}

	lead := domain.NewLead(tenantID, req.FirstName, req.LastName, req.Designation, req.Email, req.Phone)
	if !assignedTo.IsZero() {
		lead.AssignedTo = assignedTo
	}
	if sourceID != "" {
		if sid, err := primitive.ObjectIDFromHex(sourceID); err == nil {
			lead.SourceID = sid
		}
	}
	if err := s.leadRepo.Create(ctx, lead); err != nil {
		// Unique-index race: fetch what another delivery just created.
		if dup, ferr := s.leadRepo.FindByEmailOrPhone(ctx, tenantID, email, phone); ferr == nil && dup != nil {
			return &ports.FacebookLeadIngestResult{Accepted: true, LeadID: dup.ID.Hex(), AssignedTo: assignedToStr, Created: false, Reason: "duplicate — matched existing lead"}, nil
		}
		return &ports.FacebookLeadIngestResult{Accepted: false, Reason: "failed to create lead"}, nil
	}
	return &ports.FacebookLeadIngestResult{
		Accepted:   true,
		LeadID:     lead.ID.Hex(),
		AssignedTo: assignedToStr,
		Created:    true,
	}, nil
}

func campaignIDOrUnknown(id string) string {
	if strings.TrimSpace(id) == "" {
		return "unknown"
	}
	return id
}

// facebookSourceID returns the id (hex) of the tenant's "Facebook" lead source,
// creating it on first use.
func (s *FacebookLeadService) facebookSourceID(ctx context.Context, tenantID primitive.ObjectID) string {
	src, err := s.sourceRepo.FindByName(ctx, tenantID, "Facebook")
	if err == nil && src != nil {
		return src.ID.Hex()
	}
	created := domain.NewLeadSource(tenantID, "Facebook", "Leads imported from Facebook Lead Ads")
	if err := s.sourceRepo.Create(ctx, created); err != nil {
		// Another delivery may have created it concurrently — retry the lookup.
		if src, ferr := s.sourceRepo.FindByName(ctx, tenantID, "Facebook"); ferr == nil && src != nil {
			return src.ID.Hex()
		}
		slog.Warn("Failed to create Facebook lead source", "error", err)
		return ""
	}
	return created.ID.Hex()
}

// resolveTenantForPayload maps a webhook delivery to a tenant by matching the
// delivery's page id against every connected tenant's page list.
func (s *FacebookLeadService) resolveTenantForPayload(ctx context.Context, payload map[string]interface{}) (primitive.ObjectID, *domain.FacebookConnection, error) {
	pageIDs := pageIDsFromPayload(payload)
	if len(pageIDs) == 0 {
		return primitive.NilObjectID, nil, errors.New("no page id in payload")
	}

	conns, err := s.connRepo.ListAll(ctx)
	if err != nil {
		return primitive.NilObjectID, nil, err
	}
	for _, conn := range conns {
		if conn == nil || conn.AccessToken == "" {
			continue
		}
		pages, err := s.client.GetPages(ctx, conn.AccessToken)
		if err != nil {
			slog.Warn("Failed to list pages for tenant", "tenant_id", conn.TenantID.Hex(), "error", err)
			continue
		}
		for _, p := range pages {
			if pageIDs[p.ID] {
			return conn.TenantID, conn, nil
			}
		}
	}
	return primitive.NilObjectID, nil, errors.New("no connected tenant matches this page")
}

func pageIDsFromPayload(payload map[string]interface{}) map[string]bool {
	out := make(map[string]bool)
	entries, _ := payload["entry"].([]interface{})
	for _, e := range entries {
		entry, _ := e.(map[string]interface{})
		if entry == nil {
			continue
		}
		if id, _ := entry["id"].(string); id != "" {
			out[id] = true
		}
	}
	return out
}

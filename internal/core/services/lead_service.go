package services

import (
	"context"
	"errors"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadService struct {
	leadRepo ports.LeadRepository
}

func NewLeadService(leadRepo ports.LeadRepository) *LeadService {
	return &LeadService{
		leadRepo: leadRepo,
	}
}

func (s *LeadService) CreateLead(ctx context.Context, req ports.CreateLeadRequest) (*domain.Lead, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required to create lead")
	}

	lead := domain.NewLead(
		tenantID,
		req.FirstName,
		req.LastName,
		req.Designation,
		req.Email,
		req.Phone,
	)

	if req.AssignedTo != "" {
		assignedToID, err := primitive.ObjectIDFromHex(req.AssignedTo)
		if err != nil {
			return nil, errors.New("invalid assigned_to user id format")
		}
		lead.AssignedTo = assignedToID
	}

	if req.CategoryID != "" {
		categoryID, err := primitive.ObjectIDFromHex(req.CategoryID)
		if err != nil {
			return nil, errors.New("invalid category_id form")
		}
		lead.CategoryID = categoryID
	}

	if req.SourceID != "" {
		sourceID, err := primitive.ObjectIDFromHex(req.SourceID)
		if err != nil {
			return nil, errors.New("invalid source_id format")
		}
		lead.SourceID = sourceID
	}

	if req.CountryID != "" {
		countryID, err := primitive.ObjectIDFromHex(req.CountryID)
		if err != nil {
			return nil, errors.New("invalid country_id format")
		}
		lead.CountryID = countryID
	}

	if req.QualificationID != "" {
		qualificationID, err := primitive.ObjectIDFromHex(req.QualificationID)
		if err != nil {
			return nil, errors.New("invalid qualification_id format")
		}
		lead.QualificationID = qualificationID
	}

	if req.Address.Street != "" || req.Address.City != "" || req.Address.State != "" || req.Address.Country != "" || req.Address.ZipCode != "" || req.Address.AddressLine != "" {
		lead.Address = req.Address
	}

	lead.BuildSearchText()

	if err := s.leadRepo.Create(ctx, lead); err != nil {
		return nil, err
	}

	return lead, nil
}

func (s *LeadService) GetLead(ctx context.Context, id primitive.ObjectID) (*domain.Lead, error) {
	return s.leadRepo.GetByID(ctx, id)
}

func (s *LeadService) UpdateLead(ctx context.Context, id primitive.ObjectID, req ports.UpdateLeadRequest) (*domain.Lead, error) {
	lead, err := s.leadRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.FirstName != "" {
		lead.FirstName = req.FirstName
	}
	if req.LastName != "" {
		lead.LastName = req.LastName
	}
	if req.Designation != "" {
		lead.Designation = req.Designation
	}
	if req.Email != "" {
		lead.Email = req.Email
	}
	if req.Phone != "" {
		lead.Phone = req.Phone
	}

	if req.AssignedTo != "" {
		assignedToID, err := primitive.ObjectIDFromHex(req.AssignedTo)
		if err != nil {
			return nil, errors.New("invalid assigned_to user id format")
		}
		lead.AssignedTo = assignedToID
	}

	if req.CategoryID != "" {
		categoryID, err := primitive.ObjectIDFromHex(req.CategoryID)
		if err != nil {
			return nil, errors.New("invalid category_id format")
		}
		lead.CategoryID = categoryID
	}

	if req.SourceID != "" {
		sourceID, err := primitive.ObjectIDFromHex(req.SourceID)
		if err != nil {
			return nil, errors.New("invalid source_id format")
		}
		lead.SourceID = sourceID
	}

	if req.CountryID != "" {
		countryID, err := primitive.ObjectIDFromHex(req.CountryID)
		if err != nil {
			return nil, errors.New("invalid country_id format")
		}
		lead.CountryID = countryID
	}

	if req.QualificationID != "" {
		qualificationID, err := primitive.ObjectIDFromHex(req.QualificationID)
		if err != nil {
			return nil, errors.New("invalid qualification_id format")
		}
		lead.QualificationID = qualificationID
	}

	if req.Address.Street != "" || req.Address.City != "" || req.Address.State != "" || req.Address.Country != "" || req.Address.ZipCode != "" || req.Address.AddressLine != "" {
		lead.Address = req.Address
	}

	lead.BuildSearchText()

	if err := s.leadRepo.Update(ctx, lead); err != nil {
		return nil, err
	}

	return lead, nil
}

func (s *LeadService) ListLeads(ctx context.Context, req ports.FilterRequest) ([]*ports.LeadListItem, int64, error) {
	return s.leadRepo.List(ctx, req.Filters, req.Search, req.Offset, req.Limit)
}

func (s *LeadService) UpdateLeadStatus(ctx context.Context, id primitive.ObjectID, req ports.UpdateLeadStatusRequest) (*domain.Lead, error) {
	lead, err := s.leadRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Only allow toggling between active and inactive
	if req.Status != domain.LeadStatusActive && req.Status != domain.LeadStatusInactive {
		return nil, errors.New("status must be 'active' or 'inactive'")
	}

	// Cannot toggle if still a lead (not yet converted)
	if lead.Status == domain.LeadStatusLead {
		return nil, errors.New("lead has not been converted to a client yet")
	}

	lead.Status = req.Status
	lead.UpdatedAt = time.Now()

	if err := s.leadRepo.Update(ctx, lead); err != nil {
		return nil, err
	}

	return lead, nil
}

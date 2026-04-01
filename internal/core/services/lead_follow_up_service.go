package services

import (
	"context"
	"errors"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadFollowUpService struct {
	followUpRepo ports.LeadFollowUpRepository
	leadRepo     ports.LeadRepository
}

func NewLeadFollowUpService(followUpRepo ports.LeadFollowUpRepository, leadRepo ports.LeadRepository) *LeadFollowUpService {
	return &LeadFollowUpService{
		followUpRepo: followUpRepo,
		leadRepo:     leadRepo,
	}
}

func (s *LeadFollowUpService) CreateLeadFollowUp(ctx context.Context, leadID primitive.ObjectID, req ports.CreateLeadFollowUpRequest) (*domain.LeadFollowUp, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required")
	}

	creatorID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, errors.New("user context required")
	}

	// Verify lead exists and belongs to tenant
	_, err := s.leadRepo.GetByID(ctx, leadID)
	if err != nil {
		return nil, errors.New("lead not found or unauthorized")
	}

	// Basic validation for dates
	if req.StartTime.IsZero() || req.EndTime.IsZero() {
		return nil, errors.New("start_time and end_time are required")
	}
	if req.EndTime.Before(req.StartTime) {
		return nil, errors.New("end_time must be after start_time")
	}

	// Validate status
	status := domain.FollowUpStatus(req.Status)
	if status == "" {
		status = domain.StatusActive // default
	}
	if !domain.IsValidFollowUpStatus(status) {
		return nil, errors.New("invalid status: must be 'active' or 'closed'")
	}

	followUp := domain.NewLeadFollowUp(
		tenantID,
		leadID,
		creatorID,
		req.Title,
		req.Description,
		req.StartTime,
		req.EndTime,
		status,
	)

	if err := s.followUpRepo.Create(ctx, followUp); err != nil {
		return nil, err
	}

	return followUp, nil
}

func (s *LeadFollowUpService) GetLeadFollowUp(ctx context.Context, id primitive.ObjectID) (*domain.LeadFollowUp, error) {
	return s.followUpRepo.GetByID(ctx, id)
}

func (s *LeadFollowUpService) UpdateLeadFollowUp(ctx context.Context, id primitive.ObjectID, req ports.UpdateLeadFollowUpRequest) (*domain.LeadFollowUp, error) {
	followUp, err := s.followUpRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cannot update closed follow-ups
	if followUp.Status == domain.StatusClosed {
		return nil, errors.New("cannot update closed follow-up")
	}

	// Authorization: any user with update permission can update follow-ups

	if req.Title != "" {
		followUp.Title = req.Title
	}
	if req.Description != "" {
		followUp.Description = req.Description
	}
	if !req.StartTime.IsZero() {
		followUp.StartTime = req.StartTime
	}
	if !req.EndTime.IsZero() {
		followUp.EndTime = req.EndTime
	}

	if followUp.EndTime.Before(followUp.StartTime) {
		return nil, errors.New("end_time must be after start_time")
	}

	if req.Status != "" {
		status := domain.FollowUpStatus(req.Status)
		if !domain.IsValidFollowUpStatus(status) {
			return nil, errors.New("invalid status: must be 'active' or 'closed'")
		}
		followUp.Status = status
	}

	if err := s.followUpRepo.Update(ctx, followUp); err != nil {
		return nil, err
	}

	return followUp, nil
}

func (s *LeadFollowUpService) DeleteLeadFollowUp(ctx context.Context, id primitive.ObjectID) error {
	// Verify follow-up exists
	_, err := s.followUpRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Authorization: any user with delete permission can delete follow-ups

	return s.followUpRepo.Delete(ctx, id)
}

func (s *LeadFollowUpService) ListLeadFollowUps(ctx context.Context, leadID primitive.ObjectID, req ports.FilterRequest) ([]*ports.FollowUpListItem, int64, error) {
	return s.followUpRepo.ListByLeadID(ctx, leadID, req.Filters, req.Offset, req.Limit)
}

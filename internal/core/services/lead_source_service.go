package services

import (
	"context"
	"errors"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadSourceService struct {
	repo ports.LeadSourceRepository
}

func NewLeadSourceService(repo ports.LeadSourceRepository) *LeadSourceService {
	return &LeadSourceService{
		repo: repo,
	}
}

func (s *LeadSourceService) CreateLeadSource(ctx context.Context, req ports.CreateLeadSourceRequest) (*domain.LeadSource, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required")
	}

	source := domain.NewLeadSource(tenantID, req.Name, req.Description)

	if err := s.repo.Create(ctx, source); err != nil {
		return nil, err
	}

	return source, nil
}

func (s *LeadSourceService) GetLeadSource(ctx context.Context, id primitive.ObjectID) (*domain.LeadSource, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *LeadSourceService) UpdateLeadSource(ctx context.Context, id primitive.ObjectID, req ports.UpdateLeadSourceRequest) (*domain.LeadSource, error) {
	source, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		source.Name = req.Name
	}
	if req.Description != "" {
		source.Description = req.Description
	}

	if err := s.repo.Update(ctx, source); err != nil {
		return nil, err
	}

	return source, nil
}

func (s *LeadSourceService) DeleteLeadSource(ctx context.Context, id primitive.ObjectID) error {
	return s.repo.Delete(ctx, id)
}

func (s *LeadSourceService) ListLeadSources(ctx context.Context, req ports.FilterRequest) ([]*domain.LeadSource, int64, error) {
	return s.repo.List(ctx, req.Filters, req.Offset, req.Limit)
}

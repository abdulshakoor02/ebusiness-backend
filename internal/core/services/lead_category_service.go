package services

import (
	"context"
	"errors"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadCategoryService struct {
	categoryRepo ports.LeadCategoryRepository
}

func NewLeadCategoryService(categoryRepo ports.LeadCategoryRepository) *LeadCategoryService {
	return &LeadCategoryService{
		categoryRepo: categoryRepo,
	}
}

func (s *LeadCategoryService) CreateLeadCategory(ctx context.Context, req ports.CreateLeadCategoryRequest) (*domain.LeadCategory, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required")
	}

	category := domain.NewLeadCategory(
		tenantID,
		req.Name,
		req.Description,
	)

	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return nil, err
	}

	return category, nil
}

func (s *LeadCategoryService) GetLeadCategory(ctx context.Context, id primitive.ObjectID) (*domain.LeadCategory, error) {
	return s.categoryRepo.GetByID(ctx, id)
}

func (s *LeadCategoryService) UpdateLeadCategory(ctx context.Context, id primitive.ObjectID, req ports.UpdateLeadCategoryRequest) (*domain.LeadCategory, error) {
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		category.Name = req.Name
	}
	if req.Description != "" {
		category.Description = req.Description
	}

	if err := s.categoryRepo.Update(ctx, category); err != nil {
		return nil, err
	}

	return category, nil
}

func (s *LeadCategoryService) DeleteLeadCategory(ctx context.Context, id primitive.ObjectID) error {
	// TODO: verify this category is not currently in use by a Lead
	// For now, allow deletion blindly; will circle back if strictly enforcing FK
	return s.categoryRepo.Delete(ctx, id)
}

func (s *LeadCategoryService) ListLeadCategories(ctx context.Context, req ports.FilterRequest) ([]*domain.LeadCategory, int64, error) {
	return s.categoryRepo.List(ctx, req.Filters, req.Offset, req.Limit)
}

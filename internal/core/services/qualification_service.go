package services

import (
	"context"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type QualificationService struct {
	qualificationRepo ports.QualificationRepository
}

func NewQualificationService(qualificationRepo ports.QualificationRepository) *QualificationService {
	return &QualificationService{
		qualificationRepo: qualificationRepo,
	}
}

func (s *QualificationService) CreateQualification(ctx context.Context, req ports.CreateQualificationRequest) (*domain.Qualification, error) {
	qualification := domain.NewQualification(req.Name)

	if err := s.qualificationRepo.Create(ctx, qualification); err != nil {
		return nil, err
	}

	return qualification, nil
}

func (s *QualificationService) GetQualification(ctx context.Context, id primitive.ObjectID) (*domain.Qualification, error) {
	return s.qualificationRepo.GetByID(ctx, id)
}

func (s *QualificationService) UpdateQualification(ctx context.Context, id primitive.ObjectID, req ports.UpdateQualificationRequest) (*domain.Qualification, error) {
	qualification, err := s.qualificationRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		qualification.Name = req.Name
	}
	if req.IsActive != nil {
		qualification.IsActive = *req.IsActive
	}

	if err := s.qualificationRepo.Update(ctx, qualification); err != nil {
		return nil, err
	}

	return qualification, nil
}

func (s *QualificationService) DeleteQualification(ctx context.Context, id primitive.ObjectID) error {
	return s.qualificationRepo.Delete(ctx, id)
}

func (s *QualificationService) ListQualifications(ctx context.Context, req ports.FilterRequest) ([]*domain.Qualification, int64, error) {
	filters := req.Filters
	if filters == nil {
		filters = make(map[string]interface{})
	}

	if _, exists := filters["is_active"]; !exists {
		filters["is_active"] = true
	}

	return s.qualificationRepo.List(ctx, filters, req.Offset, req.Limit)
}

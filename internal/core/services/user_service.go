package services

import (
	"context"
	"errors"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserService struct {
	userRepo ports.UserRepository
}

func NewUserService(userRepo ports.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

func (s *UserService) CreateUser(ctx context.Context, req ports.CreateUserRequest) (*domain.User, error) {
	// Extract Tenant ID from context
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required to create user")
	}

	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := domain.NewUser(
		tenantID,
		req.Name,
		req.Email,
		hashedPassword,
		req.Role,
	)

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) ListUsers(ctx context.Context, req ports.FilterRequest) ([]*domain.User, int64, error) {
	return s.userRepo.List(ctx, req.Filters, req.Offset, req.Limit)
}

func (s *UserService) GetUser(ctx context.Context, id primitive.ObjectID) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *UserService) UpdateUser(ctx context.Context, id primitive.ObjectID, req ports.UpdateUserRequest) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Role != "" {
		user.Role = req.Role
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Helper to extract tenant ID from context (duplicated from repo layer, maybe move to pkg/context or shared)
func getTenantIDFromContext(ctx context.Context) (primitive.ObjectID, bool) {
	val := ctx.Value("tenant_id")
	if val == nil {
		return primitive.NilObjectID, false
	}
	if idStr, ok := val.(string); ok {
		id, err := primitive.ObjectIDFromHex(idStr)
		if err == nil {
			return id, true
		}
	}
	if id, ok := val.(primitive.ObjectID); ok {
		return id, true
	}
	return primitive.NilObjectID, false
}

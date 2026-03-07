package services

import (
	"context"
	"errors"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type TenantService struct {
	tenantRepo ports.TenantRepository
	userRepo   ports.UserRepository
}

func NewTenantService(tenantRepo ports.TenantRepository, userRepo ports.UserRepository) *TenantService {
	return &TenantService{
		tenantRepo: tenantRepo,
		userRepo:   userRepo,
	}
}

func (s *TenantService) RegisterTenant(ctx context.Context, req ports.CreateTenantRequest) (*domain.Tenant, *domain.User, error) {
	tenant := domain.NewTenant(req.Name, req.Email)
	tenant.LogoURL = req.LogoURL
	tenant.Address = req.Address

	if req.CountryID != "" {
		countryID, err := primitive.ObjectIDFromHex(req.CountryID)
		if err != nil {
			return nil, nil, errors.New("invalid country_id format")
		}
		tenant.CountryID = countryID
	}

	if req.Tax > 0 {
		tenant.Tax = req.Tax
	}

	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, nil, err
	}

	hashedPassword, err := hashPassword(req.AdminUser.Password)
	if err != nil {
		return nil, nil, err
	}

	user := domain.NewUser(
		tenant.ID,
		req.AdminUser.Name,
		req.AdminUser.Email,
		req.AdminUser.Mobile,
		hashedPassword,
		"admin",
	)
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	return tenant, user, nil
}

func (s *TenantService) GetTenant(ctx context.Context, id primitive.ObjectID) (*domain.Tenant, error) {
	return s.tenantRepo.GetByID(ctx, id)
}

func (s *TenantService) UpdateTenant(ctx context.Context, id primitive.ObjectID, req ports.UpdateTenantRequest) (*domain.Tenant, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		tenant.Name = req.Name
	}
	if req.Email != "" {
		tenant.Email = req.Email
	}
	if req.LogoURL != "" {
		tenant.LogoURL = req.LogoURL
	}
	if req.Address.Street != "" || req.Address.City != "" || req.Address.State != "" || req.Address.Country != "" || req.Address.ZipCode != "" {
		tenant.Address = req.Address
	}

	if req.CountryID != "" {
		countryID, err := primitive.ObjectIDFromHex(req.CountryID)
		if err != nil {
			return nil, errors.New("invalid country_id format")
		}
		tenant.CountryID = countryID
	}

	if req.Tax >= 0 {
		tenant.Tax = req.Tax
	}

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

func (s *TenantService) ListTenants(ctx context.Context, req ports.FilterRequest) ([]*domain.Tenant, int64, error) {
	return s.tenantRepo.List(ctx, req.Filters, req.Offset, req.Limit)
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

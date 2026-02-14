package services

import (
	"context"
	"testing"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Mock Repositories
type MockTenantRepo struct {
	mock.Mock
}

func (m *MockTenantRepo) Create(ctx context.Context, tenant *domain.Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

func (m *MockTenantRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Tenant, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.Tenant), args.Error(1)
}

func (m *MockTenantRepo) List(ctx context.Context, filter interface{}, offset, limit int64) ([]*domain.Tenant, int64, error) {
	args := m.Called(ctx, filter, offset, limit)
	return args.Get(0).([]*domain.Tenant), args.Get(1).(int64), args.Error(2)
}

func (m *MockTenantRepo) Update(ctx context.Context, tenant *domain.Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepo) List(ctx context.Context, filter interface{}, offset, limit int64) ([]*domain.User, int64, error) {
	args := m.Called(ctx, filter, offset, limit)
	return args.Get(0).([]*domain.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepo) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func TestRegisterTenant(t *testing.T) {
	// Setup
	mockTenantRepo := new(MockTenantRepo)
	mockUserRepo := new(MockUserRepo)
	service := NewTenantService(mockTenantRepo, mockUserRepo)

	req := ports.CreateTenantRequest{
		Name:    "Acme Corp",
		Email:   "contact@acme.com",
		Address: domain.Address{Street: "123 Main St", City: "Techville"},
		AdminUser: ports.CreateUserRequest{
			Name:     "Alice Admin",
			Email:    "alice@acme.com",
			Password: "securepassword",
		},
	}

	// Expectations
	mockTenantRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Tenant")).Return(nil)
	mockUserRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)

	// Execute
	tenant, user, err := service.RegisterTenant(context.Background(), req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, tenant)
	assert.NotNil(t, user)
	assert.Equal(t, req.Name, tenant.Name)
	assert.Equal(t, req.AdminUser.Email, user.Email)
	assert.Equal(t, tenant.ID, user.TenantID)
	assert.Equal(t, "admin", user.Role)

	mockTenantRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

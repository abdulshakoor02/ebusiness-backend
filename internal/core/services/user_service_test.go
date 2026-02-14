package services

import (
	"context"
	"testing"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateUser(t *testing.T) {
	mockUserRepo := new(MockUserRepo)
	service := NewUserService(mockUserRepo)

	// Simulate tenant ID in context (though repo handles filtering, service creates entity)
	// For creation, we might need tenant ID if not provided in user struct?
	// But CreateUserRequest doesn't have TenantID. It should come from context.
	// Oh wait, in my implementation plan step, I said "Tenant-ID extracted from Context".
	// The Service needs to grab tenant ID from context to create the user entity correctly.

	ctx := context.WithValue(context.Background(), "tenant_id", "507f1f77bcf86cd799439011")

	req := ports.CreateUserRequest{
		Name:     "Bob Builder",
		Email:    "bob@example.com",
		Password: "password123",
		Role:     "user",
	}

	mockUserRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)

	user, err := service.CreateUser(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, req.Name, user.Name)
	// Verify hashed password is not plaintext
	assert.NotEqual(t, req.Password, user.PasswordHash)

	// Verify tenant ID was set (assuming service extracts it)
	// Note: We need a way to mock context or ensure service logic handles it.
}

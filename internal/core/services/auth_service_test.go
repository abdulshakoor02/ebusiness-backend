package services

import (
	"context"
	"errors"
	"testing"

	"github.com/abdulshakoor02/goCrmBackend/config"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func TestLogin_Success(t *testing.T) {
	mockUserRepo := new(MockUserRepo)
	cfg := &config.Config{JWTSecret: "test_secret", JWTExpiration: "1h"}
	service := NewAuthService(mockUserRepo, cfg)

	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := &domain.User{
		Email:        "user@example.com",
		PasswordHash: string(hashedPassword),
	}

	req := ports.LoginRequest{
		Email:    "user@example.com",
		Password: "password123",
	}

	mockUserRepo.On("GetByEmail", mock.Anything, req.Email).Return(user, nil)

	resp, err := service.Login(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, user.Email, resp.User.Email)
}

func TestLogin_InvalidPassword(t *testing.T) {
	mockUserRepo := new(MockUserRepo)
	cfg := &config.Config{JWTSecret: "test_secret", JWTExpiration: "1h"}
	service := NewAuthService(mockUserRepo, cfg)

	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := &domain.User{
		Email:        "user@example.com",
		PasswordHash: string(hashedPassword),
	}

	req := ports.LoginRequest{
		Email:    "user@example.com",
		Password: "wrongpassword",
	}

	mockUserRepo.On("GetByEmail", mock.Anything, req.Email).Return(user, nil)

	resp, err := service.Login(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "invalid credentials", err.Error())
}

func TestLogin_UserNotFound(t *testing.T) {
	mockUserRepo := new(MockUserRepo)
	cfg := &config.Config{JWTSecret: "test_secret", JWTExpiration: "1h"}
	service := NewAuthService(mockUserRepo, cfg)

	req := ports.LoginRequest{
		Email:    "unknown@example.com",
		Password: "password123",
	}

	mockUserRepo.On("GetByEmail", mock.Anything, req.Email).Return(nil, errors.New("user not found"))

	resp, err := service.Login(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

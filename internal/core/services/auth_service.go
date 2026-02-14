package services

import (
	"context"
	"errors"

	"github.com/abdulshakoor02/goCrmBackend/config"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/abdulshakoor02/goCrmBackend/pkg/utils"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo ports.UserRepository
	config   *config.Config
}

func NewAuthService(userRepo ports.UserRepository, config *config.Config) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		config:   config,
	}
}

func (s *AuthService) Login(ctx context.Context, req ports.LoginRequest) (*ports.LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		// Generic error to avoid enumeration
		return nil, errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	token, err := utils.GenerateToken(
		user.ID,
		user.TenantID,
		user.Email,
		user.Role,
		s.config.JWTSecret,
		s.config.JWTExpiration,
	)
	if err != nil {
		return nil, err
	}

	return &ports.LoginResponse{
		Token: token,
		User:  user,
	}, nil
}

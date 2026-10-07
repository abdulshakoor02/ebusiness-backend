package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/abdulshakoor02/goCrmBackend/pkg/cache"
	"github.com/abdulshakoor02/goCrmBackend/pkg/facebook"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrInvalidFacebookState = errors.New("invalid or expired facebook oauth state")

type FacebookService struct {
	repo              ports.FacebookConnectionRepository
	client            *facebook.Client
	stateCache        *cache.OAuthStateCache
	scopes            []string
	postConnectPath   string
}

func NewFacebookService(
	repo ports.FacebookConnectionRepository,
	client *facebook.Client,
	stateCache *cache.OAuthStateCache,
	scopes []string,
	postConnectPath string,
) *FacebookService {
	return &FacebookService{
		repo:            repo,
		client:          client,
		stateCache:      stateCache,
		scopes:          scopes,
		postConnectPath: postConnectPath,
	}
}

func (s *FacebookService) GetAuthURL(ctx context.Context, tenantID primitive.ObjectID) (string, error) {
	state, err := newStateToken()
	if err != nil {
		return "", err
	}
	s.stateCache.Set(state, tenantID)
	url := s.client.AuthURL(state, s.scopes)
	slog.Info("Facebook OAuth flow started", "tenant_id", tenantID.Hex())
	return url, nil
}

func (s *FacebookService) StartAuth(ctx context.Context, tenantID primitive.ObjectID) (*ports.AuthStart, error) {
	authURL, err := s.GetAuthURL(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return &ports.AuthStart{
		UsePopup: true,
		Popup:    authURL,
		AuthURL:  authURL,
	}, nil
}

// connectResultURL builds the landing URL the callback redirects the browser to.
func (s *FacebookService) connectResultURL(ok bool) string {
	base := strings.TrimRight(s.postConnectPath, "/")
	if ok {
		return base + "?facebook=connected"
	}
	return base + "?facebook=error"
}

func (s *FacebookService) HandleCallback(ctx context.Context, state, code string) (string, error) {
	if state == "" || code == "" {
		return s.connectResultURL(false), errors.New("missing state or code")
	}

	tenantID, ok := s.stateCache.Get(state)
	if !ok {
		return s.connectResultURL(false), ErrInvalidFacebookState
	}
	defer s.stateCache.Delete(state)

	shortLived, err := s.client.ExchangeCode(ctx, code)
	if err != nil {
		slog.Error("Facebook code exchange failed", "error", err)
		return s.connectResultURL(false), err
	}

	// For "api" tokens the response already includes a long-expiry token.
	longLived := shortLived
	if shortLived.TokenType == "" || shortLived.TokenType == "bearer" {
		exchanged, err := s.client.ExchangeLongLived(ctx, shortLived.AccessToken)
		if err != nil {
			slog.Warn("Facebook long-lived exchange failed, storing short-lived token", "error", err)
		} else {
			longLived = exchanged
		}
	}

	me, err := s.client.GetMe(ctx, longLived.AccessToken)
	if err != nil {
		slog.Error("Facebook /me lookup failed", "error", err)
		return s.connectResultURL(false), err
	}

	now := time.Now()
	conn := domain.NewFacebookConnection(tenantID)
	conn.FacebookUserID = me.ID
	conn.FacebookUserName = me.Name
	conn.AccessToken = longLived.AccessToken
	conn.TokenType = longLived.TokenType
	if longLived.ExpiresIn > 0 {
		expiresAt := now.Add(time.Duration(longLived.ExpiresIn) * time.Second)
		conn.TokenExpiresAt = &expiresAt
	}
	conn.CreatedAt = now
	conn.UpdatedAt = now

	if err := s.repo.Upsert(ctx, conn); err != nil {
		slog.Error("Failed to store Facebook connection", "error", err)
		return s.connectResultURL(false), err
	}

	slog.Info("Facebook connection stored", "tenant", tenantID.Hex(), "fb_user", conn.FacebookUserID)
	return s.connectResultURL(true), nil
}

func (s *FacebookService) GetStatus(ctx context.Context, tenantID primitive.ObjectID) (*ports.FacebookConnectionStatus, error) {
	conn, err := s.repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if conn == nil || conn.AccessToken == "" {
		return &ports.FacebookConnectionStatus{Connected: false}, nil
	}
	return &ports.FacebookConnectionStatus{
		Connected:      true,
		FacebookUserID: conn.FacebookUserID,
		FacebookName:   conn.FacebookUserName,
		TokenExpiresAt: conn.TokenExpiresAt,
	}, nil
}

func (s *FacebookService) Disconnect(ctx context.Context, tenantID primitive.ObjectID) error {
	return s.repo.DeleteByTenantID(ctx, tenantID)
}

func newStateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

package ports

import (
	"context"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FacebookConnectionRepository persists per-tenant Facebook OAuth connections.
type FacebookConnectionRepository interface {
	Upsert(ctx context.Context, conn *domain.FacebookConnection) error
	GetByTenantID(ctx context.Context, tenantID primitive.ObjectID) (*domain.FacebookConnection, error)
	// ListAll returns every tenant connection (used to resolve webhook deliveries to a tenant).
	ListAll(ctx context.Context) ([]*domain.FacebookConnection, error)
	DeleteByTenantID(ctx context.Context, tenantID primitive.ObjectID) error
}

// FacebookConnectionStatus is the public, token-free view of a connection.
type FacebookConnectionStatus struct {
	Connected        bool       `json:"connected"`
	FacebookUserID   string     `json:"facebook_user_id,omitempty"`
	FacebookName     string     `json:"facebook_name,omitempty"`
	TokenExpiresAt   *time.Time `json:"token_expires_at,omitempty"`
}

// AuthStart is returned to the browser to begin the OAuth login.
type AuthStart struct {
	// UsePopup tells the frontend to open a centered popup window instead of
	// navigating the current tab away.
	UsePopup bool `json:"use_popup"`
	// Popup is the URL to open in the popup window.
	Popup string `json:"popup"`
	// AuthURL is the same URL (back-compat for callers that still navigate).
	AuthURL string `json:"auth_url"`
}

// FacebookService orchestrates the Facebook OAuth flow for a tenant.
type FacebookService interface {
	// GetAuthURL returns the Facebook OAuth authorization URL for the tenant
	// (a server-side state is created and tied to the tenant before redirecting).
	GetAuthURL(ctx context.Context, tenantID primitive.ObjectID) (string, error)
	// StartAuth returns the popup URL and the landing page handle. The frontend
	// opens the popup; when the callback 302s to the landing page with a
	// ?facebook= result, the popup posts a message to its opener and closes.
	StartAuth(ctx context.Context, tenantID primitive.ObjectID) (*AuthStart, error)
	// HandleCallback exchanges the OAuth code for a long-lived token, stores it,
	// and returns the URL the browser should be sent to next.
	HandleCallback(ctx context.Context, state, code string) (string, error)
	GetStatus(ctx context.Context, tenantID primitive.ObjectID) (*FacebookConnectionStatus, error)
	Disconnect(ctx context.Context, tenantID primitive.ObjectID) error
}

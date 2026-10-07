package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FacebookConnection stores the OAuth connection a tenant establishes with
// Facebook. The token stored here is the long-lived user token, used later
// to fetch the tenant's ad campaigns and their leads.
type FacebookConnection struct {
	TenantID         primitive.ObjectID `bson:"tenant_id" json:"tenant_id"`
	FacebookUserID   string             `bson:"facebook_user_id,omitempty" json:"facebook_user_id,omitempty"`
	FacebookUserName string             `bson:"facebook_user_name,omitempty" json:"facebook_user_name,omitempty"`
	AccessToken      string             `bson:"access_token" json:"-"`
	TokenType        string             `bson:"token_type,omitempty" json:"token_type,omitempty"`
	TokenExpiresAt   *time.Time         `bson:"token_expires_at,omitempty" json:"token_expires_at,omitempty"`
	CreatedAt        time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time          `bson:"updated_at" json:"updated_at"`
}

func NewFacebookConnection(tenantID primitive.ObjectID) *FacebookConnection {
	now := time.Now()
	return &FacebookConnection{
		TenantID:   tenantID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

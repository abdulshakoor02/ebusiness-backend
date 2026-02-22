package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID     primitive.ObjectID `bson:"tenant_id" json:"tenant_id"` // Partition Key
	Name         string             `bson:"name" json:"name"`
	Email        string             `bson:"email" json:"email"`
	Mobile       string             `bson:"mobile" json:"mobile"`
	PasswordHash string             `bson:"password_hash" json:"-"`
	Role         string             `bson:"role" json:"role"` // e.g., "admin", "user"
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

func NewUser(tenantID primitive.ObjectID, name, email, mobile, passwordHash, role string) *User {
	return &User{
		ID:           primitive.NewObjectID(),
		TenantID:     tenantID,
		Name:         name,
		Email:        email,
		Mobile:       mobile,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

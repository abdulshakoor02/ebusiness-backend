package ports

import (
	"context"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TenantRepository interface {
	Create(ctx context.Context, tenant *domain.Tenant) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Tenant, error)
	List(ctx context.Context, filter interface{}, offset, limit int64) ([]*domain.Tenant, int64, error)
	Update(ctx context.Context, tenant *domain.Tenant) error
}

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, filter interface{}, offset, limit int64) ([]*domain.User, int64, error)
	Update(ctx context.Context, user *domain.User) error
}

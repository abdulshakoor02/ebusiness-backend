package ports

import (
	"context"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CreateTenantRequest struct {
	Name      string            `json:"name"`
	Email     string            `json:"email"`
	LogoURL   string            `json:"logo_url"`
	Address   domain.Address    `json:"address"`
	AdminUser CreateUserRequest `json:"admin_user"`
}

type UpdateTenantRequest struct {
	Name    string         `json:"name"`
	Email   string         `json:"email"`
	LogoURL string         `json:"logo_url"`
	Address domain.Address `json:"address"`
}

type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type UpdateUserRequest struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

type FilterRequest struct {
	Filters map[string]interface{} `json:"filters"`
	Offset  int64                  `json:"offset"`
	Limit   int64                  `json:"limit"`
}

type TenantService interface {
	RegisterTenant(ctx context.Context, req CreateTenantRequest) (*domain.Tenant, *domain.User, error)
	GetTenant(ctx context.Context, id primitive.ObjectID) (*domain.Tenant, error)
	UpdateTenant(ctx context.Context, id primitive.ObjectID, req UpdateTenantRequest) (*domain.Tenant, error)
	ListTenants(ctx context.Context, req FilterRequest) ([]*domain.Tenant, int64, error)
}

type UserService interface {
	CreateUser(ctx context.Context, req CreateUserRequest) (*domain.User, error)
	GetUser(ctx context.Context, id primitive.ObjectID) (*domain.User, error)
	UpdateUser(ctx context.Context, id primitive.ObjectID, req UpdateUserRequest) (*domain.User, error)
	ListUsers(ctx context.Context, req FilterRequest) ([]*domain.User, int64, error)
}

type AuthService interface {
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
}

type AddPermissionRequest struct {
	Role   string `json:"role"`
	Path   string `json:"path"`
	Method string `json:"method"`
}

type RemovePermissionRequest struct {
	Role   string `json:"role"`
	Path   string `json:"path"`
	Method string `json:"method"`
}

type AssignRoleInheritanceRequest struct {
	ChildRole  string `json:"child_role"`
	ParentRole string `json:"parent_role"`
}

type PermissionService interface {
	AddPermission(ctx context.Context, req AddPermissionRequest) error
	RemovePermission(ctx context.Context, req RemovePermissionRequest) error
	AssignRoleInheritance(ctx context.Context, req AssignRoleInheritanceRequest) error
	GetAllPermissions(ctx context.Context) ([][]string, error)
	GetRoleInheritances(ctx context.Context) ([][]string, error)
	CheckPermission(ctx context.Context, role, obj, act string) bool
}

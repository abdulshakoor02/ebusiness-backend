package ports

import (
	"context"
	"time"

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
	Mobile   string `json:"mobile"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type UpdateUserRequest struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Mobile string `json:"mobile"`
	Role   string `json:"role"`
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

type CreatePermissionRuleRequest struct {
	Resource      string `json:"resource"`
	ResourceLabel string `json:"resource_label"`
	Action        string `json:"action"`
	ActionLabel   string `json:"action_label"`
	Path          string `json:"path"`
	Method        string `json:"method"`
	Description   string `json:"description"`
}

type UpdatePermissionRuleRequest struct {
	ResourceLabel string `json:"resource_label,omitempty"`
	ActionLabel   string `json:"action_label,omitempty"`
	Path          string `json:"path,omitempty"`
	Method        string `json:"method,omitempty"`
	Description   string `json:"description,omitempty"`
}

type PermissionService interface {
	AddPermission(ctx context.Context, req AddPermissionRequest) error
	RemovePermission(ctx context.Context, req RemovePermissionRequest) error
	AssignRoleInheritance(ctx context.Context, req AssignRoleInheritanceRequest) error
	GetAllPermissions(ctx context.Context) ([][]string, error)
	GetRoleInheritances(ctx context.Context) ([][]string, error)
	CheckPermission(ctx context.Context, role, obj, act string) bool

	// Permission Rule Management
	CreatePermissionRule(ctx context.Context, req CreatePermissionRuleRequest) (*domain.PermissionRule, error)
	UpdatePermissionRule(ctx context.Context, id primitive.ObjectID, req UpdatePermissionRuleRequest) (*domain.PermissionRule, error)
	DeletePermissionRule(ctx context.Context, id primitive.ObjectID) error
	GetPermissionRuleByID(ctx context.Context, id primitive.ObjectID) (*domain.PermissionRule, error)
	GetAvailableRulesGrouped(ctx context.Context) ([]domain.PermissionRuleGroup, error)
}

type CreateLeadRequest struct {
	CategoryID string `json:"category_id,omitempty"`
	SourceID   string `json:"source_id,omitempty"`
	AssignedTo string `json:"assigned_to,omitempty"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Company    string `json:"company"`
	Title      string `json:"title"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Status     string `json:"status"`
}

type UpdateLeadRequest struct {
	CategoryID string `json:"category_id,omitempty"`
	SourceID   string `json:"source_id,omitempty"`
	AssignedTo string `json:"assigned_to,omitempty"`
	FirstName  string `json:"first_name,omitempty"`
	LastName   string `json:"last_name,omitempty"`
	Company    string `json:"company,omitempty"`
	Title      string `json:"title,omitempty"`
	Email      string `json:"email,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Status     string `json:"status,omitempty"`
}

type LeadService interface {
	CreateLead(ctx context.Context, req CreateLeadRequest) (*domain.Lead, error)
	GetLead(ctx context.Context, id primitive.ObjectID) (*domain.Lead, error)
	UpdateLead(ctx context.Context, id primitive.ObjectID, req UpdateLeadRequest) (*domain.Lead, error)
	ListLeads(ctx context.Context, req FilterRequest) ([]*domain.Lead, int64, error)
}

type CreateLeadCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateLeadCategoryRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type LeadCategoryService interface {
	CreateLeadCategory(ctx context.Context, req CreateLeadCategoryRequest) (*domain.LeadCategory, error)
	GetLeadCategory(ctx context.Context, id primitive.ObjectID) (*domain.LeadCategory, error)
	UpdateLeadCategory(ctx context.Context, id primitive.ObjectID, req UpdateLeadCategoryRequest) (*domain.LeadCategory, error)
	DeleteLeadCategory(ctx context.Context, id primitive.ObjectID) error
	ListLeadCategories(ctx context.Context, req FilterRequest) ([]*domain.LeadCategory, int64, error)
}

type CreateLeadSourceRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdateLeadSourceRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type LeadSourceService interface {
	CreateLeadSource(ctx context.Context, req CreateLeadSourceRequest) (*domain.LeadSource, error)
	GetLeadSource(ctx context.Context, id primitive.ObjectID) (*domain.LeadSource, error)
	UpdateLeadSource(ctx context.Context, id primitive.ObjectID, req UpdateLeadSourceRequest) (*domain.LeadSource, error)
	DeleteLeadSource(ctx context.Context, id primitive.ObjectID) error
	ListLeadSources(ctx context.Context, req FilterRequest) ([]*domain.LeadSource, int64, error)
}

type CreateLeadCommentRequest struct {
	Content string `json:"content"`
}

type UpdateLeadCommentRequest struct {
	Content string `json:"content"`
}

type LeadCommentService interface {
	CreateLeadComment(ctx context.Context, leadID primitive.ObjectID, req CreateLeadCommentRequest) (*domain.LeadComment, error)
	GetLeadComment(ctx context.Context, id primitive.ObjectID) (*domain.LeadComment, error)
	UpdateLeadComment(ctx context.Context, id primitive.ObjectID, req UpdateLeadCommentRequest) (*domain.LeadComment, error)
	DeleteLeadComment(ctx context.Context, id primitive.ObjectID) error
	ListLeadComments(ctx context.Context, leadID primitive.ObjectID, req FilterRequest) ([]*domain.LeadComment, int64, error)
}

type CreateLeadAppointmentRequest struct {
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Status      string    `json:"status"` // using string representation for JSON decoding convenience
}

type UpdateLeadAppointmentRequest struct {
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	StartTime   time.Time `json:"start_time,omitempty"`
	EndTime     time.Time `json:"end_time,omitempty"`
	Status      string    `json:"status,omitempty"`
}

type LeadAppointmentService interface {
	CreateLeadAppointment(ctx context.Context, leadID primitive.ObjectID, req CreateLeadAppointmentRequest) (*domain.LeadAppointment, error)
	GetLeadAppointment(ctx context.Context, id primitive.ObjectID) (*domain.LeadAppointment, error)
	UpdateLeadAppointment(ctx context.Context, id primitive.ObjectID, req UpdateLeadAppointmentRequest) (*domain.LeadAppointment, error)
	DeleteLeadAppointment(ctx context.Context, id primitive.ObjectID) error
	ListLeadAppointments(ctx context.Context, leadID primitive.ObjectID, req FilterRequest) ([]*domain.LeadAppointment, int64, error)
}

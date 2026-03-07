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

type LeadRepository interface {
	Create(ctx context.Context, lead *domain.Lead) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Lead, error)
	List(ctx context.Context, filter interface{}, search string, offset, limit int64) ([]*LeadListItem, int64, error)
	Update(ctx context.Context, lead *domain.Lead) error
}

type LeadCategoryRepository interface {
	Create(ctx context.Context, category *domain.LeadCategory) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.LeadCategory, error)
	List(ctx context.Context, filter interface{}, offset, limit int64) ([]*domain.LeadCategory, int64, error)
	Update(ctx context.Context, category *domain.LeadCategory) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

type LeadCommentRepository interface {
	Create(ctx context.Context, comment *domain.LeadComment) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.LeadComment, error)
	ListByLeadID(ctx context.Context, leadID primitive.ObjectID, filter interface{}, offset, limit int64) ([]*domain.LeadComment, int64, error)
	Update(ctx context.Context, comment *domain.LeadComment) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

type LeadAppointmentRepository interface {
	Create(ctx context.Context, appointment *domain.LeadAppointment) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.LeadAppointment, error)
	ListByLeadID(ctx context.Context, leadID primitive.ObjectID, filter interface{}, offset, limit int64) ([]*domain.LeadAppointment, int64, error)
	Update(ctx context.Context, appointment *domain.LeadAppointment) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

type LeadSourceRepository interface {
	Create(ctx context.Context, source *domain.LeadSource) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.LeadSource, error)
	List(ctx context.Context, filter interface{}, offset, limit int64) ([]*domain.LeadSource, int64, error)
	Update(ctx context.Context, source *domain.LeadSource) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

type PermissionRuleRepository interface {
	Create(ctx context.Context, rule *domain.PermissionRule) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.PermissionRule, error)
	GetByResourceAndAction(ctx context.Context, resource, action string) (*domain.PermissionRule, error)
	ListAll(ctx context.Context) ([]*domain.PermissionRule, error)
	ListByResource(ctx context.Context, resource string) ([]*domain.PermissionRule, error)
	Update(ctx context.Context, rule *domain.PermissionRule) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

type RolePermissionRepository interface {
	Assign(ctx context.Context, role string, permissionRuleID primitive.ObjectID) error
	Remove(ctx context.Context, role string, permissionRuleID primitive.ObjectID) error
	HasPermissionByRuleID(ctx context.Context, role string, permissionRuleID primitive.ObjectID) (bool, error)
	GetByRole(ctx context.Context, role string) ([]*domain.RolePermission, error)
	GetByRoleWithDetails(ctx context.Context, role string) ([]*domain.PermissionRule, error)
	GetAll(ctx context.Context) ([]*domain.RolePermission, error)
	GetRoleInheritances(ctx context.Context) ([]*domain.RoleInheritance, error)
	AssignInheritance(ctx context.Context, childRole, parentRole string) error
	RemoveInheritance(ctx context.Context, childRole, parentRole string) error
	GetInheritedPermissions(ctx context.Context, role string) (map[primitive.ObjectID]bool, error)
	GetPermissionRulesForInheritedRoles(ctx context.Context, role string) ([]*domain.PermissionRule, error)
	CheckPermissionWithInheritance(ctx context.Context, role string, permissionRuleID primitive.ObjectID) (bool, error)
	CheckPermissionByPathMethod(ctx context.Context, role, path, method string) (bool, error)
	InvalidateAllCache()
}

type QualificationRepository interface {
	Create(ctx context.Context, qualification *domain.Qualification) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Qualification, error)
	List(ctx context.Context, filter interface{}, offset, limit int64) ([]*domain.Qualification, int64, error)
	Update(ctx context.Context, qualification *domain.Qualification) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

type CountryRepository interface {
	Create(ctx context.Context, country *domain.Country) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Country, error)
	List(ctx context.Context, filter interface{}, offset, limit int64) ([]*domain.Country, int64, error)
	Update(ctx context.Context, country *domain.Country) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

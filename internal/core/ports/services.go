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
	StampURL  string            `json:"stamp_url"`
	Address   domain.Address    `json:"address"`
	CountryID string            `json:"country_id"`
	Tax       float64           `json:"tax"`
	AdminUser CreateUserRequest `json:"admin_user"`
}

type UpdateTenantRequest struct {
	Name      string         `json:"name"`
	Email     string         `json:"email"`
	LogoURL   string         `json:"logo_url"`
	StampURL  string         `json:"stamp_url"`
	Address   domain.Address `json:"address"`
	CountryID string         `json:"country_id"`
	Tax       float64        `json:"tax"`
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
	Token           string       `json:"token"`
	User            *domain.User `json:"user"`
	Tax             float64      `json:"tax,omitempty"`
	Currency        string       `json:"currency,omitempty"`
	NextCloudFolder string       `json:"next_cloud_folder,omitempty"`
}

type FilterRequest struct {
	Filters map[string]interface{} `json:"filters"`
	Search  string                 `json:"search"`
	Offset  int64                  `json:"offset"`
	Limit   int64                  `json:"limit"`
}

type TenantService interface {
	RegisterTenant(ctx context.Context, req CreateTenantRequest) (*domain.Tenant, *domain.User, error)
	GetTenant(ctx context.Context, id primitive.ObjectID) (*domain.Tenant, error)
	UpdateTenant(ctx context.Context, id primitive.ObjectID, req UpdateTenantRequest) (*domain.Tenant, error)
	UpdateMyTenant(ctx context.Context, id primitive.ObjectID, req UpdateTenantRequest) (*domain.Tenant, error)
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
	Role             string             `json:"role"`
	PermissionRuleID primitive.ObjectID `json:"permission_rule_id"`
}

type RemovePermissionRequest struct {
	Role             string             `json:"role"`
	PermissionRuleID primitive.ObjectID `json:"permission_rule_id"`
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
	ScopeType     string `json:"scope_type"`
	FilterField   string `json:"filter_field"`
}

type UpdatePermissionRuleRequest struct {
	ResourceLabel string `json:"resource_label,omitempty"`
	ActionLabel   string `json:"action_label,omitempty"`
	Path          string `json:"path,omitempty"`
	Method        string `json:"method,omitempty"`
	Description   string `json:"description,omitempty"`
	ScopeType     string `json:"scope_type,omitempty"`
	FilterField   string `json:"filter_field,omitempty"`
}

type RolePermission struct {
	domain.PermissionRule
	Assigned bool `json:"assigned"`
}

type RolePermissionGroup struct {
	Resource string           `json:"resource"`
	Label    string           `json:"label"`
	Rules    []RolePermission `json:"rules"`
}

type BulkUpdateRolePermissionsRequest struct {
	Permissions []RolePermission `json:"permissions"`
}

type BulkPermissionItem struct {
	Path     string `json:"path"`
	Method   string `json:"method"`
	Assigned bool   `json:"assigned"`
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
	GetAvailableRulesGrouped(ctx context.Context, role string) ([]domain.PermissionRuleGroup, error)

	// UI-Friendly Role Permissions
	GetPermissionsForRoleGrouped(ctx context.Context, role string) ([]RolePermissionGroup, error)
	BulkUpdateRolePermissions(ctx context.Context, role string, req BulkUpdateRolePermissionsRequest) error

	// Dynamic Permission Checking
	GetAllPermissionsForRole(ctx context.Context, role string) (map[string]bool, error)

	// Role Management
	GetAllRoles(ctx context.Context, currentUserRole string) ([]string, error)

	// Raw permission rules for scope configuration
	GetAllPermissionRules(ctx context.Context) ([]*domain.PermissionRule, error)

	// Debug - bypass cache
	DebugGetRolePermissionsFromDB(ctx context.Context, role string) ([]*domain.RolePermission, error)
}

type CreateLeadRequest struct {
	CategoryID      string         `json:"category_id,omitempty"`
	SourceID        string         `json:"source_id,omitempty"`
	AssignedTo      string         `json:"assigned_to,omitempty"`
	CountryID       string         `json:"country_id,omitempty"`
	QualificationID string         `json:"qualification_id,omitempty"`
	FirstName       string         `json:"first_name"`
	LastName        string         `json:"last_name"`
	Designation     string         `json:"designation,omitempty"`
	Email           string         `json:"email"`
	Phone           string         `json:"phone"`
	Address         domain.Address `json:"address,omitempty"`
}

type UpdateLeadRequest struct {
	CategoryID      string         `json:"category_id,omitempty"`
	SourceID        string         `json:"source_id,omitempty"`
	AssignedTo      string         `json:"assigned_to,omitempty"`
	CountryID       string         `json:"country_id,omitempty"`
	QualificationID string         `json:"qualification_id,omitempty"`
	FirstName       string         `json:"first_name,omitempty"`
	LastName        string         `json:"last_name,omitempty"`
	Designation     string         `json:"designation,omitempty"`
	Email           string         `json:"email,omitempty"`
	Phone           string         `json:"phone,omitempty"`
	Address         domain.Address `json:"address,omitempty"`
}

// LeadRefItem is a lightweight reference to a related entity (category, source, qualification)
type LeadRefItem struct {
	ID   primitive.ObjectID `json:"id" bson:"_id"`
	Name string             `json:"name" bson:"name"`
}

// LeadUserRef is a lightweight reference to the assigned user
type LeadUserRef struct {
	ID   primitive.ObjectID `json:"id" bson:"_id"`
	Name string             `json:"name" bson:"name"`
}

// LeadCountryRef is a lightweight reference to a country with additional display fields
type LeadCountryRef struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	Name      string             `json:"name" bson:"name"`
	ISO2      string             `json:"iso2" bson:"iso2"`
	PhoneCode string             `json:"phone_code" bson:"phone_code"`
}

// LeadListItem is the enriched response for the list endpoint with resolved references
type LeadListItem struct {
	ID             primitive.ObjectID `json:"id" bson:"_id"`
	TenantID       primitive.ObjectID `json:"tenant_id" bson:"tenant_id"`
	FirstName      string             `json:"first_name" bson:"first_name"`
	LastName       string             `json:"last_name" bson:"last_name"`
	Designation    string             `json:"designation,omitempty" bson:"designation"`
	Email          string             `json:"email" bson:"email"`
	Phone          string             `json:"phone" bson:"phone"`
	Category       *LeadRefItem       `json:"category,omitempty" bson:"category,omitempty"`
	Source         *LeadRefItem       `json:"source,omitempty" bson:"source,omitempty"`
	AssignedToUser *LeadUserRef       `json:"assigned_to_user,omitempty" bson:"assigned_to_user,omitempty"`
	Country        *LeadCountryRef    `json:"country,omitempty" bson:"country,omitempty"`
	Qualification  *LeadRefItem       `json:"qualification,omitempty" bson:"qualification,omitempty"`
	Status         string             `json:"status" bson:"status"`
	ConvertedAt    *time.Time         `json:"converted_at,omitempty" bson:"converted_at,omitempty"`
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" bson:"updated_at"`
}

type UpdateLeadStatusRequest struct {
	Status string `json:"status"`
}

type LeadService interface {
	CreateLead(ctx context.Context, req CreateLeadRequest) (*domain.Lead, error)
	GetLead(ctx context.Context, id primitive.ObjectID) (*domain.Lead, error)
	UpdateLead(ctx context.Context, id primitive.ObjectID, req UpdateLeadRequest) (*domain.Lead, error)
	UpdateLeadStatus(ctx context.Context, id primitive.ObjectID, req UpdateLeadStatusRequest) (*domain.Lead, error)
	ListLeads(ctx context.Context, req FilterRequest) ([]*LeadListItem, int64, error)
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

// CommentListItem is the enriched response for the comment list endpoint with resolved author
type CommentListItem struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	TenantID  primitive.ObjectID `json:"tenant_id" bson:"tenant_id"`
	LeadID    primitive.ObjectID `json:"lead_id" bson:"lead_id"`
	Author    *LeadUserRef       `json:"author,omitempty" bson:"author,omitempty"`
	Content   string             `json:"content" bson:"content"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

type LeadCommentService interface {
	CreateLeadComment(ctx context.Context, leadID primitive.ObjectID, req CreateLeadCommentRequest) (*domain.LeadComment, error)
	GetLeadComment(ctx context.Context, id primitive.ObjectID) (*domain.LeadComment, error)
	UpdateLeadComment(ctx context.Context, id primitive.ObjectID, req UpdateLeadCommentRequest) (*domain.LeadComment, error)
	DeleteLeadComment(ctx context.Context, id primitive.ObjectID) error
	ListLeadComments(ctx context.Context, leadID primitive.ObjectID, req FilterRequest) ([]*CommentListItem, int64, error)
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

// AppointmentListItem is the enriched response for the appointment list endpoint with resolved organizer
type AppointmentListItem struct {
	ID          primitive.ObjectID       `json:"id" bson:"_id"`
	TenantID    primitive.ObjectID       `json:"tenant_id" bson:"tenant_id"`
	LeadID      primitive.ObjectID       `json:"lead_id" bson:"lead_id"`
	Organizer   *LeadUserRef             `json:"organizer,omitempty" bson:"organizer,omitempty"`
	Title       string                   `json:"title" bson:"title"`
	Description string                   `json:"description" bson:"description"`
	StartTime   time.Time                `json:"start_time" bson:"start_time"`
	EndTime     time.Time                `json:"end_time" bson:"end_time"`
	Status      domain.AppointmentStatus `json:"status" bson:"status"`
	CreatedAt   time.Time                `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at" bson:"updated_at"`
}

type LeadAppointmentService interface {
	CreateLeadAppointment(ctx context.Context, leadID primitive.ObjectID, req CreateLeadAppointmentRequest) (*domain.LeadAppointment, error)
	GetLeadAppointment(ctx context.Context, id primitive.ObjectID) (*domain.LeadAppointment, error)
	UpdateLeadAppointment(ctx context.Context, id primitive.ObjectID, req UpdateLeadAppointmentRequest) (*domain.LeadAppointment, error)
	DeleteLeadAppointment(ctx context.Context, id primitive.ObjectID) error
	ListLeadAppointments(ctx context.Context, leadID primitive.ObjectID, req FilterRequest) ([]*AppointmentListItem, int64, error)
}

type CreateQualificationRequest struct {
	Name string `json:"name"`
}

type UpdateQualificationRequest struct {
	Name     string `json:"name,omitempty"`
	IsActive *bool  `json:"is_active,omitempty"`
}

type QualificationService interface {
	CreateQualification(ctx context.Context, req CreateQualificationRequest) (*domain.Qualification, error)
	GetQualification(ctx context.Context, id primitive.ObjectID) (*domain.Qualification, error)
	UpdateQualification(ctx context.Context, id primitive.ObjectID, req UpdateQualificationRequest) (*domain.Qualification, error)
	DeleteQualification(ctx context.Context, id primitive.ObjectID) error
	ListQualifications(ctx context.Context, req FilterRequest) ([]*domain.Qualification, int64, error)
}

type CreateCountryRequest struct {
	Name         string `json:"name"`
	ISO2         string `json:"iso2"`
	ISO3         string `json:"iso3"`
	PhoneCode    string `json:"phone_code"`
	Currency     string `json:"currency"`
	CurrencyName string `json:"currency_name"`
}

type UpdateCountryRequest struct {
	Name         string `json:"name,omitempty"`
	ISO2         string `json:"iso2,omitempty"`
	ISO3         string `json:"iso3,omitempty"`
	PhoneCode    string `json:"phone_code,omitempty"`
	Currency     string `json:"currency,omitempty"`
	CurrencyName string `json:"currency_name,omitempty"`
	IsActive     *bool  `json:"is_active,omitempty"`
}

type CountryService interface {
	CreateCountry(ctx context.Context, req CreateCountryRequest) (*domain.Country, error)
	GetCountry(ctx context.Context, id primitive.ObjectID) (*domain.Country, error)
	UpdateCountry(ctx context.Context, id primitive.ObjectID, req UpdateCountryRequest) (*domain.Country, error)
	DeleteCountry(ctx context.Context, id primitive.ObjectID) error
	ListCountries(ctx context.Context, req FilterRequest) ([]*domain.Country, int64, error)
}

type CreateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type UpdateProductRequest struct {
	Name        string  `json:"name,omitempty"`
	Description string  `json:"description,omitempty"`
	Price       float64 `json:"price,omitempty"`
}

type ProductService interface {
	CreateProduct(ctx context.Context, req CreateProductRequest) (*domain.Product, error)
	GetProduct(ctx context.Context, id primitive.ObjectID) (*domain.Product, error)
	UpdateProduct(ctx context.Context, id primitive.ObjectID, req UpdateProductRequest) (*domain.Product, error)
	DeleteProduct(ctx context.Context, id primitive.ObjectID) error
	ListProducts(ctx context.Context, req FilterRequest) ([]*domain.Product, int64, error)
}

type CreateInvoiceItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type CreateInvoiceRequest struct {
	LeadID   string                     `json:"lead_id"`
	Items    []CreateInvoiceItemRequest `json:"items"`
	Discount float64                    `json:"discount"`
	DueDate  *time.Time                 `json:"due_date,omitempty"`
}

type UpdateInvoiceDueDateRequest struct {
	DueDate time.Time `json:"due_date"`
}

type UpdateInvoiceRequest struct {
	Items    []CreateInvoiceItemRequest `json:"items,omitempty"`
	Discount float64                    `json:"discount,omitempty"`
	DueDate  *time.Time                 `json:"due_date,omitempty"`
}

type InvoiceService interface {
	CreateInvoice(ctx context.Context, req CreateInvoiceRequest) (*domain.Invoice, error)
	GetInvoice(ctx context.Context, id primitive.ObjectID) (*domain.Invoice, error)
	UpdateInvoice(ctx context.Context, id primitive.ObjectID, req UpdateInvoiceRequest) (*domain.Invoice, error)
	UpdateInvoiceDueDate(ctx context.Context, id primitive.ObjectID, req UpdateInvoiceDueDateRequest) (*domain.Invoice, error)
	ListInvoices(ctx context.Context, req FilterRequest) ([]*domain.Invoice, int64, error)
	GetInvoicesByLeadID(ctx context.Context, leadID primitive.ObjectID) ([]*domain.Invoice, error)
}

type CreateReceiptRequest struct {
	AmountPaid  float64   `json:"amount_paid"`
	PaymentDate time.Time `json:"payment_date"`
}

type UpdateReceiptRequest struct {
	AmountPaid  float64   `json:"amount_paid,omitempty"`
	PaymentDate time.Time `json:"payment_date,omitempty"`
}

type ReceiptService interface {
	CreateReceipt(ctx context.Context, invoiceID primitive.ObjectID, req CreateReceiptRequest) (*domain.Receipt, error)
	GetReceipt(ctx context.Context, id primitive.ObjectID) (*domain.Receipt, error)
	UpdateReceipt(ctx context.Context, id primitive.ObjectID, req UpdateReceiptRequest) (*domain.Receipt, error)
	DeleteReceipt(ctx context.Context, id primitive.ObjectID) error
	ListReceiptsByInvoiceID(ctx context.Context, invoiceID primitive.ObjectID) ([]*domain.Receipt, error)
}

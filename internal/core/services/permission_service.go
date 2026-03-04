package services

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/abdulshakoor02/goCrmBackend/pkg/middleware"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PermissionService struct {
	permissionRuleRepo ports.PermissionRuleRepository
	rolePermissionRepo ports.RolePermissionRepository
}

func NewPermissionService(permissionRuleRepo ports.PermissionRuleRepository, rolePermissionRepo ports.RolePermissionRepository) *PermissionService {
	return &PermissionService{
		permissionRuleRepo: permissionRuleRepo,
		rolePermissionRepo: rolePermissionRepo,
	}
}

func (s *PermissionService) AddPermission(ctx context.Context, req ports.AddPermissionRequest) error {
	if req.Role == "" || req.PermissionRuleID.IsZero() {
		return errors.New("role and permission_rule_id are required")
	}

	hasPermission, err := s.rolePermissionRepo.HasPermissionByRuleID(ctx, req.Role, req.PermissionRuleID)
	if err != nil {
		return err
	}
	if hasPermission {
		return errors.New("permission already exists")
	}

	return s.rolePermissionRepo.Assign(ctx, req.Role, req.PermissionRuleID)
}

func (s *PermissionService) RemovePermission(ctx context.Context, req ports.RemovePermissionRequest) error {
	if req.Role == "" || req.PermissionRuleID.IsZero() {
		return errors.New("role and permission_rule_id are required")
	}

	hasPermission, err := s.rolePermissionRepo.HasPermissionByRuleID(ctx, req.Role, req.PermissionRuleID)
	if err != nil {
		return err
	}
	if !hasPermission {
		return errors.New("permission not found")
	}

	return s.rolePermissionRepo.Remove(ctx, req.Role, req.PermissionRuleID)
}

func (s *PermissionService) AssignRoleInheritance(ctx context.Context, req ports.AssignRoleInheritanceRequest) error {
	if req.ChildRole == "" || req.ParentRole == "" {
		return errors.New("child_role and parent_role are required")
	}

	return s.rolePermissionRepo.AssignInheritance(ctx, req.ChildRole, req.ParentRole)
}

func (s *PermissionService) GetAllPermissions(ctx context.Context) ([][]string, error) {
	rules, err := s.rolePermissionRepo.GetPermissionRulesForInheritedRoles(ctx, "admin")
	if err != nil {
		return nil, err
	}

	result := make([][]string, 0, len(rules))
	for _, r := range rules {
		result = append(result, []string{"admin", r.Path, r.Method})
	}
	return result, nil
}

func (s *PermissionService) GetRoleInheritances(ctx context.Context) ([][]string, error) {
	inheritances, err := s.rolePermissionRepo.GetRoleInheritances(ctx)
	if err != nil {
		return nil, err
	}

	result := make([][]string, 0, len(inheritances))
	for _, i := range inheritances {
		result = append(result, []string{i.ChildRole, i.ParentRole})
	}
	return result, nil
}

func (s *PermissionService) CheckPermission(ctx context.Context, role, path, method string) bool {
	ok, err := s.rolePermissionRepo.CheckPermissionByPathMethod(ctx, role, path, method)
	if err != nil {
		slog.Error("CheckPermission error", "error", err)
		return false
	}
	return ok
}

func (s *PermissionService) CreatePermissionRule(ctx context.Context, req ports.CreatePermissionRuleRequest) (*domain.PermissionRule, error) {
	if req.Resource == "" || req.Action == "" {
		return nil, errors.New("resource and action are required")
	}

	existing, _ := s.permissionRuleRepo.GetByResourceAndAction(ctx, req.Resource, req.Action)
	if existing != nil {
		slog.Debug("Permission rule already exists", "resource", req.Resource, "action", req.Action, "existing_rule", existing)
		return nil, errors.New("permission rule already exists for this resource and action")
	}

	scopeType := req.ScopeType
	if scopeType == "" {
		scopeType = "none"
	}

	slog.Debug("Creating permission rule", "resource", req.Resource, "action", req.Action, "scope_type", scopeType, "filter_field", req.FilterField)

	rule := domain.NewPermissionRule(
		req.Resource,
		req.ResourceLabel,
		req.Action,
		req.ActionLabel,
		req.Path,
		req.Method,
		req.Description,
		false,
		scopeType,
		req.FilterField,
	)

	if err := s.permissionRuleRepo.Create(ctx, rule); err != nil {
		return nil, err
	}

	// Invalidate caches and reload scope config since rules changed
	s.rolePermissionRepo.InvalidateAllCache()
	s.reloadScopeConfig()

	return rule, nil
}

func (s *PermissionService) UpdatePermissionRule(ctx context.Context, id primitive.ObjectID, req ports.UpdatePermissionRuleRequest) (*domain.PermissionRule, error) {
	rule, err := s.permissionRuleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.ResourceLabel != "" {
		rule.ResourceLabel = req.ResourceLabel
	}
	if req.ActionLabel != "" {
		rule.ActionLabel = req.ActionLabel
	}
	if req.Path != "" {
		rule.Path = req.Path
	}
	if req.Method != "" {
		rule.Method = req.Method
	}
	if req.Description != "" {
		rule.Description = req.Description
	}
	if req.ScopeType != "" {
		rule.ScopeType = req.ScopeType
	}
	if req.FilterField != "" {
		rule.FilterField = req.FilterField
	}
	rule.UpdatedAt = time.Now()

	if err := s.permissionRuleRepo.Update(ctx, rule); err != nil {
		return nil, err
	}

	// Invalidate caches and reload scope config since rules changed
	s.rolePermissionRepo.InvalidateAllCache()
	s.reloadScopeConfig()

	return rule, nil
}

func (s *PermissionService) DeletePermissionRule(ctx context.Context, id primitive.ObjectID) error {
	err := s.permissionRuleRepo.Delete(ctx, id)
	if err == nil {
		// Invalidate caches and reload scope config since rules changed
		s.rolePermissionRepo.InvalidateAllCache()
		s.reloadScopeConfig()
	}
	return err
}

// reloadScopeConfig refreshes the middleware scope config from the current permission rules
func (s *PermissionService) reloadScopeConfig() {
	rules, err := s.permissionRuleRepo.ListAll(context.Background())
	if err != nil {
		slog.Error("Failed to reload scope config after rule change", "error", err)
		return
	}
	middleware.ReloadScopeConfigFromRules(rules)
}

func (s *PermissionService) GetPermissionRuleByID(ctx context.Context, id primitive.ObjectID) (*domain.PermissionRule, error) {
	return s.permissionRuleRepo.GetByID(ctx, id)
}

func (s *PermissionService) GetAvailableRulesGrouped(ctx context.Context, role string) ([]domain.PermissionRuleGroup, error) {
	rules, err := s.permissionRuleRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	resourceMap := make(map[string]*domain.PermissionRuleGroup)
	for _, rule := range rules {
		if rule.RequiresRole != "" && rule.RequiresRole != role {
			continue
		}

		group, exists := resourceMap[rule.Resource]
		if !exists {
			group = &domain.PermissionRuleGroup{
				Resource: rule.Resource,
				Label:    rule.ResourceLabel,
				Rules:    []domain.PermissionRule{},
			}
			resourceMap[rule.Resource] = group
		}
		group.Rules = append(group.Rules, *rule)
	}

	groups := make([]domain.PermissionRuleGroup, 0, len(resourceMap))
	for _, group := range resourceMap {
		groups = append(groups, *group)
	}

	return groups, nil
}

func (s *PermissionService) GetAllPermissionsForRole(ctx context.Context, role string) (map[string]bool, error) {
	rules, err := s.permissionRuleRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	// Get all assigned permission rule IDs for this role (including inherited)
	assignedRuleIDs, err := s.rolePermissionRepo.GetInheritedPermissions(ctx, role)
	if err != nil {
		return nil, err
	}

	permissions := make(map[string]bool)
	for _, rule := range rules {
		if rule.RequiresRole != "" && rule.RequiresRole != role {
			continue
		}
		key := "can_" + rule.Action + "_" + rule.Resource
		permissions[key] = assignedRuleIDs[rule.ID]
	}

	return permissions, nil
}

func (s *PermissionService) GetPermissionsForRoleGrouped(ctx context.Context, role string) ([]ports.RolePermissionGroup, error) {
	rules, err := s.permissionRuleRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	// Get all assigned permission rule IDs for this role (including inherited)
	assignedRuleIDs, err := s.rolePermissionRepo.GetInheritedPermissions(ctx, role)
	if err != nil {
		return nil, err
	}

	// Debug: log the raw DB state
	directPerms, _ := s.rolePermissionRepo.GetByRole(ctx, role)
	slog.Debug("GetPermissionsForRoleGrouped - direct role_permissions from DB",
		"role", role,
		"direct_count", len(directPerms))
	for _, dp := range directPerms {
		slog.Debug("  direct permission", "rule_id", dp.PermissionRuleID.Hex())
	}
	slog.Debug("GetPermissionsForRoleGrouped - resolved assigned IDs (incl. inherited)",
		"role", role,
		"total_assigned", len(assignedRuleIDs))

	resourceMap := make(map[string]*ports.RolePermissionGroup)
	for _, rule := range rules {
		if rule.RequiresRole != "" && rule.RequiresRole != role {
			continue
		}

		group, exists := resourceMap[rule.Resource]
		if !exists {
			group = &ports.RolePermissionGroup{
				Resource: rule.Resource,
				Label:    rule.ResourceLabel,
				Rules:    []ports.RolePermission{},
			}
			resourceMap[rule.Resource] = group
		}

		rolePerm := ports.RolePermission{
			PermissionRule: *rule,
			Assigned:       assignedRuleIDs[rule.ID],
		}

		group.Rules = append(group.Rules, rolePerm)
	}

	groups := make([]ports.RolePermissionGroup, 0, len(resourceMap))
	for _, group := range resourceMap {
		groups = append(groups, *group)
	}

	return groups, nil
}

func (s *PermissionService) BulkUpdateRolePermissions(ctx context.Context, role string, req ports.BulkUpdateRolePermissionsRequest) error {
	if role == "" {
		return errors.New("role is required")
	}

	for _, perm := range req.Permissions {
		if perm.ID.IsZero() {
			continue
		}

		if perm.Assigned {
			if err := s.rolePermissionRepo.Assign(ctx, role, perm.ID); err != nil {
				slog.Error("Failed to assign permission", "role", role, "permission_rule_id", perm.ID, "error", err)
			}
		} else {
			if err := s.rolePermissionRepo.Remove(ctx, role, perm.ID); err != nil {
				slog.Error("Failed to remove permission", "role", role, "permission_rule_id", perm.ID, "error", err)
			}
		}
	}

	return nil
}

func (s *PermissionService) GetAllRoles(ctx context.Context, currentUserRole string) ([]string, error) {
	permissions, err := s.rolePermissionRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	roleSet := make(map[string]bool)
	for _, p := range permissions {
		roleSet[p.Role] = true
	}

	roles := make([]string, 0, len(roleSet))
	for role := range roleSet {
		// Hide 'superadmin' role from non-superadmin users
		if role == "superadmin" && currentUserRole != "superadmin" {
			continue
		}
		roles = append(roles, role)
	}

	return roles, nil
}

func (s *PermissionService) GetAllPermissionRules(ctx context.Context) ([]*domain.PermissionRule, error) {
	return s.permissionRuleRepo.ListAll(ctx)
}

func (s *PermissionService) DebugGetRolePermissionsFromDB(ctx context.Context, role string) ([]*domain.RolePermission, error) {
	return s.rolePermissionRepo.GetByRole(ctx, role)
}

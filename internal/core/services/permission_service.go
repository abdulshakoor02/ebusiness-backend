package services

import (
	"context"
	"errors"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/casbin/casbin/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PermissionService struct {
	enforcer *casbin.Enforcer
	repo     ports.PermissionRuleRepository
}

func NewPermissionService(enforcer *casbin.Enforcer, repo ports.PermissionRuleRepository) *PermissionService {
	return &PermissionService{
		enforcer: enforcer,
		repo:     repo,
	}
}

func (s *PermissionService) AddPermission(ctx context.Context, req ports.AddPermissionRequest) error {
	if req.Role == "" || req.Path == "" || req.Method == "" {
		return errors.New("role, path, and method are required")
	}

	success, err := s.enforcer.AddPolicy(req.Role, req.Path, req.Method)
	if err != nil {
		return err
	}
	if !success {
		return errors.New("policy already exists")
	}
	return nil
}

func (s *PermissionService) RemovePermission(ctx context.Context, req ports.RemovePermissionRequest) error {
	if req.Role == "" || req.Path == "" || req.Method == "" {
		return errors.New("role, path, and method are required")
	}

	success, err := s.enforcer.RemovePolicy(req.Role, req.Path, req.Method)
	if err != nil {
		return err
	}
	if !success {
		return errors.New("policy not found")
	}
	return nil
}

func (s *PermissionService) AssignRoleInheritance(ctx context.Context, req ports.AssignRoleInheritanceRequest) error {
	if req.ChildRole == "" || req.ParentRole == "" {
		return errors.New("child_role and parent_role are required")
	}

	success, err := s.enforcer.AddGroupingPolicy(req.ChildRole, req.ParentRole)
	if err != nil {
		return err
	}
	if !success {
		return errors.New("inheritance already exists")
	}
	return nil
}

func (s *PermissionService) GetAllPermissions(ctx context.Context) ([][]string, error) {
	return s.enforcer.GetPolicy()
}

func (s *PermissionService) GetRoleInheritances(ctx context.Context) ([][]string, error) {
	return s.enforcer.GetGroupingPolicy()
}

func (s *PermissionService) CheckPermission(ctx context.Context, role, obj, act string) bool {
	ok, err := s.enforcer.Enforce(role, obj, act)
	if err != nil {
		return false
	}
	return ok
}

func (s *PermissionService) CreatePermissionRule(ctx context.Context, req ports.CreatePermissionRuleRequest) (*domain.PermissionRule, error) {
	if req.Resource == "" || req.Action == "" {
		return nil, errors.New("resource and action are required")
	}

	existing, _ := s.repo.GetByResourceAndAction(ctx, req.Resource, req.Action)
	if existing != nil {
		return nil, errors.New("permission rule already exists for this resource and action")
	}

	rule := domain.NewPermissionRule(
		req.Resource,
		req.ResourceLabel,
		req.Action,
		req.ActionLabel,
		req.Path,
		req.Method,
		req.Description,
		false, // Custom rules are not system rules
	)

	if err := s.repo.Create(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

func (s *PermissionService) UpdatePermissionRule(ctx context.Context, id primitive.ObjectID, req ports.UpdatePermissionRuleRequest) (*domain.PermissionRule, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Only allow updating labels, path, method, and description
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
	rule.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

func (s *PermissionService) DeletePermissionRule(ctx context.Context, id primitive.ObjectID) error {
	return s.repo.Delete(ctx, id)
}

func (s *PermissionService) GetPermissionRuleByID(ctx context.Context, id primitive.ObjectID) (*domain.PermissionRule, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *PermissionService) GetAvailableRulesGrouped(ctx context.Context) ([]domain.PermissionRuleGroup, error) {
	rules, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	// Group rules by resource
	resourceMap := make(map[string]*domain.PermissionRuleGroup)

	for _, rule := range rules {
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

	// Convert map to slice
	groups := make([]domain.PermissionRuleGroup, 0, len(resourceMap))
	for _, group := range resourceMap {
		groups = append(groups, *group)
	}

	return groups, nil
}

func (s *PermissionService) GetAllPermissionsForRole(ctx context.Context, role string) (map[string]bool, error) {
	rules, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	permissions := make(map[string]bool)
	for _, rule := range rules {
		// Create permission key: can_{action}_{resource}
		key := "can_" + rule.Action + "_" + rule.Resource

		// Check permission using Casbin
		// For frontend-only rules (empty path/method), check if role has any policy with this resource
		if rule.Path == "" || rule.Method == "" {
			// Check if role name matches resource pattern for custom permissions
			// This is a simplified check - in production you might want more sophisticated logic
			permissions[key] = false // Custom rules need to be explicitly assigned
		} else {
			permissions[key] = s.CheckPermission(ctx, role, rule.Path, rule.Method)
		}
	}

	return permissions, nil
}

package services

import (
	"context"
	"errors"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/casbin/casbin/v2"
)

type PermissionService struct {
	enforcer *casbin.Enforcer
}

func NewPermissionService(enforcer *casbin.Enforcer) *PermissionService {
	return &PermissionService{enforcer: enforcer}
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
	// Returns a 2D string slice of all 'p' rules
	return s.enforcer.GetPolicy()
}

func (s *PermissionService) GetRoleInheritances(ctx context.Context) ([][]string, error) {
	// Returns a 2D string slice of all 'g' rules
	return s.enforcer.GetGroupingPolicy()
}

func (s *PermissionService) CheckPermission(ctx context.Context, role, obj, act string) bool {
	ok, err := s.enforcer.Enforce(role, obj, act)
	if err != nil {
		return false
	}
	return ok
}

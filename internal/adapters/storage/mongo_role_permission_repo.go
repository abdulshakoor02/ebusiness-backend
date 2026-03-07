package storage

import (
	"context"
	"strings"
	"sync"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRolePermissionRepository struct {
	rolePermissionsCollection  *mongo.Collection
	roleInheritancesCollection *mongo.Collection
	permissionRuleCollection   *mongo.Collection

	// On-demand cache: only invalidated when permissions are modified
	mu              sync.RWMutex
	permissionCache map[string][]*domain.PermissionRule
}

func NewMongoRolePermissionRepository(db *mongo.Database) *MongoRolePermissionRepository {
	return &MongoRolePermissionRepository{
		rolePermissionsCollection:  db.Collection("role_permissions"),
		roleInheritancesCollection: db.Collection("role_inheritances"),
		permissionRuleCollection:   db.Collection("permission_rules"),
		permissionCache:            make(map[string][]*domain.PermissionRule),
	}
}

func (r *MongoRolePermissionRepository) Assign(ctx context.Context, role string, permissionRuleID primitive.ObjectID) error {
	filter := bson.M{"role": role, "permission_rule_id": permissionRuleID}
	update := bson.M{"$setOnInsert": bson.M{
		"role":               role,
		"permission_rule_id": permissionRuleID,
		"created_at":         primitive.NewObjectID().Timestamp(),
	}}

	_, err := r.rolePermissionsCollection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err == nil {
		r.invalidateCache(role)
	}
	return err
}

func (r *MongoRolePermissionRepository) Remove(ctx context.Context, role string, permissionRuleID primitive.ObjectID) error {
	filter := bson.M{"role": role, "permission_rule_id": permissionRuleID}
	_, err := r.rolePermissionsCollection.DeleteOne(ctx, filter)
	if err == nil {
		r.invalidateCache(role)
	}
	return err
}

func (r *MongoRolePermissionRepository) invalidateCache(role string) {
	r.mu.Lock()
	delete(r.permissionCache, role)
	r.mu.Unlock()
}

// InvalidateAllCache clears the entire permission cache.
// Should be called when permission rules are created, updated, or deleted.
func (r *MongoRolePermissionRepository) InvalidateAllCache() {
	r.mu.Lock()
	r.permissionCache = make(map[string][]*domain.PermissionRule)
	r.mu.Unlock()
}

func (r *MongoRolePermissionRepository) HasPermissionByRuleID(ctx context.Context, role string, permissionRuleID primitive.ObjectID) (bool, error) {
	filter := bson.M{"role": role, "permission_rule_id": permissionRuleID}
	count, err := r.rolePermissionsCollection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *MongoRolePermissionRepository) GetByRole(ctx context.Context, role string) ([]*domain.RolePermission, error) {
	filter := bson.M{"role": role}
	cursor, err := r.rolePermissionsCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var permissions []*domain.RolePermission
	if err := cursor.All(ctx, &permissions); err != nil {
		return nil, err
	}
	return permissions, nil
}

func (r *MongoRolePermissionRepository) GetByRoleWithDetails(ctx context.Context, role string) ([]*domain.PermissionRule, error) {
	rolePerms, err := r.GetByRole(ctx, role)
	if err != nil {
		return nil, err
	}

	if len(rolePerms) == 0 {
		return []*domain.PermissionRule{}, nil
	}

	ruleIDs := make([]primitive.ObjectID, 0, len(rolePerms))
	for _, rp := range rolePerms {
		ruleIDs = append(ruleIDs, rp.PermissionRuleID)
	}

	filter := bson.M{"_id": bson.M{"$in": ruleIDs}}
	cursor, err := r.permissionRuleCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rules []*domain.PermissionRule
	if err := cursor.All(ctx, &rules); err != nil {
		return nil, err
	}

	return rules, nil
}

func (r *MongoRolePermissionRepository) GetAll(ctx context.Context) ([]*domain.RolePermission, error) {
	cursor, err := r.rolePermissionsCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var permissions []*domain.RolePermission
	if err := cursor.All(ctx, &permissions); err != nil {
		return nil, err
	}
	return permissions, nil
}

func (r *MongoRolePermissionRepository) GetRoleInheritances(ctx context.Context) ([]*domain.RoleInheritance, error) {
	cursor, err := r.roleInheritancesCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var inheritances []*domain.RoleInheritance
	if err := cursor.All(ctx, &inheritances); err != nil {
		return nil, err
	}
	return inheritances, nil
}

func (r *MongoRolePermissionRepository) AssignInheritance(ctx context.Context, childRole, parentRole string) error {
	filter := bson.M{"child_role": childRole, "parent_role": parentRole}
	update := bson.M{"$setOnInsert": bson.M{
		"child_role":  childRole,
		"parent_role": parentRole,
		"created_at":  primitive.NewObjectID().Timestamp(),
	}}

	_, err := r.roleInheritancesCollection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err == nil {
		// Inheritance changes affect multiple roles, clear all caches
		r.InvalidateAllCache()
	}
	return err
}

func (r *MongoRolePermissionRepository) RemoveInheritance(ctx context.Context, childRole, parentRole string) error {
	filter := bson.M{"child_role": childRole, "parent_role": parentRole}
	_, err := r.roleInheritancesCollection.DeleteOne(ctx, filter)
	if err == nil {
		// Inheritance changes affect multiple roles, clear all caches
		r.InvalidateAllCache()
	}
	return err
}

func (r *MongoRolePermissionRepository) GetInheritedPermissions(ctx context.Context, role string) (map[primitive.ObjectID]bool, error) {
	result := make(map[primitive.ObjectID]bool)

	visited := make(map[string]bool)
	var visit func(r string) error
	visit = func(currentRole string) error {
		if visited[currentRole] {
			return nil
		}
		visited[currentRole] = true

		perms, err := r.GetByRole(ctx, currentRole)
		if err != nil {
			return err
		}
		for _, p := range perms {
			result[p.PermissionRuleID] = true
		}

		inheritances, err := r.GetRoleInheritances(ctx)
		if err != nil {
			return err
		}
		for _, inherit := range inheritances {
			if inherit.ChildRole == currentRole {
				if err := visit(inherit.ParentRole); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := visit(role); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *MongoRolePermissionRepository) GetPermissionRulesForInheritedRoles(ctx context.Context, role string) ([]*domain.PermissionRule, error) {
	// Check cache first (no TTL — cache is only invalidated on mutations)
	r.mu.RLock()
	if cached, ok := r.permissionCache[role]; ok {
		r.mu.RUnlock()
		return cached, nil
	}
	r.mu.RUnlock()

	// Not in cache, fetch from database
	rolePerms, err := r.GetByRole(ctx, role)
	if err != nil {
		return nil, err
	}

	ruleIDsSet := make(map[primitive.ObjectID]bool)
	for _, rp := range rolePerms {
		ruleIDsSet[rp.PermissionRuleID] = true
	}

	visited := make(map[string]bool)
	var visit func(r string) error
	visit = func(currentRole string) error {
		if visited[currentRole] {
			return nil
		}
		visited[currentRole] = true

		perms, err := r.GetByRole(ctx, currentRole)
		if err != nil {
			return err
		}
		for _, p := range perms {
			ruleIDsSet[p.PermissionRuleID] = true
		}

		inheritances, err := r.GetRoleInheritances(ctx)
		if err != nil {
			return err
		}
		for _, inherit := range inheritances {
			if inherit.ChildRole == currentRole {
				if err := visit(inherit.ParentRole); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := visit(role); err != nil {
		return nil, err
	}

	ruleIDs := make([]primitive.ObjectID, 0)
	for id := range ruleIDsSet {
		ruleIDs = append(ruleIDs, id)
	}

	if len(ruleIDs) == 0 {
		return []*domain.PermissionRule{}, nil
	}

	filter := bson.M{"_id": bson.M{"$in": ruleIDs}}
	cursor, err := r.permissionRuleCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rules []*domain.PermissionRule
	if err := cursor.All(ctx, &rules); err != nil {
		return nil, err
	}

	// Update cache (no TTL — stays until explicitly invalidated)
	r.mu.Lock()
	r.permissionCache[role] = rules
	r.mu.Unlock()

	return rules, nil
}

func (r *MongoRolePermissionRepository) CheckPermissionWithInheritance(ctx context.Context, role string, permissionRuleID primitive.ObjectID) (bool, error) {
	hasDirect, err := r.HasPermissionByRuleID(ctx, role, permissionRuleID)
	if err != nil {
		return false, err
	}
	if hasDirect {
		return true, nil
	}

	inherited, err := r.GetInheritedPermissions(ctx, role)
	if err != nil {
		return false, err
	}

	return inherited[permissionRuleID], nil
}

func (r *MongoRolePermissionRepository) CheckPermissionByPathMethod(ctx context.Context, role, path, method string) (bool, error) {
	rules, err := r.GetPermissionRulesForInheritedRoles(ctx, role)
	if err != nil {
		return false, err
	}

	for _, rule := range rules {
		matched := matchesPathMethod(rule.Path, rule.Method, path, method)
		if matched {
			return true, nil
		}
	}

	return false, nil
}

func matchesPathMethod(rulePath, ruleMethod, requestPath, requestMethod string) bool {
	// Check for wildcard matches first
	if ruleMethod == "*" || ruleMethod == requestMethod {
		if rulePath == "*" {
			return true
		}
		// Exact match
		if rulePath == requestPath {
			return true
		}
		// Template match: /api/v1/tenants/:id matches /api/v1/tenants/abc123
		if strings.HasPrefix(rulePath, "/api/v1/") {
			// Extract the base path and parameters
			ruleParts := strings.Split(rulePath, "/")
			requestParts := strings.Split(requestPath, "/")
			if len(ruleParts) == len(requestParts) {
				match := true
				for i, part := range ruleParts {
					if strings.HasPrefix(part, ":") {
						// This is a parameter placeholder, skip it
						continue
					}
					if part != requestParts[i] {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
		}
		// Prefix match for wildcard paths like /api/v1/leads/*
		if strings.HasSuffix(rulePath, "*") {
			prefix := strings.TrimSuffix(rulePath, "*")
			if strings.HasPrefix(requestPath, prefix) {
				return true
			}
		}
	}
	return false
}

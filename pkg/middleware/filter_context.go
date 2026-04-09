package middleware

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Use a plain string key so fasthttp's RequestCtx.Value() can find it.
const scopeFilterKey = "scope_filter_data"

var (
	// scopeRulesByPath maps path+method to all permission rules that have a scope.
	// Multiple rules can share the same path+method (e.g., "list" with no scope and "list_own" with self scope).
	scopeRulesByPath   map[string][]ScopeRuleEntry
	scopeRulesByPathMu sync.RWMutex
)

type ScopeFilter struct {
	SelfUserID    string
	GroupUserIDs  []string
	FilterField   string
	TenantID      string
	IsSystemAdmin bool
	ScopeType     string
}

// ScopeRuleEntry stores a scoped permission rule's details
type ScopeRuleEntry struct {
	RuleID      primitive.ObjectID
	ScopeType   string
	FilterField string
}

func NewFilterContextMiddleware(permissionService ports.PermissionService, rolePermissionRepo ports.RolePermissionRepository) fiber.Handler {
	rules, err := permissionService.GetAllPermissionRules(context.Background())
	if err != nil {
		slog.Error("Failed to load permission rules for scope config", "error", err)
		scopeRulesByPath = make(map[string][]ScopeRuleEntry)
	} else {
		ReloadScopeConfigFromRules(rules)
	}

	return func(c *fiber.Ctx) error {
		role := c.Locals("role").(string)
		userID := c.Locals("user_id").(string)
		tenantID := c.Locals("tenant_id").(string)

		fc := ScopeFilter{
			SelfUserID:    userID,
			TenantID:      tenantID,
			IsSystemAdmin: role == "superadmin",
		}

		// Only apply scope if the current role is specifically assigned a scoped rule
		scopeRulesByPathMu.RLock()
		path := c.Path()
		method := c.Method()

		scopeEntries := findScopeEntries(path, method)
		scopeRulesByPathMu.RUnlock()

		// Check if this role is assigned any of the scoped rules for this path
		if len(scopeEntries) > 0 && !fc.IsSystemAdmin {
			assignedRuleIDs, err := rolePermissionRepo.GetInheritedPermissions(c.Context(), role)
			if err == nil {
				for _, entry := range scopeEntries {
					if assignedRuleIDs[entry.RuleID] {
						fc.ScopeType = entry.ScopeType
						fc.FilterField = entry.FilterField
						slog.Debug("FilterContextMiddleware - role has scoped rule",
							"role", role, "rule_id", entry.RuleID.Hex(),
							"scope_type", entry.ScopeType, "filter_field", entry.FilterField)
						break
					}
				}
			}
		}

		slog.Debug("FilterContextMiddleware - scope set",
			"path", path, "method", method,
			"scope_type", fc.ScopeType, "filter_field", fc.FilterField,
			"user_id", fc.SelfUserID, "role", role)

		c.Locals(scopeFilterKey, fc)

		return c.Next()
	}
}

// findScopeEntries returns scope entries for the given request path+method.
// It uses template matching to handle parameterized paths (e.g., /api/v1/leads/:id).
// Must be called under RLock.
func findScopeEntries(requestPath, requestMethod string) []ScopeRuleEntry {
	// Fast path: try exact match first (works for static paths like /api/v1/leads/list)
	key := requestPath + "_" + requestMethod
	if entries, ok := scopeRulesByPath[key]; ok {
		return entries
	}

	// Slow path: template matching for parameterized paths
	requestParts := strings.Split(requestPath, "/")
	for mapKey, entries := range scopeRulesByPath {
		// Each key is "rulePath_METHOD"
		lastUnderscore := strings.LastIndex(mapKey, "_")
		if lastUnderscore == -1 {
			continue
		}
		ruleMethod := mapKey[lastUnderscore+1:]
		rulePath := mapKey[:lastUnderscore]

		if ruleMethod != requestMethod && ruleMethod != "*" {
			continue
		}

		ruleParts := strings.Split(rulePath, "/")
		if len(ruleParts) != len(requestParts) {
			continue
		}

		match := true
		for i, part := range ruleParts {
			if strings.HasPrefix(part, ":") {
				continue // parameter placeholder — matches any segment
			}
			if part != requestParts[i] {
				match = false
				break
			}
		}
		if match {
			return entries
		}
	}

	return nil
}

func ReloadScopeConfig(service ports.PermissionService) error {
	rules, err := service.GetAllPermissionRules(context.Background())
	if err != nil {
		slog.Error("Failed to reload scope config", "error", err)
		return err
	}
	ReloadScopeConfigFromRules(rules)
	return nil
}

func ReloadScopeConfigFromRules(rules []*domain.PermissionRule) {
	scopeRulesByPathMu.Lock()
	defer scopeRulesByPathMu.Unlock()

	scopeRulesByPath = make(map[string][]ScopeRuleEntry)
	for _, rule := range rules {
		if (rule.ScopeType == "self" || rule.ScopeType == "group") && rule.Path != "" && rule.Method != "" {
			key := rule.Path + "_" + rule.Method
			scopeRulesByPath[key] = append(scopeRulesByPath[key], ScopeRuleEntry{
				RuleID:      rule.ID,
				ScopeType:   rule.ScopeType,
				FilterField: rule.FilterField,
			})
			slog.Debug("Scope config entry", "key", key, "rule_id", rule.ID.Hex(),
				"scope_type", rule.ScopeType, "filter_field", rule.FilterField)
		}
	}
	slog.Info("Scope config reloaded", "scoped_paths", len(scopeRulesByPath))
}

type ScopeConfig struct {
	ScopeType   string
	FilterField string
}

// GetScopeFilter retrieves the ScopeFilter from context.
func GetScopeFilter(ctx context.Context) ScopeFilter {
	if fc, ok := ctx.Value(scopeFilterKey).(ScopeFilter); ok {
		return fc
	}
	return ScopeFilter{}
}

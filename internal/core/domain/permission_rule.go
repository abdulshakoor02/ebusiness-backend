package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PermissionRule struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Resource      string             `bson:"resource" json:"resource"`
	ResourceLabel string             `bson:"resource_label" json:"resource_label"`
	Action        string             `bson:"action" json:"action"`
	ActionLabel   string             `bson:"action_label" json:"action_label"`
	Path          string             `bson:"path" json:"path"`
	Method        string             `bson:"method" json:"method"`
	Description   string             `bson:"description" json:"description"`
	IsSystem      bool               `bson:"is_system" json:"is_system"`
	ScopeType     string             `bson:"scope_type" json:"scope_type"`     // "none" | "self" | "group"
	FilterField   string             `bson:"filter_field" json:"filter_field"` // e.g., "assigned_to", "created_by"
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

type PermissionRuleGroup struct {
	Resource string           `json:"resource"`
	Label    string           `json:"label"`
	Rules    []PermissionRule `json:"rules"`
}

func NewPermissionRule(resource, resourceLabel, action, actionLabel, path, method, description string, isSystem bool, scopeTypeAndFilterField ...string) *PermissionRule {
	scopeType := "none"
	filterField := ""
	if len(scopeTypeAndFilterField) >= 1 {
		scopeType = scopeTypeAndFilterField[0]
	}
	if len(scopeTypeAndFilterField) >= 2 {
		filterField = scopeTypeAndFilterField[1]
	}

	return &PermissionRule{
		ID:            primitive.NewObjectID(),
		Resource:      resource,
		ResourceLabel: resourceLabel,
		Action:        action,
		ActionLabel:   actionLabel,
		Path:          path,
		Method:        method,
		Description:   description,
		IsSystem:      isSystem,
		ScopeType:     scopeType,
		FilterField:   filterField,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

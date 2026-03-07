package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RolePermission struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Role             string             `bson:"role" json:"role"`
	PermissionRuleID primitive.ObjectID `bson:"permission_rule_id" json:"permission_rule_id"`
	CreatedAt        time.Time          `bson:"created_at" json:"created_at"`
}

type RoleInheritance struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChildRole  string             `bson:"child_role" json:"child_role"`
	ParentRole string             `bson:"parent_role" json:"parent_role"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
}

func NewRolePermission(role string, permissionRuleID primitive.ObjectID) *RolePermission {
	return &RolePermission{
		ID:               primitive.NewObjectID(),
		Role:             role,
		PermissionRuleID: permissionRuleID,
		CreatedAt:        time.Now(),
	}
}

func NewRoleInheritance(childRole, parentRole string) *RoleInheritance {
	return &RoleInheritance{
		ID:         primitive.NewObjectID(),
		ChildRole:  childRole,
		ParentRole: parentRole,
		CreatedAt:  time.Now(),
	}
}

package handler

import (
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PermissionHandler struct {
	service ports.PermissionService
}

func NewPermissionHandler(service ports.PermissionService) *PermissionHandler {
	return &PermissionHandler{service: service}
}

// GetPermissions godoc
// @Summary      Get current user permissions
// @Description  Returns all permissions for the currently logged-in user dynamically based on available permission rules
// @Tags         permissions
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /permissions/my-permissions [get]
func (h *PermissionHandler) GetPermissions(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Role not found in context"})
	}

	// Dynamically get all permissions for the user's role
	permissions, err := h.service.GetAllPermissionsForRole(c.Context(), role)
	if err != nil {
		slog.Error("Failed to fetch permissions for role", "role", role, "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch permissions"})
	}

	return c.JSON(fiber.Map{
		"role":        role,
		"permissions": permissions,
	})
}

// AddPermission godoc
// @Summary      Add a permission to a role
// @Description  Allows assigning a new path and HTTP method permission to a role.
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Param        request body ports.AddPermissionRequest true "Add Permission Request"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /permissions [post]
func (h *PermissionHandler) AddPermission(c *fiber.Ctx) error {
	var req ports.AddPermissionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := h.service.AddPermission(c.Context(), req); err != nil {
		slog.Error("Failed to add permission", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Permission added successfully"})
}

// RemovePermission godoc
// @Summary      Remove a permission from a role
// @Description  Removes an existing path and HTTP method permission from a role.
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Param        request body ports.RemovePermissionRequest true "Remove Permission Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /permissions [delete]
func (h *PermissionHandler) RemovePermission(c *fiber.Ctx) error {
	var req ports.RemovePermissionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := h.service.RemovePermission(c.Context(), req); err != nil {
		slog.Error("Failed to remove permission", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Permission removed successfully"})
}

// AssignRoleInheritance godoc
// @Summary      Assign role inheritance
// @Description  Makes a child role inherit all permissions of a parent role.
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Param        request body ports.AssignRoleInheritanceRequest true "Assign Role Inheritance Request"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /permissions/roles/inherit [post]
func (h *PermissionHandler) AssignRoleInheritance(c *fiber.Ctx) error {
	var req ports.AssignRoleInheritanceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := h.service.AssignRoleInheritance(c.Context(), req); err != nil {
		slog.Error("Failed to assign role inheritance", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Role inheritance assigned successfully"})
}

// GetAllPermissions godoc
// @Summary      Get all permissions
// @Description  Retrieves all existing permissions (policies) in the system.
// @Tags         permissions
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /permissions [get]
func (h *PermissionHandler) GetAllPermissions(c *fiber.Ctx) error {
	permissions, err := h.service.GetAllPermissions(c.Context())
	if err != nil {
		slog.Error("Failed to fetch all permissions", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data": permissions,
	})
}

// GetAllRoles godoc
// @Summary      Get all roles
// @Description  Retrieves all available roles in the system.
// @Tags         permissions
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /permissions/roles [get]
func (h *PermissionHandler) GetAllRoles(c *fiber.Ctx) error {
	roles, err := h.service.GetAllRoles(c.Context())
	if err != nil {
		slog.Error("Failed to fetch roles", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data": roles,
	})
}

// GetRoleInheritances godoc
// @Summary      Get all role inheritances
// @Description  Retrieves all inherited roles (groupings) within the system.
// @Tags         permissions
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /permissions/roles/inherit [get]
func (h *PermissionHandler) GetRoleInheritances(c *fiber.Ctx) error {
	inheritances, err := h.service.GetRoleInheritances(c.Context())
	if err != nil {
		slog.Error("Failed to fetch abstract roles", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data": inheritances,
	})
}

// GetAvailableRules godoc
// @Summary      Get available permission rules grouped by resource
// @Description  Returns all available permission rules organized by resource with human-readable labels
// @Tags         permissions
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /permissions/available-rules [get]
func (h *PermissionHandler) GetAvailableRules(c *fiber.Ctx) error {
	groups, err := h.service.GetAvailableRulesGrouped(c.Context())
	if err != nil {
		slog.Error("Failed to fetch available rules", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"resources": groups,
	})
}

// CreatePermissionRule godoc
// @Summary      Create a new permission rule
// @Description  Creates a custom permission rule that can be frontend-only or endpoint-based
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Param        request body ports.CreatePermissionRuleRequest true "Create Permission Rule Request"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /permissions/rules [post]
func (h *PermissionHandler) CreatePermissionRule(c *fiber.Ctx) error {
	var req ports.CreatePermissionRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	rule, err := h.service.CreatePermissionRule(c.Context(), req)
	if err != nil {
		slog.Error("Failed to create permission rule", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Permission rule created successfully",
		"data":    rule,
	})
}

// UpdatePermissionRule godoc
// @Summary      Update a permission rule
// @Description  Updates an existing custom permission rule (system rules can only update labels)
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Param        id path string true "Rule ID"
// @Param        request body ports.UpdatePermissionRuleRequest true "Update Permission Rule Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /permissions/rules/{id} [put]
func (h *PermissionHandler) UpdatePermissionRule(c *fiber.Ctx) error {
	id := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid rule ID"})
	}

	var req ports.UpdatePermissionRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	rule, err := h.service.UpdatePermissionRule(c.Context(), objectID, req)
	if err != nil {
		slog.Error("Failed to update permission rule", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Permission rule updated successfully",
		"data":    rule,
	})
}

// DeletePermissionRule godoc
// @Summary      Delete a permission rule
// @Description  Deletes a custom permission rule (system rules cannot be deleted)
// @Tags         permissions
// @Produce      json
// @Param        id path string true "Rule ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /permissions/rules/{id} [delete]
func (h *PermissionHandler) DeletePermissionRule(c *fiber.Ctx) error {
	id := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid rule ID"})
	}

	if err := h.service.DeletePermissionRule(c.Context(), objectID); err != nil {
		slog.Error("Failed to delete permission rule", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Permission rule deleted successfully",
	})
}

// GetRolePermissions godoc
// @Summary      Get all available permissions mapped to a specific role
// @Description  Returns all permission rules grouped by resource, including a boolean indicating if the specified role is assigned each rule.
// @Tags         permissions
// @Produce      json
// @Param        role path string true "Role Name"
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /permissions/roles/{role} [get]
func (h *PermissionHandler) GetRolePermissions(c *fiber.Ctx) error {
	role := c.Params("role")
	if role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Role parameter is required"})
	}

	groups, err := h.service.GetPermissionsForRoleGrouped(c.Context(), role)
	if err != nil {
		slog.Error("Failed to fetch permissions for role", "role", role, "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"role":      role,
		"resources": groups,
	})
}

// BulkUpdateRolePermissions godoc
// @Summary      Bulk update permissions for a role
// @Description  Synchronizes a role's permissions by adding or removing rules based on the provided assigned status.
// @Tags         permissions
// @Produce      json
// @Param        role path string true "Role Name"
// @Param        request body ports.BulkUpdateRolePermissionsRequest true "Bulk Update Permissions Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /permissions/roles/{role}/bulk [post]
func (h *PermissionHandler) BulkUpdateRolePermissions(c *fiber.Ctx) error {
	role := c.Params("role")
	if role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Role parameter is required"})
	}

	var req ports.BulkUpdateRolePermissionsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := h.service.BulkUpdateRolePermissions(c.Context(), role, req); err != nil {
		slog.Error("Failed to bulk update permissions for role", "role", role, "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Role permissions synchronized successfully",
	})
}

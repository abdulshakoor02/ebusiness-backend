package handler

import (
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
)

type PermissionHandler struct {
	service ports.PermissionService
}

func NewPermissionHandler(service ports.PermissionService) *PermissionHandler {
	return &PermissionHandler{service: service}
}

// GetPermissions godoc
// @Summary      Get current user permissions
// @Description  Returns the permissions of the currently logged-in user based on their role
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

	// Define all permission checks the frontend needs
	permissions := fiber.Map{
		"can_view_tenant":   h.service.CheckPermission(c.Context(), role, "/api/v1/tenants/:id", "GET"),
		"can_update_tenant": h.service.CheckPermission(c.Context(), role, "/api/v1/tenants/:id", "PUT"),
		"can_list_tenants":  h.service.CheckPermission(c.Context(), role, "/api/v1/tenants/list", "POST"),
		"can_create_user":   h.service.CheckPermission(c.Context(), role, "/api/v1/users", "POST"),
		"can_view_user":     h.service.CheckPermission(c.Context(), role, "/api/v1/users/:id", "GET"),
		"can_update_user":   h.service.CheckPermission(c.Context(), role, "/api/v1/users/:id", "PUT"),
		"can_list_users":    h.service.CheckPermission(c.Context(), role, "/api/v1/users/list", "POST"),
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

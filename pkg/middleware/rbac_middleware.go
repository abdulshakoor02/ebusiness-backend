package middleware

import (
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
)

func NewAuthMiddleware(rolePermissionRepo ports.RolePermissionRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(string)
		if !ok || role == "" {
			slog.Warn("Auth: No role found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized: authentication required",
			})
		}

		path := c.Path()
		method := c.Method()

		slog.Debug("Auth check", "role", role, "path", path, "method", method)

		hasPermission, err := rolePermissionRepo.CheckPermissionByPathMethod(c.Context(), role, path, method)
		if err != nil {
			slog.Error("Auth permission check error", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		if !hasPermission {
			slog.Warn("Auth: Permission denied", "role", role, "path", path, "method", method)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden: insufficient permissions",
			})
		}

		return c.Next()
	}
}

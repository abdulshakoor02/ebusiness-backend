package middleware

import (
	"log/slog"

	"github.com/casbin/casbin/v2"
	fibercasbin "github.com/gofiber/contrib/casbin"
	"github.com/gofiber/fiber/v2"
)

// NewCasbinMiddleware creates a new Casbin authorization middleware for Fiber.
// It accepts a pre-configured enforcer so that the same enforcer is shared
// with any handlers that need to query permissions directly.
func NewCasbinMiddleware(enforcer *casbin.Enforcer) *fibercasbin.Middleware {
	authz := fibercasbin.New(fibercasbin.Config{
		Enforcer: enforcer,
		Lookup: func(c *fiber.Ctx) string {
			role, ok := c.Locals("role").(string)
			if !ok || role == "" {
				slog.Warn("RBAC: No role found in context")
				return ""
			}
			return role
		},
		Unauthorized: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized: authentication required",
			})
		},
		Forbidden: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden: insufficient permissions",
			})
		},
	})

	return authz
}

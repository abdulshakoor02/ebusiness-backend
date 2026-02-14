package middleware

import (
	"strings"

	"github.com/abdulshakoor02/goCrmBackend/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

func Protected(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing authorization token"})
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader { // No replacement happened
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token format"})
		}

		claims, err := utils.ValidateToken(tokenString, jwtSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
		}

		// Set locals for Fiber context
		c.Locals("user_id", claims.UserID)
		c.Locals("tenant_id", claims.TenantID)
		c.Locals("email", claims.Email)
		c.Locals("role", claims.Role)

		// Inject into Go Context for Service/Repo layers
		// Note: Fiber ctx is not the same as context.Context regarding values propagation automatically
		// Handlers must ensure they pass a customized context or rely on Fiber's Locals if services accept Fiber ctx (which they shouldn't to stay decoupled).
		// Best practice: Middleware sets Locals, Handler converts Locals to context keys before calling service.

		return c.Next()
	}
}

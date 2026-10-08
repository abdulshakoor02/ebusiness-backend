package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DashboardHandler struct{ service ports.DashboardService }

func NewDashboardHandler(service ports.DashboardService) *DashboardHandler {
	return &DashboardHandler{service: service}
}

// GetSummary godoc
// @Summary Get the authenticated role's live dashboard
// @Description Server-scoped platform, tenant, or personal metrics. No identity filters are accepted.
// @Tags dashboard
// @Produce json
// @Param timezone query string false "IANA time zone (default UTC)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security Bearer
// @Router /dashboard/summary [get]
func (h *DashboardHandler) GetSummary(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	role, _ := c.Locals("role").(string)
	if role == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authentication required"})
	}
	if role != "superadmin" && role != "admin" && role != "user" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Dashboard is not available for this role"})
	}
	scope := ports.DashboardScope{Role: role}
	var err error
	if scope.UserID, err = dashboardObjectID(c, "user_id"); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user identity"})
	}
	if role != "superadmin" {
		if scope.TenantID, err = dashboardObjectID(c, "tenant_id"); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid tenant identity"})
		}
	}
	invalidQuery := false
	c.Context().QueryArgs().VisitAll(func(key, _ []byte) {
		if string(key) != "timezone" {
			invalidQuery = true
		}
	})
	if invalidQuery {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Only the timezone parameter is supported; scope is determined by your account"})
	}
	ctx, cancel := context.WithTimeout(c.UserContext(), 8*time.Second)
	defer cancel()
	result, err := h.service.GetSummary(ctx, scope, c.Query("timezone", "UTC"), time.Now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, ports.ErrDashboardZone):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid IANA time zone"})
		case errors.Is(err, ports.ErrDashboardRole):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Dashboard is not available for this role"})
		case errors.Is(err, ports.ErrDashboardIdentity):
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid account identity"})
		default:
			slog.Error("Dashboard summary failed", "role", role, "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Dashboard metrics could not be loaded"})
		}
	}
	return c.JSON(fiber.Map{"data": result})
}

func dashboardObjectID(c *fiber.Ctx, name string) (primitive.ObjectID, error) {
	value, _ := c.Locals(name).(string)
	id, err := primitive.ObjectIDFromHex(value)
	if err != nil || id.IsZero() {
		return primitive.NilObjectID, ports.ErrDashboardIdentity
	}
	return id, nil
}

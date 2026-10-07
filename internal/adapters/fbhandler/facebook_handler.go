package fbhandler

import (
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FacebookHandler struct {
	service ports.FacebookService
}

func NewFacebookHandler(service ports.FacebookService) *FacebookHandler {
	return &FacebookHandler{service: service}
}

func (h *FacebookHandler) GetAuthURL(c *fiber.Ctx) error {
	tenantIDStr, ok := c.Locals("tenant_id").(string)
	if !ok || tenantIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	tenantID, err := primitive.ObjectIDFromHex(tenantIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid tenant id"})
	}
	authURL, err := h.service.GetAuthURL(c.Context(), tenantID)
	if err != nil {
		slog.Error("Failed to build Facebook auth URL", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"auth_url": authURL})
}

func (h *FacebookHandler) StartAuth(c *fiber.Ctx) error {
	tenantIDStr, ok := c.Locals("tenant_id").(string)
	if !ok || tenantIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	tenantID, err := primitive.ObjectIDFromHex(tenantIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid tenant id"})
	}
	start, err := h.service.StartAuth(c.Context(), tenantID)
	if err != nil {
		slog.Error("Failed to start Facebook auth", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(start)
}

func (h *FacebookHandler) Callback(c *fiber.Ctx) error {
	state := c.Query("state")
	code := c.Query("code")
	// The service always returns the browser landing URL (success or error)
	// so the popup can report back to the app and close itself.
	landing, err := h.service.HandleCallback(c.Context(), state, code)
	if err != nil {
		slog.Error("Facebook callback failed", "error", err)
	}
	if landing == "" {
		landing = "/?facebook=error"
	}
	return c.Redirect(landing, fiber.StatusFound)
}

func (h *FacebookHandler) Status(c *fiber.Ctx) error {
	tenantIDStr, ok := c.Locals("tenant_id").(string)
	if !ok || tenantIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	tenantID, err := primitive.ObjectIDFromHex(tenantIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid tenant id"})
	}
	status, err := h.service.GetStatus(c.Context(), tenantID)
	if err != nil {
		slog.Error("Failed to get Facebook status", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(status)
}

func (h *FacebookHandler) Disconnect(c *fiber.Ctx) error {
	tenantIDStr, ok := c.Locals("tenant_id").(string)
	if !ok || tenantIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	tenantID, err := primitive.ObjectIDFromHex(tenantIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid tenant id"})
	}
	if err := h.service.Disconnect(c.Context(), tenantID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "disconnected"})
}

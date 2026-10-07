package fbhandler

import (
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FacebookAdsHandler struct {
	service ports.FacebookAdsService
}

func NewFacebookAdsHandler(service ports.FacebookAdsService) *FacebookAdsHandler {
	return &FacebookAdsHandler{service: service}
}

func tenantFromLocals(c *fiber.Ctx) (primitive.ObjectID, error) {
	tenantIDStr, ok := c.Locals("tenant_id").(string)
	if !ok || tenantIDStr == "" {
		return primitive.NilObjectID, fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}
	return primitive.ObjectIDFromHex(tenantIDStr)
}

func (h *FacebookAdsHandler) ListCampaigns(c *fiber.Ctx) error {
	tenantID, err := tenantFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	views, err := h.service.ListCampaigns(c.Context(), tenantID)
	if err != nil {
		slog.Error("Failed to list Facebook campaigns", "error", err)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(views)
}

func (h *FacebookAdsHandler) AssignCampaign(c *fiber.Ctx) error {
	tenantID, err := tenantFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	var req ports.AssignFacebookCampaignRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	a, err := h.service.AssignCampaign(c.Context(), tenantID, req)
	if err != nil {
		slog.Error("Failed to assign Facebook campaign", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(a)
}

func (h *FacebookAdsHandler) UnassignCampaign(c *fiber.Ctx) error {
	tenantID, err := tenantFromLocals(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	campaignID := c.Params("campaign_id")
	if err := h.service.UnassignCampaign(c.Context(), tenantID, campaignID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "unassigned"})
}

package handler

import (
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AIChatHandler struct {
	service ports.AIChatService
}

func NewAIChatHandler(service ports.AIChatService) *AIChatHandler {
	return &AIChatHandler{service: service}
}

func (h *AIChatHandler) Chat(c *fiber.Ctx) error {
	tenantIDStr, ok := c.Locals("tenant_id").(string)
	if !ok || tenantIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "tenant context required"})
	}

	tenantID, err := primitive.ObjectIDFromHex(tenantIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid tenant ID"})
	}

	var req ports.ChatRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "message is required"})
	}

	resp, err := h.service.Chat(c.Context(), tenantID, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"data": resp})
}

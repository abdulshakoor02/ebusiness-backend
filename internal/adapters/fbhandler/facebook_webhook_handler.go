package fbhandler

import (
	"encoding/json"
	"io"
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
)

type FacebookWebhookHandler struct {
	service ports.FacebookLeadService
}

func NewFacebookWebhookHandler(service ports.FacebookLeadService) *FacebookWebhookHandler {
	return &FacebookWebhookHandler{service: service}
}

func (h *FacebookWebhookHandler) Verify(c *fiber.Ctx) error {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")
	out, err := h.service.VerifyWebhook(mode, token, challenge)
	if err != nil {
		slog.Warn("Facebook webhook verification failed")
		return c.Status(fiber.StatusForbidden).SendString("verification failed")
	}
	return c.Status(fiber.StatusOK).SendString(out)
}

func (h *FacebookWebhookHandler) Receive(c *fiber.Ctx) error {
	body, err := io.ReadAll(c.Request().BodyStream())
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot read body"})
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON body"})
	}
	result, err := h.service.IngestLead(c.Context(), payload)
	if err != nil {
		slog.Error("Facebook lead ingest error", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	// Always 200: Facebook retries non-2xx, and dedup makes replays safe.
	return c.JSON(result)
}

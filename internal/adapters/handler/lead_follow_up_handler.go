package handler

import (
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadFollowUpHandler struct {
	service ports.LeadFollowUpService
}

func NewLeadFollowUpHandler(service ports.LeadFollowUpService) *LeadFollowUpHandler {
	return &LeadFollowUpHandler{service: service}
}

// CreateLeadFollowUp godoc
// @Summary      Create a lead follow-up
// @Description  Creates a new lead follow-up associated with a specific lead
// @Tags         lead-follow-ups
// @Accept       json
// @Produce      json
// @Param        lead_id path   string  true  "Lead ID"
// @Param        request body   ports.CreateLeadFollowUpRequest true "Create Lead Follow-Up Request"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{lead_id}/follow-ups [post]
func (h *LeadFollowUpHandler) CreateLeadFollowUp(c *fiber.Ctx) error {
	leadIDParam := c.Params("lead_id")
	leadID, err := primitive.ObjectIDFromHex(leadIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead ID"})
	}

	var req ports.CreateLeadFollowUpRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	followUp, err := h.service.CreateLeadFollowUp(c.Context(), leadID, req)
	if err != nil {
		slog.Error("Failed to create lead follow-up", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(followUp)
}

// GetLeadFollowUp godoc
// @Summary      Get a lead follow-up by ID
// @Description  Retrieves a specific lead follow-up
// @Tags         lead-follow-ups
// @Accept       json
// @Produce      json
// @Param        lead_id path   string  true  "Lead ID"
// @Param        id      path   string  true  "Follow-Up ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{lead_id}/follow-ups/{id} [get]
func (h *LeadFollowUpHandler) GetLeadFollowUp(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead follow-up ID"})
	}

	followUp, err := h.service.GetLeadFollowUp(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Lead follow-up not found"})
	}

	return c.JSON(followUp)
}

// UpdateLeadFollowUp godoc
// @Summary      Update a lead follow-up
// @Description  Updates an existing lead follow-up
// @Tags         lead-follow-ups
// @Accept       json
// @Produce      json
// @Param        lead_id path   string  true  "Lead ID"
// @Param        id      path   string  true  "Follow-Up ID"
// @Param        request body   ports.UpdateLeadFollowUpRequest true "Update Lead Follow-Up Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{lead_id}/follow-ups/{id} [put]
func (h *LeadFollowUpHandler) UpdateLeadFollowUp(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead follow-up ID"})
	}

	var req ports.UpdateLeadFollowUpRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	followUp, err := h.service.UpdateLeadFollowUp(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(followUp)
}

// DeleteLeadFollowUp godoc
// @Summary      Delete a lead follow-up
// @Description  Deletes an existing lead follow-up
// @Tags         lead-follow-ups
// @Accept       json
// @Produce      json
// @Param        lead_id path   string  true  "Lead ID"
// @Param        id      path   string  true  "Follow-Up ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{lead_id}/follow-ups/{id} [delete]
func (h *LeadFollowUpHandler) DeleteLeadFollowUp(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead follow-up ID"})
	}

	if err := h.service.DeleteLeadFollowUp(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Lead follow-up deleted successfully"})
}

// ListLeadFollowUps godoc
// @Summary      List lead follow-ups
// @Description  Retrieves a paginated list of lead follow-ups for a specific lead
// @Tags         lead-follow-ups
// @Accept       json
// @Produce      json
// @Param        lead_id path   string  true  "Lead ID"
// @Param        request body   ports.FilterRequest true "Filter Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{lead_id}/follow-ups/list [post]
func (h *LeadFollowUpHandler) ListLeadFollowUps(c *fiber.Ctx) error {
	leadIDParam := c.Params("lead_id")
	leadID, err := primitive.ObjectIDFromHex(leadIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead ID"})
	}

	var req ports.FilterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	followUps, total, err := h.service.ListLeadFollowUps(c.Context(), leadID, req)
	if err != nil {
		slog.Error("Failed to list lead follow-ups", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":   followUps,
		"total":  total,
		"offset": req.Offset,
		"limit":  req.Limit,
	})
}

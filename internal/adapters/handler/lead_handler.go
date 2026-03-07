package handler

import (
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadHandler struct {
	service ports.LeadService
}

func NewLeadHandler(service ports.LeadService) *LeadHandler {
	return &LeadHandler{service: service}
}

// CreateLead godoc
// @Summary      Create a new lead
// @Description  Creates a new lead associated with the current tenant
// @Tags         leads
// @Accept       json
// @Produce      json
// @Param        request body ports.CreateLeadRequest true "Create Lead Request"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads [post]
func (h *LeadHandler) CreateLead(c *fiber.Ctx) error {
	var req ports.CreateLeadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	lead, err := h.service.CreateLead(c.Context(), req)
	if err != nil {
		slog.Error("Failed to create lead", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(lead)
}

// GetLead godoc
// @Summary      Get a lead by ID
// @Description  Retrieves a specific lead belonging to the current tenant
// @Tags         leads
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Lead ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{id} [get]
func (h *LeadHandler) GetLead(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead ID"})
	}

	lead, err := h.service.GetLead(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Lead not found"})
	}

	return c.JSON(lead)
}

// UpdateLead godoc
// @Summary      Update a lead
// @Description  Updates an existing lead's details
// @Tags         leads
// @Accept       json
// @Produce      json
// @Param        id      path   string                    true  "Lead ID"
// @Param        request body   ports.UpdateLeadRequest true  "Update Lead Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{id} [put]
func (h *LeadHandler) UpdateLead(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead ID"})
	}

	var req ports.UpdateLeadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	lead, err := h.service.UpdateLead(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(lead)
}

// ListLeads godoc
// @Summary      List leads
// @Description  Retrieves a paginated list of leads based on filters
// @Tags         leads
// @Accept       json
// @Produce      json
// @Param        request body ports.FilterRequest true "Filter Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/list [post]
func (h *LeadHandler) ListLeads(c *fiber.Ctx) error {
	var req ports.FilterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	leads, total, err := h.service.ListLeads(c.Context(), req)
	if err != nil {
		slog.Error("Failed to list leads", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":   leads,
		"total":  total,
		"offset": req.Offset,
		"limit":  req.Limit,
	})
}

// UpdateLeadStatus godoc
// @Summary      Toggle lead client status
// @Description  Toggle a converted lead's status between active and inactive
// @Tags         leads
// @Accept       json
// @Produce      json
// @Param        id      path   string                           true  "Lead ID"
// @Param        request body   ports.UpdateLeadStatusRequest  true  "Update Lead Status Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{id}/status [put]
func (h *LeadHandler) UpdateLeadStatus(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead ID"})
	}

	var req ports.UpdateLeadStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	lead, err := h.service.UpdateLeadStatus(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(lead)
}

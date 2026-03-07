package handler

import (
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadSourceHandler struct {
	service ports.LeadSourceService
}

func NewLeadSourceHandler(service ports.LeadSourceService) *LeadSourceHandler {
	return &LeadSourceHandler{service: service}
}

// CreateLeadSource godoc
// @Summary      Create a new lead source
// @Description  Creates a new lead source category/tag associated with the current tenant
// @Tags         lead-sources
// @Accept       json
// @Produce      json
// @Param        request body ports.CreateLeadSourceRequest true "Create Lead Source Request"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /lead-sources [post]
func (h *LeadSourceHandler) CreateLeadSource(c *fiber.Ctx) error {
	var req ports.CreateLeadSourceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	source, err := h.service.CreateLeadSource(c.Context(), req)
	if err != nil {
		slog.Error("Failed to create lead source", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(source)
}

// GetLeadSource godoc
// @Summary      Get a lead source by ID
// @Description  Retrieves a specific lead source belonging to the current tenant
// @Tags         lead-sources
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Lead Source ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /lead-sources/{id} [get]
func (h *LeadSourceHandler) GetLeadSource(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead source ID"})
	}

	source, err := h.service.GetLeadSource(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Lead source not found"})
	}

	return c.JSON(source)
}

// UpdateLeadSource godoc
// @Summary      Update a lead source
// @Description  Updates an existing lead source's details
// @Tags         lead-sources
// @Accept       json
// @Produce      json
// @Param        id      path   string                    true  "Lead Source ID"
// @Param        request body   ports.UpdateLeadSourceRequest true  "Update Lead Source Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /lead-sources/{id} [put]
func (h *LeadSourceHandler) UpdateLeadSource(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead source ID"})
	}

	var req ports.UpdateLeadSourceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	source, err := h.service.UpdateLeadSource(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(source)
}

// DeleteLeadSource godoc
// @Summary      Delete a lead source
// @Description  Deletes an existing lead source
// @Tags         lead-sources
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Lead Source ID"
// @Success      204  "No Content"
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /lead-sources/{id} [delete]
func (h *LeadSourceHandler) DeleteLeadSource(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead source ID"})
	}

	if err := h.service.DeleteLeadSource(c.Context(), id); err != nil {
		slog.Error("Failed to delete lead source", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ListLeadSources godoc
// @Summary      List lead sources
// @Description  Retrieves a paginated list of lead sources based on filters
// @Tags         lead-sources
// @Accept       json
// @Produce      json
// @Param        request body ports.FilterRequest true "Filter Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /lead-sources/list [post]
func (h *LeadSourceHandler) ListLeadSources(c *fiber.Ctx) error {
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

	sources, total, err := h.service.ListLeadSources(c.Context(), req)
	if err != nil {
		slog.Error("Failed to list lead sources", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":   sources,
		"total":  total,
		"offset": req.Offset,
		"limit":  req.Limit,
	})
}

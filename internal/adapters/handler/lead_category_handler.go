package handler

import (
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadCategoryHandler struct {
	service ports.LeadCategoryService
}

func NewLeadCategoryHandler(service ports.LeadCategoryService) *LeadCategoryHandler {
	return &LeadCategoryHandler{service: service}
}

// CreateLeadCategory godoc
// @Summary      Create a new lead category
// @Description  Creates a new lead category associated with the current tenant
// @Tags         lead-categories
// @Accept       json
// @Produce      json
// @Param        request body ports.CreateLeadCategoryRequest true "Create Lead Category Request"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /lead-categories [post]
func (h *LeadCategoryHandler) CreateLeadCategory(c *fiber.Ctx) error {
	var req ports.CreateLeadCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	category, err := h.service.CreateLeadCategory(c.Context(), req)
	if err != nil {
		slog.Error("Failed to create lead category", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(category)
}

// GetLeadCategory godoc
// @Summary      Get a lead category by ID
// @Description  Retrieves a specific lead category belonging to the current tenant
// @Tags         lead-categories
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Lead Category ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /lead-categories/{id} [get]
func (h *LeadCategoryHandler) GetLeadCategory(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead category ID"})
	}

	category, err := h.service.GetLeadCategory(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Lead category not found"})
	}

	return c.JSON(category)
}

// UpdateLeadCategory godoc
// @Summary      Update a lead category
// @Description  Updates an existing lead category's details
// @Tags         lead-categories
// @Accept       json
// @Produce      json
// @Param        id      path   string                             true  "Lead Category ID"
// @Param        request body   ports.UpdateLeadCategoryRequest true  "Update Lead Category Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /lead-categories/{id} [put]
func (h *LeadCategoryHandler) UpdateLeadCategory(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead category ID"})
	}

	var req ports.UpdateLeadCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	category, err := h.service.UpdateLeadCategory(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(category)
}

// DeleteLeadCategory godoc
// @Summary      Delete a lead category
// @Description  Deletes an existing lead category's details
// @Tags         lead-categories
// @Accept       json
// @Produce      json
// @Param        id      path   string  true  "Lead Category ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /lead-categories/{id} [delete]
func (h *LeadCategoryHandler) DeleteLeadCategory(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead category ID"})
	}

	if err := h.service.DeleteLeadCategory(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Lead category deleted successfully"})
}

// ListLeadCategories godoc
// @Summary      List lead categories
// @Description  Retrieves a paginated list of lead categories based on filters
// @Tags         lead-categories
// @Accept       json
// @Produce      json
// @Param        request body ports.FilterRequest true "Filter Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /lead-categories/list [post]
func (h *LeadCategoryHandler) ListLeadCategories(c *fiber.Ctx) error {
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

	categories, total, err := h.service.ListLeadCategories(c.Context(), req)
	if err != nil {
		slog.Error("Failed to list lead categories", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":   categories,
		"total":  total,
		"offset": req.Offset,
		"limit":  req.Limit,
	})
}

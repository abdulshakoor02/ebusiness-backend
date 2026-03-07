package handler

import (
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadCommentHandler struct {
	service ports.LeadCommentService
}

func NewLeadCommentHandler(service ports.LeadCommentService) *LeadCommentHandler {
	return &LeadCommentHandler{service: service}
}

// CreateLeadComment godoc
// @Summary      Add comment to a lead
// @Description  Creates a new lead comment associated with a specific lead
// @Tags         lead-comments
// @Accept       json
// @Produce      json
// @Param        lead_id path   string  true  "Lead ID"
// @Param        request body   ports.CreateLeadCommentRequest true "Create Lead Comment Request"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{lead_id}/comments [post]
func (h *LeadCommentHandler) CreateLeadComment(c *fiber.Ctx) error {
	leadIDParam := c.Params("lead_id")
	leadID, err := primitive.ObjectIDFromHex(leadIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead ID"})
	}

	var req ports.CreateLeadCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	comment, err := h.service.CreateLeadComment(c.Context(), leadID, req)
	if err != nil {
		slog.Error("Failed to create lead comment", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(comment)
}

// GetLeadComment godoc
// @Summary      Get a lead comment by ID
// @Description  Retrieves a specific lead comment
// @Tags         lead-comments
// @Accept       json
// @Produce      json
// @Param        lead_id path   string  true  "Lead ID"
// @Param        id      path   string  true  "Comment ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{lead_id}/comments/{id} [get]
func (h *LeadCommentHandler) GetLeadComment(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead comment ID"})
	}

	comment, err := h.service.GetLeadComment(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Lead comment not found"})
	}

	return c.JSON(comment)
}

// UpdateLeadComment godoc
// @Summary      Update a lead comment
// @Description  Updates an existing lead comment's content (only by author)
// @Tags         lead-comments
// @Accept       json
// @Produce      json
// @Param        lead_id path   string  true  "Lead ID"
// @Param        id      path   string  true  "Comment ID"
// @Param        request body   ports.UpdateLeadCommentRequest true "Update Lead Comment Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{lead_id}/comments/{id} [put]
func (h *LeadCommentHandler) UpdateLeadComment(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead comment ID"})
	}

	var req ports.UpdateLeadCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	comment, err := h.service.UpdateLeadComment(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()}) // Will return 500 even for unauthorized for simplicity, could wrap specific domain errors
	}

	return c.JSON(comment)
}

// DeleteLeadComment godoc
// @Summary      Delete a lead comment
// @Description  Deletes an existing lead comment (only by author)
// @Tags         lead-comments
// @Accept       json
// @Produce      json
// @Param        lead_id path   string  true  "Lead ID"
// @Param        id      path   string  true  "Comment ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{lead_id}/comments/{id} [delete]
func (h *LeadCommentHandler) DeleteLeadComment(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead comment ID"})
	}

	if err := h.service.DeleteLeadComment(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Lead comment deleted successfully"})
}

// ListLeadComments godoc
// @Summary      List lead comments
// @Description  Retrieves a paginated list of lead comments for a specific lead
// @Tags         lead-comments
// @Accept       json
// @Produce      json
// @Param        lead_id path   string  true  "Lead ID"
// @Param        request body   ports.FilterRequest true "Filter Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{lead_id}/comments/list [post]
func (h *LeadCommentHandler) ListLeadComments(c *fiber.Ctx) error {
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
		req.Limit = 50 // default slightly higher for comments thread
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	comments, total, err := h.service.ListLeadComments(c.Context(), leadID, req)
	if err != nil {
		slog.Error("Failed to list lead comments", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":   comments,
		"total":  total,
		"offset": req.Offset,
		"limit":  req.Limit,
	})
}

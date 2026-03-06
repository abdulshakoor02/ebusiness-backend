package handler

import (
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type QualificationHandler struct {
	service ports.QualificationService
}

func NewQualificationHandler(service ports.QualificationService) *QualificationHandler {
	return &QualificationHandler{service: service}
}

// CreateQualification godoc
// @Summary      Create a new qualification
// @Description  Creates a new qualification
// @Tags         qualifications
// @Accept       json
// @Produce      json
// @Param        request body ports.CreateQualificationRequest true "Create Qualification Request"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /qualifications [post]
func (h *QualificationHandler) CreateQualification(c *fiber.Ctx) error {
	var req ports.CreateQualificationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	qualification, err := h.service.CreateQualification(c.Context(), req)
	if err != nil {
		slog.Error("Failed to create qualification", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(qualification)
}

// GetQualification godoc
// @Summary      Get a qualification by ID
// @Description  Retrieves a specific qualification
// @Tags         qualifications
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Qualification ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /qualifications/{id} [get]
func (h *QualificationHandler) GetQualification(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid qualification ID"})
	}

	qualification, err := h.service.GetQualification(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Qualification not found"})
	}

	return c.JSON(qualification)
}

// UpdateQualification godoc
// @Summary      Update a qualification
// @Description  Updates an existing qualification's details
// @Tags         qualifications
// @Accept       json
// @Produce      json
// @Param        id      path   string                             true  "Qualification ID"
// @Param        request body   ports.UpdateQualificationRequest true  "Update Qualification Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /qualifications/{id} [put]
func (h *QualificationHandler) UpdateQualification(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid qualification ID"})
	}

	var req ports.UpdateQualificationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	qualification, err := h.service.UpdateQualification(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(qualification)
}

// DeleteQualification godoc
// @Summary      Delete a qualification
// @Description  Deletes an existing qualification
// @Tags         qualifications
// @Accept       json
// @Produce      json
// @Param        id      path   string  true  "Qualification ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /qualifications/{id} [delete]
func (h *QualificationHandler) DeleteQualification(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid qualification ID"})
	}

	if err := h.service.DeleteQualification(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Qualification deleted successfully"})
}

// ListQualifications godoc
// @Summary      List qualifications
// @Description  Retrieves a paginated list of qualifications based on filters
// @Tags         qualifications
// @Accept       json
// @Produce      json
// @Param        request body ports.FilterRequest true "Filter Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /qualifications/list [post]
func (h *QualificationHandler) ListQualifications(c *fiber.Ctx) error {
	var req ports.FilterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	qualifications, total, err := h.service.ListQualifications(c.Context(), req)
	if err != nil {
		slog.Error("Failed to list qualifications", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":   qualifications,
		"total":  total,
		"offset": req.Offset,
		"limit":  req.Limit,
	})
}

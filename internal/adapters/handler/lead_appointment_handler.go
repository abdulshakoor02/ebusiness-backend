package handler

import (
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadAppointmentHandler struct {
	service ports.LeadAppointmentService
}

func NewLeadAppointmentHandler(service ports.LeadAppointmentService) *LeadAppointmentHandler {
	return &LeadAppointmentHandler{service: service}
}

// CreateLeadAppointment godoc
// @Summary      Schedule a lead appointment
// @Description  Creates a new lead appointment associated with a specific lead
// @Tags         lead-appointments
// @Accept       json
// @Produce      json
// @Param        lead_id path   string  true  "Lead ID"
// @Param        request body   ports.CreateLeadAppointmentRequest true "Create Lead Appointment Request"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{lead_id}/appointments [post]
func (h *LeadAppointmentHandler) CreateLeadAppointment(c *fiber.Ctx) error {
	leadIDParam := c.Params("lead_id")
	leadID, err := primitive.ObjectIDFromHex(leadIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead ID"})
	}

	var req ports.CreateLeadAppointmentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	appointment, err := h.service.CreateLeadAppointment(c.Context(), leadID, req)
	if err != nil {
		slog.Error("Failed to create lead appointment", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(appointment)
}

// GetLeadAppointment godoc
// @Summary      Get a lead appointment by ID
// @Description  Retrieves a specific lead appointment
// @Tags         lead-appointments
// @Accept       json
// @Produce      json
// @Param        lead_id path   string  true  "Lead ID"
// @Param        id      path   string  true  "Appointment ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{lead_id}/appointments/{id} [get]
func (h *LeadAppointmentHandler) GetLeadAppointment(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead appointment ID"})
	}

	appointment, err := h.service.GetLeadAppointment(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Lead appointment not found"})
	}

	return c.JSON(appointment)
}

// UpdateLeadAppointment godoc
// @Summary      Update a lead appointment
// @Description  Updates an existing lead appointment (only by organizer)
// @Tags         lead-appointments
// @Accept       json
// @Produce      json
// @Param        lead_id path   string  true  "Lead ID"
// @Param        id      path   string  true  "Appointment ID"
// @Param        request body   ports.UpdateLeadAppointmentRequest true "Update Lead Appointment Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{lead_id}/appointments/{id} [put]
func (h *LeadAppointmentHandler) UpdateLeadAppointment(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead appointment ID"})
	}

	var req ports.UpdateLeadAppointmentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	appointment, err := h.service.UpdateLeadAppointment(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(appointment)
}

// DeleteLeadAppointment godoc
// @Summary      Delete a lead appointment
// @Description  Deletes an existing lead appointment (only by organizer)
// @Tags         lead-appointments
// @Accept       json
// @Produce      json
// @Param        lead_id path   string  true  "Lead ID"
// @Param        id      path   string  true  "Appointment ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{lead_id}/appointments/{id} [delete]
func (h *LeadAppointmentHandler) DeleteLeadAppointment(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead appointment ID"})
	}

	if err := h.service.DeleteLeadAppointment(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Lead appointment deleted successfully"})
}

// ListLeadAppointments godoc
// @Summary      List lead appointments
// @Description  Retrieves a paginated list of lead appointments for a specific lead
// @Tags         lead-appointments
// @Accept       json
// @Produce      json
// @Param        lead_id path   string  true  "Lead ID"
// @Param        request body   ports.FilterRequest true "Filter Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /leads/{lead_id}/appointments/list [post]
func (h *LeadAppointmentHandler) ListLeadAppointments(c *fiber.Ctx) error {
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

	appointments, total, err := h.service.ListLeadAppointments(c.Context(), leadID, req)
	if err != nil {
		slog.Error("Failed to list lead appointments", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":   appointments,
		"total":  total,
		"offset": req.Offset,
		"limit":  req.Limit,
	})
}

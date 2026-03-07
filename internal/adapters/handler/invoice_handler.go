package handler

import (
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InvoiceHandler struct {
	service ports.InvoiceService
}

func NewInvoiceHandler(service ports.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{service: service}
}

func (h *InvoiceHandler) CreateInvoice(c *fiber.Ctx) error {
	leadIDParam := c.Params("lead_id")
	if leadIDParam == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "lead_id is required"})
	}

	var req ports.CreateInvoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	req.LeadID = leadIDParam

	invoice, err := h.service.CreateInvoice(c.Context(), req)
	if err != nil {
		slog.Error("Failed to create invoice", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(invoice)
}

func (h *InvoiceHandler) GetInvoice(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid invoice ID"})
	}

	invoice, err := h.service.GetInvoice(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Invoice not found"})
	}

	return c.JSON(invoice)
}

func (h *InvoiceHandler) UpdateDueDate(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid invoice ID"})
	}

	var req ports.UpdateInvoiceDueDateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	invoice, err := h.service.UpdateInvoiceDueDate(c.Context(), id, req)
	if err != nil {
		slog.Error("Failed to update invoice due date", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(invoice)
}

func (h *InvoiceHandler) ListInvoices(c *fiber.Ctx) error {
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

	invoices, total, err := h.service.ListInvoices(c.Context(), req)
	if err != nil {
		slog.Error("Failed to list invoices", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":   invoices,
		"total":  total,
		"offset": req.Offset,
		"limit":  req.Limit,
	})
}

func (h *InvoiceHandler) GetInvoicesByLeadID(c *fiber.Ctx) error {
	leadIDParam := c.Params("lead_id")
	leadID, err := primitive.ObjectIDFromHex(leadIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lead ID"})
	}

	invoices, err := h.service.GetInvoicesByLeadID(c.Context(), leadID)
	if err != nil {
		slog.Error("Failed to get invoices for lead", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data": invoices,
	})
}

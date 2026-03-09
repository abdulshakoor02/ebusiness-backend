package handler

import (
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ReceiptHandler struct {
	service ports.ReceiptService
}

func NewReceiptHandler(service ports.ReceiptService) *ReceiptHandler {
	return &ReceiptHandler{service: service}
}

func (h *ReceiptHandler) CreateReceipt(c *fiber.Ctx) error {
	invoiceIDParam := c.Params("invoice_id")
	invoiceID, err := primitive.ObjectIDFromHex(invoiceIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid invoice ID"})
	}

	var req ports.CreateReceiptRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	receipt, err := h.service.CreateReceipt(c.Context(), invoiceID, req)
	if err != nil {
		slog.Error("Failed to create receipt", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(receipt)
}

func (h *ReceiptHandler) GetReceipt(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid receipt ID"})
	}

	receipt, err := h.service.GetReceipt(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Receipt not found"})
	}

	return c.JSON(receipt)
}

func (h *ReceiptHandler) UpdateReceipt(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid receipt ID"})
	}

	var req ports.UpdateReceiptRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	receipt, err := h.service.UpdateReceipt(c.Context(), id, req)
	if err != nil {
		slog.Error("Failed to update receipt", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(receipt)
}

func (h *ReceiptHandler) DeleteReceipt(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid receipt ID"})
	}

	if err := h.service.DeleteReceipt(c.Context(), id); err != nil {
		slog.Error("Failed to delete receipt", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Receipt deleted successfully"})
}

func (h *ReceiptHandler) ListReceiptsByInvoiceID(c *fiber.Ctx) error {
	invoiceIDParam := c.Params("invoice_id")
	invoiceID, err := primitive.ObjectIDFromHex(invoiceIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid invoice ID"})
	}

	receipts, err := h.service.ListReceiptsByInvoiceID(c.Context(), invoiceID)
	if err != nil {
		slog.Error("Failed to list receipts", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data": receipts,
	})
}

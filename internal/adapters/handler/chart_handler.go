package handler

import (
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
)

type ChartHandler struct {
	service ports.ChartService
}

func NewChartHandler(service ports.ChartService) *ChartHandler {
	return &ChartHandler{service: service}
}

func (h *ChartHandler) GetMonthlySummary(c *fiber.Ctx) error {
	month := c.QueryInt("month", 0)
	year := c.QueryInt("year", 0)

	data, err := h.service.GetMonthlySummary(c.Context(), month, year)
	if err != nil {
		slog.Error("Failed to get monthly summary", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"data": data})
}

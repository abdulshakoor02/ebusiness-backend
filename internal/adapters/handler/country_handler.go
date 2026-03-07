package handler

import (
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CountryHandler struct {
	service ports.CountryService
}

func NewCountryHandler(service ports.CountryService) *CountryHandler {
	return &CountryHandler{service: service}
}

// CreateCountry godoc
// @Summary      Create a new country
// @Description  Creates a new country
// @Tags         countries
// @Accept       json
// @Produce      json
// @Param        request body ports.CreateCountryRequest true "Create Country Request"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /countries [post]
func (h *CountryHandler) CreateCountry(c *fiber.Ctx) error {
	var req ports.CreateCountryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	country, err := h.service.CreateCountry(c.Context(), req)
	if err != nil {
		slog.Error("Failed to create country", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(country)
}

// GetCountry godoc
// @Summary      Get a country by ID
// @Description  Retrieves a specific country
// @Tags         countries
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Country ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /countries/{id} [get]
func (h *CountryHandler) GetCountry(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid country ID"})
	}

	country, err := h.service.GetCountry(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Country not found"})
	}

	return c.JSON(country)
}

// UpdateCountry godoc
// @Summary      Update a country
// @Description  Updates an existing country's details
// @Tags         countries
// @Accept       json
// @Produce      json
// @Param        id      path   string                        true  "Country ID"
// @Param        request body   ports.UpdateCountryRequest true  "Update Country Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /countries/{id} [put]
func (h *CountryHandler) UpdateCountry(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid country ID"})
	}

	var req ports.UpdateCountryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	country, err := h.service.UpdateCountry(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(country)
}

// DeleteCountry godoc
// @Summary      Delete a country
// @Description  Deletes an existing country
// @Tags         countries
// @Accept       json
// @Produce      json
// @Param        id      path   string  true  "Country ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /countries/{id} [delete]
func (h *CountryHandler) DeleteCountry(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid country ID"})
	}

	if err := h.service.DeleteCountry(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Country deleted successfully"})
}

// ListCountries godoc
// @Summary      List countries
// @Description  Retrieves a paginated list of countries based on filters
// @Tags         countries
// @Accept       json
// @Produce      json
// @Param        request body ports.FilterRequest true "Filter Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /countries/list [post]
func (h *CountryHandler) ListCountries(c *fiber.Ctx) error {
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

	countries, total, err := h.service.ListCountries(c.Context(), req)
	if err != nil {
		slog.Error("Failed to list countries", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":   countries,
		"total":  total,
		"offset": req.Offset,
		"limit":  req.Limit,
	})
}

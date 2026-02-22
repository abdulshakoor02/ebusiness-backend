package handler

import (
	"context"
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserHandler struct {
	service ports.UserService
}

func NewUserHandler(service ports.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// CreateUser godoc
// @Summary      Create a new user
// @Description  Creates a new user within the current tenant (scope extracted from JWT)
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body ports.CreateUserRequest true "Create User Request"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /users [post]
// @Security     Bearer
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	var req ports.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Extract context provided by Auth Middleware (Locals -> Context)
	ctx := context.WithValue(c.Context(), "tenant_id", c.Locals("tenant_id"))

	user, err := h.service.CreateUser(ctx, req)
	if err != nil {
		slog.Error("Failed to create user", "error", err)
		if mongo.IsDuplicateKeyError(err) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Email or mobile already exists"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}

// GetUser godoc
// @Summary      Get a user by ID
// @Description  Retrieves detailed information about a specific user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /users/{id} [get]
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	idHex := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	user, err := h.service.GetUser(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(user)
}

// UpdateUser godoc
// @Summary      Update a user
// @Description  Updates a user's name and/or role
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id      path      string                   true  "User ID"
// @Param        request body      ports.UpdateUserRequest   true  "Update User Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /users/{id} [put]
func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	idHex := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	var req ports.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	user, err := h.service.UpdateUser(c.Context(), id, req)
	if err != nil {
		slog.Error("Failed to update user", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(user)
}

// ListUsers godoc
// @Summary      List users
// @Description  Retrieves a list of users for the current tenant
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body ports.FilterRequest true "Filter Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /users/list [post]
// @Security     Bearer
func (h *UserHandler) ListUsers(c *fiber.Ctx) error {
	var req ports.FilterRequest
	if err := c.BodyParser(&req); err != nil {
		// If body is empty, we might want to allow it and use defaults?
	}

	if req.Limit == 0 {
		req.Limit = 10
	}

	// Extract context
	ctx := context.WithValue(c.Context(), "tenant_id", c.Locals("tenant_id"))

	users, total, err := h.service.ListUsers(ctx, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":   users,
		"total":  total,
		"offset": req.Offset,
		"limit":  req.Limit,
	})
}

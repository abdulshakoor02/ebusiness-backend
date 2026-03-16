package handler

import (
	"log/slog"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type TenantHandler struct {
	service ports.TenantService
}

func NewTenantHandler(service ports.TenantService) *TenantHandler {
	return &TenantHandler{service: service}
}

// RegisterTenant godoc
// @Summary      Register a new tenant
// @Description  Creates a new tenant and an initial admin user
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Param        request body ports.CreateTenantRequest true "Tenant Registration Request"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /tenants [post]
func (h *TenantHandler) RegisterTenant(c *fiber.Ctx) error {
	var req ports.CreateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	tenant, user, err := h.service.RegisterTenant(c.Context(), req)
	if err != nil {
		slog.Error("Failed to register tenant", "error", err)
		if mongo.IsDuplicateKeyError(err) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Email or mobile already exists"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"tenant_id": tenant.ID,
		"user_id":   user.ID,
		"message":   "Tenant created successfully",
	})
}

// GetTenant godoc
// @Summary      Get a tenant by ID
// @Description  Retrieves detailed information about a specific tenant
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Tenant ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /tenants/{id} [get]
func (h *TenantHandler) GetTenant(c *fiber.Ctx) error {
	idHex := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	tenant, err := h.service.GetTenant(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tenant not found"})
	}

	return c.JSON(tenant)
}

// UpdateTenant godoc
// @Summary      Update a tenant
// @Description  Updates a tenant's information (name, email, logo, address)
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Param        id      path      string                    true  "Tenant ID"
// @Param        request body      ports.UpdateTenantRequest  true  "Update Tenant Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /tenants/{id} [put]
func (h *TenantHandler) UpdateTenant(c *fiber.Ctx) error {
	idHex := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	var req ports.UpdateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	tenant, err := h.service.UpdateTenant(c.Context(), id, req)
	if err != nil {
		slog.Error("Failed to update tenant", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(tenant)
}

// ListTenants godoc
// @Summary      List tenants
// @Description  Retrieves a list of tenants with optional filtering and pagination
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Param        request body ports.FilterRequest true "Filter Request"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /tenants/list [post]
func (h *TenantHandler) ListTenants(c *fiber.Ctx) error {
	var req ports.FilterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Default limit if not provided
	if req.Limit == 0 {
		req.Limit = 10
	}

	tenants, total, err := h.service.ListTenants(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":   tenants,
		"total":  total,
		"offset": req.Offset,
		"limit":  req.Limit,
	})
}

// GetUserTenant godoc
// @Summary      Get current user's tenant
// @Description  Retrieves the tenant information for the currently authenticated user
// @Tags         user
// @Accept       json
// @Produce      json
// @Success      200  {object}  domain.Tenant
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /user/tenant [get]
func (h *TenantHandler) GetUserTenant(c *fiber.Ctx) error {
	tenantIDStr, ok := c.Locals("tenant_id").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	tenantID, err := primitive.ObjectIDFromHex(tenantIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid tenant id"})
	}

	tenant, err := h.service.GetTenant(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "tenant not found"})
	}

	return c.JSON(tenant)
}

// UpdateMyTenant godoc
// @Summary      Update current user's tenant
// @Description  Admin updates their own tenant's information (logo, stamp, address)
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        request body ports.UpdateTenantRequest true "Tenant Update Request"
// @Success      200  {object}  domain.Tenant
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /user/tenant [put]
func (h *TenantHandler) UpdateMyTenant(c *fiber.Ctx) error {
	tenantIDStr, ok := c.Locals("tenant_id").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	tenantID, err := primitive.ObjectIDFromHex(tenantIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid tenant id"})
	}

	var req ports.UpdateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	tenant, err := h.service.UpdateMyTenant(c.Context(), tenantID, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(tenant)
}

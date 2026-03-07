package services

import (
	"context"
	"errors"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProductService struct {
	productRepo ports.ProductRepository
	tenantRepo  ports.TenantRepository
}

func NewProductService(productRepo ports.ProductRepository, tenantRepo ports.TenantRepository) *ProductService {
	return &ProductService{
		productRepo: productRepo,
		tenantRepo:  tenantRepo,
	}
}

func (s *ProductService) CreateProduct(ctx context.Context, req ports.CreateProductRequest) (*domain.Product, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required to create product")
	}

	product := domain.NewProduct(
		tenantID,
		req.Name,
		req.Description,
		req.Price,
	)

	if err := s.productRepo.Create(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

func (s *ProductService) GetProduct(ctx context.Context, id primitive.ObjectID) (*domain.Product, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required to get product")
	}

	return s.productRepo.GetByIDAndTenant(ctx, id, tenantID)
}

func (s *ProductService) UpdateProduct(ctx context.Context, id primitive.ObjectID, req ports.UpdateProductRequest) (*domain.Product, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required to update product")
	}

	product, err := s.productRepo.GetByIDAndTenant(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Description != "" {
		product.Description = req.Description
	}
	if req.Price > 0 {
		product.Price = req.Price
	}

	if err := s.productRepo.Update(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

func (s *ProductService) DeleteProduct(ctx context.Context, id primitive.ObjectID) error {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return errors.New("tenant context required to delete product")
	}

	_, err := s.productRepo.GetByIDAndTenant(ctx, id, tenantID)
	if err != nil {
		return err
	}

	return s.productRepo.Delete(ctx, id)
}

func (s *ProductService) ListProducts(ctx context.Context, req ports.FilterRequest) ([]*domain.Product, int64, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, 0, errors.New("tenant context required to list products")
	}

	filters := req.Filters
	if filters == nil {
		filters = make(map[string]interface{})
	}
	filters["tenant_id"] = tenantID

	return s.productRepo.List(ctx, filters, req.Offset, req.Limit)
}

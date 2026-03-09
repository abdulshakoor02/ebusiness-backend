package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InvoiceService struct {
	invoiceRepo ports.InvoiceRepository
	productRepo ports.ProductRepository
	tenantRepo  ports.TenantRepository
	leadRepo    ports.LeadRepository
	receiptRepo ports.ReceiptRepository
}

func NewInvoiceService(invoiceRepo ports.InvoiceRepository, productRepo ports.ProductRepository, tenantRepo ports.TenantRepository, leadRepo ports.LeadRepository, receiptRepo ports.ReceiptRepository) *InvoiceService {
	return &InvoiceService{
		invoiceRepo: invoiceRepo,
		productRepo: productRepo,
		tenantRepo:  tenantRepo,
		leadRepo:    leadRepo,
		receiptRepo: receiptRepo,
	}
}

func (s *InvoiceService) CreateInvoice(ctx context.Context, req ports.CreateInvoiceRequest) (*domain.Invoice, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required to create invoice")
	}

	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, errors.New("tenant not found")
	}

	leadID, err := primitive.ObjectIDFromHex(req.LeadID)
	if err != nil {
		return nil, errors.New("invalid lead_id format")
	}

	_, err = s.leadRepo.GetByID(ctx, leadID)
	if err != nil {
		return nil, errors.New("lead not found")
	}

	invoiceNumber, err := s.invoiceRepo.IncrementInvoiceNumber(ctx, tenantID)
	if err != nil {
		return nil, errors.New("failed to generate invoice number")
	}

	var items []domain.InvoiceItem
	for _, itemReq := range req.Items {
		productID, err := primitive.ObjectIDFromHex(itemReq.ProductID)
		if err != nil {
			return nil, fmt.Errorf("invalid product_id: %s", itemReq.ProductID)
		}

		product, err := s.productRepo.GetByIDAndTenant(ctx, productID, tenantID)
		if err != nil {
			return nil, fmt.Errorf("product not found: %s", itemReq.ProductID)
		}

		items = append(items, domain.InvoiceItem{
			ProductID:   productID,
			ProductName: product.Name,
			Quantity:    itemReq.Quantity,
			UnitPrice:   product.Price,
			Total:       float64(itemReq.Quantity) * product.Price,
		})
	}

	invoice := domain.NewInvoice(
		tenantID,
		leadID,
		invoiceNumber,
		items,
		req.Discount,
		tenant.Tax,
	)

	if req.DueDate != nil {
		invoice.DueDate = req.DueDate
	}

	if err := s.invoiceRepo.Create(ctx, invoice); err != nil {
		return nil, err
	}

	return invoice, nil
}

func (s *InvoiceService) GetInvoice(ctx context.Context, id primitive.ObjectID) (*domain.Invoice, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required to get invoice")
	}

	return s.invoiceRepo.GetByIDAndTenant(ctx, id, tenantID)
}

func (s *InvoiceService) UpdateInvoice(ctx context.Context, id primitive.ObjectID, req ports.UpdateInvoiceRequest) (*domain.Invoice, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required to update invoice")
	}

	invoice, err := s.invoiceRepo.GetByIDAndTenant(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if invoice.Status == domain.InvoiceStatusPaid {
		return nil, errors.New("cannot update a fully paid invoice")
	}

	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, errors.New("tenant not found")
	}

	if len(req.Items) > 0 {
		var items []domain.InvoiceItem
		for _, itemReq := range req.Items {
			productID, err := primitive.ObjectIDFromHex(itemReq.ProductID)
			if err != nil {
				return nil, fmt.Errorf("invalid product_id: %s", itemReq.ProductID)
			}

			product, err := s.productRepo.GetByIDAndTenant(ctx, productID, tenantID)
			if err != nil {
				return nil, fmt.Errorf("product not found: %s", itemReq.ProductID)
			}

			items = append(items, domain.InvoiceItem{
				ProductID:   productID,
				ProductName: product.Name,
				Quantity:    itemReq.Quantity,
				UnitPrice:   product.Price,
				Total:       float64(itemReq.Quantity) * product.Price,
			})
		}
		invoice.Items = items
	}

	if req.Discount > 0 || req.Discount == 0 {
		invoice.Discount = req.Discount
	}

	if req.DueDate != nil {
		invoice.DueDate = req.DueDate
	}

	subtotal := domain.CalculateSubtotal(invoice.Items)
	taxableAmount := subtotal - invoice.Discount
	invoice.TaxAmount = taxableAmount * (tenant.Tax / 100)
	invoice.TotalAmount = taxableAmount + invoice.TaxAmount
	invoice.TaxPercentage = tenant.Tax
	invoice.Subtotal = subtotal
	invoice.UpdatedAt = time.Now()

	if err := s.invoiceRepo.Update(ctx, invoice); err != nil {
		return nil, err
	}

	return invoice, nil
}

func (s *InvoiceService) UpdateInvoiceDueDate(ctx context.Context, id primitive.ObjectID, req ports.UpdateInvoiceDueDateRequest) (*domain.Invoice, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required to update invoice")
	}

	invoice, err := s.invoiceRepo.GetByIDAndTenant(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	invoice.DueDate = &req.DueDate
	invoice.UpdatedAt = time.Now()

	if err := s.invoiceRepo.Update(ctx, invoice); err != nil {
		return nil, err
	}

	return invoice, nil
}

func (s *InvoiceService) ListInvoices(ctx context.Context, req ports.FilterRequest) ([]*domain.Invoice, int64, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, 0, errors.New("tenant context required to list invoices")
	}

	filters := req.Filters
	if filters == nil {
		filters = make(map[string]interface{})
	}
	filters["tenant_id"] = tenantID

	return s.invoiceRepo.List(ctx, filters, req.Offset, req.Limit)
}

func (s *InvoiceService) GetInvoicesByLeadID(ctx context.Context, leadID primitive.ObjectID) ([]*domain.Invoice, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required to get invoices")
	}

	invoices, err := s.invoiceRepo.GetByLeadID(ctx, leadID)
	if err != nil {
		return nil, err
	}

	var filteredInvoices []*domain.Invoice
	for _, inv := range invoices {
		if inv.TenantID == tenantID {
			filteredInvoices = append(filteredInvoices, inv)
		}
	}

	return filteredInvoices, nil
}

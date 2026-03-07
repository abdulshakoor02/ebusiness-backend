package services

import (
	"context"
	"errors"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ReceiptService struct {
	receiptRepo ports.ReceiptRepository
	invoiceRepo ports.InvoiceRepository
	tenantRepo  ports.TenantRepository
}

func NewReceiptService(receiptRepo ports.ReceiptRepository, invoiceRepo ports.InvoiceRepository, tenantRepo ports.TenantRepository) *ReceiptService {
	return &ReceiptService{
		receiptRepo: receiptRepo,
		invoiceRepo: invoiceRepo,
		tenantRepo:  tenantRepo,
	}
}

func (s *ReceiptService) CreateReceipt(ctx context.Context, invoiceID primitive.ObjectID, req ports.CreateReceiptRequest) (*domain.Receipt, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required to create receipt")
	}

	invoice, err := s.invoiceRepo.GetByIDAndTenant(ctx, invoiceID, tenantID)
	if err != nil {
		return nil, errors.New("invoice not found")
	}

	if invoice.Status == domain.InvoiceStatusPaid {
		return nil, errors.New("invoice is already fully paid")
	}

	existingPaidAmount, existingPaidAmountVat, err := s.receiptRepo.SumPaidAmountByInvoiceID(ctx, invoiceID)
	if err != nil {
		return nil, errors.New("failed to calculate existing payments")
	}

	taxAmountOnPayment := req.AmountPaid * (invoice.TaxPercentage / 100)
	totalPaidThisTime := req.AmountPaid + taxAmountOnPayment

	remainingAmount := invoice.TotalAmount - existingPaidAmountVat

	if totalPaidThisTime > remainingAmount {
		return nil, errors.New("payment exceeds remaining amount to be paid")
	}

	receiptNumber, err := s.receiptRepo.IncrementReceiptNumber(ctx, tenantID)
	if err != nil {
		return nil, errors.New("failed to generate receipt number")
	}

	paymentDate := req.PaymentDate
	if paymentDate.IsZero() {
		paymentDate = time.Now()
	}

	receipt := domain.NewReceipt(
		tenantID,
		invoiceID,
		float64(receiptNumber),
		req.AmountPaid,
		invoice.TaxPercentage,
		paymentDate,
	)

	if err := s.receiptRepo.Create(ctx, receipt); err != nil {
		return nil, err
	}

	newPaidAmount := existingPaidAmount + req.AmountPaid
	newPaidAmountVat := existingPaidAmountVat + totalPaidThisTime

	invoice.PaidAmount = newPaidAmount
	invoice.PaidAmountVat = newPaidAmountVat

	if newPaidAmountVat >= invoice.TotalAmount {
		invoice.Status = domain.InvoiceStatusPaid
	} else if newPaidAmount > 0 {
		invoice.Status = domain.InvoiceStatusPartial
	}

	invoice.UpdatedAt = time.Now()

	if err := s.invoiceRepo.Update(ctx, invoice); err != nil {
		return nil, err
	}

	return receipt, nil
}

func (s *ReceiptService) GetReceipt(ctx context.Context, id primitive.ObjectID) (*domain.Receipt, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required to get receipt")
	}

	return s.receiptRepo.GetByIDAndTenant(ctx, id, tenantID)
}

func (s *ReceiptService) ListReceiptsByInvoiceID(ctx context.Context, invoiceID primitive.ObjectID) ([]*domain.Receipt, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required to list receipts")
	}

	invoice, err := s.invoiceRepo.GetByIDAndTenant(ctx, invoiceID, tenantID)
	if err != nil {
		return nil, errors.New("invoice not found")
	}

	receipts, err := s.receiptRepo.GetByInvoiceID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}

	var filteredReceipts []*domain.Receipt
	for _, r := range receipts {
		if r.TenantID == tenantID {
			filteredReceipts = append(filteredReceipts, r)
		}
	}

	_ = invoice

	return filteredReceipts, nil
}

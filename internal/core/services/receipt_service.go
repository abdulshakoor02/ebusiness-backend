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
	leadRepo    ports.LeadRepository
}

func NewReceiptService(receiptRepo ports.ReceiptRepository, invoiceRepo ports.InvoiceRepository, tenantRepo ports.TenantRepository, leadRepo ports.LeadRepository) *ReceiptService {
	return &ReceiptService{
		receiptRepo: receiptRepo,
		invoiceRepo: invoiceRepo,
		tenantRepo:  tenantRepo,
		leadRepo:    leadRepo,
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

func (s *ReceiptService) UpdateReceipt(ctx context.Context, id primitive.ObjectID, req ports.UpdateReceiptRequest) (*domain.Receipt, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required to update receipt")
	}

	receipt, err := s.receiptRepo.GetByIDAndTenant(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	invoice, err := s.invoiceRepo.GetByIDAndTenant(ctx, receipt.InvoiceID, tenantID)
	if err != nil {
		return nil, errors.New("invoice not found")
	}

	if req.AmountPaid > 0 {
		receipt.AmountPaid = req.AmountPaid
		receipt.TaxAmount = req.AmountPaid * (invoice.TaxPercentage / 100)
		receipt.TotalPaid = receipt.AmountPaid + receipt.TaxAmount
	}

	if !req.PaymentDate.IsZero() {
		receipt.PaymentDate = req.PaymentDate
	}

	otherReceipts, err := s.receiptRepo.GetByInvoiceID(ctx, invoice.ID)
	if err != nil {
		return nil, errors.New("failed to get other receipts")
	}

	var otherPaidAmount float64
	for _, r := range otherReceipts {
		if r.ID != receipt.ID {
			otherPaidAmount += r.AmountPaid
		}
	}

	newTotalPaid := otherPaidAmount + receipt.AmountPaid
	newTotalPaidVat := newTotalPaid + (newTotalPaid * invoice.TaxPercentage / 100)

	if newTotalPaidVat > invoice.TotalAmount {
		return nil, errors.New("updated payment exceeds remaining invoice amount")
	}

	if err := s.receiptRepo.Update(ctx, receipt); err != nil {
		return nil, err
	}

	invoice.PaidAmount = otherPaidAmount + receipt.AmountPaid
	invoice.PaidAmountVat = invoice.PaidAmount + (invoice.PaidAmount * invoice.TaxPercentage / 100)

	if invoice.PaidAmountVat >= invoice.TotalAmount {
		invoice.Status = domain.InvoiceStatusPaid
	} else if invoice.PaidAmount > 0 {
		invoice.Status = domain.InvoiceStatusPartial
	} else {
		invoice.Status = domain.InvoiceStatusPending
	}

	invoice.UpdatedAt = time.Now()

	if err := s.invoiceRepo.Update(ctx, invoice); err != nil {
		return nil, err
	}

	return receipt, nil
}

func (s *ReceiptService) DeleteReceipt(ctx context.Context, id primitive.ObjectID) error {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return errors.New("tenant context required to delete receipt")
	}

	receipt, err := s.receiptRepo.GetByIDAndTenant(ctx, id, tenantID)
	if err != nil {
		return err
	}

	invoice, err := s.invoiceRepo.GetByIDAndTenant(ctx, receipt.InvoiceID, tenantID)
	if err != nil {
		return errors.New("invoice not found")
	}

	if err := s.receiptRepo.Delete(ctx, id); err != nil {
		return err
	}

	otherReceipts, err := s.receiptRepo.GetByInvoiceID(ctx, invoice.ID)
	if err != nil {
		return errors.New("failed to get other receipts")
	}

	var otherPaidAmount float64
	for _, r := range otherReceipts {
		if r.ID != receipt.ID {
			otherPaidAmount += r.AmountPaid
		}
	}

	invoice.PaidAmount = otherPaidAmount
	invoice.PaidAmountVat = otherPaidAmount + (otherPaidAmount * invoice.TaxPercentage / 100)

	if invoice.PaidAmountVat >= invoice.TotalAmount {
		invoice.Status = domain.InvoiceStatusPaid
	} else if invoice.PaidAmount > 0 {
		invoice.Status = domain.InvoiceStatusPartial
	} else {
		invoice.Status = domain.InvoiceStatusPending
	}

	invoice.UpdatedAt = time.Now()

	return s.invoiceRepo.Update(ctx, invoice)
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

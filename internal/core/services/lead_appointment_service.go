package services

import (
	"context"
	"errors"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadAppointmentService struct {
	appointmentRepo ports.LeadAppointmentRepository
	leadRepo        ports.LeadRepository
}

func NewLeadAppointmentService(appointmentRepo ports.LeadAppointmentRepository, leadRepo ports.LeadRepository) *LeadAppointmentService {
	return &LeadAppointmentService{
		appointmentRepo: appointmentRepo,
		leadRepo:        leadRepo,
	}
}

func (s *LeadAppointmentService) CreateLeadAppointment(ctx context.Context, leadID primitive.ObjectID, req ports.CreateLeadAppointmentRequest) (*domain.LeadAppointment, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required")
	}

	organizerID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, errors.New("user context required")
	}

	// Verify lead exists and belongs to tenant
	_, err := s.leadRepo.GetByID(ctx, leadID)
	if err != nil {
		return nil, errors.New("lead not found or unauthorized")
	}

	// Basic validation for dates
	if req.StartTime.IsZero() || req.EndTime.IsZero() {
		return nil, errors.New("start_time and end_time are required")
	}
	if req.EndTime.Before(req.StartTime) {
		return nil, errors.New("end_time must be after start_time")
	}

	status := domain.AppointmentStatus(req.Status)
	if status == "" {
		status = domain.StatusScheduled // default
	}

	appointment := domain.NewLeadAppointment(
		tenantID,
		leadID,
		organizerID,
		req.Title,
		req.Description,
		req.StartTime,
		req.EndTime,
		status,
	)

	if err := s.appointmentRepo.Create(ctx, appointment); err != nil {
		return nil, err
	}

	return appointment, nil
}

func (s *LeadAppointmentService) GetLeadAppointment(ctx context.Context, id primitive.ObjectID) (*domain.LeadAppointment, error) {
	return s.appointmentRepo.GetByID(ctx, id)
}

func (s *LeadAppointmentService) UpdateLeadAppointment(ctx context.Context, id primitive.ObjectID, req ports.UpdateLeadAppointmentRequest) (*domain.LeadAppointment, error) {
	appointment, err := s.appointmentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, errors.New("user context required")
	}

	// Ensure only the organizer can update the appointment
	if appointment.OrganizerID != userID {
		return nil, errors.New("unauthorized to update this appointment")
	}

	if req.Title != "" {
		appointment.Title = req.Title
	}
	if req.Description != "" {
		appointment.Description = req.Description
	}
	if !req.StartTime.IsZero() {
		appointment.StartTime = req.StartTime
	}
	if !req.EndTime.IsZero() {
		appointment.EndTime = req.EndTime
	}

	if appointment.EndTime.Before(appointment.StartTime) {
		return nil, errors.New("end_time must be after start_time")
	}

	if req.Status != "" {
		appointment.Status = domain.AppointmentStatus(req.Status)
	}

	if err := s.appointmentRepo.Update(ctx, appointment); err != nil {
		return nil, err
	}

	return appointment, nil
}

func (s *LeadAppointmentService) DeleteLeadAppointment(ctx context.Context, id primitive.ObjectID) error {
	appointment, err := s.appointmentRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return errors.New("user context required")
	}

	// Ensure only the organizer can delete the appointment
	if appointment.OrganizerID != userID {
		return errors.New("unauthorized to delete this appointment")
	}

	return s.appointmentRepo.Delete(ctx, id)
}

func (s *LeadAppointmentService) ListLeadAppointments(ctx context.Context, leadID primitive.ObjectID, req ports.FilterRequest) ([]*domain.LeadAppointment, int64, error) {
	return s.appointmentRepo.ListByLeadID(ctx, leadID, req.Filters, req.Offset, req.Limit)
}

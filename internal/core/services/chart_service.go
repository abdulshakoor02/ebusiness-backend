package services

import (
	"context"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
)

type ChartService struct {
	appointmentRepo ports.LeadAppointmentRepository
	commentRepo     ports.LeadCommentRepository
}

func NewChartService(appointmentRepo ports.LeadAppointmentRepository, commentRepo ports.LeadCommentRepository) *ChartService {
	return &ChartService{
		appointmentRepo: appointmentRepo,
		commentRepo:     commentRepo,
	}
}

func (s *ChartService) GetMonthlySummary(ctx context.Context, month, year int) ([]ports.MonthlyChartDataPoint, error) {
	now := time.Now()
	if month <= 0 || month > 12 {
		month = int(now.Month())
	}
	if year <= 0 {
		year = now.Year()
	}

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0).Add(-time.Second)

	appointmentsByDate, err := s.appointmentRepo.CountByDateRange(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	commentsByDate, err := s.commentRepo.CountByDateRange(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	daysInMonth := daysInMonth(year, time.Month(month))
	data := make([]ports.MonthlyChartDataPoint, daysInMonth)

	for day := 1; day <= daysInMonth; day++ {
		dateStr := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		data[day-1] = ports.MonthlyChartDataPoint{
			Date:               dateStr,
			AppointmentsBooked: appointmentsByDate[dateStr],
			CommentsAdded:      commentsByDate[dateStr],
		}
	}

	return data, nil
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

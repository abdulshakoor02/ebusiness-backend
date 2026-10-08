package services

import (
	"context"
	"strings"
	"time"
	_ "time/tzdata" // Keep IANA calendar windows available in minimal containers.

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
)

type DashboardService struct{ repo ports.DashboardRepository }

func NewDashboardService(repo ports.DashboardRepository) *DashboardService {
	return &DashboardService{repo: repo}
}

func (s *DashboardService) GetSummary(ctx context.Context, scope ports.DashboardScope, zone string, now time.Time) (*ports.DashboardSummary, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	window, err := dashboardWindow(zone, now)
	if err != nil {
		return nil, err
	}
	return s.repo.GetSummary(ctx, scope, window)
}

// dashboardWindow compares month-to-date with the same local calendar elapsed
// period last month. Shorter previous months are clamped at their exclusive end.
func dashboardWindow(zone string, now time.Time) (ports.DashboardWindow, error) {
	if zone == "" {
		zone = "UTC"
	}
	if len(zone) > 100 || zone == "Local" || strings.Contains(zone, "..") {
		return ports.DashboardWindow{}, ports.ErrDashboardZone
	}
	location, err := time.LoadLocation(zone)
	if err != nil {
		return ports.DashboardWindow{}, ports.ErrDashboardZone
	}
	if now.IsZero() {
		now = time.Now()
	}
	local := now.In(location)
	start := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, location)
	previous := start.AddDate(0, -1, 0)
	comparisonEnd := time.Date(previous.Year(), previous.Month(), local.Day(), local.Hour(), local.Minute(), local.Second(), local.Nanosecond(), location)
	if comparisonEnd.After(start) {
		comparisonEnd = start
	}
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	return ports.DashboardWindow{
		Now: now.UTC(), Start: start.UTC(), End: now.UTC(),
		ComparisonStart: previous.UTC(), ComparisonEnd: comparisonEnd.UTC(),
		TrendStart: start.AddDate(0, -5, 0).UTC(), TodayStart: today.UTC(),
		TodayEnd: today.AddDate(0, 0, 1).UTC(), UpcomingEnd: local.AddDate(0, 0, 30).UTC(), Location: location,
	}, nil
}

var _ ports.DashboardService = (*DashboardService)(nil)

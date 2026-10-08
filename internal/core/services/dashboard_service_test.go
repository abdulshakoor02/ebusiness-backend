package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type dashboardRepositoryFunc func(context.Context, ports.DashboardScope, ports.DashboardWindow) (*ports.DashboardSummary, error)

func (f dashboardRepositoryFunc) GetSummary(ctx context.Context, scope ports.DashboardScope, window ports.DashboardWindow) (*ports.DashboardSummary, error) {
	return f(ctx, scope, window)
}

func dashboardServiceTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestDashboardServiceCalendarWindows(t *testing.T) {
	tests := []struct {
		name, zone, now, start, previousStart, previousEnd, trendStart, todayStart, todayEnd, upcomingEnd string
	}{
		{
			name: "default UTC same elapsed prior month", now: "2024-06-15T12:34:56.123456789Z",
			start: "2024-06-01T00:00:00Z", previousStart: "2024-05-01T00:00:00Z", previousEnd: "2024-05-15T12:34:56.123456789Z",
			trendStart: "2024-01-01T00:00:00Z", todayStart: "2024-06-15T00:00:00Z", todayEnd: "2024-06-16T00:00:00Z", upcomingEnd: "2024-07-15T12:34:56.123456789Z",
		},
		{
			name: "IANA local month differs from UTC", zone: "Asia/Dubai", now: "2024-03-31T21:30:00Z",
			start: "2024-04-01T00:00:00+04:00", previousStart: "2024-03-01T00:00:00+04:00", previousEnd: "2024-03-01T01:30:00+04:00",
			trendStart: "2023-11-01T00:00:00+04:00", todayStart: "2024-04-01T00:00:00+04:00", todayEnd: "2024-04-02T00:00:00+04:00", upcomingEnd: "2024-05-01T01:30:00+04:00",
		},
		{
			name: "DST spring day has 23 hours", zone: "America/New_York", now: "2024-03-10T12:34:56-04:00",
			start: "2024-03-01T00:00:00-05:00", previousStart: "2024-02-01T00:00:00-05:00", previousEnd: "2024-02-10T12:34:56-05:00",
			trendStart: "2023-10-01T00:00:00-04:00", todayStart: "2024-03-10T00:00:00-05:00", todayEnd: "2024-03-11T00:00:00-04:00", upcomingEnd: "2024-04-09T12:34:56-04:00",
		},
		{
			name: "DST fall day has 25 hours", zone: "America/New_York", now: "2024-11-03T12:34:56-05:00",
			start: "2024-11-01T00:00:00-04:00", previousStart: "2024-10-01T00:00:00-04:00", previousEnd: "2024-10-03T12:34:56-04:00",
			trendStart: "2024-06-01T00:00:00-04:00", todayStart: "2024-11-03T00:00:00-04:00", todayEnd: "2024-11-04T00:00:00-05:00", upcomingEnd: "2024-12-03T12:34:56-05:00",
		},
		{
			name: "leap February clamps prior exclusive end", zone: "UTC", now: "2024-03-31T16:17:18Z",
			start: "2024-03-01T00:00:00Z", previousStart: "2024-02-01T00:00:00Z", previousEnd: "2024-03-01T00:00:00Z",
			trendStart: "2023-10-01T00:00:00Z", todayStart: "2024-03-31T00:00:00Z", todayEnd: "2024-04-01T00:00:00Z", upcomingEnd: "2024-04-30T16:17:18Z",
		},
		{
			name: "common February clamps prior exclusive end", zone: "UTC", now: "2023-03-29T08:00:00Z",
			start: "2023-03-01T00:00:00Z", previousStart: "2023-02-01T00:00:00Z", previousEnd: "2023-03-01T00:00:00Z",
			trendStart: "2022-10-01T00:00:00Z", todayStart: "2023-03-29T00:00:00Z", todayEnd: "2023-03-30T00:00:00Z", upcomingEnd: "2023-04-28T08:00:00Z",
		},
		{
			name: "leap day compares January 29 not full January", zone: "UTC", now: "2024-02-29T08:00:00Z",
			start: "2024-02-01T00:00:00Z", previousStart: "2024-01-01T00:00:00Z", previousEnd: "2024-01-29T08:00:00Z",
			trendStart: "2023-09-01T00:00:00Z", todayStart: "2024-02-29T00:00:00Z", todayEnd: "2024-03-01T00:00:00Z", upcomingEnd: "2024-03-30T08:00:00Z",
		},
		{
			name: "thirty day prior month clamps", zone: "UTC", now: "2024-05-31T08:00:00Z",
			start: "2024-05-01T00:00:00Z", previousStart: "2024-04-01T00:00:00Z", previousEnd: "2024-05-01T00:00:00Z",
			trendStart: "2023-12-01T00:00:00Z", todayStart: "2024-05-31T00:00:00Z", todayEnd: "2024-06-01T00:00:00Z", upcomingEnd: "2024-06-30T08:00:00Z",
		},
		{
			name: "year rollover at exact month boundary is empty MTD", zone: "UTC", now: "2025-01-01T00:00:00Z",
			start: "2025-01-01T00:00:00Z", previousStart: "2024-12-01T00:00:00Z", previousEnd: "2024-12-01T00:00:00Z",
			trendStart: "2024-08-01T00:00:00Z", todayStart: "2025-01-01T00:00:00Z", todayEnd: "2025-01-02T00:00:00Z", upcomingEnd: "2025-01-31T00:00:00Z",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := dashboardServiceTime(t, tt.now)
			scope := ports.DashboardScope{Role: "admin", TenantID: primitive.NewObjectID(), UserID: primitive.NewObjectID()}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			want := &ports.DashboardSummary{Role: "admin", Scope: "tenant"}
			calls := 0
			service := NewDashboardService(dashboardRepositoryFunc(func(gotCtx context.Context, gotScope ports.DashboardScope, w ports.DashboardWindow) (*ports.DashboardSummary, error) {
				calls++
				if gotCtx != ctx || gotScope != scope {
					t.Error("service did not forward verified scope and context unchanged")
				}
				for name, pair := range map[string]struct {
					got  time.Time
					want string
				}{
					"Now": {w.Now, tt.now}, "EndExclusive": {w.End, tt.now}, "Start": {w.Start, tt.start},
					"ComparisonStart": {w.ComparisonStart, tt.previousStart}, "ComparisonEndExclusive": {w.ComparisonEnd, tt.previousEnd},
					"TrendStart": {w.TrendStart, tt.trendStart}, "TodayStart": {w.TodayStart, tt.todayStart},
					"TodayEndExclusive": {w.TodayEnd, tt.todayEnd}, "UpcomingEndExclusive": {w.UpcomingEnd, tt.upcomingEnd},
				} {
					if !pair.got.Equal(dashboardServiceTime(t, pair.want)) || pair.got.Location() != time.UTC {
						t.Errorf("%s = %s (%s), want UTC instant %s", name, pair.got, pair.got.Location(), pair.want)
					}
				}
				zone := tt.zone
				if zone == "" {
					zone = "UTC"
				}
				if w.Location == nil || w.Location.String() != zone {
					t.Errorf("Location = %v, want %s", w.Location, zone)
				}
				return want, nil
			}))
			got, err := service.GetSummary(ctx, scope, tt.zone, now)
			if err != nil || got != want || calls != 1 {
				t.Fatalf("GetSummary = (%v, %v), calls = %d", got, err, calls)
			}
		})
	}
}

func TestDashboardServiceRejectsInvalidScopeAndZoneBeforeRepository(t *testing.T) {
	user, tenant := primitive.NewObjectID(), primitive.NewObjectID()
	tests := []struct {
		name  string
		scope ports.DashboardScope
		zone  string
		want  error
	}{
		{"unknown role", ports.DashboardScope{Role: "manager", UserID: user, TenantID: tenant}, "UTC", ports.ErrDashboardRole},
		{"empty role", ports.DashboardScope{UserID: user, TenantID: tenant}, "UTC", ports.ErrDashboardRole},
		{"nil admin user", ports.DashboardScope{Role: "admin", TenantID: tenant}, "UTC", ports.ErrDashboardIdentity},
		{"nil user identity", ports.DashboardScope{Role: "user", TenantID: tenant}, "UTC", ports.ErrDashboardIdentity},
		{"nil platform user", ports.DashboardScope{Role: "superadmin"}, "UTC", ports.ErrDashboardIdentity},
		{"nil admin tenant", ports.DashboardScope{Role: "admin", UserID: user}, "UTC", ports.ErrDashboardIdentity},
		{"nil personal tenant", ports.DashboardScope{Role: "user", UserID: user}, "UTC", ports.ErrDashboardIdentity},
	}
	for _, zone := range []string{"Not/A_Zone", "Local", "../UTC", "Europe/../UTC", strings.Repeat("A", 101), "+04:00", " UTC "} {
		tests = append(tests, struct {
			name  string
			scope ports.DashboardScope
			zone  string
			want  error
		}{"invalid zone " + zone, ports.DashboardScope{Role: "user", TenantID: tenant, UserID: user}, zone, ports.ErrDashboardZone})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewDashboardService(dashboardRepositoryFunc(func(context.Context, ports.DashboardScope, ports.DashboardWindow) (*ports.DashboardSummary, error) {
				t.Fatal("invalid request reached repository")
				return nil, nil
			}))
			got, err := service.GetSummary(context.Background(), tt.scope, tt.zone, time.Now())
			if got != nil || !errors.Is(err, tt.want) {
				t.Fatalf("GetSummary = (%v, %v), want (nil, %v)", got, err, tt.want)
			}
		})
	}
}

func TestDashboardServicePlatformNeedsNoTenantAndPropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("repository unavailable")
	scope := ports.DashboardScope{Role: "superadmin", UserID: primitive.NewObjectID()}
	calls := 0
	service := NewDashboardService(dashboardRepositoryFunc(func(_ context.Context, got ports.DashboardScope, _ ports.DashboardWindow) (*ports.DashboardSummary, error) {
		calls++
		if got != scope {
			t.Errorf("scope = %+v, want %+v", got, scope)
		}
		return nil, wantErr
	}))
	result, err := service.GetSummary(context.Background(), scope, "UTC", time.Now())
	if result != nil || !errors.Is(err, wantErr) || calls != 1 {
		t.Fatalf("GetSummary = (%v, %v), calls = %d", result, err, calls)
	}
}

func TestDashboardServiceZeroNowUsesCurrentInstant(t *testing.T) {
	before := time.Now().UTC()
	var window ports.DashboardWindow
	service := NewDashboardService(dashboardRepositoryFunc(func(_ context.Context, _ ports.DashboardScope, w ports.DashboardWindow) (*ports.DashboardSummary, error) {
		window = w
		return &ports.DashboardSummary{}, nil
	}))
	_, err := service.GetSummary(context.Background(), ports.DashboardScope{Role: "superadmin", UserID: primitive.NewObjectID()}, "UTC", time.Time{})
	after := time.Now().UTC()
	if err != nil || window.Now.Before(before) || window.Now.After(after) || !window.End.Equal(window.Now) {
		t.Fatalf("zero-now window = %+v, err = %v; expected current instant between %s and %s", window, err, before, after)
	}
}

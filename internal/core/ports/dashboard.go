package ports

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrDashboardIdentity = errors.New("invalid dashboard identity")
	ErrDashboardRole     = errors.New("unsupported dashboard role")
	ErrDashboardZone     = errors.New("invalid dashboard time zone")
)

// DashboardScope contains only verified identity claims, never client filters.
type DashboardScope struct {
	Role     string
	TenantID primitive.ObjectID
	UserID   primitive.ObjectID
}

func (s DashboardScope) Validate() error {
	if s.Role != "superadmin" && s.Role != "admin" && s.Role != "user" {
		return ErrDashboardRole
	}
	if s.UserID.IsZero() || (s.Role != "superadmin" && s.TenantID.IsZero()) {
		return ErrDashboardIdentity
	}
	return nil
}

func (s DashboardScope) Label() string {
	switch s.Role {
	case "superadmin":
		return "platform"
	case "admin":
		return "tenant"
	default:
		return "personal"
	}
}

// DashboardWindow uses half-open instants and local calendar boundaries.
type DashboardWindow struct {
	Now, Start, End, ComparisonStart, ComparisonEnd time.Time
	TrendStart, TodayStart, TodayEnd, UpcomingEnd   time.Time
	Location                                        *time.Location
}

type DashboardPeriod struct {
	Start           string `json:"start"`
	EndExclusive    string `json:"end_exclusive"`
	ComparisonStart string `json:"comparison_start"`
	ComparisonEnd   string `json:"comparison_end_exclusive"`
	TimeZone        string `json:"timezone"`
}

// Previous is absent for snapshots, rather than comparing totals with monthly growth.
type DashboardMetric struct {
	Value    float64  `json:"value"`
	Previous *float64 `json:"previous,omitempty"`
	Format   string   `json:"format"`
	Period   string   `json:"period"`
}

type DashboardTrendPoint struct {
	Label        string   `json:"label"`
	Start        string   `json:"start"`
	Leads        int64    `json:"leads"`
	Tenants      int64    `json:"tenants"`
	Users        int64    `json:"users"`
	Appointments int64    `json:"appointments"`
	FollowUps    int64    `json:"follow_ups"`
	Comments     int64    `json:"comments"`
	Revenue      *float64 `json:"revenue,omitempty"`
}

type DashboardBreakdown struct {
	Label string `json:"label" bson:"label"`
	Value int64  `json:"value" bson:"value"`
}

type DashboardTask struct {
	ID        string `json:"id"`
	LeadID    string `json:"lead_id,omitempty"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Status    string `json:"status"`
	Overdue   bool   `json:"overdue"`
}

type DashboardTenantHighlight struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	Leads     int64  `json:"leads"`
	Users     int64  `json:"users"`
}

type DashboardLead struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// Money exists only in a single tenant admin's view. Platform currencies are not
// added together; a user's personal workload never includes tenant financials.
type DashboardSummary struct {
	GeneratedAt      string                     `json:"generated_at"`
	Role             string                     `json:"role"`
	Scope            string                     `json:"scope"`
	TenantName       string                     `json:"tenant_name,omitempty"`
	Currency         string                     `json:"currency,omitempty"`
	Period           DashboardPeriod            `json:"period"`
	Metrics          map[string]DashboardMetric `json:"metrics"`
	Trend            []DashboardTrendPoint      `json:"trend"`
	LeadStatus       []DashboardBreakdown       `json:"lead_status"`
	Upcoming         []DashboardTask            `json:"upcoming"`
	RecentLeads      []DashboardLead            `json:"recent_leads"`
	TenantHighlights []DashboardTenantHighlight `json:"tenant_highlights"`
}

type DashboardRepository interface {
	GetSummary(context.Context, DashboardScope, DashboardWindow) (*DashboardSummary, error)
}

type DashboardService interface {
	GetSummary(context.Context, DashboardScope, string, time.Time) (*DashboardSummary, error)
}

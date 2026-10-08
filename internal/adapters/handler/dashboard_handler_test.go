package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/abdulshakoor02/goCrmBackend/pkg/middleware"
	"github.com/abdulshakoor02/goCrmBackend/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const dashboardJWTSecret = "dashboard-handler-test-secret"

type dashboardServiceRecorder struct {
	calls  int
	scope  ports.DashboardScope
	zone   string
	now    time.Time
	ctx    context.Context
	result *ports.DashboardSummary
	err    error
}

func (s *dashboardServiceRecorder) GetSummary(ctx context.Context, scope ports.DashboardScope, zone string, now time.Time) (*ports.DashboardSummary, error) {
	s.calls++
	s.scope, s.zone, s.now, s.ctx = scope, zone, now, ctx
	return s.result, s.err
}

type dashboardPermissionRecorder struct {
	ports.RolePermissionRepository
	allowed            bool
	err                error
	calls              int
	role, path, method string
}

func (r *dashboardPermissionRecorder) CheckPermissionByPathMethod(_ context.Context, role, path, method string) (bool, error) {
	r.calls++
	r.role, r.path, r.method = role, path, method
	return r.allowed, r.err
}

func dashboardHandlerApp(service ports.DashboardService, permission *dashboardPermissionRecorder) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	handlers := []fiber.Handler{middleware.Protected(dashboardJWTSecret)}
	if permission != nil {
		handlers = append(handlers, middleware.NewAuthMiddleware(permission))
	}
	handlers = append(handlers, NewDashboardHandler(service).GetSummary)
	app.Get("/api/dashboard/summary", handlers...)
	return app
}

func dashboardClaims(role string) utils.JWTClaims {
	return utils.JWTClaims{
		UserID: primitive.NewObjectID().Hex(), TenantID: primitive.NewObjectID().Hex(), Role: role,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}
}

func dashboardSignToken(t *testing.T, claims utils.JWTClaims, secret string) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + token
}

func dashboardRequest(t *testing.T, app *fiber.App, target, authorization string) (*http.Response, []byte) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	response, err := app.Test(req, 5000)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response, body
}

func dashboardAssertNoStore(t *testing.T, response *http.Response) {
	t.Helper()
	cache := response.Header.Get("Cache-Control")
	if !strings.Contains(cache, "no-store") || !strings.Contains(cache, "private") {
		t.Errorf("Cache-Control = %q, want private, no-store", cache)
	}
}

func TestDashboardHandlerProtectedIdentityAndTimezone(t *testing.T) {
	for _, tt := range []struct{ role, query, zone, label string }{
		{"admin", "", "UTC", "tenant"},
		{"user", "?timezone=America%2FNew_York", "America/New_York", "personal"},
		{"superadmin", "?timezone=Asia%2FDubai", "Asia/Dubai", "platform"},
	} {
		t.Run(tt.role, func(t *testing.T) {
			claims := dashboardClaims(tt.role)
			if tt.role == "superadmin" {
				claims.TenantID = "" // Platform access must not require a tenant claim.
			}
			service := &dashboardServiceRecorder{result: &ports.DashboardSummary{
				Role: tt.role, Scope: tt.label, Metrics: map[string]ports.DashboardMetric{},
				Trend: []ports.DashboardTrendPoint{}, LeadStatus: []ports.DashboardBreakdown{},
				Upcoming: []ports.DashboardTask{}, RecentLeads: []ports.DashboardLead{}, TenantHighlights: []ports.DashboardTenantHighlight{},
			}}
			permission := &dashboardPermissionRecorder{allowed: true}
			before := time.Now().UTC()
			response, body := dashboardRequest(t, dashboardHandlerApp(service, permission), "/api/dashboard/summary"+tt.query, dashboardSignToken(t, claims, dashboardJWTSecret))
			after := time.Now().UTC()
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, body = %s", response.StatusCode, body)
			}
			dashboardAssertNoStore(t, response)
			if !strings.HasPrefix(response.Header.Get("Content-Type"), "application/json") {
				t.Errorf("Content-Type = %s", response.Header.Get("Content-Type"))
			}
			var envelope struct {
				Data ports.DashboardSummary `json:"data"`
			}
			if err := json.Unmarshal(body, &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Data.Role != tt.role || envelope.Data.Scope != tt.label {
				t.Errorf("data envelope = %+v", envelope.Data)
			}
			if service.calls != 1 || service.scope.Role != tt.role || service.scope.UserID.Hex() != claims.UserID || service.zone != tt.zone {
				t.Fatalf("recorded call: %+v", service)
			}
			if tt.role == "superadmin" {
				if !service.scope.TenantID.IsZero() {
					t.Errorf("platform tenant = %s, want zero", service.scope.TenantID)
				}
			} else if service.scope.TenantID.Hex() != claims.TenantID {
				t.Errorf("tenant = %s, want JWT tenant %s", service.scope.TenantID.Hex(), claims.TenantID)
			}
			if service.now.Before(before) || service.now.After(after) || service.now.Location() != time.UTC {
				t.Errorf("service now = %s, outside request's UTC instants", service.now)
			}
			deadline, ok := service.ctx.Deadline()
			if !ok || deadline.Before(before.Add(8*time.Second)) || deadline.After(after.Add(8*time.Second)) {
				t.Errorf("context deadline = %s, present = %t; want eight-second request timeout", deadline, ok)
			}
			if !errors.Is(service.ctx.Err(), context.Canceled) {
				t.Errorf("request context not canceled after response: %v", service.ctx.Err())
			}
			if permission.calls != 1 || permission.role != tt.role || permission.path != "/api/dashboard/summary" || permission.method != http.MethodGet {
				t.Errorf("RBAC call = %+v", permission)
			}
		})
	}
}

func TestDashboardHandlerRejectsUnauthenticatedAndInvalidClaims(t *testing.T) {
	for _, tt := range []struct {
		name           string
		mutate         func(*utils.JWTClaims)
		authorization  string
		status         int
		handlerReached bool
	}{
		{name: "no token", status: 401},
		{name: "malformed token", authorization: "Bearer broken", status: 401},
		{name: "wrong authentication scheme", authorization: "Basic broken", status: 401},
		{name: "expired JWT", mutate: func(c *utils.JWTClaims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour)) }, status: 401},
		{name: "unknown role", mutate: func(c *utils.JWTClaims) { c.Role = "manager" }, status: 403, handlerReached: true},
		{name: "empty role", mutate: func(c *utils.JWTClaims) { c.Role = "" }, status: 401, handlerReached: true},
		{name: "bad user id", mutate: func(c *utils.JWTClaims) { c.UserID = "not-an-objectid" }, status: 401, handlerReached: true},
		{name: "empty user id", mutate: func(c *utils.JWTClaims) { c.UserID = "" }, status: 401, handlerReached: true},
		{name: "nil user id", mutate: func(c *utils.JWTClaims) { c.UserID = primitive.NilObjectID.Hex() }, status: 401, handlerReached: true},
		{name: "bad tenant id", mutate: func(c *utils.JWTClaims) { c.TenantID = "not-an-objectid" }, status: 401, handlerReached: true},
		{name: "empty tenant id", mutate: func(c *utils.JWTClaims) { c.TenantID = "" }, status: 401, handlerReached: true},
		{name: "nil tenant id", mutate: func(c *utils.JWTClaims) { c.TenantID = primitive.NilObjectID.Hex() }, status: 401, handlerReached: true},
		{name: "platform still needs user id", mutate: func(c *utils.JWTClaims) { c.Role, c.UserID, c.TenantID = "superadmin", "", "" }, status: 401, handlerReached: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			auth := tt.authorization
			if tt.mutate != nil {
				claims := dashboardClaims("user")
				tt.mutate(&claims)
				auth = dashboardSignToken(t, claims, dashboardJWTSecret)
			}
			service := &dashboardServiceRecorder{}
			response, body := dashboardRequest(t, dashboardHandlerApp(service, nil), "/api/dashboard/summary", auth)
			if response.StatusCode != tt.status || service.calls != 0 {
				t.Fatalf("status = %d, service calls = %d, body = %s", response.StatusCode, service.calls, body)
			}
			if tt.handlerReached {
				dashboardAssertNoStore(t, response)
			}
		})
	}
	t.Run("wrong signing secret", func(t *testing.T) {
		service := &dashboardServiceRecorder{}
		response, body := dashboardRequest(t, dashboardHandlerApp(service, nil), "/api/dashboard/summary", dashboardSignToken(t, dashboardClaims("admin"), "wrong-secret"))
		if response.StatusCode != 401 || service.calls != 0 {
			t.Fatalf("status = %d, calls = %d, body = %s", response.StatusCode, service.calls, body)
		}
	})
}

func TestDashboardHandlerRejectsClientScopeAndFilters(t *testing.T) {
	for _, key := range []string{"tenant_id", "user_id", "role", "scope", "assigned_to", "organizer_id", "creator_id", "author_id", "filter", "filters[tenant_id]", "start", "end", "limit", "$where"} {
		t.Run(key, func(t *testing.T) {
			service := &dashboardServiceRecorder{}
			query := url.Values{"timezone": {"Asia/Dubai"}, key: {primitive.NewObjectID().Hex()}}
			response, body := dashboardRequest(t, dashboardHandlerApp(service, nil), "/api/dashboard/summary?"+query.Encode(), dashboardSignToken(t, dashboardClaims("admin"), dashboardJWTSecret))
			if response.StatusCode != 400 || service.calls != 0 {
				t.Fatalf("status = %d, calls = %d, body = %s", response.StatusCode, service.calls, body)
			}
			dashboardAssertNoStore(t, response)
		})
	}
}

func TestDashboardHandlerMapsErrorsWithoutLeakingDatabaseDetails(t *testing.T) {
	secretErr := errors.New("mongodb://db-admin:private-password@database.internal:27018/customer_production: collection receipts aggregate failed")
	for _, tt := range []struct {
		name    string
		err     error
		status  int
		message string
	}{
		{"invalid IANA zone", fmt.Errorf("zone: %w", ports.ErrDashboardZone), 400, "Invalid IANA time zone"},
		{"invalid service identity", fmt.Errorf("identity: %w", ports.ErrDashboardIdentity), 401, "Invalid account identity"},
		{"unsupported service role", fmt.Errorf("role: %w", ports.ErrDashboardRole), 403, "Dashboard is not available for this role"},
		{"database failure", secretErr, 500, "Dashboard metrics could not be loaded"},
		{"query timeout", context.DeadlineExceeded, 500, "Dashboard metrics could not be loaded"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &dashboardServiceRecorder{err: tt.err}
			response, body := dashboardRequest(t, dashboardHandlerApp(service, nil), "/api/dashboard/summary?timezone=Not%2FA_Zone", dashboardSignToken(t, dashboardClaims("user"), dashboardJWTSecret))
			if response.StatusCode != tt.status || service.calls != 1 || service.zone != "Not/A_Zone" {
				t.Fatalf("status = %d, calls = %d, timezone = %s, body = %s", response.StatusCode, service.calls, service.zone, body)
			}
			dashboardAssertNoStore(t, response)
			var envelope map[string]json.RawMessage
			if err := json.Unmarshal(body, &envelope); err != nil {
				t.Fatal(err)
			}
			var message string
			if err := json.Unmarshal(envelope["error"], &message); err != nil || message != tt.message || len(envelope) != 1 {
				t.Errorf("error envelope = %s, want only generic error %q", body, tt.message)
			}
			for _, secret := range []string{"mongodb://", "private-password", "database.internal", "customer_production", "aggregate failed"} {
				if strings.Contains(string(body), secret) {
					t.Errorf("response leaked database detail %q: %s", secret, body)
				}
			}
		})
	}
}

func TestDashboardHandlerRBACStopsDeniedAndFailedPermissionChecks(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
	}{
		{"denied", nil, 403}, {"permission database failure", errors.New("private RBAC database detail"), 500},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &dashboardServiceRecorder{}
			permission := &dashboardPermissionRecorder{err: tt.err}
			response, body := dashboardRequest(t, dashboardHandlerApp(service, permission), "/api/dashboard/summary", dashboardSignToken(t, dashboardClaims("user"), dashboardJWTSecret))
			if response.StatusCode != tt.status || service.calls != 0 || permission.calls != 1 {
				t.Fatalf("status = %d, service calls = %d, RBAC calls = %d, body = %s", response.StatusCode, service.calls, permission.calls, body)
			}
			if strings.Contains(string(body), "private RBAC") {
				t.Errorf("permission failure leaked details: %s", body)
			}
		})
	}
}

package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
)

// Integration tests are deliberately opt-in, never use application configuration,
// and mutate only a fresh database created by this test. Example isolated run:
// DASHBOARD_TEST_MONGO_URI=mongodb://127.0.0.1:27019 go test ./internal/adapters/storage -run TestMongoDashboard -v
func dashboardTestDatabase(t *testing.T) (*mongo.Database, context.Context) {
	t.Helper()
	uri := os.Getenv("DASHBOARD_TEST_MONGO_URI")
	if uri == "" {
		t.Skip("set DASHBOARD_TEST_MONGO_URI to an isolated MongoDB to run dashboard integration tests")
	}
	parsed, err := connstring.ParseAndValidate(uri)
	if err != nil {
		t.Fatal("invalid DASHBOARD_TEST_MONGO_URI") // Do not print a URI that may contain credentials.
	}
	for _, address := range parsed.Hosts {
		_, port, _ := net.SplitHostPort(address)
		if port == "27018" {
			t.Fatal("refusing to connect dashboard tests to production MongoDB port 27018")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	t.Cleanup(cancel)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetServerSelectionTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("connect isolated MongoDB: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if err := client.Disconnect(cleanupCtx); err != nil {
			t.Errorf("disconnect test MongoDB: %v", err)
		}
	})
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		t.Fatalf("isolated MongoDB unavailable: %v", err)
	}
	name := "dashboard_test_" + primitive.NewObjectID().Hex()
	matching, err := client.ListDatabaseNames(ctx, bson.M{"name": name})
	if err != nil {
		t.Fatalf("check throwaway database name: %v", err)
	}
	if len(matching) != 0 {
		t.Fatal("refusing to mutate an existing database")
	}
	db := client.Database(name) // Intentionally ignore any database path in the URI.
	if err := db.CreateCollection(ctx, "__dashboard_test_owner"); err != nil {
		t.Fatalf("create throwaway database: %v", err)
	}
	t.Cleanup(func() {
		// Only this exact newly-created database can ever be dropped.
		if db.Name() != name || !strings.HasPrefix(name, "dashboard_test_") {
			t.Error("refusing to drop a database not owned by this test")
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if err := db.Drop(cleanupCtx); err != nil {
			t.Errorf("drop throwaway database %s: %v", name, err)
		}
	})
	return db, ctx
}

func dashboardRepoTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func dashboardRepoWindow(t *testing.T) ports.DashboardWindow {
	t.Helper()
	location, err := time.LoadLocation("Asia/Dubai")
	if err != nil {
		t.Fatal(err)
	}
	return ports.DashboardWindow{
		Now: dashboardRepoTime(t, "2024-06-15T12:00:00Z"), End: dashboardRepoTime(t, "2024-06-15T12:00:00Z"),
		Start: dashboardRepoTime(t, "2024-05-31T20:00:00Z"), ComparisonStart: dashboardRepoTime(t, "2024-04-30T20:00:00Z"),
		ComparisonEnd: dashboardRepoTime(t, "2024-05-15T12:00:00Z"), TrendStart: dashboardRepoTime(t, "2023-12-31T20:00:00Z"),
		TodayStart: dashboardRepoTime(t, "2024-06-14T20:00:00Z"), TodayEnd: dashboardRepoTime(t, "2024-06-15T20:00:00Z"),
		UpcomingEnd: dashboardRepoTime(t, "2024-07-15T12:00:00Z"), Location: location,
	}
}

func dashboardInsert(t *testing.T, ctx context.Context, db *mongo.Database, collection string, docs ...bson.M) {
	t.Helper()
	values := make([]interface{}, len(docs))
	for i := range docs {
		values[i] = docs[i]
	}
	if _, err := db.Collection(collection).InsertMany(ctx, values); err != nil {
		t.Fatalf("insert %s fixtures: %v", collection, err)
	}
}

func dashboardGet(t *testing.T, ctx context.Context, db *mongo.Database, scope ports.DashboardScope, w ports.DashboardWindow) *ports.DashboardSummary {
	t.Helper()
	result, err := NewMongoDashboardRepository(db).GetSummary(ctx, scope, w)
	if err != nil {
		t.Fatalf("GetSummary(%s): %v", scope.Role, err)
	}
	if result == nil {
		t.Fatal("GetSummary returned nil without an error")
	}
	if result.Role != scope.Role || result.Scope != scope.Label() {
		t.Errorf("summary identity = %s/%s, want %s/%s", result.Role, result.Scope, scope.Role, scope.Label())
	}
	wantPeriod := ports.DashboardPeriod{
		Start: w.Start.Format(time.RFC3339Nano), EndExclusive: w.End.Format(time.RFC3339Nano),
		ComparisonStart: w.ComparisonStart.Format(time.RFC3339Nano), ComparisonEnd: w.ComparisonEnd.Format(time.RFC3339Nano), TimeZone: "Asia/Dubai",
	}
	if result.Period != wantPeriod || result.GeneratedAt != w.Now.Format(time.RFC3339Nano) {
		t.Errorf("report period = %+v, generated = %s; want %+v at %s", result.Period, result.GeneratedAt, wantPeriod, w.Now)
	}
	return result
}

func dashboardAssertMetric(t *testing.T, result *ports.DashboardSummary, key string, value float64, format, period string, previous ...float64) {
	t.Helper()
	metric, ok := result.Metrics[key]
	if !ok {
		t.Errorf("missing metric %q", key)
		return
	}
	if math.Abs(metric.Value-value) > 0.000001 || metric.Format != format || metric.Period != period {
		t.Errorf("%s = %+v, want value %v, format %s, period %s", key, metric, value, format, period)
	}
	if len(previous) == 0 {
		if metric.Previous != nil {
			t.Errorf("snapshot %s has misleading previous value %v", key, *metric.Previous)
		}
	} else if metric.Previous == nil || math.Abs(*metric.Previous-previous[0]) > 0.000001 {
		t.Errorf("%s previous = %v, want %v", key, metric.Previous, previous[0])
	}
}

func dashboardAssertNoFinance(t *testing.T, result *ports.DashboardSummary) {
	t.Helper()
	for _, key := range []string{"revenue", "invoiced", "outstanding", "open_invoices"} {
		if _, ok := result.Metrics[key]; ok {
			t.Errorf("%s scope leaked financial metric %s", result.Scope, key)
		}
	}
	if result.Currency != "" {
		t.Errorf("%s scope leaked currency %s", result.Scope, result.Currency)
	}
	for _, point := range result.Trend {
		if point.Revenue != nil {
			t.Errorf("%s scope exposed revenue in %s", result.Scope, point.Start)
		}
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), `"revenue"`) || strings.Contains(string(body), `"currency"`) {
		t.Errorf("non-admin JSON contains finance: %s", body)
	}
}

func dashboardAssertArrays(t *testing.T, result *ports.DashboardSummary, empty bool) {
	t.Helper()
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"trend", "lead_status", "upcoming", "recent_leads", "tenant_highlights"} {
		value := string(object[key])
		if !strings.HasPrefix(value, "[") || (empty && key != "trend" && value != "[]") {
			t.Errorf("%s = %s, want non-null JSON array (empty = %t)", key, value, empty)
		}
	}
}

func TestMongoDashboardTenantAndPersonalIsolation(t *testing.T) {
	db, ctx := dashboardTestDatabase(t)
	w := dashboardRepoWindow(t)
	tenantA, tenantB, countryA, countryB := primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()
	owner, coworker, otherUser, platformUser := primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()
	dashboardInsert(t, ctx, db, "countries", bson.M{"_id": countryA, "currency": " aed "}, bson.M{"_id": countryB, "currency": "usd"})
	dashboardInsert(t, ctx, db, "tenants",
		bson.M{"_id": tenantA, "name": "Tenant A", "country_id": countryA, "created_at": w.ComparisonStart},
		bson.M{"_id": tenantB, "name": "Tenant B", "country_id": countryB, "created_at": w.Start},
		bson.M{"_id": primitive.NewObjectID(), "name": "Future tenant", "created_at": w.Now})
	dashboardInsert(t, ctx, db, "users",
		bson.M{"_id": owner, "tenant_id": tenantA, "created_at": w.ComparisonStart},
		bson.M{"_id": coworker, "tenant_id": tenantA, "created_at": w.Start},
		bson.M{"_id": otherUser, "tenant_id": tenantB, "created_at": w.Start.Add(time.Hour)},
		bson.M{"_id": platformUser, "role": "superadmin", "created_at": w.TrendStart.Add(-time.Millisecond)},
		bson.M{"_id": primitive.NewObjectID(), "tenant_id": tenantA, "created_at": w.Now})

	leadIDs := map[string]string{}
	lead := func(tag string, tenant, assigned primitive.ObjectID, at time.Time, status string) bson.M {
		id := primitive.NewObjectID()
		leadIDs[tag] = id.Hex()
		return bson.M{"_id": id, "tenant_id": tenant, "assigned_to": assigned, "first_name": "Lead", "last_name": tag, "status": status, "created_at": at}
	}
	dashboardInsert(t, ctx, db, "leads",
		lead("last-prior-month", tenantA, owner, w.Start.Add(-time.Millisecond), "active"),
		lead("month-start", tenantA, owner, w.Start, "lead"),
		lead("latest", tenantA, owner, w.Now.Add(-time.Hour), "active"),
		lead("prior-start", tenantA, owner, w.ComparisonStart, "inactive"),
		lead("prior-inside", tenantA, owner, w.ComparisonEnd.Add(-time.Millisecond), "active"),
		lead("prior-end", tenantA, owner, w.ComparisonEnd, "lead"),
		lead("old", tenantA, owner, w.TrendStart.Add(-time.Millisecond), "inactive"),
		lead("now-excluded", tenantA, owner, w.Now, "active"),
		lead("future-excluded", tenantA, owner, w.Now.Add(time.Millisecond), "active"),
		lead("coworker-current", tenantA, coworker, w.Start.Add(2*time.Hour), "active"),
		lead("coworker-prior", tenantA, coworker, w.ComparisonStart.Add(time.Hour), "lead"),
		lead("unassigned", tenantA, primitive.NilObjectID, w.Start.Add(3*time.Hour), "inactive"),
		// The same owner id in B catches an accidental owner-only (instead of tenant AND owner) filter.
		lead("foreign-same-owner", tenantB, owner, w.Start.Add(4*time.Hour), "active"),
		lead("foreign-current", tenantB, otherUser, w.Now.Add(-2*time.Hour), "lead"),
		lead("foreign-prior", tenantB, otherUser, w.ComparisonStart.Add(2*time.Hour), "inactive"))

	taskIDs := map[string]string{}
	task := func(tag, ownerField string, tenant, user primitive.ObjectID, start, end time.Time, status string) bson.M {
		id := primitive.NewObjectID()
		taskIDs[tag] = id.Hex()
		leadID, _ := primitive.ObjectIDFromHex(leadIDs["latest"])
		return bson.M{"_id": id, "tenant_id": tenant, ownerField: user, "lead_id": leadID, "title": tag, "start_time": start, "end_time": end, "status": status}
	}
	dashboardInsert(t, ctx, db, "lead_appointments",
		task("day-start-ongoing", "organizer_id", tenantA, owner, w.TodayStart, w.Now.Add(time.Hour), "scheduled"),
		task("rescheduled", "organizer_id", tenantA, owner, w.Now.Add(-2*time.Hour), w.Now.Add(30*time.Minute), "rescheduled"),
		task("completed", "organizer_id", tenantA, owner, w.Now.Add(-3*time.Hour), w.Now.Add(time.Hour), "completed"),
		task("cancelled", "organizer_id", tenantA, owner, w.Now.Add(-4*time.Hour), w.Now.Add(time.Hour), "cancelled"),
		task("day-end", "organizer_id", tenantA, owner, w.TodayEnd, w.TodayEnd.Add(time.Hour), "scheduled"),
		task("before-day", "organizer_id", tenantA, owner, w.TodayStart.Add(-time.Millisecond), w.TodayStart, "scheduled"),
		task("horizon", "organizer_id", tenantA, owner, w.UpcomingEnd, w.UpcomingEnd.Add(time.Hour), "scheduled"),
		task("next-hour", "organizer_id", tenantA, owner, w.Now.Add(time.Hour), w.Now.Add(2*time.Hour), "scheduled"),
		task("tomorrow", "organizer_id", tenantA, owner, w.Now.Add(24*time.Hour), w.Now.Add(25*time.Hour), "scheduled"),
		task("expired", "organizer_id", tenantA, owner, w.Now.Add(-6*time.Hour), w.Now.Add(-5*time.Hour), "scheduled"),
		task("coworker-appointment", "organizer_id", tenantA, coworker, w.Now.Add(-time.Hour), w.Now.Add(time.Hour), "scheduled"),
		task("foreign-appointment", "organizer_id", tenantB, owner, w.Now.Add(-time.Hour), w.Now.Add(time.Hour), "scheduled"))
	dashboardInsert(t, ctx, db, "lead_follow_ups",
		task("overdue", "creator_id", tenantA, owner, w.Now.Add(-48*time.Hour), w.Now.Add(-24*time.Hour), "active"),
		task("due-now", "creator_id", tenantA, owner, w.Now.Add(-24*time.Hour), w.Now, "active"),
		task("next-follow-up", "creator_id", tenantA, owner, w.Now.Add(30*time.Minute), w.Now.Add(2*time.Hour), "active"),
		task("closed", "creator_id", tenantA, owner, w.Now.Add(-72*time.Hour), w.Now.Add(-2*time.Hour), "closed"),
		task("follow-up-horizon", "creator_id", tenantA, owner, w.UpcomingEnd, w.UpcomingEnd.Add(time.Hour), "active"),
		task("coworker-overdue", "creator_id", tenantA, coworker, w.Now.Add(-60*time.Hour), w.Now.Add(-50*time.Hour), "active"),
		task("foreign-follow-up", "creator_id", tenantB, owner, w.Now.Add(-60*time.Hour), w.Now.Add(-50*time.Hour), "active"))
	comment := func(tenant, author primitive.ObjectID, at time.Time) bson.M {
		return bson.M{"_id": primitive.NewObjectID(), "tenant_id": tenant, "author_id": author, "created_at": at}
	}
	dashboardInsert(t, ctx, db, "lead_comments",
		comment(tenantA, owner, w.ComparisonStart), comment(tenantA, owner, w.ComparisonEnd.Add(-time.Millisecond)),
		comment(tenantA, owner, w.ComparisonEnd), comment(tenantA, owner, w.Start.Add(-time.Millisecond)),
		comment(tenantA, owner, w.Start), comment(tenantA, owner, w.Now.Add(-time.Millisecond)),
		comment(tenantA, owner, w.Now), comment(tenantA, owner, w.Now.Add(time.Millisecond)),
		comment(tenantA, coworker, w.Start.Add(time.Hour)), comment(tenantB, owner, w.Start.Add(time.Hour)))

	receipt := func(tenant primitive.ObjectID, payment time.Time, amount float64) bson.M {
		return bson.M{"_id": primitive.NewObjectID(), "tenant_id": tenant, "payment_date": payment, "amount_paid": amount, "total_paid": amount * 1.05, "tax_amount": amount * 0.05, "created_at": w.TrendStart.Add(-time.Hour)}
	}
	dashboardInsert(t, ctx, db, "receipts",
		receipt(tenantA, w.ComparisonStart, 25.20), receipt(tenantA, w.ComparisonEnd.Add(-time.Millisecond), 14.80),
		receipt(tenantA, w.ComparisonEnd, 500), receipt(tenantA, w.Start.Add(-time.Millisecond), 600),
		receipt(tenantA, w.Start, 100.10), receipt(tenantA, w.Now.Add(-time.Millisecond), 50.25),
		receipt(tenantA, w.Now, 10000), receipt(tenantA, w.Now.Add(time.Millisecond), 10000), receipt(tenantB, w.Start, 9000))
	invoice := func(tenant primitive.ObjectID, created time.Time, total, paid, grossPaid float64, status string) bson.M {
		return bson.M{"_id": primitive.NewObjectID(), "tenant_id": tenant, "created_at": created, "total_amount": total, "subtotal": total / 1.05, "paid_amount": paid, "paid_amount_vat": grossPaid, "status": status}
	}
	missingPaid := invoice(tenantA, w.Start.Add(-time.Millisecond), 20, 0, 0, "pending")
	delete(missingPaid, "paid_amount_vat")
	dashboardInsert(t, ctx, db, "invoices",
		invoice(tenantA, w.ComparisonStart, 210, 100, 105, "partial"),
		invoice(tenantA, w.ComparisonEnd.Add(-time.Millisecond), 50, 55, 60, "paid"), // Overpayment must not reduce other balances.
		invoice(tenantA, w.ComparisonEnd, 100, 0, 0, "pending"), missingPaid,
		invoice(tenantA, w.Start, 315, 100, 105, "partial"), invoice(tenantA, w.Now.Add(-time.Millisecond), 105, 100, 105, "paid"),
		invoice(tenantA, w.TrendStart.Add(-time.Millisecond), 60, 0, 0, "pending"),
		invoice(tenantA, w.Now, 10000, 0, 0, "pending"), invoice(tenantB, w.Start, 9000, 0, 0, "pending"))

	personal := dashboardGet(t, ctx, db, ports.DashboardScope{Role: "user", TenantID: tenantA, UserID: owner}, w)
	admin := dashboardGet(t, ctx, db, ports.DashboardScope{Role: "admin", TenantID: tenantA, UserID: owner}, w)
	platform := dashboardGet(t, ctx, db, ports.DashboardScope{Role: "superadmin", UserID: platformUser}, w)

	t.Run("personal uses each collection ownership and no financial or people metrics", func(t *testing.T) {
		dashboardAssertMetric(t, personal, "leads", 7, "number", "all_time")
		dashboardAssertMetric(t, personal, "new_leads", 2, "number", "month", 2)
		dashboardAssertMetric(t, personal, "active_leads", 3, "number", "current")
		dashboardAssertMetric(t, personal, "appointments_today", 5, "number", "today")
		dashboardAssertMetric(t, personal, "appointments_completed_today", 1, "number", "today")
		dashboardAssertMetric(t, personal, "open_follow_ups", 4, "number", "current")
		dashboardAssertMetric(t, personal, "overdue_follow_ups", 1, "number", "current")
		dashboardAssertMetric(t, personal, "comments_month", 2, "number", "month", 2)
		if len(personal.Metrics) != 8 {
			t.Errorf("personal has unexpected metrics: %+v", personal.Metrics)
		}
		for _, key := range []string{"users", "new_users", "tenants", "new_tenants"} {
			if _, exists := personal.Metrics[key]; exists {
				t.Errorf("personal leaked metric %s", key)
			}
		}
		dashboardAssertNoFinance(t, personal)
		if personal.TenantName != "Tenant A" || len(personal.TenantHighlights) != 0 {
			t.Errorf("personal tenant information = %s / %+v", personal.TenantName, personal.TenantHighlights)
		}
		wantStatus := []ports.DashboardBreakdown{{Label: "active", Value: 3}, {Label: "inactive", Value: 2}, {Label: "lead", Value: 2}}
		if !reflect.DeepEqual(personal.LeadStatus, wantStatus) {
			t.Errorf("personal status = %+v, want %+v", personal.LeadStatus, wantStatus)
		}
		wantRecent := []string{"latest", "month-start", "last-prior-month", "prior-end", "prior-inside", "prior-start"}
		if len(personal.RecentLeads) != len(wantRecent) {
			t.Fatalf("personal recent leads = %+v", personal.RecentLeads)
		}
		for i, tag := range wantRecent {
			if personal.RecentLeads[i].ID != leadIDs[tag] || personal.RecentLeads[i].Name != "Lead "+tag {
				t.Errorf("recent[%d] = %+v, want lead %s", i, personal.RecentLeads[i], tag)
			}
		}
	})

	t.Run("admin tenant-only finance uses paid principal and gross invoice balances", func(t *testing.T) {
		dashboardAssertMetric(t, admin, "leads", 10, "number", "all_time")
		dashboardAssertMetric(t, admin, "new_leads", 4, "number", "month", 3)
		dashboardAssertMetric(t, admin, "active_leads", 4, "number", "current")
		dashboardAssertMetric(t, admin, "users", 2, "number", "all_time")
		dashboardAssertMetric(t, admin, "new_users", 1, "number", "month", 1)
		dashboardAssertMetric(t, admin, "appointments_today", 6, "number", "today")
		dashboardAssertMetric(t, admin, "appointments_completed_today", 1, "number", "today")
		dashboardAssertMetric(t, admin, "open_follow_ups", 5, "number", "current")
		dashboardAssertMetric(t, admin, "overdue_follow_ups", 2, "number", "current")
		dashboardAssertMetric(t, admin, "comments_month", 3, "number", "month", 2)
		dashboardAssertMetric(t, admin, "revenue", 150.35, "currency", "month", 40)
		dashboardAssertMetric(t, admin, "invoiced", 420, "currency", "month", 260)
		dashboardAssertMetric(t, admin, "outstanding", 495, "currency", "current")
		dashboardAssertMetric(t, admin, "open_invoices", 5, "number", "current")
		if admin.TenantName != "Tenant A" || admin.Currency != "AED" || len(admin.TenantHighlights) != 0 {
			t.Errorf("admin tenant information = %s/%s, highlights %+v", admin.TenantName, admin.Currency, admin.TenantHighlights)
		}
		wantStatus := []ports.DashboardBreakdown{{Label: "active", Value: 4}, {Label: "inactive", Value: 3}, {Label: "lead", Value: 3}}
		if !reflect.DeepEqual(admin.LeadStatus, wantStatus) {
			t.Errorf("admin status = %+v, want %+v", admin.LeadStatus, wantStatus)
		}
		for _, lead := range admin.RecentLeads {
			if lead.ID == leadIDs["foreign-current"] || lead.ID == leadIDs["foreign-same-owner"] {
				t.Errorf("admin recent leads leaked B: %+v", lead)
			}
		}
	})

	t.Run("platform cross-tenant operational totals never sum currencies", func(t *testing.T) {
		dashboardAssertMetric(t, platform, "leads", 13, "number", "all_time")
		dashboardAssertMetric(t, platform, "new_leads", 6, "number", "month", 4)
		dashboardAssertMetric(t, platform, "users", 4, "number", "all_time")
		dashboardAssertMetric(t, platform, "new_users", 2, "number", "month", 1)
		dashboardAssertMetric(t, platform, "tenants", 2, "number", "all_time")
		dashboardAssertMetric(t, platform, "new_tenants", 1, "number", "month", 1)
		dashboardAssertNoFinance(t, platform)
		if len(platform.Metrics) != 6 || len(platform.Upcoming) != 0 || len(platform.RecentLeads) != 0 {
			t.Errorf("unexpected platform workload: %+v", platform)
		}
		want := []ports.DashboardTenantHighlight{
			{ID: tenantB.Hex(), Name: "Tenant B", CreatedAt: w.Start.Format(time.RFC3339Nano), Leads: 3, Users: 1},
			{ID: tenantA.Hex(), Name: "Tenant A", CreatedAt: w.ComparisonStart.Format(time.RFC3339Nano), Leads: 10, Users: 2},
		}
		if !reflect.DeepEqual(platform.TenantHighlights, want) {
			t.Errorf("tenant highlights = %+v, want %+v", platform.TenantHighlights, want)
		}
	})

	t.Run("upcoming merges sorts limits and includes active overdue follow-ups", func(t *testing.T) {
		for _, test := range []struct {
			summary *ports.DashboardSummary
			tags    []string
		}{
			{personal, []string{"overdue", "due-now", "day-start-ongoing", "rescheduled", "next-follow-up", "next-hour"}},
			{admin, []string{"coworker-overdue", "overdue", "due-now", "day-start-ongoing", "rescheduled", "coworker-appointment"}},
		} {
			if len(test.summary.Upcoming) != 6 {
				t.Fatalf("%s upcoming = %+v, want six tasks", test.summary.Scope, test.summary.Upcoming)
			}
			for i, tag := range test.tags {
				task := test.summary.Upcoming[i]
				if task.ID != taskIDs[tag] || task.Title != tag || task.LeadID != leadIDs["latest"] {
					t.Errorf("%s upcoming[%d] = %+v, want %s", test.summary.Scope, i, task, tag)
				}
				wantOverdue := tag == "overdue" || tag == "coworker-overdue"
				if task.Overdue != wantOverdue {
					t.Errorf("task %s overdue = %t, want %t", tag, task.Overdue, wantOverdue)
				}
			}
		}
	})

	t.Run("IANA trends zero-fill six calendar months and include only elapsed current month", func(t *testing.T) {
		for _, summary := range []*ports.DashboardSummary{personal, admin, platform} {
			if len(summary.Trend) != 6 {
				t.Fatalf("%s trend length = %d", summary.Scope, len(summary.Trend))
			}
			for i, point := range summary.Trend {
				want := ports.DashboardTrendPoint{Label: []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"}[i], Start: fmt.Sprintf("2024-%02d-01", i+1)}
				if i == 4 {
					switch summary.Scope {
					case "personal":
						want.Leads, want.Comments = 4, 4
					case "tenant":
						want.Leads, want.Users, want.Comments = 5, 1, 4
					case "platform":
						want.Leads, want.Users, want.Tenants = 6, 1, 1
					}
				}
				if i == 5 {
					switch summary.Scope {
					case "personal":
						want.Leads, want.Appointments, want.FollowUps, want.Comments = 2, 5, 3, 2
					case "tenant":
						want.Leads, want.Users, want.Appointments, want.FollowUps, want.Comments = 4, 1, 6, 4, 3
					case "platform":
						want.Leads, want.Users, want.Tenants = 6, 2, 1
					}
				}
				if summary.Scope == "tenant" {
					revenue := 0.0
					if i == 4 {
						revenue = 1140
					}
					if i == 5 {
						revenue = 150.35
					}
					want.Revenue = &revenue
				}
				if !reflect.DeepEqual(point, want) {
					t.Errorf("%s trend[%d] = %+v, want %+v", summary.Scope, i, point, want)
				}
			}
			dashboardAssertArrays(t, summary, false)
		}
	})
}

func TestMongoDashboardEmptyResultsAreZeroFilledArrays(t *testing.T) {
	db, ctx := dashboardTestDatabase(t)
	w := dashboardRepoWindow(t)
	user, tenant := primitive.NewObjectID(), primitive.NewObjectID()
	platform := dashboardGet(t, ctx, db, ports.DashboardScope{Role: "superadmin", UserID: user}, w)
	dashboardInsert(t, ctx, db, "tenants", bson.M{"_id": tenant, "name": "Empty tenant", "created_at": w.Start})
	admin := dashboardGet(t, ctx, db, ports.DashboardScope{Role: "admin", TenantID: tenant, UserID: user}, w)
	personal := dashboardGet(t, ctx, db, ports.DashboardScope{Role: "user", TenantID: tenant, UserID: user}, w)
	for _, result := range []*ports.DashboardSummary{platform, admin, personal} {
		t.Run(result.Scope, func(t *testing.T) {
			dashboardAssertArrays(t, result, true)
			wantCount := map[string]int{"platform": 6, "tenant": 14, "personal": 8}[result.Scope]
			if len(result.Metrics) != wantCount {
				t.Errorf("metrics = %+v, want %d zero metrics", result.Metrics, wantCount)
			}
			for key, metric := range result.Metrics {
				if metric.Value != 0 || (metric.Previous != nil && *metric.Previous != 0) {
					t.Errorf("%s not zero-filled: %+v", key, metric)
				}
				if (metric.Period == "month") != (metric.Previous != nil) {
					t.Errorf("%s has inconsistent comparison metadata: %+v", key, metric)
				}
			}
			if result.Currency != "" {
				t.Errorf("missing country invented a currency: %s", result.Currency)
			}
			if len(result.Trend) != 6 {
				t.Fatalf("trend length = %d, want six zero months", len(result.Trend))
			}
			for i, point := range result.Trend {
				if point.Start != fmt.Sprintf("2024-%02d-01", i+1) || point.Leads != 0 || point.Users != 0 || point.Tenants != 0 || point.Appointments != 0 || point.FollowUps != 0 || point.Comments != 0 {
					t.Errorf("trend[%d] = %+v, want zero-filled chronological month", i, point)
				}
				if result.Role == "admin" && (point.Revenue == nil || *point.Revenue != 0) {
					t.Errorf("admin trend[%d] revenue is not explicit zero: %v", i, point.Revenue)
				}
			}
			if result.Role != "admin" {
				dashboardAssertNoFinance(t, result)
			}
		})
	}
}

func TestMongoDashboardUpcomingReturnsSixOfOneKindDeterministically(t *testing.T) {
	for _, kind := range []struct{ collection, ownerField, kind, status string }{
		{"lead_appointments", "organizer_id", "appointment", "scheduled"},
		{"lead_follow_ups", "creator_id", "follow_up", "active"},
	} {
		t.Run(kind.kind, func(t *testing.T) {
			db, ctx := dashboardTestDatabase(t)
			w := dashboardRepoWindow(t)
			tenant, user := primitive.NewObjectID(), primitive.NewObjectID()
			dashboardInsert(t, ctx, db, "tenants", bson.M{"_id": tenant, "name": "Tasks only", "created_at": w.Start})
			// Equal start times and reverse insertion order exercise the stable _id tie-breaker.
			ids := make([]primitive.ObjectID, 8)
			for i := range ids {
				ids[i] = primitive.NewObjectID()
			}
			docs := make([]bson.M, 0, len(ids)+4)
			for i := len(ids) - 1; i >= 0; i-- {
				docs = append(docs, bson.M{"_id": ids[i], "tenant_id": tenant, kind.ownerField: user, "title": fmt.Sprintf("Task %d", i), "start_time": w.Now.Add(time.Hour), "end_time": w.Now.Add(2 * time.Hour), "status": kind.status})
			}
			for _, status := range []string{"completed", "cancelled", "closed"} {
				docs = append(docs, bson.M{"_id": primitive.NewObjectID(), "tenant_id": tenant, kind.ownerField: user, "title": "Must not appear", "start_time": w.Now, "end_time": w.Now.Add(time.Hour), "status": status})
			}
			// At the exclusive horizon is ineligible even when it would otherwise match.
			docs = append(docs, bson.M{"_id": primitive.NewObjectID(), "tenant_id": tenant, kind.ownerField: user, "start_time": w.UpcomingEnd, "end_time": w.UpcomingEnd.Add(time.Hour), "status": kind.status})
			dashboardInsert(t, ctx, db, kind.collection, docs...)
			scope := ports.DashboardScope{Role: "user", TenantID: tenant, UserID: user}
			result := dashboardGet(t, ctx, db, scope, w)
			if len(result.Upcoming) != 6 {
				t.Fatalf("only one kind: got %d upcoming tasks, want six", len(result.Upcoming))
			}
			for i, task := range result.Upcoming {
				if task.ID != ids[i].Hex() || task.Kind != kind.kind || task.Status != kind.status || task.Overdue || task.LeadID != "" {
					t.Errorf("upcoming[%d] = %+v, want %s %s", i, task, kind.kind, ids[i].Hex())
				}
			}
			again := dashboardGet(t, ctx, db, scope, w)
			if !reflect.DeepEqual(result, again) {
				t.Error("identical fixture and injected window produced nondeterministic summaries")
			}
		})
	}
}

func TestMongoDashboardUpcomingHalfOpenBoundariesAndMixedTies(t *testing.T) {
	db, ctx := dashboardTestDatabase(t)
	w := dashboardRepoWindow(t)
	tenant, user := primitive.NewObjectID(), primitive.NewObjectID()
	dashboardInsert(t, ctx, db, "tenants", bson.M{"_id": tenant, "name": "Boundary tasks", "created_at": w.Start})
	followID, appointmentID := primitive.NewObjectID(), primitive.NewObjectID()
	dashboardInsert(t, ctx, db, "lead_appointments",
		bson.M{"_id": appointmentID, "tenant_id": tenant, "organizer_id": user, "start_time": w.Now.Add(-time.Hour), "end_time": w.Now, "status": "rescheduled"},
		bson.M{"_id": primitive.NewObjectID(), "tenant_id": tenant, "organizer_id": user, "start_time": w.UpcomingEnd, "end_time": w.UpcomingEnd.Add(time.Hour), "status": "scheduled"},
		bson.M{"_id": primitive.NewObjectID(), "tenant_id": tenant, "organizer_id": user, "start_time": w.Now.Add(-2 * time.Hour), "end_time": w.Now.Add(-time.Millisecond), "status": "scheduled"})
	dashboardInsert(t, ctx, db, "lead_follow_ups",
		bson.M{"_id": followID, "tenant_id": tenant, "creator_id": user, "start_time": w.Now.Add(-time.Hour), "end_time": w.Now.Add(-time.Millisecond), "status": "active"},
		bson.M{"_id": primitive.NewObjectID(), "tenant_id": tenant, "creator_id": user, "start_time": w.UpcomingEnd, "end_time": w.UpcomingEnd.Add(time.Hour), "status": "active"})
	result := dashboardGet(t, ctx, db, ports.DashboardScope{Role: "user", TenantID: tenant, UserID: user}, w)
	if len(result.Upcoming) != 2 {
		t.Fatalf("upcoming boundaries: %+v, want only two eligible tasks", result.Upcoming)
	}
	if result.Upcoming[0].ID != appointmentID.Hex() || result.Upcoming[0].Kind != "appointment" || result.Upcoming[0].Overdue {
		t.Errorf("first mixed tie = %+v, want appointment ending now", result.Upcoming[0])
	}
	if result.Upcoming[1].ID != followID.Hex() || result.Upcoming[1].Kind != "follow_up" || !result.Upcoming[1].Overdue {
		t.Errorf("second mixed tie = %+v, want active overdue follow-up", result.Upcoming[1])
	}
}

func TestDashboardRepositoryRejectsInvalidScopeAndWindowWithoutMongo(t *testing.T) {
	repo := NewMongoDashboardRepository(nil)
	user, tenant := primitive.NewObjectID(), primitive.NewObjectID()
	for _, tt := range []struct {
		scope ports.DashboardScope
		want  error
	}{
		{ports.DashboardScope{Role: "manager", TenantID: tenant, UserID: user}, ports.ErrDashboardRole},
		{ports.DashboardScope{Role: "user", UserID: user}, ports.ErrDashboardIdentity},
		{ports.DashboardScope{Role: "admin", TenantID: tenant}, ports.ErrDashboardIdentity},
		{ports.DashboardScope{Role: "superadmin"}, ports.ErrDashboardIdentity},
	} {
		result, err := repo.GetSummary(context.Background(), tt.scope, ports.DashboardWindow{})
		if result != nil || !errors.Is(err, tt.want) {
			t.Errorf("invalid scope %+v = (%v, %v), want %v", tt.scope, result, err, tt.want)
		}
	}
	for _, window := range []ports.DashboardWindow{{Location: time.UTC}, {Now: time.Now()}} {
		result, err := repo.GetSummary(context.Background(), ports.DashboardScope{Role: "superadmin", UserID: user}, window)
		if result != nil || err == nil {
			t.Errorf("invalid window %+v = (%v, %v), want error before database access", window, result, err)
		}
	}
}

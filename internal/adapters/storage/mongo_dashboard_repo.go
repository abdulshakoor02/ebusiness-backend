package storage

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDashboardRepository never accepts client-supplied Mongo filters. Personal
// reads use collection-specific ownership; finance reads run only for admins.
type MongoDashboardRepository struct{ db *mongo.Database }

func NewMongoDashboardRepository(db *mongo.Database) *MongoDashboardRepository {
	return &MongoDashboardRepository{db: db}
}

func (r *MongoDashboardRepository) GetSummary(ctx context.Context, scope ports.DashboardScope, window ports.DashboardWindow) (*ports.DashboardSummary, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if window.Location == nil || window.Now.IsZero() {
		return nil, errors.New("dashboard reporting window required")
	}
	result := &ports.DashboardSummary{
		GeneratedAt: dashboardTime(window.Now), Role: scope.Role, Scope: scope.Label(),
		Period: ports.DashboardPeriod{Start: dashboardTime(window.Start), EndExclusive: dashboardTime(window.End),
			ComparisonStart: dashboardTime(window.ComparisonStart), ComparisonEnd: dashboardTime(window.ComparisonEnd), TimeZone: window.Location.String()},
		Metrics: map[string]ports.DashboardMetric{}, Trend: make([]ports.DashboardTrendPoint, 6),
		LeadStatus: []ports.DashboardBreakdown{}, Upcoming: []ports.DashboardTask{},
		RecentLeads: []ports.DashboardLead{}, TenantHighlights: []ports.DashboardTenantHighlight{},
	}
	for index := range result.Trend {
		month := window.TrendStart.In(window.Location).AddDate(0, index, 0)
		result.Trend[index] = ports.DashboardTrendPoint{Label: month.Format("Jan"), Start: month.Format("2006-01-02")}
	}
	if scope.Role != "superadmin" {
		var tenant domain.Tenant
		if err := r.db.Collection("tenants").FindOne(ctx, bson.M{"_id": scope.TenantID}, options.FindOne().SetProjection(bson.M{"name": 1, "country_id": 1})).Decode(&tenant); err != nil {
			return nil, err
		}
		result.TenantName = tenant.Name
		if scope.Role == "admin" && !tenant.CountryID.IsZero() {
			var country struct {
				Currency string `bson:"currency"`
			}
			err := r.db.Collection("countries").FindOne(ctx, bson.M{"_id": tenant.CountryID}, options.FindOne().SetProjection(bson.M{"currency": 1})).Decode(&country)
			if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
				return nil, err
			}
			// Missing currency is unknown, not an invented USD value.
			result.Currency = strings.ToUpper(strings.TrimSpace(country.Currency))
		}
	}
	if err := r.loadDashboardLeads(ctx, scope, window, result); err != nil {
		return nil, err
	}
	if scope.Role == "superadmin" || scope.Role == "admin" {
		if err := r.loadDashboardPeople(ctx, scope, window, result); err != nil {
			return nil, err
		}
	}
	if scope.Role == "superadmin" {
		highlights, err := r.dashboardTenants(ctx, window.Now)
		if err != nil {
			return nil, err
		}
		result.TenantHighlights = highlights
	} else {
		if err := r.loadDashboardWork(ctx, scope, window, result); err != nil {
			return nil, err
		}
		if scope.Role == "admin" {
			if err := r.loadDashboardFinance(ctx, scope, window, result); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

// dashboardFilter is fail-closed for collections without personal ownership.
func dashboardFilter(scope ports.DashboardScope, collection string) (bson.M, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	filter := bson.M{}
	if scope.Role == "superadmin" {
		return filter, nil
	}
	filter["tenant_id"] = scope.TenantID
	if scope.Role == "user" {
		owners := map[string]string{"leads": "assigned_to", "lead_appointments": "organizer_id", "lead_follow_ups": "creator_id", "lead_comments": "author_id"}
		field, ok := owners[collection]
		if !ok {
			return nil, ports.ErrDashboardRole
		}
		filter[field] = scope.UserID
	}
	return filter, nil
}

func (r *MongoDashboardRepository) loadDashboardLeads(ctx context.Context, scope ports.DashboardScope, w ports.DashboardWindow, result *ports.DashboardSummary) error {
	filter, err := dashboardFilter(scope, "leads")
	if err != nil {
		return err
	}
	filter["created_at"] = bson.M{"$lt": w.Now}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{"$ifNull": bson.A{"$status", "unknown"}}, "total": bson.M{"$sum": 1},
			"current":  dashboardConditionalSum(dashboardInRange("created_at", w.Start, w.End), 1),
			"previous": dashboardConditionalSum(dashboardInRange("created_at", w.ComparisonStart, w.ComparisonEnd), 1),
		}}},
	}
	rows, err := dashboardAggregate[dashboardStatusRow](ctx, r.db.Collection("leads"), pipeline)
	if err != nil {
		return err
	}
	var total, current, previous, active int64
	for _, row := range rows {
		total += row.Total
		current += row.Current
		previous += row.Previous
		if row.Status == domain.LeadStatusActive {
			active += row.Total
		}
		result.LeadStatus = append(result.LeadStatus, ports.DashboardBreakdown{Label: row.Status, Value: row.Total})
	}
	sort.Slice(result.LeadStatus, func(i, j int) bool {
		if result.LeadStatus[i].Value == result.LeadStatus[j].Value {
			return result.LeadStatus[i].Label < result.LeadStatus[j].Label
		}
		return result.LeadStatus[i].Value > result.LeadStatus[j].Value
	})
	result.Metrics["leads"] = dashboardMetric(float64(total), "number", "all_time")
	result.Metrics["new_leads"] = dashboardComparison(float64(current), float64(previous), "number")
	if scope.Role != "superadmin" {
		result.Metrics["active_leads"] = dashboardMetric(float64(active), "number", "current")
	}
	series, err := r.dashboardSeries(ctx, "leads", filter, "created_at", "", w)
	if err != nil {
		return err
	}
	for index := range result.Trend {
		result.Trend[index].Leads = int64(series[result.Trend[index].Start])
	}
	if scope.Role == "superadmin" {
		return nil
	}
	cursor, err := r.db.Collection("leads").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).SetLimit(6).SetProjection(bson.M{"first_name": 1, "last_name": 1, "status": 1, "created_at": 1}))
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	var leads []domain.Lead
	if err := cursor.All(ctx, &leads); err != nil {
		return err
	}
	for _, lead := range leads {
		result.RecentLeads = append(result.RecentLeads, ports.DashboardLead{ID: lead.ID.Hex(), Name: strings.TrimSpace(lead.FirstName + " " + lead.LastName), Status: lead.Status, CreatedAt: dashboardTime(lead.CreatedAt)})
	}
	return nil
}

func (r *MongoDashboardRepository) loadDashboardPeople(ctx context.Context, scope ports.DashboardScope, w ports.DashboardWindow, result *ports.DashboardSummary) error {
	collections := []string{"users"}
	if scope.Role == "superadmin" {
		collections = append(collections, "tenants")
	}
	for _, collection := range collections {
		filter, err := dashboardFilter(scope, collection)
		if err != nil {
			return err
		}
		filter["created_at"] = bson.M{"$lt": w.Now}
		rows, err := dashboardAggregate[dashboardStatusRow](ctx, r.db.Collection(collection), mongo.Pipeline{
			{{Key: "$match", Value: filter}},
			{{Key: "$group", Value: bson.M{"_id": nil, "total": bson.M{"$sum": 1},
				"current":  dashboardConditionalSum(dashboardInRange("created_at", w.Start, w.End), 1),
				"previous": dashboardConditionalSum(dashboardInRange("created_at", w.ComparisonStart, w.ComparisonEnd), 1)}}},
		})
		if err != nil {
			return err
		}
		var stats dashboardStatusRow
		if len(rows) > 0 {
			stats = rows[0]
		}
		result.Metrics[collection] = dashboardMetric(float64(stats.Total), "number", "all_time")
		result.Metrics["new_"+collection] = dashboardComparison(float64(stats.Current), float64(stats.Previous), "number")
		series, err := r.dashboardSeries(ctx, collection, filter, "created_at", "", w)
		if err != nil {
			return err
		}
		for index := range result.Trend {
			value := int64(series[result.Trend[index].Start])
			if collection == "users" {
				result.Trend[index].Users = value
			} else {
				result.Trend[index].Tenants = value
			}
		}
	}
	return nil
}

func (r *MongoDashboardRepository) loadDashboardWork(ctx context.Context, scope ports.DashboardScope, w ports.DashboardWindow, result *ports.DashboardSummary) error {
	appointments, err := dashboardFilter(scope, "lead_appointments")
	if err != nil {
		return err
	}
	appointments["status"] = bson.M{"$in": bson.A{domain.StatusScheduled, domain.StatusRescheduled, domain.StatusCompleted}}
	today := dashboardCopy(appointments)
	today["start_time"] = bson.M{"$gte": w.TodayStart, "$lt": w.TodayEnd}
	count, err := r.db.Collection("lead_appointments").CountDocuments(ctx, today)
	if err != nil {
		return err
	}
	result.Metrics["appointments_today"] = dashboardMetric(float64(count), "number", "today")
	today["status"] = domain.StatusCompleted
	completed, err := r.db.Collection("lead_appointments").CountDocuments(ctx, today)
	if err != nil {
		return err
	}
	result.Metrics["appointments_completed_today"] = dashboardMetric(float64(completed), "number", "today")
	followUps, err := dashboardFilter(scope, "lead_follow_ups")
	if err != nil {
		return err
	}
	active := dashboardCopy(followUps)
	active["status"] = domain.StatusActive
	rows, err := dashboardAggregate[dashboardStatusRow](ctx, r.db.Collection("lead_follow_ups"), mongo.Pipeline{
		{{Key: "$match", Value: active}},
		{{Key: "$group", Value: bson.M{"_id": nil, "total": bson.M{"$sum": 1}, "current": dashboardConditionalSum(bson.M{"$lt": bson.A{"$end_time", w.Now}}, 1)}}},
	})
	if err != nil {
		return err
	}
	var open dashboardStatusRow
	if len(rows) > 0 {
		open = rows[0]
	}
	result.Metrics["open_follow_ups"] = dashboardMetric(float64(open.Total), "number", "current")
	result.Metrics["overdue_follow_ups"] = dashboardMetric(float64(open.Current), "number", "current")
	comments, err := dashboardFilter(scope, "lead_comments")
	if err != nil {
		return err
	}
	commentTotals, err := r.dashboardPeriodTotals(ctx, "lead_comments", comments, "created_at", "", w)
	if err != nil {
		return err
	}
	result.Metrics["comments_month"] = dashboardComparison(commentTotals.Current, commentTotals.Previous, "number")
	for _, source := range []struct {
		collection string
		filter     bson.M
		date       string
	}{
		{"lead_appointments", appointments, "start_time"}, {"lead_follow_ups", followUps, "start_time"}, {"lead_comments", comments, "created_at"},
	} {
		series, err := r.dashboardSeries(ctx, source.collection, source.filter, source.date, "", w)
		if err != nil {
			return err
		}
		for index := range result.Trend {
			value := int64(series[result.Trend[index].Start])
			switch source.collection {
			case "lead_appointments":
				result.Trend[index].Appointments = value
			case "lead_follow_ups":
				result.Trend[index].FollowUps = value
			case "lead_comments":
				result.Trend[index].Comments = value
			}
		}
	}
	tasks, err := r.dashboardTasks(ctx, scope, w)
	if err != nil {
		return err
	}
	result.Upcoming = tasks
	return nil
}

func (r *MongoDashboardRepository) loadDashboardFinance(ctx context.Context, scope ports.DashboardScope, w ports.DashboardWindow, result *ports.DashboardSummary) error {
	if scope.Role != "admin" {
		return ports.ErrDashboardRole
	}
	filter, err := dashboardFilter(scope, "receipts")
	if err != nil {
		return err
	}
	totals, err := r.dashboardPeriodTotals(ctx, "receipts", filter, "payment_date", "amount_paid", w)
	if err != nil {
		return err
	}
	result.Metrics["revenue"] = dashboardComparison(dashboardMoney(totals.Current), dashboardMoney(totals.Previous), "currency")
	series, err := r.dashboardSeries(ctx, "receipts", filter, "payment_date", "amount_paid", w)
	if err != nil {
		return err
	}
	for index := range result.Trend {
		value := dashboardMoney(series[result.Trend[index].Start])
		result.Trend[index].Revenue = &value
	}
	invoiceFilter, err := dashboardFilter(scope, "invoices")
	if err != nil {
		return err
	}
	invoiceFilter["created_at"] = bson.M{"$lt": w.Now}
	rows, err := dashboardAggregate[dashboardInvoiceRow](ctx, r.db.Collection("invoices"), mongo.Pipeline{
		{{Key: "$match", Value: invoiceFilter}},
		{{Key: "$group", Value: bson.M{"_id": nil,
			"current":     dashboardConditionalSum(dashboardInRange("created_at", w.Start, w.End), "$total_amount"),
			"previous":    dashboardConditionalSum(dashboardInRange("created_at", w.ComparisonStart, w.ComparisonEnd), "$total_amount"),
			"outstanding": bson.M{"$sum": bson.M{"$max": bson.A{0, bson.M{"$subtract": bson.A{bson.M{"$ifNull": bson.A{"$total_amount", 0}}, bson.M{"$ifNull": bson.A{"$paid_amount_vat", 0}}}}}}},
			"open":        dashboardConditionalSum(bson.M{"$in": bson.A{"$status", bson.A{domain.InvoiceStatusPending, domain.InvoiceStatusPartial}}}, 1),
		}}},
	})
	if err != nil {
		return err
	}
	var invoices dashboardInvoiceRow
	if len(rows) > 0 {
		invoices = rows[0]
	}
	result.Metrics["invoiced"] = dashboardComparison(dashboardMoney(invoices.Current), dashboardMoney(invoices.Previous), "currency")
	result.Metrics["outstanding"] = dashboardMetric(dashboardMoney(invoices.Outstanding), "currency", "current")
	result.Metrics["open_invoices"] = dashboardMetric(float64(invoices.Open), "number", "current")
	return nil
}

// Each collection is aggregated once across all six months, not once per bucket.
func (r *MongoDashboardRepository) dashboardSeries(ctx context.Context, collection string, base bson.M, dateField, sumField string, w ports.DashboardWindow) (map[string]float64, error) {
	filter := dashboardCopy(base)
	filter[dateField] = bson.M{"$gte": w.TrendStart, "$lt": w.End}
	var value interface{} = 1
	if sumField != "" {
		value = "$" + sumField
	}
	rows, err := dashboardAggregate[dashboardSeriesRow](ctx, r.db.Collection(collection), mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.M{"_id": bson.M{"$dateToString": bson.M{"format": "%Y-%m-01", "date": "$" + dateField, "timezone": w.Location.String()}}, "value": bson.M{"$sum": value}}}},
	})
	if err != nil {
		return nil, err
	}
	result := make(map[string]float64, len(rows))
	for _, row := range rows {
		result[row.Key] = row.Value
	}
	return result, nil
}

func (r *MongoDashboardRepository) dashboardPeriodTotals(ctx context.Context, collection string, base bson.M, dateField, sumField string, w ports.DashboardWindow) (dashboardMoneyRow, error) {
	filter := dashboardCopy(base)
	filter[dateField] = bson.M{"$gte": w.ComparisonStart, "$lt": w.End}
	var value interface{} = 1
	if sumField != "" {
		value = "$" + sumField
	}
	rows, err := dashboardAggregate[dashboardMoneyRow](ctx, r.db.Collection(collection), mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.M{"_id": nil,
			"current":  dashboardConditionalSum(dashboardInRange(dateField, w.Start, w.End), value),
			"previous": dashboardConditionalSum(dashboardInRange(dateField, w.ComparisonStart, w.ComparisonEnd), value),
		}}},
	})
	if err != nil {
		return dashboardMoneyRow{}, err
	}
	if len(rows) == 0 {
		return dashboardMoneyRow{}, nil
	}
	return rows[0], nil
}

func dashboardAggregate[T any](ctx context.Context, collection *mongo.Collection, pipeline mongo.Pipeline) ([]T, error) {
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	rows := []T{}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func dashboardCopy(source bson.M) bson.M {
	result := bson.M{}
	for key, value := range source {
		result[key] = value
	}
	return result
}

func dashboardInRange(field string, start, end time.Time) bson.M {
	return bson.M{"$and": bson.A{bson.M{"$gte": bson.A{"$" + field, start}}, bson.M{"$lt": bson.A{"$" + field, end}}}}
}

func dashboardConditionalSum(condition, value interface{}) bson.M {
	return bson.M{"$sum": bson.M{"$cond": bson.A{condition, value, 0}}}
}

func dashboardMetric(value float64, format, period string) ports.DashboardMetric {
	return ports.DashboardMetric{Value: value, Format: format, Period: period}
}
func dashboardComparison(value, previous float64, format string) ports.DashboardMetric {
	return ports.DashboardMetric{Value: value, Previous: &previous, Format: format, Period: "month"}
}
func dashboardTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
func dashboardMoney(value float64) float64 { return math.Round(value*100) / 100 }

type dashboardStatusRow struct {
	Status   string `bson:"_id"`
	Total    int64  `bson:"total"`
	Current  int64  `bson:"current"`
	Previous int64  `bson:"previous"`
}
type dashboardSeriesRow struct {
	Key   string  `bson:"_id"`
	Value float64 `bson:"value"`
}
type dashboardMoneyRow struct {
	Current  float64 `bson:"current"`
	Previous float64 `bson:"previous"`
}
type dashboardInvoiceRow struct {
	Current     float64 `bson:"current"`
	Previous    float64 `bson:"previous"`
	Outstanding float64 `bson:"outstanding"`
	Open        int64   `bson:"open"`
}

var _ ports.DashboardRepository = (*MongoDashboardRepository)(nil)

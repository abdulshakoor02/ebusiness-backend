package storage

import (
	"context"
	"sort"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Fetch six of each kind so the global top six cannot omit the fifth or sixth
// appointment when fewer follow-ups exist. Active overdue follow-ups are included.
func (r *MongoDashboardRepository) dashboardTasks(ctx context.Context, scope ports.DashboardScope, w ports.DashboardWindow) ([]ports.DashboardTask, error) {
	appointmentFilter, err := dashboardFilter(scope, "lead_appointments")
	if err != nil {
		return nil, err
	}
	appointmentFilter["status"] = bson.M{"$in": bson.A{domain.StatusScheduled, domain.StatusRescheduled}}
	appointmentFilter["start_time"] = bson.M{"$lt": w.UpcomingEnd}
	appointmentFilter["end_time"] = bson.M{"$gte": w.Now}
	followUpFilter, err := dashboardFilter(scope, "lead_follow_ups")
	if err != nil {
		return nil, err
	}
	followUpFilter["status"] = domain.StatusActive
	followUpFilter["start_time"] = bson.M{"$lt": w.UpcomingEnd}
	tasks := make([]ports.DashboardTask, 0, 12)
	for _, source := range []struct {
		collection, kind string
		filter           bson.M
	}{
		{"lead_appointments", "appointment", appointmentFilter}, {"lead_follow_ups", "follow_up", followUpFilter},
	} {
		cursor, err := r.db.Collection(source.collection).Find(ctx, source.filter, options.Find().SetLimit(6).
			SetSort(bson.D{{Key: "start_time", Value: 1}, {Key: "_id", Value: 1}}).
			SetProjection(bson.M{"lead_id": 1, "title": 1, "start_time": 1, "end_time": 1, "status": 1}))
		if err != nil {
			return nil, err
		}
		var records []dashboardTaskRecord
		if err := cursor.All(ctx, &records); err != nil {
			return nil, err
		}
		for _, record := range records {
			leadID := ""
			if !record.LeadID.IsZero() {
				leadID = record.LeadID.Hex()
			}
			tasks = append(tasks, ports.DashboardTask{
				ID: record.ID.Hex(), LeadID: leadID, Kind: source.kind, Title: record.Title,
				StartTime: dashboardTime(record.Start), EndTime: dashboardTime(record.End), Status: record.Status,
				Overdue: source.kind == "follow_up" && record.End.Before(w.Now),
			})
		}
	}
	sort.Slice(tasks, func(i, j int) bool {
		left, _ := time.Parse(time.RFC3339Nano, tasks[i].StartTime)
		right, _ := time.Parse(time.RFC3339Nano, tasks[j].StartTime)
		if left.Equal(right) {
			return tasks[i].Kind+tasks[i].ID < tasks[j].Kind+tasks[j].ID
		}
		return left.Before(right)
	})
	if len(tasks) > 6 {
		tasks = tasks[:6]
	}
	return tasks, nil
}

type dashboardTaskRecord struct {
	ID     primitive.ObjectID `bson:"_id"`
	LeadID primitive.ObjectID `bson:"lead_id"`
	Title  string             `bson:"title"`
	Start  time.Time          `bson:"start_time"`
	End    time.Time          `bson:"end_time"`
	Status string             `bson:"status"`
}

// Limit tenants before joins. Each lookup returns only a count, never complete
// users, their credentials, or an unbounded array of lead IDs.
func (r *MongoDashboardRepository) dashboardTenants(ctx context.Context, now time.Time) ([]ports.DashboardTenantHighlight, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"created_at": bson.M{"$lt": now}}}},
		{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}}},
		{{Key: "$limit", Value: 6}},
	}
	for _, collection := range []string{"leads", "users"} {
		pipeline = append(pipeline, bson.D{{Key: "$lookup", Value: bson.M{
			"from": collection, "let": bson.M{"tenant": "$_id"}, "as": collection,
			"pipeline": mongo.Pipeline{
				{{Key: "$match", Value: bson.M{"$expr": bson.M{"$eq": bson.A{"$tenant_id", "$$tenant"}}, "created_at": bson.M{"$lt": now}}}},
				{{Key: "$count", Value: "value"}},
			},
		}}})
	}
	pipeline = append(pipeline, bson.D{{Key: "$project", Value: bson.M{
		"name": 1, "created_at": 1,
		"leads": bson.M{"$ifNull": bson.A{bson.M{"$arrayElemAt": bson.A{"$leads.value", 0}}, 0}},
		"users": bson.M{"$ifNull": bson.A{bson.M{"$arrayElemAt": bson.A{"$users.value", 0}}, 0}},
	}}})
	rows, err := dashboardAggregate[struct {
		ID        primitive.ObjectID `bson:"_id"`
		Name      string             `bson:"name"`
		CreatedAt time.Time          `bson:"created_at"`
		Leads     int64              `bson:"leads"`
		Users     int64              `bson:"users"`
	}](ctx, r.db.Collection("tenants"), pipeline)
	if err != nil {
		return nil, err
	}
	result := make([]ports.DashboardTenantHighlight, 0, len(rows))
	for _, row := range rows {
		result = append(result, ports.DashboardTenantHighlight{ID: row.ID.Hex(), Name: row.Name, CreatedAt: dashboardTime(row.CreatedAt), Leads: row.Leads, Users: row.Users})
	}
	return result, nil
}

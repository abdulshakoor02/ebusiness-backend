package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AppointmentStatus string

const (
	StatusScheduled   AppointmentStatus = "scheduled"
	StatusCompleted   AppointmentStatus = "completed"
	StatusRescheduled AppointmentStatus = "rescheduled"
	StatusCancelled   AppointmentStatus = "cancelled"
)

type LeadAppointment struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID    primitive.ObjectID `bson:"tenant_id" json:"tenant_id"`
	LeadID      primitive.ObjectID `bson:"lead_id" json:"lead_id"`
	OrganizerID primitive.ObjectID `bson:"organizer_id" json:"organizer_id"` // user who scheduled it
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	StartTime   time.Time          `bson:"start_time" json:"start_time"`
	EndTime     time.Time          `bson:"end_time" json:"end_time"`
	Status      AppointmentStatus  `bson:"status" json:"status"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

func NewLeadAppointment(tenantID, leadID, organizerID primitive.ObjectID, title, description string, startTime, endTime time.Time, status AppointmentStatus) *LeadAppointment {
	return &LeadAppointment{
		ID:          primitive.NewObjectID(),
		TenantID:    tenantID,
		LeadID:      leadID,
		OrganizerID: organizerID,
		Title:       title,
		Description: description,
		StartTime:   startTime,
		EndTime:     endTime,
		Status:      status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

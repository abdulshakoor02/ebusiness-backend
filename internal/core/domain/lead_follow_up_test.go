package domain

import (
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestFollowUpStatus_Values(t *testing.T) {
	if StatusActive != "active" {
		t.Errorf("StatusActive = %v, want %v", StatusActive, "active")
	}
	if StatusClosed != "closed" {
		t.Errorf("StatusClosed = %v, want %v", StatusClosed, "closed")
	}
}

func TestIsValidFollowUpStatus(t *testing.T) {
	tests := []struct {
		name   string
		status FollowUpStatus
		want   bool
	}{
		{"valid active", StatusActive, true},
		{"valid closed", StatusClosed, true},
		{"invalid status", "invalid", false},
		{"empty status", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidFollowUpStatus(tt.status); got != tt.want {
				t.Errorf("IsValidFollowUpStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLeadFollowUp_FieldNames(t *testing.T) {
	typ := reflect.TypeOf(LeadFollowUp{})

	// Check all expected fields exist
	expectedFields := []string{
		"ID", "TenantID", "LeadID", "CreatorID", "Title",
		"Description", "StartTime", "EndTime", "Status",
		"CreatedAt", "UpdatedAt",
	}

	for _, fieldName := range expectedFields {
		if _, found := typ.FieldByName(fieldName); !found {
			t.Errorf("LeadFollowUp missing field: %s", fieldName)
		}
	}

	// Ensure OrganizerID is NOT present (we renamed to CreatorID)
	if _, found := typ.FieldByName("OrganizerID"); found {
		t.Error("LeadFollowUp should NOT have OrganizerID field (use CreatorID instead)")
	}

	// Verify CreatorID exists
	if _, found := typ.FieldByName("CreatorID"); !found {
		t.Error("LeadFollowUp must have CreatorID field")
	}
}

func TestNewLeadFollowUp(t *testing.T) {
	tenantID := primitive.NewObjectID()
	leadID := primitive.NewObjectID()
	creatorID := primitive.NewObjectID()
	title := "Test Follow-up"
	description := "Test description"
	startTime := time.Now()
	endTime := startTime.Add(time.Hour)
	status := StatusActive

	followUp := NewLeadFollowUp(tenantID, leadID, creatorID, title, description, startTime, endTime, status)

	if followUp.ID.IsZero() {
		t.Error("NewLeadFollowUp() should generate non-zero ID")
	}
	if followUp.TenantID != tenantID {
		t.Errorf("TenantID = %v, want %v", followUp.TenantID, tenantID)
	}
	if followUp.LeadID != leadID {
		t.Errorf("LeadID = %v, want %v", followUp.LeadID, leadID)
	}
	if followUp.CreatorID != creatorID {
		t.Errorf("CreatorID = %v, want %v", followUp.CreatorID, creatorID)
	}
	if followUp.Title != title {
		t.Errorf("Title = %v, want %v", followUp.Title, title)
	}
	if followUp.Description != description {
		t.Errorf("Description = %v, want %v", followUp.Description, description)
	}
	if followUp.Status != status {
		t.Errorf("Status = %v, want %v", followUp.Status, status)
	}
	if followUp.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if followUp.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

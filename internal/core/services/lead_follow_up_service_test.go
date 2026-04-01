package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Mock repository
type MockLeadFollowUpRepo struct {
	mock.Mock
}

func (m *MockLeadFollowUpRepo) Create(ctx context.Context, followUp *domain.LeadFollowUp) error {
	args := m.Called(ctx, followUp)
	return args.Error(0)
}

func (m *MockLeadFollowUpRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.LeadFollowUp, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.LeadFollowUp), args.Error(1)
}

func (m *MockLeadFollowUpRepo) ListByLeadID(ctx context.Context, leadID primitive.ObjectID, filter interface{}, offset, limit int64) ([]*ports.FollowUpListItem, int64, error) {
	args := m.Called(ctx, leadID, filter, offset, limit)
	return args.Get(0).([]*ports.FollowUpListItem), args.Get(1).(int64), args.Error(2)
}

func (m *MockLeadFollowUpRepo) Update(ctx context.Context, followUp *domain.LeadFollowUp) error {
	args := m.Called(ctx, followUp)
	return args.Error(0)
}

func (m *MockLeadFollowUpRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Mock lead repository
type MockLeadRepo struct {
	mock.Mock
}

func (m *MockLeadRepo) Create(ctx context.Context, lead *domain.Lead) error {
	args := m.Called(ctx, lead)
	return args.Error(0)
}

func (m *MockLeadRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Lead, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Lead), args.Error(1)
}

func (m *MockLeadRepo) List(ctx context.Context, filter interface{}, search string, offset, limit int64) ([]*ports.LeadListItem, int64, error) {
	args := m.Called(ctx, filter, search, offset, limit)
	return args.Get(0).([]*ports.LeadListItem), args.Get(1).(int64), args.Error(2)
}

func (m *MockLeadRepo) Update(ctx context.Context, lead *domain.Lead) error {
	args := m.Called(ctx, lead)
	return args.Error(0)
}

func (m *MockLeadRepo) UpdateComments(ctx context.Context, leadID primitive.ObjectID, comments string) error {
	args := m.Called(ctx, leadID, comments)
	return args.Error(0)
}

func TestCreateLeadFollowUp_Success(t *testing.T) {
	mockFollowUpRepo := new(MockLeadFollowUpRepo)
	mockLeadRepo := new(MockLeadRepo)
	service := NewLeadFollowUpService(mockFollowUpRepo, mockLeadRepo)

	tenantID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	leadID := primitive.NewObjectID()
	ctx := context.WithValue(context.WithValue(context.Background(), "tenant_id", tenantID.Hex()), "user_id", userID.Hex())

	mockLeadRepo.On("GetByID", mock.Anything, leadID).Return(&domain.Lead{ID: leadID, TenantID: tenantID}, nil)
	mockFollowUpRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.LeadFollowUp")).Return(nil)

	req := ports.CreateLeadFollowUpRequest{
		Title:       "Test Follow-up",
		Description: "Test description",
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(time.Hour),
		Status:      "active",
	}

	followUp, err := service.CreateLeadFollowUp(ctx, leadID, req)

	assert.NoError(t, err)
	assert.NotNil(t, followUp)
	assert.Equal(t, req.Title, followUp.Title)
	assert.Equal(t, tenantID, followUp.TenantID)
	assert.Equal(t, leadID, followUp.LeadID)
	mockLeadRepo.AssertExpectations(t)
	mockFollowUpRepo.AssertExpectations(t)
}

func TestCreateLeadFollowUp_MissingTenant(t *testing.T) {
	mockFollowUpRepo := new(MockLeadFollowUpRepo)
	mockLeadRepo := new(MockLeadRepo)
	service := NewLeadFollowUpService(mockFollowUpRepo, mockLeadRepo)

	ctx := context.Background()
	leadID := primitive.NewObjectID()

	req := ports.CreateLeadFollowUpRequest{
		Title:     "Test",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Hour),
	}

	followUp, err := service.CreateLeadFollowUp(ctx, leadID, req)

	assert.Error(t, err)
	assert.Equal(t, "tenant context required", err.Error())
	assert.Nil(t, followUp)
}

func TestCreateLeadFollowUp_MissingUser(t *testing.T) {
	mockFollowUpRepo := new(MockLeadFollowUpRepo)
	mockLeadRepo := new(MockLeadRepo)
	service := NewLeadFollowUpService(mockFollowUpRepo, mockLeadRepo)

	tenantID := primitive.NewObjectID()
	ctx := context.WithValue(context.Background(), "tenant_id", tenantID.Hex())
	leadID := primitive.NewObjectID()

	req := ports.CreateLeadFollowUpRequest{
		Title:     "Test",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Hour),
	}

	followUp, err := service.CreateLeadFollowUp(ctx, leadID, req)

	assert.Error(t, err)
	assert.Equal(t, "user context required", err.Error())
	assert.Nil(t, followUp)
}

func TestCreateLeadFollowUp_InvalidLead(t *testing.T) {
	mockFollowUpRepo := new(MockLeadFollowUpRepo)
	mockLeadRepo := new(MockLeadRepo)
	service := NewLeadFollowUpService(mockFollowUpRepo, mockLeadRepo)

	tenantID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	leadID := primitive.NewObjectID()
	ctx := context.WithValue(context.WithValue(context.Background(), "tenant_id", tenantID.Hex()), "user_id", userID.Hex())

	mockLeadRepo.On("GetByID", mock.Anything, leadID).Return(nil, errors.New("lead not found"))

	req := ports.CreateLeadFollowUpRequest{
		Title:     "Test",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Hour),
	}

	followUp, err := service.CreateLeadFollowUp(ctx, leadID, req)

	assert.Error(t, err)
	assert.Equal(t, "lead not found or unauthorized", err.Error())
	assert.Nil(t, followUp)
	mockLeadRepo.AssertExpectations(t)
}

func TestCreateLeadFollowUp_InvalidStatus(t *testing.T) {
	mockFollowUpRepo := new(MockLeadFollowUpRepo)
	mockLeadRepo := new(MockLeadRepo)
	service := NewLeadFollowUpService(mockFollowUpRepo, mockLeadRepo)

	tenantID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	leadID := primitive.NewObjectID()
	ctx := context.WithValue(context.WithValue(context.Background(), "tenant_id", tenantID.Hex()), "user_id", userID.Hex())

	mockLeadRepo.On("GetByID", mock.Anything, leadID).Return(&domain.Lead{ID: leadID, TenantID: tenantID}, nil)

	req := ports.CreateLeadFollowUpRequest{
		Title:     "Test",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Hour),
		Status:    "invalid_status",
	}

	followUp, err := service.CreateLeadFollowUp(ctx, leadID, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
	assert.Nil(t, followUp)
}

func TestUpdateLeadFollowUp_AnyUser(t *testing.T) {
	mockFollowUpRepo := new(MockLeadFollowUpRepo)
	mockLeadRepo := new(MockLeadRepo)
	service := NewLeadFollowUpService(mockFollowUpRepo, mockLeadRepo)

	// Setup: User A creates the follow-up
	creatorID := primitive.NewObjectID()
	updaterID := primitive.NewObjectID() // Different user!
	tenantID := primitive.NewObjectID()
	followUpID := primitive.NewObjectID()
	leadID := primitive.NewObjectID()

	ctx := context.WithValue(context.WithValue(context.Background(), "tenant_id", tenantID.Hex()), "user_id", updaterID.Hex())

	existingFollowUp := &domain.LeadFollowUp{
		ID:        followUpID,
		TenantID:  tenantID,
		LeadID:    leadID,
		CreatorID: creatorID, // User A created it
		Title:     "Original Title",
		Status:    domain.StatusActive,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Hour),
	}

	mockFollowUpRepo.On("GetByID", mock.Anything, followUpID).Return(existingFollowUp, nil)
	mockFollowUpRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.LeadFollowUp")).Return(nil)

	req := ports.UpdateLeadFollowUpRequest{
		Title: "Updated by different user",
	}

	updatedFollowUp, err := service.UpdateLeadFollowUp(ctx, followUpID, req)

	// KEY TEST: Should succeed even though updater is NOT the creator
	assert.NoError(t, err)
	assert.NotNil(t, updatedFollowUp)
	assert.Equal(t, "Updated by different user", updatedFollowUp.Title)
	mockFollowUpRepo.AssertExpectations(t)
}

func TestUpdateLeadFollowUp_ClosedFollowUp(t *testing.T) {
	mockFollowUpRepo := new(MockLeadFollowUpRepo)
	mockLeadRepo := new(MockLeadRepo)
	service := NewLeadFollowUpService(mockFollowUpRepo, mockLeadRepo)

	userID := primitive.NewObjectID()
	tenantID := primitive.NewObjectID()
	followUpID := primitive.NewObjectID()
	ctx := context.WithValue(context.WithValue(context.Background(), "tenant_id", tenantID.Hex()), "user_id", userID.Hex())

	closedFollowUp := &domain.LeadFollowUp{
		ID:       followUpID,
		TenantID: tenantID,
		Title:    "Closed Follow-up",
		Status:   domain.StatusClosed, // Closed!
	}

	mockFollowUpRepo.On("GetByID", mock.Anything, followUpID).Return(closedFollowUp, nil)

	req := ports.UpdateLeadFollowUpRequest{
		Title: "Attempted Update",
	}

	updatedFollowUp, err := service.UpdateLeadFollowUp(ctx, followUpID, req)

	assert.Error(t, err)
	assert.Equal(t, "cannot update closed follow-up", err.Error())
	assert.Nil(t, updatedFollowUp)
	mockFollowUpRepo.AssertExpectations(t)
}

func TestUpdateLeadFollowUp_InvalidStatus(t *testing.T) {
	mockFollowUpRepo := new(MockLeadFollowUpRepo)
	mockLeadRepo := new(MockLeadRepo)
	service := NewLeadFollowUpService(mockFollowUpRepo, mockLeadRepo)

	userID := primitive.NewObjectID()
	tenantID := primitive.NewObjectID()
	followUpID := primitive.NewObjectID()
	ctx := context.WithValue(context.WithValue(context.Background(), "tenant_id", tenantID.Hex()), "user_id", userID.Hex())

	existingFollowUp := &domain.LeadFollowUp{
		ID:       followUpID,
		TenantID: tenantID,
		Title:    "Test",
		Status:   domain.StatusActive,
	}

	mockFollowUpRepo.On("GetByID", mock.Anything, followUpID).Return(existingFollowUp, nil)

	req := ports.UpdateLeadFollowUpRequest{
		Status: "invalid_status",
	}

	updatedFollowUp, err := service.UpdateLeadFollowUp(ctx, followUpID, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
	assert.Nil(t, updatedFollowUp)
}

func TestDeleteLeadFollowUp_AnyUser(t *testing.T) {
	mockFollowUpRepo := new(MockLeadFollowUpRepo)
	mockLeadRepo := new(MockLeadRepo)
	service := NewLeadFollowUpService(mockFollowUpRepo, mockLeadRepo)

	// Setup: User A creates the follow-up
	creatorID := primitive.NewObjectID()
	deleterID := primitive.NewObjectID() // Different user!
	tenantID := primitive.NewObjectID()
	followUpID := primitive.NewObjectID()
	leadID := primitive.NewObjectID()

	ctx := context.WithValue(context.WithValue(context.Background(), "tenant_id", tenantID.Hex()), "user_id", deleterID.Hex())

	existingFollowUp := &domain.LeadFollowUp{
		ID:        followUpID,
		TenantID:  tenantID,
		LeadID:    leadID,
		CreatorID: creatorID, // User A created it
		Title:     "Test",
		Status:    domain.StatusActive,
	}

	mockFollowUpRepo.On("GetByID", mock.Anything, followUpID).Return(existingFollowUp, nil)
	mockFollowUpRepo.On("Delete", mock.Anything, followUpID).Return(nil)

	err := service.DeleteLeadFollowUp(ctx, followUpID)

	// KEY TEST: Should succeed even though deleter is NOT the creator
	assert.NoError(t, err)
	mockFollowUpRepo.AssertExpectations(t)
}

func TestListLeadFollowUps_Pagination(t *testing.T) {
	mockFollowUpRepo := new(MockLeadFollowUpRepo)
	mockLeadRepo := new(MockLeadRepo)
	service := NewLeadFollowUpService(mockFollowUpRepo, mockLeadRepo)

	tenantID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	leadID := primitive.NewObjectID()
	ctx := context.WithValue(context.WithValue(context.Background(), "tenant_id", tenantID.Hex()), "user_id", userID.Hex())

	expectedItems := []*ports.FollowUpListItem{
		{ID: primitive.NewObjectID(), Title: "Follow-up 1"},
		{ID: primitive.NewObjectID(), Title: "Follow-up 2"},
	}

	mockFollowUpRepo.On("ListByLeadID", mock.Anything, leadID, mock.Anything, int64(0), int64(10)).Return(expectedItems, int64(2), nil)

	req := ports.FilterRequest{
		Offset: 0,
		Limit:  10,
	}

	items, total, err := service.ListLeadFollowUps(ctx, leadID, req)

	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, items, 2)
	mockFollowUpRepo.AssertExpectations(t)
}

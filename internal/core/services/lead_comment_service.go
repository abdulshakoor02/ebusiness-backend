package services

import (
	"context"
	"errors"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"github.com/abdulshakoor02/goCrmBackend/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeadCommentService struct {
	commentRepo ports.LeadCommentRepository
	leadRepo    ports.LeadRepository
}

func NewLeadCommentService(commentRepo ports.LeadCommentRepository, leadRepo ports.LeadRepository) *LeadCommentService {
	return &LeadCommentService{
		commentRepo: commentRepo,
		leadRepo:    leadRepo,
	}
}

func (s *LeadCommentService) CreateLeadComment(ctx context.Context, leadID primitive.ObjectID, req ports.CreateLeadCommentRequest) (*domain.LeadComment, error) {
	tenantID, ok := getTenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required")
	}

	authorID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, errors.New("user context required")
	}

	// Verify lead exists and belongs to tenant
	_, err := s.leadRepo.GetByID(ctx, leadID)
	if err != nil {
		return nil, errors.New("lead not found or unauthorized")
	}

	comment := domain.NewLeadComment(
		tenantID,
		leadID,
		authorID,
		req.Content,
	)

	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, err
	}

	s.leadRepo.UpdateComments(ctx, leadID, comment.Content)

	return comment, nil
}

func (s *LeadCommentService) GetLeadComment(ctx context.Context, id primitive.ObjectID) (*domain.LeadComment, error) {
	return s.commentRepo.GetByID(ctx, id)
}

func (s *LeadCommentService) UpdateLeadComment(ctx context.Context, id primitive.ObjectID, req ports.UpdateLeadCommentRequest) (*domain.LeadComment, error) {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, errors.New("user context required")
	}

	if comment.AuthorID != userID {
		return nil, errors.New("unauthorized to update this comment")
	}

	if req.Content != "" {
		comment.Content = req.Content
	}

	if err := s.commentRepo.Update(ctx, comment); err != nil {
		return nil, err
	}

	s.leadRepo.UpdateComments(ctx, comment.LeadID, comment.Content)

	return comment, nil
}

func (s *LeadCommentService) DeleteLeadComment(ctx context.Context, id primitive.ObjectID) error {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return errors.New("user context required")
	}

	if comment.AuthorID != userID {
		return errors.New("unauthorized to delete this comment")
	}

	if err := s.commentRepo.Delete(ctx, id); err != nil {
		return err
	}

	latestComment, err := s.commentRepo.GetLatestByLeadID(ctx, comment.LeadID)
	if err != nil {
		return nil
	}

	if latestComment != nil {
		s.leadRepo.UpdateComments(ctx, comment.LeadID, latestComment.Content)
	} else {
		s.leadRepo.UpdateComments(ctx, comment.LeadID, "")
	}

	return nil
}

func (s *LeadCommentService) ListLeadComments(ctx context.Context, leadID primitive.ObjectID, req ports.FilterRequest) ([]*ports.CommentListItem, int64, error) {
	return s.commentRepo.ListByLeadID(ctx, leadID, req.Filters, req.Offset, req.Limit)
}

// Helper to extract user ID from context
func getUserIDFromContext(ctx context.Context) (primitive.ObjectID, bool) {
	val := ctx.Value("user_id")
	if val == nil {
		return primitive.NilObjectID, false
	}
	if idStr, ok := val.(string); ok {
		id, err := primitive.ObjectIDFromHex(idStr)
		if err == nil {
			return id, true
		}
	}
	if id, ok := val.(primitive.ObjectID); ok {
		return id, true
	}
	return primitive.NilObjectID, false
}

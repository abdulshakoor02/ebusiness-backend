package storage

import (
	"context"

	"github.com/abdulshakoor02/goCrmBackend/internal/core/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoFacebookAssignmentRepository struct {
	collection *mongo.Collection
}

func NewMongoFacebookAssignmentRepository(db *mongo.Database) *MongoFacebookAssignmentRepository {
	return &MongoFacebookAssignmentRepository{
		collection: db.Collection("facebook_campaign_assignments"),
	}
}

func (r *MongoFacebookAssignmentRepository) Upsert(ctx context.Context, a *domain.FacebookCampaignAssignment) error {
	filter := bson.M{"tenant_id": a.TenantID, "campaign_id": a.CampaignID}
	update := bson.M{"$set": a}
	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *MongoFacebookAssignmentRepository) GetByCampaign(ctx context.Context, tenantID primitive.ObjectID, campaignID string) (*domain.FacebookCampaignAssignment, error) {
	var a domain.FacebookCampaignAssignment
	err := r.collection.FindOne(ctx, bson.M{"tenant_id": tenantID, "campaign_id": campaignID}).Decode(&a)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *MongoFacebookAssignmentRepository) ListByTenant(ctx context.Context, tenantID primitive.ObjectID) ([]*domain.FacebookCampaignAssignment, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"tenant_id": tenantID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var out []*domain.FacebookCampaignAssignment
	if err := cursor.All(ctx, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []*domain.FacebookCampaignAssignment{}
	}
	return out, nil
}

func (r *MongoFacebookAssignmentRepository) Delete(ctx context.Context, tenantID primitive.ObjectID, campaignID string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"tenant_id": tenantID, "campaign_id": campaignID})
	return err
}

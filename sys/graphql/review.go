package graphql

import (
	"context"
	"errors"

	"cleanbuddy-api/res/store"
	"cleanbuddy-api/sys/graphql/gen"
	"cleanbuddy-api/sys/graphql/scalar"
)

// QUERY RESOLVERS
func (qr *queryResolver) Review(ctx context.Context, id string) (*store.Review, error) {
	return nil, errors.New("not yet implemented")
}

func (qr *queryResolver) ReviewByBooking(ctx context.Context, bookingID string) (*store.Review, error) {
	return nil, errors.New("not yet implemented")
}

func (qr *queryResolver) ReviewsForCleaner(ctx context.Context, cleanerProfileID string, filters *gen.ReviewFiltersInput, limit, offset *int, orderBy *string) (*gen.ReviewConnection, error) {
	return nil, errors.New("not yet implemented")
}

func (qr *queryResolver) MyReviews(ctx context.Context, filters *gen.ReviewFiltersInput, limit, offset *int, orderBy *string) (*gen.ReviewConnection, error) {
	return nil, errors.New("not yet implemented")
}

func (qr *queryResolver) ReviewsPendingModeration(ctx context.Context, limit *int) ([]*store.Review, error) {
	return nil, errors.New("not yet implemented")
}

// MUTATION RESOLVERS
func (mr *mutationResolver) CreateReview(ctx context.Context, input gen.CreateReviewInput) (*store.Review, error) {
	return nil, errors.New("not yet implemented")
}

func (mr *mutationResolver) UpdateReview(ctx context.Context, input gen.UpdateReviewInput) (*store.Review, error) {
	return nil, errors.New("not yet implemented")
}

func (mr *mutationResolver) DeleteReview(ctx context.Context, id string) (*scalar.Void, error) {
	return nil, errors.New("not yet implemented")
}

func (mr *mutationResolver) AddCleanerResponse(ctx context.Context, input gen.AddCleanerResponseInput) (*store.Review, error) {
	return nil, errors.New("not yet implemented")
}

func (mr *mutationResolver) FlagReview(ctx context.Context, input gen.FlagReviewInput) (*store.Review, error) {
	return nil, errors.New("not yet implemented")
}

func (mr *mutationResolver) ModerateReview(ctx context.Context, input gen.ModerateReviewInput) (*store.Review, error) {
	return nil, errors.New("not yet implemented")
}

func (mr *mutationResolver) MarkReviewHelpful(ctx context.Context, reviewID string, helpful bool) (*store.Review, error) {
	return nil, errors.New("not yet implemented")
}

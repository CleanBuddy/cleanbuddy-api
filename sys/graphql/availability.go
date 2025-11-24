package graphql

import (
	"context"
	"errors"

	"cleanbuddy-api/res/store"
	"cleanbuddy-api/sys/graphql/gen"
	"cleanbuddy-api/sys/graphql/scalar"
)

// QUERY RESOLVERS
func (qr *queryResolver) Availability(ctx context.Context, id string) (*store.Availability, error) {
	return nil, errors.New("not yet implemented")
}

func (qr *queryResolver) AvailabilityForCleaner(ctx context.Context, cleanerProfileID string, filters *gen.AvailabilityFiltersInput, limit, offset *int) ([]*store.Availability, error) {
	return nil, errors.New("not yet implemented")
}

func (qr *queryResolver) MyAvailability(ctx context.Context, filters *gen.AvailabilityFiltersInput, limit, offset *int) ([]*store.Availability, error) {
	return nil, errors.New("not yet implemented")
}

func (qr *queryResolver) IsCleanerAvailable(ctx context.Context, input gen.CheckAvailabilityInput) (bool, error) {
	return false, errors.New("not yet implemented")
}

// MUTATION RESOLVERS
func (mr *mutationResolver) CreateAvailability(ctx context.Context, input gen.CreateAvailabilityInput) (*store.Availability, error) {
	return nil, errors.New("not yet implemented")
}

func (mr *mutationResolver) UpdateAvailability(ctx context.Context, input gen.UpdateAvailabilityInput) (*store.Availability, error) {
	return nil, errors.New("not yet implemented")
}

func (mr *mutationResolver) DeleteAvailability(ctx context.Context, id string) (*scalar.Void, error) {
	return nil, errors.New("not yet implemented")
}

func (mr *mutationResolver) BulkCreateAvailability(ctx context.Context, inputs []*gen.CreateAvailabilityInput) ([]*store.Availability, error) {
	return nil, errors.New("not yet implemented")
}

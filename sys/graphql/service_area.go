package graphql

import (
	"context"
	"errors"

	"cleanbuddy-api/res/store"
	"cleanbuddy-api/sys/graphql/gen"
	"cleanbuddy-api/sys/graphql/scalar"
	"cleanbuddy-api/sys/http/middleware"

	"github.com/google/uuid"
)

// QUERY RESOLVERS

func (qr *queryResolver) ServiceArea(ctx context.Context, id string) (*store.ServiceArea, error) {
	area, err := qr.Store.ServiceAreas().Get(ctx, id)
	if err != nil {
		qr.Logger.Printf("Error retrieving service area: %s", err)
		return nil, errors.New("service area not found")
	}
	return area, nil
}

func (qr *queryResolver) ServiceAreasByCleanerProfile(ctx context.Context, cleanerProfileID string) ([]*store.ServiceArea, error) {
	areas, err := qr.Store.ServiceAreas().GetByCleanerProfile(ctx, cleanerProfileID)
	if err != nil {
		qr.Logger.Printf("Error retrieving service areas: %s", err)
		return nil, errors.New("error retrieving service areas")
	}
	return areas, nil
}

func (qr *queryResolver) MyServiceAreas(ctx context.Context) ([]*store.ServiceArea, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Get cleaner profile
	profile, err := qr.Store.CleanerProfiles().GetByUserID(ctx, currentUser.ID)
	if err != nil {
		return nil, errors.New("cleaner profile not found")
	}

	areas, err := qr.Store.ServiceAreas().GetByCleanerProfile(ctx, profile.ID)
	if err != nil {
		qr.Logger.Printf("Error retrieving service areas: %s", err)
		return nil, errors.New("error retrieving service areas")
	}

	return areas, nil
}

func (qr *queryResolver) CleanersInArea(ctx context.Context, city, neighborhood string) ([]*store.CleanerProfile, error) {
	cleaners, err := qr.Store.ServiceAreas().FindCleanersInArea(ctx, city, neighborhood)
	if err != nil {
		qr.Logger.Printf("Error finding cleaners in area: %s", err)
		return nil, errors.New("error finding cleaners")
	}
	return cleaners, nil
}

func (qr *queryResolver) CleanersByPostalCode(ctx context.Context, postalCode string) ([]*store.CleanerProfile, error) {
	cleaners, err := qr.Store.ServiceAreas().FindCleanersByPostalCode(ctx, postalCode)
	if err != nil {
		qr.Logger.Printf("Error finding cleaners by postal code: %s", err)
		return nil, errors.New("error finding cleaners")
	}
	return cleaners, nil
}

// MUTATION RESOLVERS

func (mr *mutationResolver) AddServiceArea(ctx context.Context, input gen.CreateServiceAreaInput) (*store.ServiceArea, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Get cleaner profile
	profile, err := mr.Store.CleanerProfiles().GetByUserID(ctx, currentUser.ID)
	if err != nil {
		return nil, errors.New("cleaner profile not found")
	}

	area := &store.ServiceArea{
		ID:               uuid.New().String(),
		CleanerProfileID: profile.ID,
		City:             input.City,
		Neighborhood:     input.Neighborhood,
		PostalCode:       input.PostalCode,
	}

	if input.TravelFee != nil {
		area.TravelFee = *input.TravelFee
	}
	if input.IsPreferred != nil {
		area.IsPreferred = *input.IsPreferred
	}

	if err := mr.Store.ServiceAreas().Create(ctx, area); err != nil {
		mr.Logger.Printf("Error creating service area: %s", err)
		return nil, errors.New("error creating service area")
	}

	return area, nil
}

func (mr *mutationResolver) UpdateServiceArea(ctx context.Context, input gen.UpdateServiceAreaInput) (*store.ServiceArea, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Get service area
	area, err := mr.Store.ServiceAreas().Get(ctx, input.ID)
	if err != nil {
		return nil, errors.New("service area not found")
	}

	// Verify ownership
	profile, err := mr.Store.CleanerProfiles().GetByUserID(ctx, currentUser.ID)
	if err != nil || profile.ID != area.CleanerProfileID {
		return nil, errors.New("access forbidden")
	}

	// Update fields
	if input.City != nil {
		area.City = *input.City
	}
	if input.Neighborhood != nil {
		area.Neighborhood = *input.Neighborhood
	}
	if input.PostalCode != nil {
		area.PostalCode = *input.PostalCode
	}
	if input.TravelFee != nil {
		area.TravelFee = *input.TravelFee
	}
	if input.IsPreferred != nil {
		area.IsPreferred = *input.IsPreferred
	}

	if err := mr.Store.ServiceAreas().Update(ctx, area); err != nil {
		mr.Logger.Printf("Error updating service area: %s", err)
		return nil, errors.New("error updating service area")
	}

	return area, nil
}

func (mr *mutationResolver) DeleteServiceArea(ctx context.Context, id string) (*scalar.Void, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Get service area
	area, err := mr.Store.ServiceAreas().Get(ctx, id)
	if err != nil {
		return nil, errors.New("service area not found")
	}

	// Verify ownership
	profile, err := mr.Store.CleanerProfiles().GetByUserID(ctx, currentUser.ID)
	if err != nil || profile.ID != area.CleanerProfileID {
		return nil, errors.New("access forbidden")
	}

	if err := mr.Store.ServiceAreas().Delete(ctx, id); err != nil {
		mr.Logger.Printf("Error deleting service area: %s", err)
		return nil, errors.New("error deleting service area")
	}

	return &scalar.Void{}, nil
}

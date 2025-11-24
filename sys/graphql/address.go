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

func (qr *queryResolver) Address(ctx context.Context, id string) (*store.Address, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	address, err := qr.Store.Addresses().Get(ctx, id)
	if err != nil {
		qr.Logger.Printf("Error retrieving address: %s", err)
		return nil, errors.New("address not found")
	}

	// Ensure user owns this address
	if address.UserID != currentUser.ID {
		return nil, errors.New("access forbidden")
	}

	return address, nil
}

func (qr *queryResolver) MyAddresses(ctx context.Context) ([]*store.Address, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	addresses, err := qr.Store.Addresses().GetByUser(ctx, currentUser.ID)
	if err != nil {
		qr.Logger.Printf("Error retrieving addresses: %s", err)
		return nil, errors.New("error retrieving addresses")
	}

	return addresses, nil
}

func (qr *queryResolver) MyDefaultAddress(ctx context.Context) (*store.Address, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	address, err := qr.Store.Addresses().GetDefaultByUser(ctx, currentUser.ID)
	if err != nil {
		// No default address is not an error, return nil
		return nil, nil
	}

	return address, nil
}

// MUTATION RESOLVERS

func (mr *mutationResolver) CreateAddress(ctx context.Context, input gen.CreateAddressInput) (*store.Address, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Validate Google Place ID is provided
	if input.GooglePlaceID == "" {
		return nil, errors.New("Google Place ID is required")
	}

	// Create address from Google Maps API data
	address := &store.Address{
		ID:                 uuid.New().String(),
		UserID:             currentUser.ID,
		Street:             input.Street,
		City:               input.City,
		PostalCode:         input.PostalCode,
		Country:            input.Country,
		Latitude:           &input.Latitude,
		Longitude:          &input.Longitude,
		GooglePlaceID:      &input.GooglePlaceID,
	}

	// Optional fields from user
	if input.Label != nil {
		address.Label = *input.Label
	}
	if input.Building != nil {
		address.Building = *input.Building
	}
	if input.Apartment != nil {
		address.Apartment = *input.Apartment
	}
	if input.Floor != nil {
		address.Floor = input.Floor
	}
	if input.Neighborhood != nil {
		address.Neighborhood = *input.Neighborhood
	}
	if input.AccessInstructions != nil {
		address.AccessInstructions = *input.AccessInstructions
	}
	if input.County != nil {
		address.County = *input.County
	}

	// Set as default if requested or if it's the first address
	if input.IsDefault != nil && *input.IsDefault {
		address.IsDefault = true
	} else {
		// Check if user has any addresses
		existingAddresses, _ := mr.Store.Addresses().GetByUser(ctx, currentUser.ID)
		if len(existingAddresses) == 0 {
			address.IsDefault = true
		}
	}

	if err := mr.Store.Addresses().Create(ctx, address); err != nil {
		mr.Logger.Printf("Error creating address: %s", err)
		return nil, errors.New("error creating address")
	}

	// If set as default, unset other defaults
	if address.IsDefault {
		if err := mr.Store.Addresses().SetDefault(ctx, address.ID, currentUser.ID); err != nil {
			mr.Logger.Printf("Error setting default address: %s", err)
		}
	}

	return address, nil
}

func (mr *mutationResolver) UpdateAddress(ctx context.Context, input gen.UpdateAddressInput) (*store.Address, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Get existing address
	address, err := mr.Store.Addresses().Get(ctx, input.ID)
	if err != nil {
		return nil, errors.New("address not found")
	}

	// Ensure user owns this address
	if address.UserID != currentUser.ID {
		return nil, errors.New("access forbidden")
	}

	// Update only user-provided fields (not Google Maps fields)
	if input.Label != nil {
		address.Label = *input.Label
	}
	if input.Building != nil {
		address.Building = *input.Building
	}
	if input.Apartment != nil {
		address.Apartment = *input.Apartment
	}
	if input.Floor != nil {
		address.Floor = input.Floor
	}
	if input.Neighborhood != nil {
		address.Neighborhood = *input.Neighborhood
	}
	if input.AccessInstructions != nil {
		address.AccessInstructions = *input.AccessInstructions
	}

	if err := mr.Store.Addresses().Update(ctx, address); err != nil {
		mr.Logger.Printf("Error updating address: %s", err)
		return nil, errors.New("error updating address")
	}

	return address, nil
}

func (mr *mutationResolver) DeleteAddress(ctx context.Context, id string) (*scalar.Void, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Get address
	address, err := mr.Store.Addresses().Get(ctx, id)
	if err != nil {
		return nil, errors.New("address not found")
	}

	// Ensure user owns this address
	if address.UserID != currentUser.ID {
		return nil, errors.New("access forbidden")
	}

	if err := mr.Store.Addresses().Delete(ctx, id); err != nil {
		mr.Logger.Printf("Error deleting address: %s", err)
		return nil, errors.New("error deleting address")
	}

	return &scalar.Void{}, nil
}

func (mr *mutationResolver) SetDefaultAddress(ctx context.Context, id string) (*store.Address, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Get address
	address, err := mr.Store.Addresses().Get(ctx, id)
	if err != nil {
		return nil, errors.New("address not found")
	}

	// Ensure user owns this address
	if address.UserID != currentUser.ID {
		return nil, errors.New("access forbidden")
	}

	if err := mr.Store.Addresses().SetDefault(ctx, id, currentUser.ID); err != nil {
		mr.Logger.Printf("Error setting default address: %s", err)
		return nil, errors.New("error setting default address")
	}

	// Fetch updated address
	address, err = mr.Store.Addresses().Get(ctx, id)
	if err != nil {
		return nil, errors.New("error fetching updated address")
	}

	return address, nil
}

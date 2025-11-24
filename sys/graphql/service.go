package graphql

import (
	"context"
	"errors"
	"os"
	"strconv"

	"cleanbuddy-api/res/store"
	"cleanbuddy-api/sys/graphql/gen"
)

// QUERY RESOLVERS

func (qr *queryResolver) ServiceDefinition(ctx context.Context, serviceType store.ServiceType) (*store.ServiceDefinition, error) {
	service, err := qr.Store.Services().GetServiceDefinition(ctx, serviceType)
	if err != nil {
		qr.Logger.Printf("Error retrieving service definition: %s", err)
		return nil, errors.New("service definition not found")
	}
	return service, nil
}

func (qr *queryResolver) ServiceDefinitions(ctx context.Context, activeOnly *bool) ([]*store.ServiceDefinition, error) {
	active := true
	if activeOnly != nil {
		active = *activeOnly
	}

	services, err := qr.Store.Services().ListServiceDefinitions(ctx, active)
	if err != nil {
		qr.Logger.Printf("Error listing service definitions: %s", err)
		return nil, errors.New("error retrieving services")
	}
	return services, nil
}

func (qr *queryResolver) AddOnDefinition(ctx context.Context, addOn store.ServiceAddOn) (*store.ServiceAddOnDefinition, error) {
	addOnDef, err := qr.Store.Services().GetAddOnDefinition(ctx, addOn)
	if err != nil {
		qr.Logger.Printf("Error retrieving add-on definition: %s", err)
		return nil, errors.New("add-on definition not found")
	}
	return addOnDef, nil
}

func (qr *queryResolver) AddOnDefinitions(ctx context.Context, activeOnly *bool) ([]*store.ServiceAddOnDefinition, error) {
	active := true
	if activeOnly != nil {
		active = *activeOnly
	}

	addOns, err := qr.Store.Services().ListAddOnDefinitions(ctx, active)
	if err != nil {
		qr.Logger.Printf("Error listing add-on definitions: %s", err)
		return nil, errors.New("error retrieving add-ons")
	}
	return addOns, nil
}

func (qr *queryResolver) CalculateServicePrice(ctx context.Context, input gen.CalculateServicePriceInput) (*gen.ServicePriceCalculation, error) {
	// 1. Get cleaner profile and hourly rate
	cleanerProfile, err := qr.Store.CleanerProfiles().Get(ctx, input.CleanerProfileID)
	if err != nil {
		qr.Logger.Printf("Error retrieving cleaner profile: %s", err)
		return nil, errors.New("cleaner profile not found")
	}

	if !cleanerProfile.IsActive {
		return nil, errors.New("cleaner is not currently accepting bookings")
	}

	// 2. Get service definition
	serviceDefinition, err := qr.Store.Services().GetServiceDefinition(ctx, input.ServiceType)
	if err != nil {
		qr.Logger.Printf("Error retrieving service definition: %s", err)
		return nil, errors.New("service definition not found")
	}

	if !serviceDefinition.IsActive {
		return nil, errors.New("service is not currently available")
	}

	// 3. Get customer address to determine travel fee
	address, err := qr.Store.Addresses().Get(ctx, input.AddressID)
	if err != nil {
		qr.Logger.Printf("Error retrieving address: %s", err)
		return nil, errors.New("address not found")
	}

	// 4. Find service area and travel fee
	travelFee := 0
	serviceAreas, err := qr.Store.ServiceAreas().GetByCleanerProfile(ctx, cleanerProfile.ID)
	if err == nil && len(serviceAreas) > 0 {
		// Find matching service area by city and optionally postal code
		for _, area := range serviceAreas {
			if area.City == address.City {
				// Exact match with postal code if available
				if address.PostalCode != "" && area.PostalCode != "" && area.PostalCode == address.PostalCode {
					travelFee = area.TravelFee
					break
				}
				// Match by city and neighborhood
				if address.Neighborhood != "" && area.Neighborhood != "" && area.Neighborhood == address.Neighborhood {
					travelFee = area.TravelFee
					break
				}
				// Fallback to city match
				if area.PostalCode == "" && area.Neighborhood == "" {
					travelFee = area.TravelFee
				}
			}
		}
	}

	// 5. Calculate base service price
	// Price = (Base Hours * Hourly Rate * Price Multiplier)
	baseHours := serviceDefinition.BaseHours
	hourlyRate := cleanerProfile.HourlyRate
	priceMultiplier := serviceDefinition.PriceMultiplier

	servicePrice := int(float64(hourlyRate) * baseHours * priceMultiplier)

	// 6. Calculate add-ons price
	addOnsPrice := 0
	totalAddOnHours := 0.0
	if len(input.AddOns) > 0 {
		addOnDefs, err := qr.Store.Services().ListAddOnDefinitions(ctx, true)
		if err != nil {
			qr.Logger.Printf("Error retrieving add-on definitions: %s", err)
		} else {
			// Create map for quick lookup
			addOnMap := make(map[store.ServiceAddOn]*store.ServiceAddOnDefinition)
			for _, def := range addOnDefs {
				addOnMap[def.AddOn] = def
			}

			// Sum up add-on prices and hours
			for _, addOn := range input.AddOns {
				if def, exists := addOnMap[addOn]; exists {
					addOnsPrice += def.FixedPrice
					totalAddOnHours += def.EstimatedHours
				}
			}
		}
	}

	// Total estimated duration including add-ons
	totalEstimatedHours := baseHours + totalAddOnHours

	// 7. Calculate platform fee (percentage from ENV, default 15%)
	platformFeePercentage := 15.0 // default
	if feeEnv := os.Getenv("PLATFORM_FEE_PERCENTAGE"); feeEnv != "" {
		if fee, err := strconv.ParseFloat(feeEnv, 64); err == nil {
			platformFeePercentage = fee
		}
	}

	subtotal := servicePrice + addOnsPrice + travelFee
	platformFee := int(float64(subtotal) * platformFeePercentage / 100.0)
	totalPrice := subtotal + platformFee

	// 8. Calculate cleaner payout (subtotal minus platform fee)
	cleanerPayout := subtotal - platformFee

	return &gen.ServicePriceCalculation{
		ServicePrice:      servicePrice,
		AddOnsPrice:       addOnsPrice,
		TravelFee:         travelFee,
		PlatformFee:       platformFee,
		TotalPrice:        totalPrice,
		CleanerPayout:     cleanerPayout,
		EstimatedDuration: totalEstimatedHours,
	}, nil
}

// MUTATION RESOLVERS (Admin only)

func (mr *mutationResolver) CreateServiceDefinition(ctx context.Context, input gen.CreateServiceDefinitionInput) (*store.ServiceDefinition, error) {
	// TODO: Implement admin-only service definition creation
	return nil, errors.New("not yet implemented")
}

func (mr *mutationResolver) UpdateServiceDefinition(ctx context.Context, input gen.UpdateServiceDefinitionInput) (*store.ServiceDefinition, error) {
	// TODO: Implement admin-only service definition update
	return nil, errors.New("not yet implemented")
}

func (mr *mutationResolver) CreateAddOnDefinition(ctx context.Context, input gen.CreateAddOnDefinitionInput) (*store.ServiceAddOnDefinition, error) {
	// TODO: Implement admin-only add-on definition creation
	return nil, errors.New("not yet implemented")
}

func (mr *mutationResolver) UpdateAddOnDefinition(ctx context.Context, input gen.UpdateAddOnDefinitionInput) (*store.ServiceAddOnDefinition, error) {
	// TODO: Implement admin-only add-on definition update
	return nil, errors.New("not yet implemented")
}

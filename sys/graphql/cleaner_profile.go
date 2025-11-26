package graphql

import (
	"context"
	"errors"
	"time"

	"cleanbuddy-api/res/store"
	"cleanbuddy-api/sys/graphql/gen"
	"cleanbuddy-api/sys/graphql/scalar"
	"cleanbuddy-api/sys/http/middleware"

	"github.com/google/uuid"
)

// FIELD RESOLVERS

type cleanerProfileResolver struct{ *Resolver }

func (r *Resolver) CleanerProfile() gen.CleanerProfileResolver {
	return &cleanerProfileResolver{r}
}

func (cpr *cleanerProfileResolver) User(ctx context.Context, profile *store.CleanerProfile) (*store.User, error) {
	user, err := cpr.Store.Users().Get(ctx, profile.UserID)
	if err != nil {
		cpr.Logger.Printf("Error retrieving user for cleaner profile: %s", err)
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (cpr *cleanerProfileResolver) ServiceAreas(ctx context.Context, profile *store.CleanerProfile) ([]*store.ServiceArea, error) {
	areas, err := cpr.Store.ServiceAreas().GetByCleanerProfile(ctx, profile.ID)
	if err != nil {
		cpr.Logger.Printf("Error retrieving service areas: %s", err)
		return nil, errors.New("error retrieving service areas")
	}
	return areas, nil
}

func (cpr *cleanerProfileResolver) Reviews(ctx context.Context, profile *store.CleanerProfile) ([]*store.Review, error) {
	// Get approved reviews only for display
	statusApproved := store.ReviewStatusApproved
	filters := store.ReviewFilters{
		Status:  &statusApproved,
		OrderBy: "created_at DESC",
		Limit:   10, // Limit to recent 10 reviews
	}

	reviews, err := cpr.Store.Reviews().GetByCleanerProfile(ctx, profile.ID, filters)
	if err != nil {
		cpr.Logger.Printf("Error retrieving reviews: %s", err)
		return nil, errors.New("error retrieving reviews")
	}
	return reviews, nil
}

func (cpr *cleanerProfileResolver) Availability(ctx context.Context, profile *store.CleanerProfile) ([]*store.Availability, error) {
	filters := store.AvailabilityFilters{
		OrderBy: "date ASC, start_time ASC",
		Limit:   50, // Limit to next 50 availability entries
	}

	availability, err := cpr.Store.Availability().GetByCleanerProfile(ctx, profile.ID, filters)
	if err != nil {
		cpr.Logger.Printf("Error retrieving availability: %s", err)
		return nil, errors.New("error retrieving availability")
	}
	return availability, nil
}

func (cpr *cleanerProfileResolver) Company(ctx context.Context, profile *store.CleanerProfile) (*store.Company, error) {
	if profile.CompanyID == nil {
		// Try to find by admin_user_id for individual cleaners (fallback)
		company, err := cpr.Store.Companies().GetByAdminUserID(ctx, profile.UserID)
		if err != nil {
			return nil, nil // No company found
		}
		return company, nil
	}

	company, err := cpr.Store.Companies().Get(ctx, *profile.CompanyID)
	if err != nil {
		cpr.Logger.Printf("Error retrieving company for cleaner: %s", err)
		return nil, nil
	}
	return company, nil
}

// QUERY RESOLVERS

func (qr *queryResolver) CleanerProfile(ctx context.Context, id string) (*store.CleanerProfile, error) {
	profile, err := qr.Store.CleanerProfiles().Get(ctx, id)
	if err != nil {
		qr.Logger.Printf("Error retrieving cleaner profile: %s", err)
		return nil, errors.New("cleaner profile not found")
	}
	return profile, nil
}

func (qr *queryResolver) CleanerProfileByUserID(ctx context.Context, userID string) (*store.CleanerProfile, error) {
	profile, err := qr.Store.CleanerProfiles().GetByUserID(ctx, userID)
	if err != nil {
		qr.Logger.Printf("Error retrieving cleaner profile by user ID: %s", err)
		return nil, errors.New("cleaner profile not found")
	}
	return profile, nil
}

func (qr *queryResolver) MyCleanerProfile(ctx context.Context) (*store.CleanerProfile, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Check if user is a cleaner
	if !currentUser.IsCleaner() {
		return nil, errors.New("user is not a cleaner")
	}

	profile, err := qr.Store.CleanerProfiles().GetByUserID(ctx, currentUser.ID)
	if err != nil {
		qr.Logger.Printf("Error retrieving cleaner profile: %s", err)
		return nil, errors.New("cleaner profile not found")
	}

	return profile, nil
}

func (qr *queryResolver) SearchCleaners(ctx context.Context, filters *gen.CleanerProfileFiltersInput, limit *int, offset *int, orderBy *string) (*gen.CleanerProfileConnection, error) {
	// Build filters
	storeFilters := store.CleanerProfileFilters{}

	if filters != nil {
		storeFilters.Tier = filters.Tier
		storeFilters.MinRating = filters.MinRating
		storeFilters.MaxRating = filters.MaxRating
		storeFilters.IsActive = filters.IsActive
		storeFilters.IsVerified = filters.IsVerified
		storeFilters.IsAvailableToday = filters.IsAvailableToday
		storeFilters.ServiceAreaIDs = filters.ServiceAreaIds

		// Location-based filtering
		if filters.City != nil || filters.Neighborhood != nil || filters.PostalCode != nil {
			// Get cleaners by location first
			var cleaners []*store.CleanerProfile
			var err error

			if filters.PostalCode != nil {
				cleaners, err = qr.Store.ServiceAreas().FindCleanersByPostalCode(ctx, *filters.PostalCode)
			} else if filters.City != nil {
				neighborhood := ""
				if filters.Neighborhood != nil {
					neighborhood = *filters.Neighborhood
				}
				cleaners, err = qr.Store.ServiceAreas().FindCleanersInArea(ctx, *filters.City, neighborhood)
			}

			if err != nil {
				qr.Logger.Printf("Error finding cleaners by location: %s", err)
				return nil, errors.New("error searching cleaners")
			}

			// Extract cleaner profile IDs
			var profileIDs []string
			for _, c := range cleaners {
				profileIDs = append(profileIDs, c.ID)
			}
			storeFilters.ServiceAreaIDs = profileIDs
		}
	}

	if limit != nil {
		storeFilters.Limit = *limit
	} else {
		storeFilters.Limit = 20
	}

	if offset != nil {
		storeFilters.Offset = *offset
	}

	if orderBy != nil {
		storeFilters.OrderBy = *orderBy
	}

	profiles, err := qr.Store.CleanerProfiles().List(ctx, storeFilters)
	if err != nil {
		qr.Logger.Printf("Error listing cleaner profiles: %s", err)
		return nil, errors.New("error searching cleaners")
	}

	// Build connection
	edges := make([]*gen.CleanerProfileEdge, len(profiles))
	for i, profile := range profiles {
		edges[i] = &gen.CleanerProfileEdge{
			Node:   profile,
			Cursor: profile.ID,
		}
	}

	return &gen.CleanerProfileConnection{
		Edges:      edges,
		TotalCount: len(profiles),
	}, nil
}

func (qr *queryResolver) AvailableCleaners(ctx context.Context, date time.Time, startTime string, duration float64, city string, neighborhood, postalCode *string, filters *gen.CleanerProfileFiltersInput) ([]*store.CleanerProfile, error) {
	// TODO: Implement complex availability checking
	// For now, return basic search results
	return nil, errors.New("not yet implemented")
}

// MUTATION RESOLVERS

func (mr *mutationResolver) CreateCleanerProfile(ctx context.Context, input gen.CreateCleanerProfileInput) (*store.CleanerProfile, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Check if user is a cleaner or cleaner admin (cleaner admin can also create their own profile)
	if !currentUser.IsCleaner() && !currentUser.IsCleanerAdmin() {
		return nil, errors.New("user must have cleaner or cleaner admin role to create a profile")
	}

	// Check if profile already exists
	existing, _ := mr.Store.CleanerProfiles().GetByUserID(ctx, currentUser.ID)
	if existing != nil {
		return nil, errors.New("cleaner profile already exists for this user")
	}

	// Create profile
	profile := &store.CleanerProfile{
		ID:         uuid.New().String(),
		UserID:     currentUser.ID,
		Tier:       store.CleanerTierNew,
		IsActive:   true,
		IsVerified: false,
	}

	// Check if user was invited to a company
	var inviteToDelete *store.CleanerInvite
	invite, err := mr.Store.CleanerInvites().GetByAcceptedUserID(ctx, currentUser.ID)
	if err == nil && invite != nil {
		// User was invited - link to that company
		profile.CompanyID = &invite.CompanyID
		inviteToDelete = invite
	} else {
		// Original logic: Link cleaner to their company (created during application approval)
		company, err := mr.Store.Companies().GetByAdminUserID(ctx, currentUser.ID)
		if err == nil && company != nil {
			profile.CompanyID = &company.ID
		}
	}

	if input.Bio != nil {
		profile.Bio = *input.Bio
	}

	if input.ProfilePicture != nil {
		profile.ProfilePicture = input.ProfilePicture
	}

	if err := mr.Store.CleanerProfiles().Create(ctx, profile); err != nil {
		mr.Logger.Printf("Error creating cleaner profile: %s", err)
		return nil, errors.New("error creating cleaner profile")
	}

	// Create service areas
	for _, areaInput := range input.ServiceAreaInputs {
		area := &store.ServiceArea{
			ID:               uuid.New().String(),
			CleanerProfileID: profile.ID,
			City:             areaInput.City,
			Neighborhood:     areaInput.Neighborhood,
			PostalCode:       areaInput.PostalCode,
		}

		if areaInput.TravelFee != nil {
			area.TravelFee = *areaInput.TravelFee
		}
		if areaInput.IsPreferred != nil {
			area.IsPreferred = *areaInput.IsPreferred
		}

		if err := mr.Store.ServiceAreas().Create(ctx, area); err != nil {
			mr.Logger.Printf("Error creating service area: %s", err)
			// Continue creating other areas
		}
	}

	// Delete the invite after profile is successfully created
	if inviteToDelete != nil {
		if err := mr.Store.CleanerInvites().Delete(ctx, inviteToDelete.ID); err != nil {
			mr.Logger.Printf("Error deleting accepted invite: %s", err)
			// Don't fail the profile creation if invite deletion fails
		}
	}

	return profile, nil
}

func (mr *mutationResolver) UpdateCleanerProfile(ctx context.Context, input gen.UpdateCleanerProfileInput) (*store.CleanerProfile, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Get existing profile
	profile, err := mr.Store.CleanerProfiles().GetByUserID(ctx, currentUser.ID)
	if err != nil {
		return nil, errors.New("cleaner profile not found")
	}

	// Update fields
	if input.Bio != nil {
		profile.Bio = *input.Bio
	}
	if input.ProfilePicture != nil {
		profile.ProfilePicture = input.ProfilePicture
	}
	if input.IsActive != nil {
		profile.IsActive = *input.IsActive
	}
	if input.IsAvailableToday != nil {
		profile.IsAvailableToday = *input.IsAvailableToday
	}

	if err := mr.Store.CleanerProfiles().Update(ctx, profile); err != nil {
		mr.Logger.Printf("Error updating cleaner profile: %s", err)
		return nil, errors.New("error updating cleaner profile")
	}

	return profile, nil
}

func (mr *mutationResolver) DeleteCleanerProfile(ctx context.Context) (*scalar.Void, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Get existing profile
	profile, err := mr.Store.CleanerProfiles().GetByUserID(ctx, currentUser.ID)
	if err != nil {
		return nil, errors.New("cleaner profile not found")
	}

	if err := mr.Store.CleanerProfiles().Delete(ctx, profile.ID); err != nil {
		mr.Logger.Printf("Error deleting cleaner profile: %s", err)
		return nil, errors.New("error deleting cleaner profile")
	}

	return &scalar.Void{}, nil
}

func (mr *mutationResolver) UpdateCleanerTier(ctx context.Context, profileID string, tier store.CleanerTier) (*store.CleanerProfile, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Only global admins can update tiers
	if !currentUser.IsGlobalAdmin() {
		return nil, errors.New("access forbidden, admin privileges required")
	}

	// Get profile
	profile, err := mr.Store.CleanerProfiles().Get(ctx, profileID)
	if err != nil {
		return nil, errors.New("cleaner profile not found")
	}

	// Update tier
	if err := mr.Store.CleanerProfiles().UpdateTier(ctx, profileID, tier); err != nil {
		mr.Logger.Printf("Error updating cleaner tier: %s", err)
		return nil, errors.New("error updating cleaner tier")
	}

	// Fetch updated profile
	profile, err = mr.Store.CleanerProfiles().Get(ctx, profileID)
	if err != nil {
		return nil, errors.New("error fetching updated profile")
	}

	return profile, nil
}

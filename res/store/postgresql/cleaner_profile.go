package postgresql

import (
	"context"
	"fmt"

	"cleanbuddy-api/res/store"
)

type cleanerProfileStore struct {
	*storeImpl
}

func NewCleanerProfileStore(rootStore *storeImpl) *cleanerProfileStore {
	return &cleanerProfileStore{storeImpl: rootStore}
}

// MUTATIONS

func (cps *cleanerProfileStore) Create(ctx context.Context, profile *store.CleanerProfile) error {
	// Validate rate is within tier range
	if !profile.IsRateValidForTier() {
		minRate := store.GetMinRateForTier(profile.Tier)
		maxRate := store.GetMaxRateForTier(profile.Tier)
		return fmt.Errorf("hourly rate %d is outside valid range for tier %s (%d-%d bani)",
			profile.HourlyRate, profile.Tier, minRate, maxRate)
	}

	result := cps.db.WithContext(ctx).Create(profile)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("failed to create cleaner profile")
	}
	return nil
}

func (cps *cleanerProfileStore) Get(ctx context.Context, id string) (*store.CleanerProfile, error) {
	var profile store.CleanerProfile
	result := cps.db.WithContext(ctx).Where("id = ?", id).First(&profile)
	if result.Error != nil {
		return nil, result.Error
	}
	return &profile, nil
}

func (cps *cleanerProfileStore) GetByUserID(ctx context.Context, userID string) (*store.CleanerProfile, error) {
	var profile store.CleanerProfile
	result := cps.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile)
	if result.Error != nil {
		return nil, result.Error
	}
	return &profile, nil
}

func (cps *cleanerProfileStore) Update(ctx context.Context, profile *store.CleanerProfile) error {
	// Validate rate is within tier range if rate changed
	if !profile.IsRateValidForTier() {
		minRate := store.GetMinRateForTier(profile.Tier)
		maxRate := store.GetMaxRateForTier(profile.Tier)
		return fmt.Errorf("hourly rate %d is outside valid range for tier %s (%d-%d bani)",
			profile.HourlyRate, profile.Tier, minRate, maxRate)
	}

	result := cps.db.WithContext(ctx).Save(profile)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("cleaner profile not found (id: %s)", profile.ID)
	}
	return nil
}

func (cps *cleanerProfileStore) Delete(ctx context.Context, id string) error {
	result := cps.db.WithContext(ctx).Delete(&store.CleanerProfile{ID: id})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("cleaner profile not found (id: %s)", id)
	}
	return nil
}

func (cps *cleanerProfileStore) List(ctx context.Context, filters store.CleanerProfileFilters) ([]*store.CleanerProfile, error) {
	query := cps.db.WithContext(ctx).Model(&store.CleanerProfile{})

	// Apply filters
	if filters.Tier != nil {
		query = query.Where("tier = ?", *filters.Tier)
	}
	if filters.MinRating != nil {
		query = query.Where("average_rating >= ?", *filters.MinRating)
	}
	if filters.MaxRating != nil {
		query = query.Where("average_rating <= ?", *filters.MaxRating)
	}
	if filters.MinHourlyRate != nil {
		query = query.Where("hourly_rate >= ?", *filters.MinHourlyRate)
	}
	if filters.MaxHourlyRate != nil {
		query = query.Where("hourly_rate <= ?", *filters.MaxHourlyRate)
	}
	if filters.IsActive != nil {
		query = query.Where("is_active = ?", *filters.IsActive)
	}
	if filters.IsVerified != nil {
		query = query.Where("is_verified = ?", *filters.IsVerified)
	}
	if filters.IsAvailableToday != nil {
		query = query.Where("is_available_today = ?", *filters.IsAvailableToday)
	}

	// Filter by service areas if provided
	if len(filters.ServiceAreaIDs) > 0 {
		query = query.Joins("INNER JOIN service_areas ON service_areas.cleaner_profile_id = cleaner_profiles.id").
			Where("service_areas.id IN ?", filters.ServiceAreaIDs).
			Distinct()
	}

	// Apply ordering
	if filters.OrderBy != "" {
		query = query.Order(filters.OrderBy)
	} else {
		query = query.Order("average_rating DESC, created_at DESC")
	}

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	var profiles []*store.CleanerProfile
	if err := query.Find(&profiles).Error; err != nil {
		return nil, err
	}

	return profiles, nil
}

func (cps *cleanerProfileStore) UpdateStats(ctx context.Context, profileID string, stats store.CleanerStats) error {
	updates := make(map[string]interface{})

	if stats.TotalBookings != nil {
		updates["total_bookings"] = *stats.TotalBookings
	}
	if stats.CompletedBookings != nil {
		updates["completed_bookings"] = *stats.CompletedBookings
	}
	if stats.CancelledBookings != nil {
		updates["cancelled_bookings"] = *stats.CancelledBookings
	}
	if stats.AverageRating != nil {
		updates["average_rating"] = *stats.AverageRating
	}
	if stats.TotalReviews != nil {
		updates["total_reviews"] = *stats.TotalReviews
	}
	if stats.TotalEarnings != nil {
		updates["total_earnings"] = *stats.TotalEarnings
	}

	if len(updates) == 0 {
		return fmt.Errorf("no stats to update")
	}

	result := cps.db.WithContext(ctx).Model(&store.CleanerProfile{}).
		Where("id = ?", profileID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("cleaner profile not found (id: %s)", profileID)
	}

	return nil
}

func (cps *cleanerProfileStore) UpdateTier(ctx context.Context, profileID string, newTier store.CleanerTier) error {
	result := cps.db.WithContext(ctx).Model(&store.CleanerProfile{}).
		Where("id = ?", profileID).
		Update("tier", newTier)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("cleaner profile not found (id: %s)", profileID)
	}

	return nil
}

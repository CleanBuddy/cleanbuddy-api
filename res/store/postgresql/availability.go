package postgresql

import (
	"context"
	"fmt"
	"time"

	"cleanbuddy-api/res/store"

	"gorm.io/gorm"
)

type availabilityStore struct {
	*storeImpl
}

func NewAvailabilityStore(rootStore *storeImpl) *availabilityStore {
	return &availabilityStore{storeImpl: rootStore}
}

func (as *availabilityStore) Create(ctx context.Context, availability *store.Availability) error {
	result := as.db.WithContext(ctx).Create(availability)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("failed to create availability")
	}
	return nil
}

func (as *availabilityStore) Get(ctx context.Context, id string) (*store.Availability, error) {
	var availability store.Availability
	result := as.db.WithContext(ctx).Where("id = ?", id).First(&availability)
	if result.Error != nil {
		return nil, result.Error
	}
	return &availability, nil
}

func (as *availabilityStore) Update(ctx context.Context, availability *store.Availability) error {
	result := as.db.WithContext(ctx).Save(availability)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("availability not found (id: %s)", availability.ID)
	}
	return nil
}

func (as *availabilityStore) Delete(ctx context.Context, id string) error {
	result := as.db.WithContext(ctx).Delete(&store.Availability{ID: id})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("availability not found (id: %s)", id)
	}
	return nil
}

func (as *availabilityStore) GetByCleanerProfile(ctx context.Context, cleanerProfileID string, filters store.AvailabilityFilters) ([]*store.Availability, error) {
	query := as.db.WithContext(ctx).Where("cleaner_profile_id = ?", cleanerProfileID)
	query = as.applyFilters(query, filters)

	var availabilities []*store.Availability
	if err := query.Find(&availabilities).Error; err != nil {
		return nil, err
	}
	return availabilities, nil
}

func (as *availabilityStore) GetByDateRange(ctx context.Context, cleanerProfileID string, startDate, endDate time.Time) ([]*store.Availability, error) {
	var availabilities []*store.Availability
	err := as.db.WithContext(ctx).
		Where("cleaner_profile_id = ?", cleanerProfileID).
		Where("date >= ? AND date <= ?", startDate, endDate).
		Order("date ASC, start_time ASC").
		Find(&availabilities).Error

	if err != nil {
		return nil, err
	}
	return availabilities, nil
}

func (as *availabilityStore) IsCleanerAvailable(ctx context.Context, cleanerProfileID string, date time.Time, startTime, endTime string) (bool, error) {
	// Check for unavailable entries that overlap
	var count int64
	err := as.db.WithContext(ctx).
		Model(&store.Availability{}).
		Where("cleaner_profile_id = ?", cleanerProfileID).
		Where("type = ?", store.AvailabilityTypeUnavailable).
		Where("date = ?", date).
		Where("NOT (end_time <= ? OR start_time >= ?)", startTime, endTime).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	// If there are any unavailable entries that overlap, cleaner is not available
	return count == 0, nil
}

func (as *availabilityStore) GetAvailableCleaners(ctx context.Context, date time.Time, startTime, endTime string, serviceAreaIDs []string) ([]*store.CleanerProfile, error) {
	var profiles []*store.CleanerProfile

	query := as.db.WithContext(ctx).
		Model(&store.CleanerProfile{}).
		Where("is_active = ?", true)

	// Filter by service areas if provided
	if len(serviceAreaIDs) > 0 {
		query = query.Joins("INNER JOIN service_areas ON service_areas.cleaner_profile_id = cleaner_profiles.id").
			Where("service_areas.id IN ?", serviceAreaIDs)
	}

	// Exclude cleaners who have unavailable entries for this time slot
	query = query.Where(`
		NOT EXISTS (
			SELECT 1 FROM availabilities
			WHERE availabilities.cleaner_profile_id = cleaner_profiles.id
			AND availabilities.type = ?
			AND availabilities.date = ?
			AND NOT (availabilities.end_time <= ? OR availabilities.start_time >= ?)
		)
	`, store.AvailabilityTypeUnavailable, date, startTime, endTime)

	// Exclude cleaners who have confirmed bookings at this time
	query = query.Where(`
		NOT EXISTS (
			SELECT 1 FROM bookings
			WHERE bookings.cleaner_id = cleaner_profiles.user_id
			AND bookings.scheduled_date = ?
			AND bookings.status IN ?
			AND NOT (
				TIME(bookings.scheduled_time) + (bookings.duration || ' hours')::interval <= ?::time
				OR TIME(bookings.scheduled_time) >= ?::time
			)
		)
	`, date, []store.BookingStatus{
		store.BookingStatusConfirmed,
		store.BookingStatusInProgress,
	}, startTime, endTime)

	query = query.Distinct().Order("average_rating DESC")

	if err := query.Find(&profiles).Error; err != nil {
		return nil, err
	}

	return profiles, nil
}

// Helper method to apply filters
func (as *availabilityStore) applyFilters(query *gorm.DB, filters store.AvailabilityFilters) *gorm.DB {
	if filters.Type != nil {
		query = query.Where("type = ?", *filters.Type)
	}
	if filters.StartDate != nil {
		query = query.Where("date >= ?", *filters.StartDate)
	}
	if filters.EndDate != nil {
		query = query.Where("date <= ?", *filters.EndDate)
	}

	if filters.OrderBy != "" {
		query = query.Order(filters.OrderBy)
	} else {
		query = query.Order("date ASC, start_time ASC")
	}

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	return query
}

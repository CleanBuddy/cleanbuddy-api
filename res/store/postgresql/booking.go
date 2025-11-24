package postgresql

import (
	"gorm.io/gorm"

	"context"
	"fmt"
	"time"

	"cleanbuddy-api/res/store"
)

type bookingStore struct {
	*storeImpl
}

func NewBookingStore(rootStore *storeImpl) *bookingStore {
	return &bookingStore{storeImpl: rootStore}
}

func (bs *bookingStore) Create(ctx context.Context, booking *store.Booking) error {
	result := bs.db.WithContext(ctx).Create(booking)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("failed to create booking")
	}
	return nil
}

func (bs *bookingStore) Get(ctx context.Context, id string) (*store.Booking, error) {
	var booking store.Booking
	result := bs.db.WithContext(ctx).Where("id = ?", id).First(&booking)
	if result.Error != nil {
		return nil, result.Error
	}
	return &booking, nil
}

func (bs *bookingStore) Update(ctx context.Context, booking *store.Booking) error {
	result := bs.db.WithContext(ctx).Save(booking)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("booking not found (id: %s)", booking.ID)
	}
	return nil
}

func (bs *bookingStore) Delete(ctx context.Context, id string) error {
	result := bs.db.WithContext(ctx).Delete(&store.Booking{ID: id})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("booking not found (id: %s)", id)
	}
	return nil
}

func (bs *bookingStore) GetByCustomer(ctx context.Context, customerID string, filters store.BookingFilters) ([]*store.Booking, error) {
	query := bs.db.WithContext(ctx).Where("customer_id = ?", customerID)
	query = bs.applyFilters(query, filters)

	var bookings []*store.Booking
	if err := query.Find(&bookings).Error; err != nil {
		return nil, err
	}
	return bookings, nil
}

func (bs *bookingStore) GetByCleaner(ctx context.Context, cleanerID string, filters store.BookingFilters) ([]*store.Booking, error) {
	query := bs.db.WithContext(ctx).Where("cleaner_id = ?", cleanerID)
	query = bs.applyFilters(query, filters)

	var bookings []*store.Booking
	if err := query.Find(&bookings).Error; err != nil {
		return nil, err
	}
	return bookings, nil
}

func (bs *bookingStore) UpdateStatus(ctx context.Context, bookingID string, status store.BookingStatus) error {
	result := bs.db.WithContext(ctx).Model(&store.Booking{}).
		Where("id = ?", bookingID).
		Update("status", status)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("booking not found (id: %s)", bookingID)
	}
	return nil
}

func (bs *bookingStore) CancelBooking(ctx context.Context, bookingID string, cancelledBy string, reason store.CancellationReason, note string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":              store.BookingStatusCancelled,
		"cancellation_reason": reason,
		"cancellation_note":   note,
		"cancelled_by_id":     cancelledBy,
		"cancelled_at":        now,
	}

	result := bs.db.WithContext(ctx).Model(&store.Booking{}).
		Where("id = ?", bookingID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("booking not found (id: %s)", bookingID)
	}
	return nil
}

func (bs *bookingStore) GetUpcoming(ctx context.Context, userID string, limit int) ([]*store.Booking, error) {
	now := time.Now()
	var bookings []*store.Booking

	query := bs.db.WithContext(ctx).
		Where("(customer_id = ? OR cleaner_id = ?)", userID, userID).
		Where("scheduled_date >= ?", now).
		Where("status IN ?", []store.BookingStatus{
			store.BookingStatusPending,
			store.BookingStatusConfirmed,
			store.BookingStatusInProgress,
		}).
		Order("scheduled_date ASC, scheduled_time ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&bookings).Error; err != nil {
		return nil, err
	}
	return bookings, nil
}

func (bs *bookingStore) GetByDateRange(ctx context.Context, cleanerID string, startDate, endDate time.Time) ([]*store.Booking, error) {
	var bookings []*store.Booking

	err := bs.db.WithContext(ctx).
		Where("cleaner_id = ?", cleanerID).
		Where("scheduled_date >= ? AND scheduled_date <= ?", startDate, endDate).
		Order("scheduled_date ASC, scheduled_time ASC").
		Find(&bookings).Error

	if err != nil {
		return nil, err
	}
	return bookings, nil
}

func (bs *bookingStore) ListAll(ctx context.Context, filters store.BookingFilters) ([]*store.Booking, error) {
	query := bs.db.WithContext(ctx)
	query = bs.applyFilters(query, filters)

	var bookings []*store.Booking
	if err := query.Find(&bookings).Error; err != nil {
		return nil, err
	}
	return bookings, nil
}

// Helper method to apply filters
func (bs *bookingStore) applyFilters(query *gorm.DB, filters store.BookingFilters) *gorm.DB {
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}
	if filters.ServiceType != nil {
		query = query.Where("service_type = ?", *filters.ServiceType)
	}
	if filters.StartDate != nil {
		query = query.Where("scheduled_date >= ?", *filters.StartDate)
	}
	if filters.EndDate != nil {
		query = query.Where("scheduled_date <= ?", *filters.EndDate)
	}
	if filters.MinPrice != nil {
		query = query.Where("total_price >= ?", *filters.MinPrice)
	}
	if filters.MaxPrice != nil {
		query = query.Where("total_price <= ?", *filters.MaxPrice)
	}
	if filters.IsRecurring != nil {
		query = query.Where("is_recurring = ?", *filters.IsRecurring)
	}

	if filters.OrderBy != "" {
		query = query.Order(filters.OrderBy)
	} else {
		query = query.Order("scheduled_date DESC, created_at DESC")
	}

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	return query
}

package postgresql

import (
	"gorm.io/gorm"

	"context"
	"fmt"
	"time"

	"cleanbuddy-api/res/store"
)

type reviewStore struct {
	*storeImpl
}

func NewReviewStore(rootStore *storeImpl) *reviewStore {
	return &reviewStore{storeImpl: rootStore}
}

func (rs *reviewStore) Create(ctx context.Context, review *store.Review) error {
	result := rs.db.WithContext(ctx).Create(review)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("failed to create review")
	}
	return nil
}

func (rs *reviewStore) Get(ctx context.Context, id string) (*store.Review, error) {
	var review store.Review
	result := rs.db.WithContext(ctx).Where("id = ?", id).First(&review)
	if result.Error != nil {
		return nil, result.Error
	}
	return &review, nil
}

func (rs *reviewStore) GetByBooking(ctx context.Context, bookingID string) (*store.Review, error) {
	var review store.Review
	result := rs.db.WithContext(ctx).Where("booking_id = ?", bookingID).First(&review)
	if result.Error != nil {
		return nil, result.Error
	}
	return &review, nil
}

func (rs *reviewStore) Update(ctx context.Context, review *store.Review) error {
	result := rs.db.WithContext(ctx).Save(review)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("review not found (id: %s)", review.ID)
	}
	return nil
}

func (rs *reviewStore) Delete(ctx context.Context, id string) error {
	result := rs.db.WithContext(ctx).Delete(&store.Review{ID: id})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("review not found (id: %s)", id)
	}
	return nil
}

func (rs *reviewStore) GetByCleanerProfile(ctx context.Context, cleanerProfileID string, filters store.ReviewFilters) ([]*store.Review, error) {
	query := rs.db.WithContext(ctx).Where("cleaner_profile_id = ?", cleanerProfileID)
	query = rs.applyFilters(query, filters)

	var reviews []*store.Review
	if err := query.Find(&reviews).Error; err != nil {
		return nil, err
	}
	return reviews, nil
}

func (rs *reviewStore) GetByCustomer(ctx context.Context, customerID string, filters store.ReviewFilters) ([]*store.Review, error) {
	query := rs.db.WithContext(ctx).Where("customer_id = ?", customerID)
	query = rs.applyFilters(query, filters)

	var reviews []*store.Review
	if err := query.Find(&reviews).Error; err != nil {
		return nil, err
	}
	return reviews, nil
}

func (rs *reviewStore) GetPendingModeration(ctx context.Context, limit int) ([]*store.Review, error) {
	query := rs.db.WithContext(ctx).
		Where("status = ?", store.ReviewStatusPending).
		Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	var reviews []*store.Review
	if err := query.Find(&reviews).Error; err != nil {
		return nil, err
	}
	return reviews, nil
}

func (rs *reviewStore) UpdateStatus(ctx context.Context, reviewID string, status store.ReviewStatus, moderatorID string, note string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":          status,
		"moderation_note": note,
		"moderated_by_id": moderatorID,
		"moderated_at":    now,
	}

	result := rs.db.WithContext(ctx).Model(&store.Review{}).
		Where("id = ?", reviewID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("review not found (id: %s)", reviewID)
	}
	return nil
}

func (rs *reviewStore) FlagReview(ctx context.Context, reviewID string, reason string) error {
	result := rs.db.WithContext(ctx).Model(&store.Review{}).
		Where("id = ?", reviewID).
		Updates(map[string]interface{}{
			"status":      store.ReviewStatusFlagged,
			"flag_reason": reason,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("review not found (id: %s)", reviewID)
	}
	return nil
}

func (rs *reviewStore) AddCleanerResponse(ctx context.Context, reviewID string, response string) error {
	now := time.Now()
	result := rs.db.WithContext(ctx).Model(&store.Review{}).
		Where("id = ?", reviewID).
		Updates(map[string]interface{}{
			"cleaner_response": response,
			"responded_at":     now,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("review not found (id: %s)", reviewID)
	}
	return nil
}

func (rs *reviewStore) UpdateHelpfulness(ctx context.Context, reviewID string, helpful bool) error {
	var field string
	if helpful {
		field = "helpful_count"
	} else {
		field = "not_helpful_count"
	}

	result := rs.db.WithContext(ctx).Model(&store.Review{}).
		Where("id = ?", reviewID).
		UpdateColumn(field, gorm.Expr(field+" + 1"))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("review not found (id: %s)", reviewID)
	}
	return nil
}

func (rs *reviewStore) GetAverageRatingForCleaner(ctx context.Context, cleanerProfileID string) (float64, int, error) {
	var result struct {
		AverageRating float64
		Count         int
	}

	err := rs.db.WithContext(ctx).
		Model(&store.Review{}).
		Select("AVG(rating) as average_rating, COUNT(*) as count").
		Where("cleaner_profile_id = ? AND status = ?", cleanerProfileID, store.ReviewStatusApproved).
		Scan(&result).Error

	if err != nil {
		return 0, 0, err
	}

	return result.AverageRating, result.Count, nil
}

// Helper method to apply filters
func (rs *reviewStore) applyFilters(query *gorm.DB, filters store.ReviewFilters) *gorm.DB {
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}
	if filters.MinRating != nil {
		query = query.Where("rating >= ?", *filters.MinRating)
	}
	if filters.MaxRating != nil {
		query = query.Where("rating <= ?", *filters.MaxRating)
	}
	if filters.HasComment != nil {
		if *filters.HasComment {
			query = query.Where("comment != ''")
		} else {
			query = query.Where("comment = ''")
		}
	}

	if filters.OrderBy != "" {
		query = query.Order(filters.OrderBy)
	} else {
		query = query.Order("created_at DESC")
	}

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	return query
}

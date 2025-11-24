package store

import (
	"context"
	"time"
)

// ReviewStatus represents the moderation status of a review
type ReviewStatus string

const (
	ReviewStatusPending  ReviewStatus = "pending"  // Awaiting moderation
	ReviewStatusApproved ReviewStatus = "approved" // Approved and visible
	ReviewStatusRejected ReviewStatus = "rejected" // Rejected by moderator
	ReviewStatusFlagged  ReviewStatus = "flagged"  // Flagged for review
)

// Review represents a customer review for a cleaner after a booking
type Review struct {
	ID               string          `gorm:"primaryKey;size:50;unique"`
	Booking          *Booking        `gorm:"foreignKey:BookingID"`
	BookingID        string          `gorm:"size:50;not null;unique;index:idx_review_booking"`
	Customer         *User           `gorm:"foreignKey:CustomerID"`
	CustomerID       string          `gorm:"size:50;not null;index:idx_review_customer"`
	Cleaner          *User           `gorm:"foreignKey:CleanerID"`
	CleanerID        string          `gorm:"size:50;not null;index:idx_review_cleaner"`
	CleanerProfile   *CleanerProfile `gorm:"foreignKey:CleanerProfileID"`
	CleanerProfileID string          `gorm:"size:50;not null"`

	// Rating (1-5 stars)
	Rating          int     `gorm:"not null;check:rating >= 1 AND rating <= 5"`

	// Review Content
	Title           string  `gorm:"size:200"` // Optional review title
	Comment         string  `gorm:"type:text"` // Review text

	// Detailed Ratings (optional breakdown)
	QualityRating       *int `gorm:"check:quality_rating >= 1 AND quality_rating <= 5"`
	PunctualityRating   *int `gorm:"check:punctuality_rating >= 1 AND punctuality_rating <= 5"`
	ProfessionalismRating *int `gorm:"check:professionalism_rating >= 1 AND professionalism_rating <= 5"`
	ValueRating         *int `gorm:"check:value_rating >= 1 AND value_rating <= 5"`

	// Moderation
	Status           ReviewStatus `gorm:"size:20;not null;default:'pending';index:idx_review_status"`
	FlagReason       string       `gorm:"type:text"` // Reason if flagged
	ModerationNote   string       `gorm:"type:text"` // Admin notes
	ModeratedBy      *User        `gorm:"foreignKey:ModeratedByID"`
	ModeratedByID    *string      `gorm:"size:50"`
	ModeratedAt      *time.Time

	// Response from Cleaner (optional feature)
	CleanerResponse  string     `gorm:"type:text"`
	RespondedAt      *time.Time

	// Helpfulness tracking
	HelpfulCount     int `gorm:"not null;default:0"` // How many users found this helpful
	NotHelpfulCount  int `gorm:"not null;default:0"`

	CreatedAt time.Time `gorm:"autoCreateTime;not null;index:idx_review_created"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;not null"`
}

// ReviewStore defines the data access interface for reviews
type ReviewStore interface {
	// Create creates a new review
	Create(ctx context.Context, review *Review) error

	// Get retrieves a review by ID
	Get(ctx context.Context, id string) (*Review, error)

	// GetByBooking retrieves a review for a specific booking
	GetByBooking(ctx context.Context, bookingID string) (*Review, error)

	// Update updates a review
	Update(ctx context.Context, review *Review) error

	// Delete deletes a review
	Delete(ctx context.Context, id string) error

	// GetByCleanerProfile retrieves all reviews for a cleaner profile
	GetByCleanerProfile(ctx context.Context, cleanerProfileID string, filters ReviewFilters) ([]*Review, error)

	// GetByCustomer retrieves all reviews written by a customer
	GetByCustomer(ctx context.Context, customerID string, filters ReviewFilters) ([]*Review, error)

	// GetPendingModeration retrieves reviews pending moderation
	GetPendingModeration(ctx context.Context, limit int) ([]*Review, error)

	// UpdateStatus updates the moderation status of a review
	UpdateStatus(ctx context.Context, reviewID string, status ReviewStatus, moderatorID string, note string) error

	// FlagReview flags a review for moderation
	FlagReview(ctx context.Context, reviewID string, reason string) error

	// AddCleanerResponse adds a cleaner's response to a review
	AddCleanerResponse(ctx context.Context, reviewID string, response string) error

	// UpdateHelpfulness updates the helpfulness counters
	UpdateHelpfulness(ctx context.Context, reviewID string, helpful bool) error

	// GetAverageRatingForCleaner calculates the average rating for a cleaner
	GetAverageRatingForCleaner(ctx context.Context, cleanerProfileID string) (float64, int, error) // returns (average, count, error)
}

// ReviewFilters contains filter options for listing reviews
type ReviewFilters struct {
	Status      *ReviewStatus
	MinRating   *int
	MaxRating   *int
	HasComment  *bool
	Limit       int
	Offset      int
	OrderBy     string // e.g., "created_at DESC", "rating DESC"
}

package store

import (
	"context"
	"time"
)

// CleanerTier represents the tier level of a cleaner
type CleanerTier string

const (
	CleanerTierNew      CleanerTier = "new"
	CleanerTierStandard CleanerTier = "standard"
	CleanerTierPremium  CleanerTier = "premium"
	CleanerTierPro      CleanerTier = "pro"
)

// CleanerProfile represents the extended profile for users with cleaner role
type CleanerProfile struct {
	ID        string   `gorm:"primaryKey;size:50;unique"`
	User      *User    `gorm:"foreignKey:UserID"`
	UserID    string   `gorm:"size:50;not null;unique;index:idx_cleaner_profile_user"`
	Company   *Company `gorm:"foreignKey:CompanyID"`
	CompanyID *string  `gorm:"size:50;index:idx_cleaner_profiles_company"`

	// Profile Information
	Bio            string  `gorm:"type:text"`
	ProfilePicture *string `gorm:"size:512"` // URL to profile picture

	// Tier and Performance
	Tier              CleanerTier `gorm:"size:20;not null;default:'new';index:idx_cleaner_tier"`
	TotalBookings     int         `gorm:"not null;default:0"`
	CompletedBookings   int         `gorm:"not null;default:0"`
	CancelledBookings   int         `gorm:"not null;default:0"`
	AverageRating       float64     `gorm:"type:decimal(3,2);default:0.00"` // 0.00 to 5.00
	TotalReviews        int         `gorm:"not null;default:0"`
	TotalEarnings       int64       `gorm:"not null;default:0"` // Total earnings in bani

	// Availability
	IsActive         bool `gorm:"not null;default:true"`  // Can receive new bookings
	IsAvailableToday bool `gorm:"not null;default:false"` // Quick filter for same-day bookings

	// Verification
	IsVerified       bool       `gorm:"not null;default:false"`
	VerifiedAt       *time.Time
	BackgroundCheck  bool       `gorm:"not null;default:false"`
	IdentityVerified bool       `gorm:"not null;default:false"`

	CreatedAt time.Time `gorm:"autoCreateTime;not null;index:idx_cleaner_created"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;not null"`
}

// CleanerProfileStore defines the data access interface for cleaner profiles
type CleanerProfileStore interface {
	// Create creates a new cleaner profile
	Create(ctx context.Context, profile *CleanerProfile) error

	// Get retrieves a cleaner profile by ID
	Get(ctx context.Context, id string) (*CleanerProfile, error)

	// GetByUserID retrieves a cleaner profile by user ID
	GetByUserID(ctx context.Context, userID string) (*CleanerProfile, error)

	// Update updates a cleaner profile
	Update(ctx context.Context, profile *CleanerProfile) error

	// Delete deletes a cleaner profile
	Delete(ctx context.Context, id string) error

	// List retrieves cleaner profiles with filters
	List(ctx context.Context, filters CleanerProfileFilters) ([]*CleanerProfile, error)

	// UpdateStats updates the performance statistics of a cleaner
	UpdateStats(ctx context.Context, profileID string, stats CleanerStats) error

	// UpdateTier updates the tier of a cleaner
	UpdateTier(ctx context.Context, profileID string, newTier CleanerTier) error
}

// CleanerProfileFilters contains filter options for listing cleaner profiles
type CleanerProfileFilters struct {
	Tier             *CleanerTier
	MinRating        *float64
	MaxRating        *float64
	IsActive         *bool
	IsVerified       *bool
	IsAvailableToday *bool
	ServiceAreaIDs   []string // Filter by service areas
	CompanyID        *string  // Filter by company
	Limit            int
	Offset           int
	OrderBy          string // e.g., "average_rating DESC"
}

// CleanerStats represents statistics to update for a cleaner
type CleanerStats struct {
	TotalBookings     *int
	CompletedBookings *int
	CancelledBookings *int
	AverageRating     *float64
	TotalReviews      *int
	TotalEarnings     *int64
}

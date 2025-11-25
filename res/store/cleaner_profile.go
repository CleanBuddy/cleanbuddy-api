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
	Tier                CleanerTier `gorm:"size:20;not null;default:'new';index:idx_cleaner_tier"`
	HourlyRate          int         `gorm:"not null"` // Rate in bani (smallest RON unit, 1 RON = 100 bani)
	TotalBookings       int         `gorm:"not null;default:0"`
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

// GetMinRateForTier returns the minimum hourly rate for a tier (in bani)
func GetMinRateForTier(tier CleanerTier) int {
	switch tier {
	case CleanerTierNew:
		return 4000 // 40 RON
	case CleanerTierStandard:
		return 5000 // 50 RON
	case CleanerTierPremium:
		return 7000 // 70 RON
	case CleanerTierPro:
		return 10000 // 100 RON
	default:
		return 4000
	}
}

// GetMaxRateForTier returns the maximum hourly rate for a tier (in bani)
func GetMaxRateForTier(tier CleanerTier) int {
	switch tier {
	case CleanerTierNew:
		return 5000 // 50 RON
	case CleanerTierStandard:
		return 7000 // 70 RON
	case CleanerTierPremium:
		return 10000 // 100 RON
	case CleanerTierPro:
		return 15000 // 150 RON
	default:
		return 5000
	}
}

// IsRateValidForTier checks if a rate is within the valid range for a tier
func (cp *CleanerProfile) IsRateValidForTier() bool {
	minRate := GetMinRateForTier(cp.Tier)
	maxRate := GetMaxRateForTier(cp.Tier)
	return cp.HourlyRate >= minRate && cp.HourlyRate <= maxRate
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
	Tier              *CleanerTier
	MinRating         *float64
	MaxRating         *float64
	MinHourlyRate     *int
	MaxHourlyRate     *int
	IsActive          *bool
	IsVerified        *bool
	IsAvailableToday  *bool
	ServiceAreaIDs    []string // Filter by service areas
	CompanyID         *string  // Filter by company
	Limit             int
	Offset            int
	OrderBy           string // e.g., "average_rating DESC", "hourly_rate ASC"
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

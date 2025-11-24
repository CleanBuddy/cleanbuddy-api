package store

import (
	"context"
	"time"
)

// AvailabilityType represents the type of availability entry
type AvailabilityType string

const (
	AvailabilityTypeAvailable   AvailabilityType = "available"   // Cleaner is available
	AvailabilityTypeUnavailable AvailabilityType = "unavailable" // Cleaner is blocked/unavailable
)

// RecurrencePattern represents how often availability repeats
type RecurrencePattern string

const (
	RecurrencePatternNone   RecurrencePattern = "none"   // Single day only
	RecurrencePatternWeekly RecurrencePattern = "weekly" // Repeats every week
)

// Availability represents a cleaner's availability schedule
type Availability struct {
	ID               string          `gorm:"primaryKey;size:50;unique"`
	CleanerProfile   *CleanerProfile `gorm:"foreignKey:CleanerProfileID"`
	CleanerProfileID string          `gorm:"size:50;not null;index:idx_availability_cleaner"`

	// Type
	Type AvailabilityType `gorm:"size:20;not null"`

	// Date and Time
	Date      time.Time `gorm:"not null;index:idx_availability_date"` // Specific date
	StartTime string    `gorm:"size:10;not null"` // e.g., "09:00"
	EndTime   string    `gorm:"size:10;not null"` // e.g., "17:00"

	// Recurrence
	IsRecurring       bool              `gorm:"not null;default:false"`
	RecurrencePattern RecurrencePattern `gorm:"size:20"`
	RecurrenceEnd     *time.Time        // When recurrence ends (null = indefinite)

	// Notes
	Notes string `gorm:"type:text"` // e.g., "Vacation", "Personal appointment"

	CreatedAt time.Time `gorm:"autoCreateTime;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;not null"`
}

// AvailabilityStore defines the data access interface for availability
type AvailabilityStore interface {
	// Create creates a new availability entry
	Create(ctx context.Context, availability *Availability) error

	// Get retrieves an availability entry by ID
	Get(ctx context.Context, id string) (*Availability, error)

	// Update updates an availability entry
	Update(ctx context.Context, availability *Availability) error

	// Delete deletes an availability entry
	Delete(ctx context.Context, id string) error

	// GetByCleanerProfile retrieves all availability entries for a cleaner
	GetByCleanerProfile(ctx context.Context, cleanerProfileID string, filters AvailabilityFilters) ([]*Availability, error)

	// GetByDateRange retrieves availability entries within a date range
	GetByDateRange(ctx context.Context, cleanerProfileID string, startDate, endDate time.Time) ([]*Availability, error)

	// IsCleanerAvailable checks if a cleaner is available at a specific date and time
	IsCleanerAvailable(ctx context.Context, cleanerProfileID string, date time.Time, startTime, endTime string) (bool, error)

	// GetAvailableCleaners finds cleaners available at a specific date and time
	GetAvailableCleaners(ctx context.Context, date time.Time, startTime, endTime string, serviceAreaIDs []string) ([]*CleanerProfile, error)
}

// AvailabilityFilters contains filter options for listing availability
type AvailabilityFilters struct {
	Type      *AvailabilityType
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
	Offset    int
	OrderBy   string // e.g., "date ASC"
}

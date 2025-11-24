package store

import (
	"context"
	"time"
)

// ServiceArea represents a geographic area where a cleaner provides services
type ServiceArea struct {
	ID               string          `gorm:"primaryKey;size:50;unique"`
	CleanerProfile   *CleanerProfile `gorm:"foreignKey:CleanerProfileID"`
	CleanerProfileID string          `gorm:"size:50;not null;index:idx_service_area_cleaner"`

	// Location Information
	City         string `gorm:"size:100;not null;index:idx_service_area_city"`
	Neighborhood string `gorm:"size:100;not null"`
	PostalCode   string `gorm:"size:20;index:idx_service_area_postal"`

	// Travel Settings
	TravelFee int  `gorm:"not null;default:0"` // Travel fee in bani
	IsPreferred bool `gorm:"not null;default:false"` // Preferred area (cleaner lives here or nearby)

	CreatedAt time.Time `gorm:"autoCreateTime;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;not null"`
}

// ServiceAreaStore defines the data access interface for service areas
type ServiceAreaStore interface {
	// Create creates a new service area
	Create(ctx context.Context, area *ServiceArea) error

	// Get retrieves a service area by ID
	Get(ctx context.Context, id string) (*ServiceArea, error)

	// GetByCleanerProfile retrieves all service areas for a cleaner profile
	GetByCleanerProfile(ctx context.Context, cleanerProfileID string) ([]*ServiceArea, error)

	// Update updates a service area
	Update(ctx context.Context, area *ServiceArea) error

	// Delete deletes a service area
	Delete(ctx context.Context, id string) error

	// FindCleanersInArea finds all cleaners serving a specific area
	FindCleanersInArea(ctx context.Context, city, neighborhood string) ([]*CleanerProfile, error)

	// FindCleanersByPostalCode finds all cleaners serving a specific postal code
	FindCleanersByPostalCode(ctx context.Context, postalCode string) ([]*CleanerProfile, error)
}

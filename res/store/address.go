package store

import (
	"context"
	"time"
)

// Address represents a physical address for bookings or user profiles
type Address struct {
	ID     string `gorm:"primaryKey;size:50;unique"`
	User   *User  `gorm:"foreignKey:UserID"`
	UserID string `gorm:"size:50;not null;index:idx_address_user"`

	// Address Fields
	Label        string  `gorm:"size:100"`            // e.g., "Home", "Office"
	Street       string  `gorm:"size:200;not null"`
	Building     string  `gorm:"size:50"`             // Building number or name
	Apartment    string  `gorm:"size:20"`             // Apartment/Unit number
	Floor        *int    `gorm:""`                    // Floor number
	City         string  `gorm:"size:100;not null;index:idx_address_city"`
	Neighborhood string  `gorm:"size:100"`
	PostalCode   string  `gorm:"size:20;not null"`
	County       string  `gorm:"size:100"`            // Romanian: Județ
	Country      string  `gorm:"size:100;not null;default:'Romania'"`

	// Additional Information
	AccessInstructions string `gorm:"type:text"` // e.g., "Ring bell 3 times", "Gate code: 1234"
	IsDefault          bool   `gorm:"not null;default:false;index:idx_address_default"`

	// Coordinates (from Google Maps Platform API)
	Latitude  *float64 `gorm:"type:decimal(10,8)"`
	Longitude *float64 `gorm:"type:decimal(11,8)"`

	// Google Maps Place ID (for verification and future lookups)
	GooglePlaceID *string `gorm:"size:256;index:idx_address_place_id"`

	CreatedAt time.Time `gorm:"autoCreateTime;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;not null"`
}

// AddressStore defines the data access interface for addresses
type AddressStore interface {
	// Create creates a new address
	Create(ctx context.Context, address *Address) error

	// Get retrieves an address by ID
	Get(ctx context.Context, id string) (*Address, error)

	// GetByUser retrieves all addresses for a user
	GetByUser(ctx context.Context, userID string) ([]*Address, error)

	// GetDefaultByUser retrieves the default address for a user
	GetDefaultByUser(ctx context.Context, userID string) (*Address, error)

	// Update updates an address
	Update(ctx context.Context, address *Address) error

	// Delete deletes an address
	Delete(ctx context.Context, id string) error

	// SetDefault sets an address as the default for a user (unsets other defaults)
	SetDefault(ctx context.Context, addressID, userID string) error
}

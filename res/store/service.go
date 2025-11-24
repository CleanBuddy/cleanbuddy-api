package store

import (
	"context"
	"time"
)

// ServiceType represents the type of cleaning service
type ServiceType string

const (
	ServiceTypeGeneral   ServiceType = "general"    // General cleaning
	ServiceTypeDeep      ServiceType = "deep"       // Deep cleaning
	ServiceTypeMoveInOut ServiceType = "move_in_out" // Move-in/Move-out cleaning
)

// ServiceFrequency represents how often a service is booked
type ServiceFrequency string

const (
	ServiceFrequencyOneTime   ServiceFrequency = "one_time"
	ServiceFrequencyWeekly    ServiceFrequency = "weekly"
	ServiceFrequencyBiMonthly ServiceFrequency = "bi_monthly" // 2 times per month
	ServiceFrequencyMonthly   ServiceFrequency = "monthly"
)

// ServiceAddOn represents additional services that can be added to a booking
type ServiceAddOn string

const (
	ServiceAddOnOven    ServiceAddOn = "oven"
	ServiceAddOnWindows ServiceAddOn = "windows"
	ServiceAddOnFridge  ServiceAddOn = "fridge"
	ServiceAddOnGarage  ServiceAddOn = "garage"
)

// ServiceDefinition represents the definition of a service with pricing
type ServiceDefinition struct {
	ID   string      `gorm:"primaryKey;size:50;unique"`
	Type ServiceType `gorm:"size:20;not null;unique;index:idx_service_type"`

	// Service Details
	Name        string `gorm:"size:100;not null"` // e.g., "General Cleaning", "Deep Cleaning"
	Description string `gorm:"type:text"`

	// Duration and Pricing Modifiers
	BaseHours        float64 `gorm:"not null"` // Base hours for the service
	PriceMultiplier  float64 `gorm:"not null;default:1.0"` // Multiplier for cleaner's hourly rate

	// Availability
	IsActive bool `gorm:"not null;default:true"`

	CreatedAt time.Time `gorm:"autoCreateTime;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;not null"`
}

// ServiceAddOnDefinition represents the definition of an add-on service
type ServiceAddOnDefinition struct {
	ID     string       `gorm:"primaryKey;size:50;unique"`
	AddOn  ServiceAddOn `gorm:"size:20;not null;unique;index:idx_addon_type"`

	// Add-On Details
	Name        string `gorm:"size:100;not null"` // e.g., "Oven Cleaning"
	Description string `gorm:"type:text"`

	// Pricing
	FixedPrice      int     `gorm:"not null"` // Fixed price in bani (can be 0 if time-based)
	EstimatedHours  float64 `gorm:"not null;default:0"` // Additional hours needed

	// Availability
	IsActive bool `gorm:"not null;default:true"`

	CreatedAt time.Time `gorm:"autoCreateTime;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;not null"`
}

// ServiceStore defines the data access interface for service definitions
type ServiceStore interface {
	// GetServiceDefinition retrieves a service definition by type
	GetServiceDefinition(ctx context.Context, serviceType ServiceType) (*ServiceDefinition, error)

	// ListServiceDefinitions retrieves all active service definitions
	ListServiceDefinitions(ctx context.Context, activeOnly bool) ([]*ServiceDefinition, error)

	// GetAddOnDefinition retrieves an add-on definition by type
	GetAddOnDefinition(ctx context.Context, addOn ServiceAddOn) (*ServiceAddOnDefinition, error)

	// ListAddOnDefinitions retrieves all active add-on definitions
	ListAddOnDefinitions(ctx context.Context, activeOnly bool) ([]*ServiceAddOnDefinition, error)

	// CreateServiceDefinition creates a new service definition
	CreateServiceDefinition(ctx context.Context, service *ServiceDefinition) error

	// UpdateServiceDefinition updates a service definition
	UpdateServiceDefinition(ctx context.Context, service *ServiceDefinition) error

	// CreateAddOnDefinition creates a new add-on definition
	CreateAddOnDefinition(ctx context.Context, addOn *ServiceAddOnDefinition) error

	// UpdateAddOnDefinition updates an add-on definition
	UpdateAddOnDefinition(ctx context.Context, addOn *ServiceAddOnDefinition) error
}

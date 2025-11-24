package store

import (
	"context"
	"time"
)

// BookingStatus represents the status of a booking
type BookingStatus string

const (
	BookingStatusPending    BookingStatus = "pending"     // Initial state, awaiting cleaner confirmation
	BookingStatusConfirmed  BookingStatus = "confirmed"   // Cleaner confirmed
	BookingStatusInProgress BookingStatus = "in_progress" // Service is being performed
	BookingStatusCompleted  BookingStatus = "completed"   // Service completed successfully
	BookingStatusCancelled  BookingStatus = "cancelled"   // Cancelled by customer or cleaner
	BookingStatusNoShow     BookingStatus = "no_show"     // Customer was not present
)

// CancellationReason represents why a booking was cancelled
type CancellationReason string

const (
	CancellationReasonCustomerRequest CancellationReason = "customer_request"
	CancellationReasonCleanerRequest  CancellationReason = "cleaner_request"
	CancellationReasonEmergency       CancellationReason = "emergency"
	CancellationReasonWeather         CancellationReason = "weather"
	CancellationReasonOther           CancellationReason = "other"
)

// Booking represents a service booking
type Booking struct {
	ID               string          `gorm:"primaryKey;size:50;unique"`
	Customer         *User           `gorm:"foreignKey:CustomerID"`
	CustomerID       string          `gorm:"size:50;not null;index:idx_booking_customer"`
	Cleaner          *User           `gorm:"foreignKey:CleanerID"`
	CleanerID        string          `gorm:"size:50;not null;index:idx_booking_cleaner"`
	CleanerProfile   *CleanerProfile `gorm:"foreignKey:CleanerProfileID"`
	CleanerProfileID string          `gorm:"size:50;not null"`

	// Service Details
	ServiceType      ServiceType      `gorm:"size:20;not null"`
	ServiceFrequency ServiceFrequency `gorm:"size:20;not null"`
	ServiceAddOns    string           `gorm:"type:text"` // JSON array of ServiceAddOn values

	// Scheduling
	ScheduledDate time.Time `gorm:"not null;index:idx_booking_date"`
	ScheduledTime string    `gorm:"size:10;not null"` // e.g., "14:00"
	Duration      float64   `gorm:"not null"`         // Duration in hours

	// Address
	Address   *Address `gorm:"foreignKey:AddressID"`
	AddressID string   `gorm:"size:50;not null"`

	// Pricing (stored at booking time to preserve historical pricing)
	CleanerHourlyRate  int   `gorm:"not null"` // Rate in bani at time of booking
	ServicePrice       int   `gorm:"not null"` // Total service price in bani
	AddOnsPrice        int   `gorm:"not null;default:0"` // Total add-ons price in bani
	TravelFee          int   `gorm:"not null;default:0"` // Travel fee in bani
	PlatformFee        int   `gorm:"not null"` // Platform fee in bani
	TotalPrice         int   `gorm:"not null"` // Total price charged to customer in bani
	CleanerPayout      int   `gorm:"not null"` // Amount cleaner receives in bani

	// Status and Progress
	Status             BookingStatus       `gorm:"size:20;not null;default:'pending';index:idx_booking_status"`
	CancellationReason *CancellationReason `gorm:"size:30"`
	CancellationNote   string              `gorm:"type:text"`
	CancelledBy        *User               `gorm:"foreignKey:CancelledByID"`
	CancelledByID      *string             `gorm:"size:50"`
	CancelledAt        *time.Time

	// Timestamps
	ConfirmedAt      *time.Time
	StartedAt        *time.Time // When cleaner marks as in progress
	CompletedAt      *time.Time

	// Recurring Booking Support
	IsRecurring      bool    `gorm:"not null;default:false"`
	ParentBookingID  *string `gorm:"size:50;index:idx_booking_parent"` // References parent for recurring bookings
	NextBookingID    *string `gorm:"size:50"` // References next booking in series

	// Special Instructions
	CustomerNotes string `gorm:"type:text"` // Customer's notes for the cleaner
	CleanerNotes  string `gorm:"type:text"` // Cleaner's notes after service

	CreatedAt time.Time `gorm:"autoCreateTime;not null;index:idx_booking_created"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;not null"`
}

// BookingStore defines the data access interface for bookings
type BookingStore interface {
	// Create creates a new booking
	Create(ctx context.Context, booking *Booking) error

	// Get retrieves a booking by ID
	Get(ctx context.Context, id string) (*Booking, error)

	// Update updates a booking
	Update(ctx context.Context, booking *Booking) error

	// Delete deletes a booking (soft delete recommended)
	Delete(ctx context.Context, id string) error

	// GetByCustomer retrieves all bookings for a customer
	GetByCustomer(ctx context.Context, customerID string, filters BookingFilters) ([]*Booking, error)

	// GetByCleaner retrieves all bookings for a cleaner
	GetByCleaner(ctx context.Context, cleanerID string, filters BookingFilters) ([]*Booking, error)

	// UpdateStatus updates the status of a booking
	UpdateStatus(ctx context.Context, bookingID string, status BookingStatus) error

	// CancelBooking cancels a booking with reason
	CancelBooking(ctx context.Context, bookingID string, cancelledBy string, reason CancellationReason, note string) error

	// GetUpcoming retrieves upcoming bookings for a user (customer or cleaner)
	GetUpcoming(ctx context.Context, userID string, limit int) ([]*Booking, error)

	// GetByDateRange retrieves bookings within a date range for a cleaner (for availability)
	GetByDateRange(ctx context.Context, cleanerID string, startDate, endDate time.Time) ([]*Booking, error)

	// ListAll retrieves all bookings with filters (for admin)
	ListAll(ctx context.Context, filters BookingFilters) ([]*Booking, error)
}

// BookingFilters contains filter options for listing bookings
type BookingFilters struct {
	Status        *BookingStatus
	ServiceType   *ServiceType
	StartDate     *time.Time
	EndDate       *time.Time
	MinPrice      *int
	MaxPrice      *int
	IsRecurring   *bool
	Limit         int
	Offset        int
	OrderBy       string // e.g., "scheduled_date DESC"
}

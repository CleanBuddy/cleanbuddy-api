package store

import (
	"context"
	"time"
)

// TransactionType represents the type of financial transaction
type TransactionType string

const (
	TransactionTypePayment TransactionType = "payment" // Customer payment
	TransactionTypePayout  TransactionType = "payout"  // Cleaner payout
	TransactionTypeRefund  TransactionType = "refund"  // Refund to customer
)

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"   // Transaction initiated
	TransactionStatusProcessing TransactionStatus = "processing" // Being processed
	TransactionStatusCompleted TransactionStatus = "completed" // Successfully completed
	TransactionStatusFailed    TransactionStatus = "failed"    // Failed
	TransactionStatusCancelled TransactionStatus = "cancelled" // Cancelled
)

// PaymentMethod represents how payment was made
type PaymentMethod string

const (
	PaymentMethodCard         PaymentMethod = "card"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodCash         PaymentMethod = "cash"
)

// Transaction represents a financial transaction in the system
type Transaction struct {
	ID        string            `gorm:"primaryKey;size:50;unique"`
	Type      TransactionType   `gorm:"size:20;not null;index:idx_transaction_type"`
	Status    TransactionStatus `gorm:"size:20;not null;index:idx_transaction_status"`

	// Related Entities
	Booking   *Booking `gorm:"foreignKey:BookingID"`
	BookingID *string  `gorm:"size:50;index:idx_transaction_booking"`

	// Payer (customer for payments, platform for payouts)
	Payer   *User  `gorm:"foreignKey:PayerID"`
	PayerID string `gorm:"size:50;not null;index:idx_transaction_payer"`

	// Payee (platform for payments, cleaner for payouts)
	Payee   *User  `gorm:"foreignKey:PayeeID"`
	PayeeID string `gorm:"size:50;not null;index:idx_transaction_payee"`

	// Amount Details (all in bani)
	Amount          int `gorm:"not null"` // Total transaction amount
	PlatformFee     int `gorm:"not null;default:0"` // Platform fee taken
	NetAmount       int `gorm:"not null"` // Amount after fees

	// Payment Details
	PaymentMethod      PaymentMethod `gorm:"size:30;not null"`
	Currency           string        `gorm:"size:10;not null;default:'RON'"`

	// External Payment Provider Data
	StripePaymentID    *string `gorm:"size:256;unique;index:idx_transaction_stripe"` // Stripe payment intent ID
	StripeTransferID   *string `gorm:"size:256;unique"` // Stripe transfer ID for payouts
	StripeRefundID     *string `gorm:"size:256;unique"` // Stripe refund ID

	// Metadata
	Description        string `gorm:"type:text"`
	Metadata           string `gorm:"type:text"` // JSON for additional data

	// Failure Information
	FailureReason      string `gorm:"type:text"`
	FailureCode        string `gorm:"size:100"`

	// Timestamps
	ProcessedAt time.Time  `gorm:"index:idx_transaction_processed"`
	CompletedAt *time.Time
	FailedAt    *time.Time

	CreatedAt time.Time `gorm:"autoCreateTime;not null;index:idx_transaction_created"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;not null"`
}

// PayoutBatch represents a batch of payouts to cleaners
type PayoutBatch struct {
	ID             string            `gorm:"primaryKey;size:50;unique"`
	Status         TransactionStatus `gorm:"size:20;not null"`

	// Batch Details
	TotalAmount    int       `gorm:"not null"` // Total amount in batch (in bani)
	TotalPayouts   int       `gorm:"not null"` // Number of payouts in batch
	PeriodStart    time.Time `gorm:"not null"` // Start of payout period
	PeriodEnd      time.Time `gorm:"not null"` // End of payout period

	// Processing
	InitiatedBy   *User   `gorm:"foreignKey:InitiatedByID"`
	InitiatedByID string  `gorm:"size:50;not null"`
	ProcessedAt   *time.Time
	CompletedAt   *time.Time

	// Metadata
	Notes          string `gorm:"type:text"`

	CreatedAt time.Time `gorm:"autoCreateTime;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;not null"`
}

// TransactionStore defines the data access interface for transactions
type TransactionStore interface {
	// Create creates a new transaction
	Create(ctx context.Context, transaction *Transaction) error

	// Get retrieves a transaction by ID
	Get(ctx context.Context, id string) (*Transaction, error)

	// GetByStripePaymentID retrieves a transaction by Stripe payment ID
	GetByStripePaymentID(ctx context.Context, stripePaymentID string) (*Transaction, error)

	// Update updates a transaction
	Update(ctx context.Context, transaction *Transaction) error

	// UpdateStatus updates the status of a transaction
	UpdateStatus(ctx context.Context, transactionID string, status TransactionStatus) error

	// GetByBooking retrieves all transactions for a booking
	GetByBooking(ctx context.Context, bookingID string) ([]*Transaction, error)

	// GetByUser retrieves all transactions for a user (as payer or payee)
	GetByUser(ctx context.Context, userID string, filters TransactionFilters) ([]*Transaction, error)

	// GetPayoutsDue retrieves transactions that are due for payout
	GetPayoutsDue(ctx context.Context, beforeDate time.Time) ([]*Transaction, error)

	// ListAll retrieves all transactions with filters (for admin)
	ListAll(ctx context.Context, filters TransactionFilters) ([]*Transaction, error)

	// CreatePayoutBatch creates a new payout batch
	CreatePayoutBatch(ctx context.Context, batch *PayoutBatch) error

	// GetPayoutBatch retrieves a payout batch by ID
	GetPayoutBatch(ctx context.Context, id string) (*PayoutBatch, error)

	// ListPayoutBatches lists all payout batches
	ListPayoutBatches(ctx context.Context, limit, offset int) ([]*PayoutBatch, error)

	// GetCleanerEarnings calculates total earnings for a cleaner
	GetCleanerEarnings(ctx context.Context, cleanerID string, startDate, endDate time.Time) (int64, error)
}

// TransactionFilters contains filter options for listing transactions
type TransactionFilters struct {
	Type          *TransactionType
	Status        *TransactionStatus
	PaymentMethod *PaymentMethod
	StartDate     *time.Time
	EndDate       *time.Time
	MinAmount     *int
	MaxAmount     *int
	Limit         int
	Offset        int
	OrderBy       string // e.g., "created_at DESC"
}

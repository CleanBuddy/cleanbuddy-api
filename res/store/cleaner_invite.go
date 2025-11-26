package store

import (
	"context"
	"time"
)

// CleanerInviteStatus represents the status of a cleaner invite
type CleanerInviteStatus string

const (
	CleanerInviteStatusPending  CleanerInviteStatus = "PENDING"
	CleanerInviteStatusAccepted CleanerInviteStatus = "ACCEPTED"
	CleanerInviteStatusExpired  CleanerInviteStatus = "EXPIRED"
	CleanerInviteStatusRevoked  CleanerInviteStatus = "REVOKED"
)

// CleanerInvite represents an invite for a cleaner to join a company
type CleanerInvite struct {
	ID    string `gorm:"primaryKey;size:50;unique"`
	Token string `gorm:"size:64;not null;unique;index:idx_cleaner_invites_token"`

	// Company that created the invite
	Company   *Company `gorm:"foreignKey:CompanyID"`
	CompanyID string   `gorm:"size:50;not null;index:idx_cleaner_invites_company"`

	// Admin who created the invite
	CreatedBy   *User  `gorm:"foreignKey:CreatedByID"`
	CreatedByID string `gorm:"size:50;not null"`

	// Optional: Pre-fill email for tracking
	Email *string `gorm:"size:256;index:idx_cleaner_invites_email"`

	// Optional: Custom message from admin
	Message *string `gorm:"type:text"`

	// Status tracking
	Status CleanerInviteStatus `gorm:"size:20;not null;default:'PENDING';index:idx_cleaner_invites_status"`

	// User who accepted the invite (if accepted)
	AcceptedBy   *User   `gorm:"foreignKey:AcceptedByID"`
	AcceptedByID *string `gorm:"size:50"`
	AcceptedAt   *time.Time

	// Expiration (default: 7 days from creation)
	ExpiresAt time.Time `gorm:"not null;index:idx_cleaner_invites_expires"`

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;not null"`
}

// IsExpired checks if the invite has expired
func (ci *CleanerInvite) IsExpired() bool {
	return time.Now().After(ci.ExpiresAt)
}

// CleanerInviteStore defines the data access interface for cleaner invites
type CleanerInviteStore interface {
	// Create creates a new cleaner invite
	Create(ctx context.Context, invite *CleanerInvite) error

	// Get retrieves an invite by ID
	Get(ctx context.Context, id string) (*CleanerInvite, error)

	// GetByToken retrieves an invite by its unique token
	GetByToken(ctx context.Context, token string) (*CleanerInvite, error)

	// GetByCompany retrieves all invites for a company
	GetByCompany(ctx context.Context, companyID string) ([]*CleanerInvite, error)

	// GetPendingByCompany retrieves pending invites for a company
	GetPendingByCompany(ctx context.Context, companyID string) ([]*CleanerInvite, error)

	// GetByAcceptedUserID retrieves the invite accepted by a specific user
	GetByAcceptedUserID(ctx context.Context, userID string) (*CleanerInvite, error)

	// Update updates an invite
	Update(ctx context.Context, invite *CleanerInvite) error

	// MarkAsAccepted marks an invite as accepted
	MarkAsAccepted(ctx context.Context, id string, acceptedByID string) error

	// MarkAsRevoked marks an invite as revoked
	MarkAsRevoked(ctx context.Context, id string) error

	// Delete removes an invite by ID
	Delete(ctx context.Context, id string) error

	// DeleteExpired removes expired invites older than a given time
	DeleteExpired(ctx context.Context, olderThan time.Time) error
}

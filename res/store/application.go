package store

import (
	"context"
	"time"
)

type ApplicationType string

const (
	ApplicationTypeCleaner      ApplicationType = "cleaner"
	ApplicationTypeCompanyAdmin ApplicationType = "company_admin"
)

type ApplicationStatus string

const (
	ApplicationStatusPending  ApplicationStatus = "pending"
	ApplicationStatusApproved ApplicationStatus = "approved"
	ApplicationStatusRejected ApplicationStatus = "rejected"
)

type Application struct {
	ID   string          `gorm:"primaryKey;size:50;unique"`
	User *User           `gorm:"foreignKey:UserID"`
	UserID string        `gorm:"size:50;not null;index:idx_applications_user_status"`

	ApplicationType ApplicationType   `gorm:"size:20;not null"`
	Status          ApplicationStatus `gorm:"size:20;not null;default:'pending';index:idx_applications_user_status"`

	Message string `gorm:"type:text"` // Expandable field for future structured data (JSON or text)

	ReviewedBy   *User   `gorm:"foreignKey:ReviewedByID"`
	ReviewedByID *string `gorm:"size:50"`
	ReviewedAt   *time.Time

	CreatedAt time.Time `gorm:"autoCreateTime;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;not null"`
}

type ApplicationStore interface {
	// Create creates a new application
	Create(ctx context.Context, application *Application) error

	// Get retrieves an application by ID
	Get(ctx context.Context, id string) (*Application, error)

	// GetByUser retrieves all applications for a specific user
	GetByUser(ctx context.Context, userID string) ([]*Application, error)

	// GetPending retrieves all pending applications (for global admin review)
	GetPending(ctx context.Context) ([]*Application, error)

	// UpdateStatus updates the application status and review information
	// This should be called when a global admin approves or rejects an application
	UpdateStatus(ctx context.Context, id string, status ApplicationStatus, reviewerID string) error

	// GetByUserAndType retrieves applications by user and type (to check for duplicates)
	GetByUserAndType(ctx context.Context, userID string, appType ApplicationType) ([]*Application, error)
}

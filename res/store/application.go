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

type CompanyInfo struct {
	CompanyName         string  `json:"companyName"`
	RegistrationNumber  string  `json:"registrationNumber"`
	TaxID               string  `json:"taxId"`
	CompanyStreet       string  `json:"companyStreet"`
	CompanyCity         string  `json:"companyCity"`
	CompanyPostalCode   string  `json:"companyPostalCode"`
	CompanyCounty       *string `json:"companyCounty,omitempty"`
	CompanyCountry      string  `json:"companyCountry"`
	BusinessType        *string `json:"businessType,omitempty"`
}

type ApplicationDocuments struct {
	IdentityDocumentUrl      string   `json:"identityDocumentUrl"`
	BusinessRegistrationUrl  *string  `json:"businessRegistrationUrl,omitempty"`
	InsuranceCertificateUrl  *string  `json:"insuranceCertificateUrl,omitempty"`
	AdditionalDocuments      []string `json:"additionalDocuments,omitempty"`
}

type Application struct {
	ID   string          `gorm:"primaryKey;size:50;unique"`
	User *User           `gorm:"foreignKey:UserID"`
	UserID string        `gorm:"size:50;not null;index:idx_applications_user_status"`

	ApplicationType ApplicationType   `gorm:"size:20;not null"`
	Status          ApplicationStatus `gorm:"size:20;not null;default:'pending';index:idx_applications_user_status"`

	Message string `gorm:"type:text"` // Expandable field for future structured data (JSON or text)

	CompanyInfo      *CompanyInfo          `gorm:"type:jsonb;serializer:json"`
	Documents        *ApplicationDocuments `gorm:"type:jsonb;serializer:json"`
	RejectionReason  *string               `gorm:"type:text"`

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

	// UpdateStatusWithReason updates the application status with rejection reason
	// This should be called when a global admin rejects an application with a reason
	UpdateStatusWithReason(ctx context.Context, id string, status ApplicationStatus, reviewerID string, reason *string) error

	// GetByUserAndType retrieves applications by user and type (to check for duplicates)
	GetByUserAndType(ctx context.Context, userID string, appType ApplicationType) ([]*Application, error)
}

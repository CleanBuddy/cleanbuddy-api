package store

import (
	"context"
	"time"
)

// CompanyType represents the type of company
type CompanyType string

const (
	// CompanyTypeIndividual is for individual cleaner companies (solo operators)
	CompanyTypeIndividual CompanyType = "individual"
	// CompanyTypeBusiness is for business companies (managed by company admin, can have multiple cleaners)
	CompanyTypeBusiness CompanyType = "business"
)

// CompanyStatus represents the approval status of a company
type CompanyStatus string

// ApplicationDocuments stores the URLs of uploaded documents
type ApplicationDocuments struct {
	IdentityDocumentUrl     string   `json:"identityDocumentUrl"`
	BusinessRegistrationUrl *string  `json:"businessRegistrationUrl,omitempty"`
	InsuranceCertificateUrl *string  `json:"insuranceCertificateUrl,omitempty"`
	AdditionalDocuments     []string `json:"additionalDocuments,omitempty"`
}

const (
	// CompanyStatusPending is for companies awaiting admin approval
	CompanyStatusPending CompanyStatus = "PENDING"
	// CompanyStatusApproved is for companies that have been approved by global admin
	CompanyStatusApproved CompanyStatus = "APPROVED"
	// CompanyStatusRejected is for companies that have been rejected by global admin
	CompanyStatusRejected CompanyStatus = "REJECTED"
)

// Company represents a cleaning company on the platform
type Company struct {
	ID          string      `gorm:"primaryKey;size:50;unique"`
	AdminUser   *User       `gorm:"foreignKey:AdminUserID"`
	AdminUserID string      `gorm:"size:50;not null;unique;index:idx_companies_admin_user"`
	CompanyType CompanyType `gorm:"size:20;not null;default:'business';index:idx_companies_type"`

	// Approval Status (drives UI state)
	Status          CompanyStatus `gorm:"size:20;not null;default:'pending';index:idx_companies_status"`
	RejectionReason *string       `gorm:"type:text"`

	// Company Information
	CompanyName        string  `gorm:"size:256;not null;index:idx_companies_name"`
	RegistrationNumber string  `gorm:"size:100;not null"`
	TaxID              string  `gorm:"size:100;not null"`
	CompanyStreet      string  `gorm:"size:256;not null"`
	CompanyCity        string  `gorm:"size:100;not null"`
	CompanyPostalCode  string  `gorm:"size:20;not null"`
	CompanyCounty      *string `gorm:"size:100"`
	CompanyCountry     string  `gorm:"size:100;not null"`
	BusinessType       *string `gorm:"size:50"`

	// Documents (same structure as ApplicationDocuments)
	Documents *ApplicationDocuments `gorm:"type:jsonb;serializer:json"`

	// Message from applicant
	Message *string `gorm:"type:text"`

	// Status
	IsActive bool `gorm:"not null;default:true"`

	// Stats (for future use with company cleaners)
	TotalCleaners  int `gorm:"not null;default:0"`
	ActiveCleaners int `gorm:"not null;default:0"`

	CreatedAt time.Time `gorm:"autoCreateTime;not null;index:idx_companies_created"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;not null"`
}

// IsPending checks if the company is pending approval
func (c *Company) IsPending() bool {
	return c.Status == CompanyStatusPending
}

// IsApproved checks if the company is approved
func (c *Company) IsApproved() bool {
	return c.Status == CompanyStatusApproved
}

// IsRejected checks if the company is rejected
func (c *Company) IsRejected() bool {
	return c.Status == CompanyStatusRejected
}

// CompanyStore defines the data access interface for companies
type CompanyStore interface {
	// Create creates a new company
	Create(ctx context.Context, company *Company) error

	// Get retrieves a company by ID
	Get(ctx context.Context, id string) (*Company, error)

	// GetByAdminUserID retrieves a company by the admin user's ID
	GetByAdminUserID(ctx context.Context, adminUserID string) (*Company, error)

	// Update updates a company
	Update(ctx context.Context, company *Company) error

	// Delete deletes a company
	Delete(ctx context.Context, id string) error

	// List retrieves all companies
	List(ctx context.Context) ([]*Company, error)

	// ListByStatus retrieves companies by status
	ListByStatus(ctx context.Context, status CompanyStatus) ([]*Company, error)

	// UpdateStatus updates the status of a company
	UpdateStatus(ctx context.Context, id string, status CompanyStatus, rejectionReason *string) error

	// UpdateStats updates the cleaner statistics of a company
	UpdateStats(ctx context.Context, companyID string, stats CompanyStats) error
}

// CompanyStats represents statistics to update for a company
type CompanyStats struct {
	TotalCleaners  *int
	ActiveCleaners *int
}

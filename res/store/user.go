package store

import "time"

type UserRole string

const (
	UserRoleClient             UserRole = "CLIENT"
	UserRolePendingApplication UserRole = "PENDING_APPLICATION"
	UserRolePendingCleaner     UserRole = "PENDING_CLEANER"
	UserRoleRejectedCleaner    UserRole = "REJECTED_CLEANER"
	UserRoleCleaner            UserRole = "CLEANER"
	UserRoleCompanyAdmin       UserRole = "COMPANY_ADMIN"
	UserRoleGlobalAdmin        UserRole = "GLOBAL_ADMIN"
)

type User struct {
	ID          string   `gorm:"primaryKey;size:50;unique"`
	DisplayName string   `gorm:"size:50;not null"`
	Role        UserRole `gorm:"size:20;not null;default:'CLIENT'"`

	GoogleIdentity *string `gorm:"size:256;unique"`
	Email          string  `gorm:"size:256;not null"`

	UpdatedAt time.Time `gorm:"autoUpdateTime;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime;not null"`
}

// IsGlobalAdmin checks if the user has global admin privileges
func (u *User) IsGlobalAdmin() bool {
	return u.Role == UserRoleGlobalAdmin
}

// IsCompanyAdmin checks if the user is a company admin
func (u *User) IsCompanyAdmin() bool {
	return u.Role == UserRoleCompanyAdmin
}

// IsCleaner checks if the user is a cleaner
func (u *User) IsCleaner() bool {
	return u.Role == UserRoleCleaner
}

// IsClient checks if the user is a basic client
func (u *User) IsClient() bool {
	return u.Role == UserRoleClient
}

// IsPendingCleaner checks if the user has a pending cleaner application
func (u *User) IsPendingCleaner() bool {
	return u.Role == UserRolePendingCleaner
}

// IsRejectedCleaner checks if the user has a rejected cleaner application
func (u *User) IsRejectedCleaner() bool {
	return u.Role == UserRoleRejectedCleaner
}

// IsPendingApplication checks if the user is waiting to submit application
func (u *User) IsPendingApplication() bool {
	return u.Role == UserRolePendingApplication
}

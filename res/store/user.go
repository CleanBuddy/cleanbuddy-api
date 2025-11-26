package store

import "time"

type UserRole string

const (
	UserRoleClient       UserRole = "CLIENT"        // Regular customer (default sign-in)
	UserRoleCleanerAdmin UserRole = "CLEANER_ADMIN" // Company admin (sign-in from "become a cleaner" flow)
	UserRoleCleaner      UserRole = "CLEANER"       // Cleaner working for a company (sign-in from invite link)
	UserRoleGlobalAdmin  UserRole = "GLOBAL_ADMIN"  // Platform administrator (set via env var)
)

type User struct {
	ID          string   `gorm:"primaryKey;size:50;unique"`
	DisplayName string   `gorm:"size:50;not null"`
	Role        UserRole `gorm:"size:50;not null;default:'CLIENT'"`

	GoogleIdentity *string `gorm:"size:256;unique"`
	Email          string  `gorm:"size:256;not null"`

	UpdatedAt time.Time `gorm:"autoUpdateTime;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime;not null"`
}

// IsGlobalAdmin checks if the user has global admin privileges
func (u *User) IsGlobalAdmin() bool {
	return u.Role == UserRoleGlobalAdmin
}

// IsCleanerAdmin checks if the user is a cleaner admin (company owner)
func (u *User) IsCleanerAdmin() bool {
	return u.Role == UserRoleCleanerAdmin
}

// IsCleaner checks if the user is a cleaner (invited to a company)
func (u *User) IsCleaner() bool {
	return u.Role == UserRoleCleaner
}

// IsClient checks if the user is a basic client
func (u *User) IsClient() bool {
	return u.Role == UserRoleClient
}

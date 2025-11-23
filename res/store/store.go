package store

import (
	"context"
	"time"
)

type Store interface {
	AuthSessions() AuthSessionStore
	Users() UserStore
	Applications() ApplicationStore

	// Database access for advanced operations
	GetDB() interface{} // Returns the underlying database connection
}

type AuthSessionStore interface {
	Get(ctx context.Context, ID string) (*AuthSession, error)

	Create(ctx context.Context, ID, userID string) (*AuthSession, error)
	Delete(ctx context.Context, IDs []string) error
	DeleteExpired(ctx context.Context, expirationPoint time.Time) error
	DeleteAllByUser(ctx context.Context, userID string) error
}

type UserStore interface {
	Get(ctx context.Context, id string) (*User, error)
	GetByGoogleIdentity(ctx context.Context, googleIdentity string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)

	Create(ctx context.Context, ID, displayName, email string, role UserRole, googleIdentity *string) (*User, error)
	Update(ctx context.Context, userID string, displayName *string, role *UserRole) (*User, error)
	Delete(ctx context.Context, userID string) error
}

package postgresql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"saas-starter-api/res/store"

	"gorm.io/gorm"
)

type authSessionStore struct {
	*storeImpl
}

func NewAuthSessionStore(rootStore *storeImpl) *authSessionStore {
	return &authSessionStore{storeImpl: rootStore}
}

// MUTATIONS

func (asStore *authSessionStore) Create(ctx context.Context, ID, userID string) (*store.AuthSession, error) {
	newAuthSession := &store.AuthSession{ID: ID, UserID: userID}

	result := asStore.db.WithContext(ctx).Create(newAuthSession)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return nil, store.ErrUniqueViolation
		}
		return nil, result.Error
	} else if result.RowsAffected != 1 {
		return nil, fmt.Errorf("failed to create auth session (id: %s)", ID)
	}

	return newAuthSession, nil
}

func (asStore *authSessionStore) Delete(ctx context.Context, IDs []string) error {
	result := asStore.db.WithContext(ctx).Where("id IN ?", IDs).Delete(&store.AuthSession{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (asStore *authSessionStore) DeleteExpired(ctx context.Context, expirationPoint time.Time) error {
	result := asStore.db.WithContext(ctx).Where("created_at < ?", expirationPoint).Delete(&store.AuthSession{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (asStore *authSessionStore) DeleteAllByUser(ctx context.Context, userID string) error {
	result := asStore.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&store.AuthSession{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// QUERIES

func (asStore *authSessionStore) Get(ctx context.Context, ID string) (*store.AuthSession, error) {
	var session store.AuthSession
	result := asStore.db.WithContext(ctx).Where("id = ?", ID).First(&session)
	if result.Error != nil {
		return nil, result.Error
	}
	return &session, nil
}

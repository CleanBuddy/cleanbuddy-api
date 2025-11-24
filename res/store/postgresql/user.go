package postgresql

import (
	"context"
	"fmt"
	"net/mail"
	"unicode/utf8"

	"cleanbuddy-api/res/store"
)

type userStore struct {
	*storeImpl
}

func NewUserStore(rootStore *storeImpl) *userStore {
	return &userStore{storeImpl: rootStore}
}

// MUTATIONS

func (uStore *userStore) Create(
	ctx context.Context,
	ID string,
	displayName string,
	email string,
	role store.UserRole,
	googleIdentity *string,
) (*store.User, error) {
	newUser := &store.User{ID: ID}

	// Role validation
	if role != store.UserRoleClient && role != store.UserRoleCleaner && role != store.UserRoleCompanyAdmin && role != store.UserRoleGlobalAdmin {
		return nil, fmt.Errorf("invalid user role (%s)", role)
	}
	newUser.Role = role

	// Display name validation

	if !utf8.ValidString(displayName) {
		return nil, fmt.Errorf("invalid user display name string (%s)", displayName)
	}

	displayNameLength := utf8.RuneCountInString(displayName)
	if displayNameLength == 0 {
		return nil, fmt.Errorf("invalid user display name string (empty)")
	} else if displayNameLength > 50 {
		return nil, fmt.Errorf("invalid user display name length (%d > 50)", displayNameLength)
	}

	newUser.DisplayName = displayName

	// Email validation

	if utf8.ValidString(email) {
		if emailAddr, err := mail.ParseAddress(email); err == nil {
			newUser.Email = emailAddr.Address
		} else {
			return nil, fmt.Errorf("invalid user email address")
		}
	} else {
		return nil, fmt.Errorf("invalid user email address string")
	}

	// Google identity validation

	if googleIdentity != nil {
		if !utf8.ValidString(*googleIdentity) {
			return nil, fmt.Errorf("invalid user google identity (%s)", *googleIdentity)
		}

		googleIdentityLength := utf8.RuneCountInString(*googleIdentity)
		if googleIdentityLength == 0 {
			return nil, fmt.Errorf("invalid user google identity (empty)")
		}

		newUser.GoogleIdentity = googleIdentity
	}

	result := uStore.db.WithContext(ctx).Create(newUser)
	if result.Error != nil {
		return nil, result.Error
	} else if result.RowsAffected != 1 {
		return nil, fmt.Errorf("failed to create user (id: %s)", ID)
	}

	return newUser, nil
}

func (uStore *userStore) Update(ctx context.Context, userID string, displayName *string, role *store.UserRole) (*store.User, error) {
	updates := make(map[string]interface{})

	// Validate and add displayName if provided
	if displayName != nil {
		if !utf8.ValidString(*displayName) {
			return nil, fmt.Errorf("invalid user display name string (%s)", *displayName)
		}

		displayNameLength := utf8.RuneCountInString(*displayName)
		if displayNameLength == 0 {
			return nil, fmt.Errorf("invalid user display name string (empty)")
		} else if displayNameLength > 50 {
			return nil, fmt.Errorf("invalid user display name length (%d > 50)", displayNameLength)
		}

		updates["display_name"] = *displayName
	}

	// Validate and add role if provided
	if role != nil {
		if *role != store.UserRoleClient && *role != store.UserRoleCleaner && *role != store.UserRoleCompanyAdmin && *role != store.UserRoleGlobalAdmin {
			return nil, fmt.Errorf("invalid user role (%s)", *role)
		}

		updates["role"] = *role
	}

	// If no updates provided, return error
	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	// Perform the update
	result := uStore.db.WithContext(ctx).Model(&store.User{}).
		Where("id = ?", userID).
		Updates(updates)

	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("user not found (id: %s)", userID)
	}

	// Fetch and return the updated user
	var user store.User
	if err := uStore.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch updated user: %w", err)
	}

	return &user, nil
}

func (uStore *userStore) Delete(ctx context.Context, userID string) error {
	// With CASCADE delete constraints properly configured in the database,
	// deleting a user will automatically cascade to:
	// - Auth sessions (CASCADE)
	// - Applications (CASCADE)
	result := uStore.db.WithContext(ctx).Delete(&store.User{ID: userID})
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("user not found (id: %s)", userID)
	}
	return nil
}

// QUERIES

func (uStore *userStore) Get(ctx context.Context, id string) (*store.User, error) {
	var user store.User
	result := uStore.db.WithContext(ctx).Where("id = ?", id).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (uStore *userStore) GetByGoogleIdentity(ctx context.Context, googleIdentity string) (*store.User, error) {
	var user store.User
	result := uStore.db.WithContext(ctx).Where("google_identity = ?", googleIdentity).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (uStore *userStore) GetByEmail(ctx context.Context, email string) (*store.User, error) {
	var user store.User
	result := uStore.db.WithContext(ctx).Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

package graphql

import (
	"context"
	"errors"

	"cleanbuddy-api/res/store"
	"cleanbuddy-api/sys/graphql/gen"
	"cleanbuddy-api/sys/graphql/scalar"
	"cleanbuddy-api/sys/http/middleware"
)

// FIELD RESOLVERS

type userResolver struct{ *Resolver }

func (r *Resolver) User() gen.UserResolver { return &userResolver{r} }

func (ur *userResolver) Company(ctx context.Context, user *store.User) (*store.Company, error) {
	// Only cleaner admins have a company
	if !user.IsCleanerAdmin() {
		return nil, nil
	}

	company, err := ur.Store.Companies().GetByAdminUserID(ctx, user.ID)
	if err != nil {
		// Company may not exist yet
		return nil, nil
	}

	return company, nil
}

func (ur *userResolver) CleanerProfile(ctx context.Context, user *store.User) (*store.CleanerProfile, error) {
	// Only cleaners and cleaner admins can have profiles
	if !user.IsCleaner() && !user.IsCleanerAdmin() {
		return nil, nil
	}

	profile, err := ur.Store.CleanerProfiles().GetByUserID(ctx, user.ID)
	if err != nil {
		// Profile may not exist yet
		return nil, nil
	}

	return profile, nil
}

// QUERIES RESOLVERS

func (qr *queryResolver) CurrentUser(ctx context.Context) (*store.User, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, nil
	}

	return currentUser, nil
}

// MUTATION RESOLVERS

func (mr *mutationResolver) SignOut(ctx context.Context) (*scalar.Void, error) {
	// Get current user
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		mr.Logger.Printf("Error retrieving current user for sign out")
		return nil, errors.New("access forbidden, authorization required")
	}

	// Delete all auth sessions for the current user
	if err := mr.Store.AuthSessions().DeleteAllByUser(ctx, currentUser.ID); err != nil {
		mr.Logger.Printf("Error deleting auth sessions: %s", err)
		return nil, errors.New("error signing out")
	}

	return &scalar.Void{}, nil
}

func (mr *mutationResolver) DeleteCurrentUser(ctx context.Context) (*scalar.Void, error) {
	// Get current user
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		mr.Logger.Printf("Error retrieving current user")
		return nil, errors.New("access forbidden, authorization required")
	}

	// Remove user from email service before deleting from database (if available)
	if mr.MailService != nil {
		if err := mr.MailService.RemoveUserByEmail(ctx, currentUser.Email); err != nil {
			// Log the error but don't fail the operation - email service is optional
			mr.Logger.Printf("Warning: Failed to remove user from email service: %v", err)
		}
	}

	// Delete user and all associated data
	if err := mr.Store.Users().Delete(ctx, currentUser.ID); err != nil {
		mr.Logger.Printf("Error deleting user: %s", err)
		return nil, errors.New("error deleting user")
	}

	return &scalar.Void{}, nil
}

func (mr *mutationResolver) UpdateCurrentUser(ctx context.Context, input gen.UpdateCurrentUserInput) (*store.User, error) {
	// Get current user
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		mr.Logger.Printf("Error retrieving current user for update")
		return nil, errors.New("access forbidden, authorization required")
	}

	// Prepare update parameters - only displayName can be updated by users
	var displayName *string

	if input.DisplayName != nil {
		displayName = input.DisplayName
	}

	// Update the user in the database (role is nil - only admins can change roles)
	updatedUser, err := mr.Store.Users().Update(ctx, currentUser.ID, displayName, nil)
	if err != nil {
		mr.Logger.Printf("Error updating user: %s", err)
		return nil, errors.New("error updating user")
	}

	return updatedUser, nil
}

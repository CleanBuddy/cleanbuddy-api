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

func (ur *userResolver) Applications(ctx context.Context, user *store.User) ([]*store.Application, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	applications, err := ur.Store.Applications().GetByUser(ctx, currentUser.ID)
	if err != nil {
		ur.Logger.Printf("Error retrieving applications: %s", err)
		return nil, errors.New("internal server error")
	}

	return applications, nil
}

func (ur *userResolver) PendingCleanerApplication(ctx context.Context, user *store.User) (*store.Application, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Get all applications for the user
	applications, err := ur.Store.Applications().GetByUser(ctx, currentUser.ID)
	if err != nil {
		ur.Logger.Printf("Error retrieving applications: %s", err)
		return nil, errors.New("internal server error")
	}

	// Find the most recent pending cleaner application
	for _, app := range applications {
		if app.ApplicationType == store.ApplicationTypeCleaner && app.Status == store.ApplicationStatusPending {
			return app, nil
		}
	}

	// No pending cleaner application found
	return nil, nil
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

func (mr *mutationResolver) UpdateUserRole(ctx context.Context, role store.UserRole) (*store.User, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Define valid role transitions
	validTransitions := map[store.UserRole][]store.UserRole{
		store.UserRoleClient:          {store.UserRolePendingApplication},
		store.UserRoleRejectedCleaner: {store.UserRolePendingApplication},
	}

	newRole := role

	// Validate transition
	allowedRoles, exists := validTransitions[currentUser.Role]
	if !exists {
		mr.Logger.Printf("Role transition not allowed from %s", currentUser.Role)
		return nil, errors.New("role transition not allowed from your current role")
	}

	isValid := false
	for _, allowed := range allowedRoles {
		if newRole == allowed {
			isValid = true
			break
		}
	}

	if !isValid {
		mr.Logger.Printf("Cannot transition from %s to %s", currentUser.Role, newRole)
		return nil, errors.New("cannot transition to the requested role")
	}

	// Update role
	updatedUser, err := mr.Store.Users().Update(ctx, currentUser.ID, nil, &newRole)
	if err != nil {
		mr.Logger.Printf("Error updating user role: %s", err)
		return nil, errors.New("error updating user role")
	}

	return updatedUser, nil
}


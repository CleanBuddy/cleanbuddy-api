package graphql

import (
	"context"
	"errors"
	"fmt"

	"cleanbuddy-api/res/store"
	"cleanbuddy-api/sys/graphql/gen"
	"cleanbuddy-api/sys/http/middleware"

	"github.com/rs/xid"
)

// FIELD RESOLVERS

type applicationResolver struct{ *Resolver }

func (r *Resolver) Application() gen.ApplicationResolver { return &applicationResolver{r} }

func (ar *applicationResolver) User(ctx context.Context, obj *store.Application) (*store.User, error) {
	user, err := ar.Store.Users().Get(ctx, obj.UserID)
	if err != nil {
		ar.Logger.Printf("Error retrieving application user: %s", err)
		return nil, errors.New("internal server error")
	}

	return user, nil
}

func (ar *applicationResolver) ReviewedBy(ctx context.Context, obj *store.Application) (*store.User, error) {
	if obj.ReviewedByID == nil {
		return nil, nil
	}

	user, err := ar.Store.Users().Get(ctx, *obj.ReviewedByID)
	if err != nil {
		ar.Logger.Printf("Error retrieving reviewer user: %s", err)
		return nil, errors.New("internal server error")
	}

	return user, nil
}

// QUERY RESOLVERS

func (qr *queryResolver) Application(ctx context.Context, id string) (*store.Application, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	application, err := qr.Store.Applications().Get(ctx, id)
	if err != nil {
		qr.Logger.Printf("Error retrieving application: %s", err)
		return nil, errors.New("internal server error")
	}

	// Users can only see their own applications, global admins can see all
	if application.UserID != currentUser.ID && !currentUser.IsGlobalAdmin() {
		return nil, errors.New("access forbidden")
	}

	return application, nil
}

func (qr *queryResolver) MyApplications(ctx context.Context) ([]*store.Application, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	applications, err := qr.Store.Applications().GetByUser(ctx, currentUser.ID)
	if err != nil {
		qr.Logger.Printf("Error retrieving user applications: %s", err)
		return nil, errors.New("internal server error")
	}

	return applications, nil
}

func (qr *queryResolver) PendingApplications(ctx context.Context) ([]*store.Application, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Only global admins can view pending applications
	if !currentUser.IsGlobalAdmin() {
		return nil, errors.New("access forbidden, global admin access required")
	}

	applications, err := qr.Store.Applications().GetPending(ctx)
	if err != nil {
		qr.Logger.Printf("Error retrieving pending applications: %s", err)
		return nil, errors.New("internal server error")
	}

	return applications, nil
}

// MUTATION RESOLVERS

func (mr *mutationResolver) SubmitApplication(ctx context.Context, input gen.SubmitApplicationInput) (*store.Application, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Check if user already has a pending application of this type
	existingApps, err := mr.Store.Applications().GetByUserAndType(ctx, currentUser.ID, input.ApplicationType)
	if err != nil {
		mr.Logger.Printf("Error checking existing applications: %s", err)
		return nil, errors.New("internal server error")
	}

	for _, app := range existingApps {
		if app.Status == store.ApplicationStatusPending {
			return nil, errors.New("you already have a pending application of this type")
		}
	}

	// Create the application
	application := &store.Application{
		ID:              fmt.Sprintf("app_%s", xid.New().String()),
		UserID:          currentUser.ID,
		ApplicationType: input.ApplicationType,
		Status:          store.ApplicationStatusPending,
	}

	if input.Message != nil {
		application.Message = *input.Message
	}

	if err := mr.Store.Applications().Create(ctx, application); err != nil {
		mr.Logger.Printf("Error creating application: %s", err)
		return nil, errors.New("error creating application")
	}

	// Notify admins about new application (if notification service available)
	if mr.NotificationService != nil {
		notifMsg := fmt.Sprintf("New %s application from %s (%s)", input.ApplicationType, currentUser.DisplayName, currentUser.Email)
		if err := mr.NotificationService.SendFeedback(ctx, notifMsg, currentUser.ID, currentUser.Email); err != nil {
			mr.Logger.Printf("Warning: Failed to send application notification: %v", err)
		}
	}

	return application, nil
}

func (mr *mutationResolver) ApproveApplication(ctx context.Context, applicationID string) (*store.Application, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Only global admins can approve applications
	if !currentUser.IsGlobalAdmin() {
		return nil, errors.New("access forbidden, global admin access required")
	}

	// Get the application
	application, err := mr.Store.Applications().Get(ctx, applicationID)
	if err != nil {
		mr.Logger.Printf("Error retrieving application: %s", err)
		return nil, errors.New("internal server error")
	}

	// Check if already processed
	if application.Status != store.ApplicationStatusPending {
		return nil, fmt.Errorf("application already %s", application.Status)
	}

	// Update application status
	if err := mr.Store.Applications().UpdateStatus(ctx, applicationID, store.ApplicationStatusApproved, currentUser.ID); err != nil {
		mr.Logger.Printf("Error updating application status: %s", err)
		return nil, errors.New("error approving application")
	}

	// Update user role based on application type
	var newRole store.UserRole
	switch application.ApplicationType {
	case store.ApplicationTypeCleaner:
		newRole = store.UserRoleCleaner
	case store.ApplicationTypeCompanyAdmin:
		newRole = store.UserRoleCompanyAdmin
	default:
		return nil, fmt.Errorf("invalid application type: %s", application.ApplicationType)
	}

	// Update the user's role
	if _, err := mr.Store.Users().Update(ctx, application.UserID, nil, &newRole); err != nil {
		mr.Logger.Printf("Error updating user role: %s", err)
		return nil, errors.New("error updating user role")
	}

	// Fetch the updated application
	updatedApplication, err := mr.Store.Applications().Get(ctx, applicationID)
	if err != nil {
		mr.Logger.Printf("Error retrieving updated application: %s", err)
		return nil, errors.New("internal server error")
	}

	return updatedApplication, nil
}

func (mr *mutationResolver) RejectApplication(ctx context.Context, applicationID string, reason *string) (*store.Application, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Only global admins can reject applications
	if !currentUser.IsGlobalAdmin() {
		return nil, errors.New("access forbidden, global admin access required")
	}

	// Get the application
	application, err := mr.Store.Applications().Get(ctx, applicationID)
	if err != nil {
		mr.Logger.Printf("Error retrieving application: %s", err)
		return nil, errors.New("internal server error")
	}

	// Check if already processed
	if application.Status != store.ApplicationStatusPending {
		return nil, fmt.Errorf("application already %s", application.Status)
	}

	// Update application status
	if err := mr.Store.Applications().UpdateStatus(ctx, applicationID, store.ApplicationStatusRejected, currentUser.ID); err != nil {
		mr.Logger.Printf("Error updating application status: %s", err)
		return nil, errors.New("error rejecting application")
	}

	// TODO: Optionally store rejection reason in application message or separate field
	// For now, we just update the status

	// Fetch the updated application
	updatedApplication, err := mr.Store.Applications().Get(ctx, applicationID)
	if err != nil {
		mr.Logger.Printf("Error retrieving updated application: %s", err)
		return nil, errors.New("internal server error")
	}

	return updatedApplication, nil
}

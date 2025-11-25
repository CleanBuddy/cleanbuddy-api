package graphql

import (
	"context"
	"errors"
	"fmt"

	"cleanbuddy-api/res/storage"
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

func (ar *applicationResolver) CompanyInfo(ctx context.Context, obj *store.Application) (*store.CompanyInfo, error) {
	return obj.CompanyInfo, nil
}

func (ar *applicationResolver) Documents(ctx context.Context, obj *store.Application) (*store.ApplicationDocuments, error) {
	return obj.Documents, nil
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

func (qr *queryResolver) GenerateDocumentSignedURL(ctx context.Context, documentURL string) (string, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return "", errors.New("access forbidden, authorization required")
	}

	// Only global admins can generate signed URLs for documents
	if !currentUser.IsGlobalAdmin() {
		return "", errors.New("access forbidden, global admin access required")
	}

	if qr.StorageService == nil {
		return "", errors.New("storage service not available")
	}

	// Generate signed URL valid for 24 hours
	signedURL, err := qr.StorageService.GenerateSignedURL(ctx, documentURL, 24*60*60*1000000000) // 24 hours in nanoseconds
	if err != nil {
		qr.Logger.Printf("Error generating signed URL: %s", err)
		return "", errors.New("failed to generate signed URL")
	}

	return signedURL, nil
}

// MUTATION RESOLVERS

func (mr *mutationResolver) SubmitApplication(ctx context.Context, input gen.SubmitApplicationInput) (*store.Application, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Check if user is in a valid state to submit application
	if input.ApplicationType == store.ApplicationTypeCleaner {
		// Only CLIENT or PENDING_APPLICATION roles can submit cleaner applications
		if !currentUser.IsClient() && !currentUser.IsPendingApplication() {
			return nil, errors.New("you cannot submit a cleaner application in your current state")
		}
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

	// Validate required fields for cleaner applications
	if input.ApplicationType == store.ApplicationTypeCleaner {
		if input.CompanyInfo == nil {
			return nil, errors.New("company information is required for cleaner applications")
		}
		if input.Documents == nil {
			return nil, errors.New("documents are required for cleaner applications")
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

	// Map company info if provided
	if input.CompanyInfo != nil {
		application.CompanyInfo = &store.CompanyInfo{
			CompanyName:        input.CompanyInfo.CompanyName,
			RegistrationNumber: input.CompanyInfo.RegistrationNumber,
			TaxID:              input.CompanyInfo.TaxID,
			CompanyStreet:      input.CompanyInfo.CompanyStreet,
			CompanyCity:        input.CompanyInfo.CompanyCity,
			CompanyPostalCode:  input.CompanyInfo.CompanyPostalCode,
			CompanyCounty:      input.CompanyInfo.CompanyCounty,
			CompanyCountry:     input.CompanyInfo.CompanyCountry,
			BusinessType:       input.CompanyInfo.BusinessType,
		}
	}

	// Handle document uploads if provided
	if input.Documents != nil {
		if mr.StorageService == nil {
			return nil, errors.New("storage service not available")
		}

		docs := &store.ApplicationDocuments{}

		// Upload identity document (required)
		path := storage.BuildApplicationDocumentPath(application.ID, "identity-document", input.Documents.IdentityDocument.Filename)
		url, err := mr.StorageService.UploadFromReader(
			ctx,
			input.Documents.IdentityDocument.File,
			input.Documents.IdentityDocument.Filename,
			input.Documents.IdentityDocument.Size,
			input.Documents.IdentityDocument.ContentType,
			path,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to upload identity document: %w", err)
		}
		docs.IdentityDocumentUrl = url

		// Upload business registration (optional)
		if input.Documents.BusinessRegistration != nil {
			path := storage.BuildApplicationDocumentPath(application.ID, "business-registration", input.Documents.BusinessRegistration.Filename)
			url, err := mr.StorageService.UploadFromReader(
				ctx,
				input.Documents.BusinessRegistration.File,
				input.Documents.BusinessRegistration.Filename,
				input.Documents.BusinessRegistration.Size,
				input.Documents.BusinessRegistration.ContentType,
				path,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to upload business registration: %w", err)
			}
			docs.BusinessRegistrationUrl = &url
		}

		// Upload insurance certificate (optional)
		if input.Documents.InsuranceCertificate != nil {
			path := storage.BuildApplicationDocumentPath(application.ID, "insurance-certificate", input.Documents.InsuranceCertificate.Filename)
			url, err := mr.StorageService.UploadFromReader(
				ctx,
				input.Documents.InsuranceCertificate.File,
				input.Documents.InsuranceCertificate.Filename,
				input.Documents.InsuranceCertificate.Size,
				input.Documents.InsuranceCertificate.ContentType,
				path,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to upload insurance certificate: %w", err)
			}
			docs.InsuranceCertificateUrl = &url
		}

		// Upload additional documents (optional)
		if len(input.Documents.AdditionalDocuments) > 0 {
			additionalUrls := make([]string, 0, len(input.Documents.AdditionalDocuments))
			for i, upload := range input.Documents.AdditionalDocuments {
				if upload == nil {
					continue
				}
				path := storage.BuildApplicationDocumentPath(application.ID, fmt.Sprintf("additional-%d", i+1), upload.Filename)
				url, err := mr.StorageService.UploadFromReader(
					ctx,
					upload.File,
					upload.Filename,
					upload.Size,
					upload.ContentType,
					path,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to upload additional document %d: %w", i+1, err)
				}
				additionalUrls = append(additionalUrls, url)
			}
			docs.AdditionalDocuments = additionalUrls
		}

		application.Documents = docs
	}

	if err := mr.Store.Applications().Create(ctx, application); err != nil {
		mr.Logger.Printf("Error creating application: %s", err)
		return nil, errors.New("error creating application")
	}

	// Update user role to PENDING_CLEANER for cleaner applications
	if input.ApplicationType == store.ApplicationTypeCleaner {
		newRole := store.UserRolePendingCleaner
		if _, err := mr.Store.Users().Update(ctx, currentUser.ID, nil, &newRole); err != nil {
			mr.Logger.Printf("Error updating user role to pending_cleaner: %s", err)
			// Don't fail the application submission, just log the error
		}
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

	// Update application status with rejection reason
	if err := mr.Store.Applications().UpdateStatusWithReason(ctx, applicationID, store.ApplicationStatusRejected, currentUser.ID, reason); err != nil {
		mr.Logger.Printf("Error updating application status: %s", err)
		return nil, errors.New("error rejecting application")
	}

	// Set user role to REJECTED_CLEANER if they were PENDING_CLEANER
	applicant, err := mr.Store.Users().Get(ctx, application.UserID)
	if err == nil && applicant != nil && applicant.IsPendingCleaner() {
		rejectedRole := store.UserRoleRejectedCleaner
		if _, err := mr.Store.Users().Update(ctx, application.UserID, nil, &rejectedRole); err != nil {
			mr.Logger.Printf("Error setting user role to rejected_cleaner: %s", err)
			// Don't fail the rejection, just log the error
		}
	}

	// Fetch the updated application
	updatedApplication, err := mr.Store.Applications().Get(ctx, applicationID)
	if err != nil {
		mr.Logger.Printf("Error retrieving updated application: %s", err)
		return nil, errors.New("internal server error")
	}

	return updatedApplication, nil
}

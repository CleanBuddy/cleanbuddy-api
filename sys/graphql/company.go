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

type companyResolver struct{ *Resolver }

func (r *Resolver) Company() gen.CompanyResolver { return &companyResolver{r} }

func (cr *companyResolver) AdminUser(ctx context.Context, obj *store.Company) (*store.User, error) {
	user, err := cr.Store.Users().Get(ctx, obj.AdminUserID)
	if err != nil {
		cr.Logger.Printf("Error retrieving company admin user: %s", err)
		return nil, errors.New("internal server error")
	}

	return user, nil
}

func (cr *companyResolver) Documents(ctx context.Context, obj *store.Company) (*store.ApplicationDocuments, error) {
	return obj.Documents, nil
}

func (cr *companyResolver) Cleaners(ctx context.Context, obj *store.Company) ([]*store.CleanerProfile, error) {
	if obj.CompanyType == store.CompanyTypeIndividual {
		// For individual companies, the admin user is the cleaner
		profile, err := cr.Store.CleanerProfiles().GetByUserID(ctx, obj.AdminUserID)
		if err != nil {
			// Profile may not exist yet
			return []*store.CleanerProfile{}, nil
		}
		return []*store.CleanerProfile{profile}, nil
	}

	// For business companies, return all cleaners with this company_id
	filters := store.CleanerProfileFilters{
		CompanyID: &obj.ID,
	}
	cleaners, err := cr.Store.CleanerProfiles().List(ctx, filters)
	if err != nil {
		cr.Logger.Printf("Error retrieving company cleaners: %s", err)
		return []*store.CleanerProfile{}, nil
	}
	return cleaners, nil
}

// QUERY RESOLVERS

func (qr *queryResolver) MyCompany(ctx context.Context) (*store.Company, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Company admins and cleaners can access their company
	if currentUser.IsCleanerAdmin() || currentUser.IsCleaner() {
		company, err := qr.Store.Companies().GetByAdminUserID(ctx, currentUser.ID)
		if err != nil {
			qr.Logger.Printf("Error retrieving company for user %s: %s", currentUser.ID, err)
			return nil, errors.New("company not found")
		}
		return company, nil
	}

	return nil, errors.New("access forbidden, you don't have a company")
}

func (qr *queryResolver) Company(ctx context.Context, id string) (*store.Company, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Only global admins can view any company by ID
	if !currentUser.IsGlobalAdmin() {
		return nil, errors.New("access forbidden, global admin access required")
	}

	company, err := qr.Store.Companies().Get(ctx, id)
	if err != nil {
		qr.Logger.Printf("Error retrieving company: %s", err)
		return nil, errors.New("company not found")
	}

	return company, nil
}

func (qr *queryResolver) Companies(ctx context.Context) ([]*store.Company, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Only global admins can list all companies
	if !currentUser.IsGlobalAdmin() {
		return nil, errors.New("access forbidden, global admin access required")
	}

	companies, err := qr.Store.Companies().List(ctx)
	if err != nil {
		qr.Logger.Printf("Error listing companies: %s", err)
		return nil, errors.New("internal server error")
	}

	return companies, nil
}

// MUTATION RESOLVERS

func (mr *mutationResolver) UpdateCompany(ctx context.Context, input gen.UpdateCompanyInput) (*store.Company, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Only cleaner admins can update their company
	if !currentUser.IsCleanerAdmin() {
		return nil, errors.New("access forbidden, you are not a company admin")
	}

	// Get current company
	company, err := mr.Store.Companies().GetByAdminUserID(ctx, currentUser.ID)
	if err != nil {
		mr.Logger.Printf("Error retrieving company for user %s: %s", currentUser.ID, err)
		return nil, errors.New("company not found")
	}

	// Apply updates
	if input.CompanyName != nil {
		company.CompanyName = *input.CompanyName
	}
	if input.CompanyStreet != nil {
		company.CompanyStreet = *input.CompanyStreet
	}
	if input.CompanyCity != nil {
		company.CompanyCity = *input.CompanyCity
	}
	if input.CompanyPostalCode != nil {
		company.CompanyPostalCode = *input.CompanyPostalCode
	}
	if input.CompanyCounty != nil {
		company.CompanyCounty = input.CompanyCounty
	}
	if input.BusinessType != nil {
		company.BusinessType = input.BusinessType
	}
	if input.IsActive != nil {
		company.IsActive = *input.IsActive
	}

	// Save updates
	if err := mr.Store.Companies().Update(ctx, company); err != nil {
		mr.Logger.Printf("Error updating company: %s", err)
		return nil, errors.New("error updating company")
	}

	return company, nil
}

func (qr *queryResolver) PendingCompanies(ctx context.Context) ([]*store.Company, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Only global admins can list pending companies
	if !currentUser.IsGlobalAdmin() {
		return nil, errors.New("access forbidden, global admin access required")
	}

	companies, err := qr.Store.Companies().ListByStatus(ctx, store.CompanyStatusPending)
	if err != nil {
		qr.Logger.Printf("Error listing pending companies: %s", err)
		return nil, errors.New("internal server error")
	}

	return companies, nil
}

func (mr *mutationResolver) CreateCompany(ctx context.Context, input gen.CreateCompanyInput) (*store.Company, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Only cleaner admins can create a company
	if !currentUser.IsCleanerAdmin() {
		return nil, errors.New("access forbidden, only cleaner admins can create a company")
	}

	// Check if user already has a company
	existingCompany, _ := mr.Store.Companies().GetByAdminUserID(ctx, currentUser.ID)
	if existingCompany != nil {
		return nil, errors.New("you already have a company")
	}

	// Build the company from input
	companyID := fmt.Sprintf("company_%s", xid.New().String())
	company := &store.Company{
		ID:                 companyID,
		AdminUserID:        currentUser.ID,
		CompanyType:        input.CompanyType,
		Status:             store.CompanyStatusPending,
		CompanyName:        input.CompanyInfo.CompanyName,
		RegistrationNumber: input.CompanyInfo.RegistrationNumber,
		TaxID:              input.CompanyInfo.TaxID,
		CompanyStreet:      input.CompanyInfo.CompanyStreet,
		CompanyCity:        input.CompanyInfo.CompanyCity,
		CompanyPostalCode:  input.CompanyInfo.CompanyPostalCode,
		CompanyCounty:      input.CompanyInfo.CompanyCounty,
		CompanyCountry:     input.CompanyInfo.CompanyCountry,
		BusinessType:       input.CompanyInfo.BusinessType,
		Message:            input.Message,
		IsActive:           false, // Not active until approved
	}

	// Handle document uploads if provided
	if input.Documents != nil {
		// Upload identity document (required)
		path := storage.BuildApplicationDocumentPath(companyID, "identity-document", input.Documents.IdentityDocument.Filename)
		identityDocURL, err := mr.StorageService.UploadFromReader(
			ctx,
			input.Documents.IdentityDocument.File,
			input.Documents.IdentityDocument.Filename,
			input.Documents.IdentityDocument.Size,
			input.Documents.IdentityDocument.ContentType,
			path,
		)
		if err != nil {
			mr.Logger.Printf("Error uploading identity document: %s", err)
			return nil, errors.New("error uploading identity document")
		}

		company.Documents = &store.ApplicationDocuments{
			IdentityDocumentUrl: identityDocURL,
		}

		// Upload business registration (optional)
		if input.Documents.BusinessRegistration != nil {
			path := storage.BuildApplicationDocumentPath(companyID, "business-registration", input.Documents.BusinessRegistration.Filename)
			businessRegURL, err := mr.StorageService.UploadFromReader(
				ctx,
				input.Documents.BusinessRegistration.File,
				input.Documents.BusinessRegistration.Filename,
				input.Documents.BusinessRegistration.Size,
				input.Documents.BusinessRegistration.ContentType,
				path,
			)
			if err != nil {
				mr.Logger.Printf("Error uploading business registration: %s", err)
				return nil, errors.New("error uploading business registration")
			}
			company.Documents.BusinessRegistrationUrl = &businessRegURL
		}

		// Upload insurance certificate (optional)
		if input.Documents.InsuranceCertificate != nil {
			path := storage.BuildApplicationDocumentPath(companyID, "insurance-certificate", input.Documents.InsuranceCertificate.Filename)
			insuranceURL, err := mr.StorageService.UploadFromReader(
				ctx,
				input.Documents.InsuranceCertificate.File,
				input.Documents.InsuranceCertificate.Filename,
				input.Documents.InsuranceCertificate.Size,
				input.Documents.InsuranceCertificate.ContentType,
				path,
			)
			if err != nil {
				mr.Logger.Printf("Error uploading insurance certificate: %s", err)
				return nil, errors.New("error uploading insurance certificate")
			}
			company.Documents.InsuranceCertificateUrl = &insuranceURL
		}

		// Upload additional documents (optional)
		if len(input.Documents.AdditionalDocuments) > 0 {
			additionalURLs := make([]string, 0, len(input.Documents.AdditionalDocuments))
			for i, doc := range input.Documents.AdditionalDocuments {
				if doc != nil {
					path := storage.BuildApplicationDocumentPath(companyID, fmt.Sprintf("additional-%d", i+1), doc.Filename)
					docURL, err := mr.StorageService.UploadFromReader(
						ctx,
						doc.File,
						doc.Filename,
						doc.Size,
						doc.ContentType,
						path,
					)
					if err != nil {
						mr.Logger.Printf("Error uploading additional document: %s", err)
						continue
					}
					additionalURLs = append(additionalURLs, docURL)
				}
			}
			company.Documents.AdditionalDocuments = additionalURLs
		}
	}

	// Create the company
	if err := mr.Store.Companies().Create(ctx, company); err != nil {
		mr.Logger.Printf("Error creating company: %s", err)
		return nil, errors.New("error creating company")
	}

	mr.Logger.Printf("Company %s created by user %s", company.ID, currentUser.ID)

	return company, nil
}

func (mr *mutationResolver) ApproveCompany(ctx context.Context, companyID string) (*store.Company, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Only global admins can approve companies
	if !currentUser.IsGlobalAdmin() {
		return nil, errors.New("access forbidden, global admin access required")
	}

	// Get the company
	company, err := mr.Store.Companies().Get(ctx, companyID)
	if err != nil {
		mr.Logger.Printf("Error retrieving company %s: %s", companyID, err)
		return nil, errors.New("company not found")
	}

	// Check if company is pending
	if !company.IsPending() {
		return nil, errors.New("company is not pending approval")
	}

	// Update status to approved
	if err := mr.Store.Companies().UpdateStatus(ctx, companyID, store.CompanyStatusApproved, nil); err != nil {
		mr.Logger.Printf("Error approving company %s: %s", companyID, err)
		return nil, errors.New("error approving company")
	}

	// Activate the company
	company.Status = store.CompanyStatusApproved
	company.IsActive = true
	company.RejectionReason = nil
	if err := mr.Store.Companies().Update(ctx, company); err != nil {
		mr.Logger.Printf("Error activating company %s: %s", companyID, err)
	}

	// Get updated company
	company, _ = mr.Store.Companies().Get(ctx, companyID)

	mr.Logger.Printf("Company %s approved by admin %s", companyID, currentUser.ID)

	return company, nil
}

func (mr *mutationResolver) RejectCompany(ctx context.Context, companyID string, reason *string) (*store.Company, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Only global admins can reject companies
	if !currentUser.IsGlobalAdmin() {
		return nil, errors.New("access forbidden, global admin access required")
	}

	// Get the company
	company, err := mr.Store.Companies().Get(ctx, companyID)
	if err != nil {
		mr.Logger.Printf("Error retrieving company %s: %s", companyID, err)
		return nil, errors.New("company not found")
	}

	// Check if company is pending
	if !company.IsPending() {
		return nil, errors.New("company is not pending approval")
	}

	// Update status to rejected
	if err := mr.Store.Companies().UpdateStatus(ctx, companyID, store.CompanyStatusRejected, reason); err != nil {
		mr.Logger.Printf("Error rejecting company %s: %s", companyID, err)
		return nil, errors.New("error rejecting company")
	}

	// Get updated company
	company, _ = mr.Store.Companies().Get(ctx, companyID)

	mr.Logger.Printf("Company %s rejected by admin %s", companyID, currentUser.ID)

	return company, nil
}

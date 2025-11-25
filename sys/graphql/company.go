package graphql

import (
	"context"
	"errors"

	"cleanbuddy-api/res/store"
	"cleanbuddy-api/sys/graphql/gen"
	"cleanbuddy-api/sys/http/middleware"
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
	if currentUser.IsCompanyAdmin() || currentUser.IsCleaner() {
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

	// Only company admins can update their company
	if !currentUser.IsCompanyAdmin() {
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

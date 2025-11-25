package graphql

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"cleanbuddy-api/res/store"
	"cleanbuddy-api/sys/graphql/gen"
	"cleanbuddy-api/sys/http/middleware"

	"github.com/rs/xid"
)

const (
	defaultInviteExpiryDays = 7
	maxInviteExpiryDays     = 30
	inviteTokenLength       = 32 // 32 bytes = 64 hex characters
)

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken() (string, error) {
	bytes := make([]byte, inviteTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// FIELD RESOLVERS

type cleanerInviteResolver struct{ *Resolver }

func (r *Resolver) CleanerInvite() gen.CleanerInviteResolver {
	return &cleanerInviteResolver{r}
}

func (cir *cleanerInviteResolver) Company(ctx context.Context, obj *store.CleanerInvite) (*store.Company, error) {
	company, err := cir.Store.Companies().Get(ctx, obj.CompanyID)
	if err != nil {
		cir.Logger.Printf("Error retrieving company for invite: %s", err)
		return nil, errors.New("company not found")
	}
	return company, nil
}

func (cir *cleanerInviteResolver) CreatedBy(ctx context.Context, obj *store.CleanerInvite) (*store.User, error) {
	user, err := cir.Store.Users().Get(ctx, obj.CreatedByID)
	if err != nil {
		cir.Logger.Printf("Error retrieving invite creator: %s", err)
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (cir *cleanerInviteResolver) AcceptedBy(ctx context.Context, obj *store.CleanerInvite) (*store.User, error) {
	if obj.AcceptedByID == nil {
		return nil, nil
	}
	user, err := cir.Store.Users().Get(ctx, *obj.AcceptedByID)
	if err != nil {
		cir.Logger.Printf("Error retrieving invite acceptor: %s", err)
		return nil, nil
	}
	return user, nil
}

// QUERY RESOLVERS

func (qr *queryResolver) ValidateCleanerInviteToken(ctx context.Context, token string) (*gen.ValidateCleanerInviteResult, error) {
	invite, err := qr.Store.CleanerInvites().GetByToken(ctx, token)
	if err != nil || invite == nil {
		return &gen.ValidateCleanerInviteResult{
			Valid:        false,
			ErrorMessage: stringPtr("Invalid invite token"),
		}, nil
	}

	// Check if expired
	if time.Now().After(invite.ExpiresAt) {
		return &gen.ValidateCleanerInviteResult{
			Valid:        false,
			Invite:       invite,
			ErrorMessage: stringPtr("Invite has expired"),
		}, nil
	}

	// Check if already used or revoked
	if invite.Status != store.CleanerInviteStatusPending {
		return &gen.ValidateCleanerInviteResult{
			Valid:        false,
			Invite:       invite,
			ErrorMessage: stringPtr(fmt.Sprintf("Invite has already been %s", string(invite.Status))),
		}, nil
	}

	// Get company info
	company, err := qr.Store.Companies().Get(ctx, invite.CompanyID)
	if err != nil {
		return &gen.ValidateCleanerInviteResult{
			Valid:        false,
			ErrorMessage: stringPtr("Company not found"),
		}, nil
	}

	return &gen.ValidateCleanerInviteResult{
		Valid:   true,
		Invite:  invite,
		Company: company,
	}, nil
}

func (qr *queryResolver) CleanerInvite(ctx context.Context, id string) (*store.CleanerInvite, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	if !currentUser.IsCompanyAdmin() && !currentUser.IsGlobalAdmin() {
		return nil, errors.New("access forbidden, company admin access required")
	}

	invite, err := qr.Store.CleanerInvites().Get(ctx, id)
	if err != nil {
		qr.Logger.Printf("Error retrieving cleaner invite: %s", err)
		return nil, errors.New("invite not found")
	}

	// Company admins can only see their own company's invites
	if currentUser.IsCompanyAdmin() {
		company, err := qr.Store.Companies().GetByAdminUserID(ctx, currentUser.ID)
		if err != nil || company.ID != invite.CompanyID {
			return nil, errors.New("access forbidden")
		}
	}

	return invite, nil
}

func (qr *queryResolver) MyCompanyInvites(ctx context.Context) ([]*store.CleanerInvite, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	if !currentUser.IsCompanyAdmin() {
		return nil, errors.New("access forbidden, company admin access required")
	}

	company, err := qr.Store.Companies().GetByAdminUserID(ctx, currentUser.ID)
	if err != nil {
		qr.Logger.Printf("Error retrieving company: %s", err)
		return nil, errors.New("company not found")
	}

	invites, err := qr.Store.CleanerInvites().GetByCompany(ctx, company.ID)
	if err != nil {
		qr.Logger.Printf("Error retrieving company invites: %s", err)
		return nil, errors.New("error retrieving invites")
	}

	return invites, nil
}

func (qr *queryResolver) MyCompanyCleaners(ctx context.Context) ([]*store.CleanerProfile, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	if !currentUser.IsCompanyAdmin() {
		return nil, errors.New("access forbidden, company admin access required")
	}

	company, err := qr.Store.Companies().GetByAdminUserID(ctx, currentUser.ID)
	if err != nil {
		qr.Logger.Printf("Error retrieving company: %s", err)
		return nil, errors.New("company not found")
	}

	// Get all cleaners linked to this company
	filters := store.CleanerProfileFilters{
		CompanyID: &company.ID,
	}
	cleaners, err := qr.Store.CleanerProfiles().List(ctx, filters)
	if err != nil {
		qr.Logger.Printf("Error retrieving company cleaners: %s", err)
		return nil, errors.New("error retrieving cleaners")
	}

	return cleaners, nil
}

// MUTATION RESOLVERS

func (mr *mutationResolver) CreateCleanerInvite(ctx context.Context, input *gen.CreateCleanerInviteInput) (*gen.CleanerInviteResult, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	if !currentUser.IsCompanyAdmin() {
		return nil, errors.New("access forbidden, only company admins can create invites")
	}

	// Get the admin's company
	company, err := mr.Store.Companies().GetByAdminUserID(ctx, currentUser.ID)
	if err != nil {
		mr.Logger.Printf("Error retrieving company: %s", err)
		return nil, errors.New("company not found")
	}

	// Only business companies can invite cleaners
	if company.CompanyType != store.CompanyTypeBusiness {
		return nil, errors.New("only business companies can invite cleaners")
	}

	// Generate secure token
	token, err := generateSecureToken()
	if err != nil {
		mr.Logger.Printf("Error generating token: %s", err)
		return nil, errors.New("error creating invite")
	}

	// Calculate expiry
	expiryDays := defaultInviteExpiryDays
	if input != nil && input.ExpiresInDays != nil {
		if *input.ExpiresInDays > maxInviteExpiryDays {
			expiryDays = maxInviteExpiryDays
		} else if *input.ExpiresInDays > 0 {
			expiryDays = *input.ExpiresInDays
		}
	}

	invite := &store.CleanerInvite{
		ID:          fmt.Sprintf("inv_%s", xid.New().String()),
		Token:       token,
		CompanyID:   company.ID,
		CreatedByID: currentUser.ID,
		Status:      store.CleanerInviteStatusPending,
		ExpiresAt:   time.Now().AddDate(0, 0, expiryDays),
	}

	if input != nil {
		invite.Email = input.Email
		invite.Message = input.Message
	}

	if err := mr.Store.CleanerInvites().Create(ctx, invite); err != nil {
		mr.Logger.Printf("Error creating invite: %s", err)
		return nil, errors.New("error creating invite")
	}

	// Build invite URL (frontend URL with token)
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	inviteURL := fmt.Sprintf("%s/invite/%s", frontendURL, token)

	return &gen.CleanerInviteResult{
		Invite:    invite,
		InviteURL: inviteURL,
	}, nil
}

func (mr *mutationResolver) AcceptCleanerInvite(ctx context.Context, token string) (*gen.AcceptCleanerInviteResult, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	// Get and validate invite
	invite, err := mr.Store.CleanerInvites().GetByToken(ctx, token)
	if err != nil || invite == nil {
		return nil, errors.New("invalid invite token")
	}

	// Check status
	if invite.Status != store.CleanerInviteStatusPending {
		return nil, fmt.Errorf("invite has already been %s", string(invite.Status))
	}

	// Check expiration
	if time.Now().After(invite.ExpiresAt) {
		return nil, errors.New("invite has expired")
	}

	// Check if user already has a cleaner profile
	existingProfile, _ := mr.Store.CleanerProfiles().GetByUserID(ctx, currentUser.ID)
	if existingProfile != nil {
		return nil, errors.New("you already have a cleaner profile")
	}

	// Check user role - they should not already be a cleaner or company admin
	if currentUser.IsCleaner() || currentUser.IsCompanyAdmin() {
		return nil, errors.New("you are already a cleaner or company admin")
	}

	// Get company
	company, err := mr.Store.Companies().Get(ctx, invite.CompanyID)
	if err != nil {
		mr.Logger.Printf("Error retrieving company: %s", err)
		return nil, errors.New("company not found")
	}

	// Update user role to CLEANER
	newRole := store.UserRoleCleaner
	_, err = mr.Store.Users().Update(ctx, currentUser.ID, nil, &newRole)
	if err != nil {
		mr.Logger.Printf("Error updating user role: %s", err)
		return nil, errors.New("error updating user role")
	}

	// Mark invite as accepted
	if err := mr.Store.CleanerInvites().MarkAsAccepted(ctx, invite.ID, currentUser.ID); err != nil {
		mr.Logger.Printf("Error marking invite as accepted: %s", err)
		// Continue - the important part (role update) succeeded
	}

	// Get updated user
	updatedUser, _ := mr.Store.Users().Get(ctx, currentUser.ID)

	// Update company stats
	totalCleaners := company.TotalCleaners + 1
	activeCleaners := company.ActiveCleaners + 1
	mr.Store.Companies().UpdateStats(ctx, company.ID, store.CompanyStats{
		TotalCleaners:  &totalCleaners,
		ActiveCleaners: &activeCleaners,
	})

	return &gen.AcceptCleanerInviteResult{
		Success: true,
		User:    updatedUser,
		Company: company,
	}, nil
}

func (mr *mutationResolver) RevokeCleanerInvite(ctx context.Context, id string) (*store.CleanerInvite, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("access forbidden, authorization required")
	}

	if !currentUser.IsCompanyAdmin() && !currentUser.IsGlobalAdmin() {
		return nil, errors.New("access forbidden, company admin access required")
	}

	invite, err := mr.Store.CleanerInvites().Get(ctx, id)
	if err != nil {
		mr.Logger.Printf("Error retrieving invite: %s", err)
		return nil, errors.New("invite not found")
	}

	// Company admins can only revoke their own company's invites
	if currentUser.IsCompanyAdmin() {
		company, err := mr.Store.Companies().GetByAdminUserID(ctx, currentUser.ID)
		if err != nil || company.ID != invite.CompanyID {
			return nil, errors.New("access forbidden")
		}
	}

	if invite.Status != store.CleanerInviteStatusPending {
		return nil, errors.New("can only revoke pending invites")
	}

	if err := mr.Store.CleanerInvites().MarkAsRevoked(ctx, invite.ID); err != nil {
		mr.Logger.Printf("Error revoking invite: %s", err)
		return nil, errors.New("error revoking invite")
	}

	// Fetch updated invite
	invite, _ = mr.Store.CleanerInvites().Get(ctx, id)
	return invite, nil
}

// Helper function
func stringPtr(s string) *string { return &s }

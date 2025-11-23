package graphql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"saas-starter-api/res/auth"
	"saas-starter-api/res/store"
	"saas-starter-api/sys/graphql/gen"
	"saas-starter-api/sys/http/middleware"

	"github.com/rs/xid"
)

const (
	userDisplayNamePlaceholderDefault string = "User"
)

// MUTATION RESOLVERS

func (mr *mutationResolver) AuthWithRefreshToken(ctx context.Context, token string) (*gen.AuthResult, error) {
	// 1. Validate refresh token and associated session/user

	var claims auth.RefreshTokenClaims
	err := mr.Auth.ValidateToken(token, &claims)
	if err != nil {
		mr.Logger.Printf("Error validating refresh token: %s", err)
		return nil, errors.New("invalid request, refresh token expired or malformed")
	}

	user, err := mr.Store.Users().Get(ctx, claims.UserID)
	if err != nil {
		mr.Logger.Printf("Error retrieving user associated with the refresh token: %s", err)
		return nil, errors.New("invalid request, refresh token expired or malformed")
	}
	if user == nil {
		mr.Logger.Printf("Error retrieving user associated with the refresh token: %s", err)
		return nil, errors.New("invalid request, refresh token expired or malformed")
	}

	err = mr.Store.AuthSessions().DeleteExpired(ctx, (time.Now().Add(-auth.RefreshTokenLifespanInHours * time.Hour)))
	if err != nil {
		mr.Logger.Printf("Error removing expired refresh session: %s", err)
		return nil, errors.New("error creating auth session")
	}

	currentRefreshSession, err := mr.Store.AuthSessions().Get(ctx, claims.RefreshTokenValue)
	if err != nil {
		mr.Logger.Printf("Error retrieving refresh session: %s", err)
		return nil, errors.New("invalid request, refresh token expired or malformed")
	}
	if currentRefreshSession == nil {
		mr.Logger.Printf("Error retrieving refresh session: %s", err)
		return nil, errors.New("invalid request, refresh token expired or malformed")
	}

	// 2. Create and store the refresh token (and remove any expired ones)
	refreshTokenValue := fmt.Sprintf("auth_refresh_tok:%s", xid.New().String())

	refreshSession, err := mr.Store.AuthSessions().Create(ctx, refreshTokenValue, user.ID)
	if err != nil {
		mr.Logger.Printf("Error creating refresh session: %s", err)
		return nil, errors.New("error creating auth session")
	}

	// 3. Create the JWT wrappers around refreshToken & accessToken

	refreshToken, err := mr.Auth.GenerateRefreshToken(user.ID, refreshSession.ID)
	if err != nil {
		mr.Logger.Printf("Error generating refresh token: %s", err)
		return nil, errors.New("error creating auth session")
	}

	accessToken, err := mr.Auth.GenerateAccessToken(user.ID)
	if err != nil {
		mr.Logger.Printf("Error generating access token: %s", err)
		return nil, errors.New("error creating auth session")
	}

	return &gen.AuthResult{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (mr *mutationResolver) AuthWithIdentityProvider(ctx context.Context, code string, kind gen.AuthIdentityKind) (*gen.AuthResult, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser != nil {
		return nil, errors.New("access forbidden, session already associated with a user")
	}

	// 1. Social identity validation

	var userMetadata *auth.AuthUserMetadata
	var err error

	switch kind {
	case gen.AuthIdentityKindGoogleOAuth2:
		userMetadata, err = mr.Auth.AuthorizationWithGoogle(ctx, code)
		if err != nil {
			mr.Logger.Printf("Error authorizing Google access code: %s", err)
			return nil, errors.New("invalid request, error authorizing google access code")
		}
	}

	// 2. Detect existing user

	var associatedUser *store.User
	var (
		googleIdentity *string
	)
	var finalUserID string

	switch kind {
	case gen.AuthIdentityKindGoogleOAuth2:
		googleIdentity = &userMetadata.Identifier
		associatedUser, err = mr.Store.Users().GetByGoogleIdentity(ctx, userMetadata.Identifier)
		if err != nil {
			mr.Logger.Printf("Error retrieving user through google identifier: %s", err)
		}
	}

	if associatedUser != nil { // user already registered, this is a login
		finalUserID = associatedUser.ID
	} else { // no existing user associated with the used social identity, register the user
		userID := fmt.Sprintf("%s_%s", "user", xid.New().String())
		userName := userDisplayNamePlaceholderDefault
		if userMetadata.DisplayName != nil && len(*userMetadata.DisplayName) > 0 {
			userName = *userMetadata.DisplayName
		}

		newUser, err := mr.Store.Users().Create(ctx, userID, userName, userMetadata.Email, store.UserRoleClient, googleIdentity)
		if err != nil {
			mr.Logger.Printf("Error creating user: %s", err)
			return nil, errors.New("error creating user")
		}

		// Register user with mail service if available
		if mr.MailService != nil {
			if err := mr.MailService.RegisterUser(ctx, newUser.ID, newUser.Email, newUser.DisplayName); err != nil {
				mr.Logger.Printf("Warning: Failed to register user %s with mail service: %v", newUser.ID, err)
			}
		}

		// Send notification for new user signup if available
		if mr.NotificationService != nil {
			if err := mr.NotificationService.NotifyNewUserSignup(ctx, newUser.Email, newUser.DisplayName, newUser.ID); err != nil {
				mr.Logger.Printf("Warning: Failed to send notification for user %s: %v", newUser.ID, err)
			}
		}

		finalUserID = newUser.ID
	}

	// 3. Create and store the refresh token (and remove any expired ones)

	err = mr.Store.AuthSessions().DeleteExpired(ctx, (time.Now().Add(-auth.RefreshTokenLifespanInHours * time.Hour)))
	if err != nil {
		mr.Logger.Printf("Error removing expired refresh session: %s", err)
		return nil, errors.New("error creating auth session")
	}

	refreshTokenValue := fmt.Sprintf("%s:%s", "auth_refresh_tok", xid.New().String())

	refreshSession, err := mr.Store.AuthSessions().Create(ctx, refreshTokenValue, finalUserID)
	if err != nil {
		mr.Logger.Printf("Error creating refresh session: %s", err)
		return nil, errors.New("error creating auth session")
	}

	// 4. Create the JWT wrappers around refreshToken & accessToken

	refreshToken, err := mr.Auth.GenerateRefreshToken(finalUserID, refreshSession.ID)
	if err != nil {
		mr.Logger.Printf("Error generating refresh token: %s", err)
		return nil, errors.New("error creating auth session")
	}

	accessToken, err := mr.Auth.GenerateAccessToken(finalUserID)
	if err != nil {
		mr.Logger.Printf("Error generating access token: %s", err)
		return nil, errors.New("error creating auth session")
	}

	return &gen.AuthResult{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

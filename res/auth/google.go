package auth

import (
	"context"
	"errors"

	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
)

var webSpecificRedirectMask oauth2.AuthCodeOption = oauth2.SetAuthURLParam("redirect_uri", "postmessage")

func (a *authImpl) AuthorizationWithGoogle(ctx context.Context, code string) (*AuthUserMetadata, error) {
	// Ref https://developers.google.com/identity/sign-in/web/server-side-flow
	// 1. Exchange auth code for access token, refresh token, and ID token

	tok, err := a.googleOAuth2Config.Exchange(ctx, code, webSpecificRedirectMask)
	if err != nil {
		return nil, err
	}

	// 2. Parse the ID token for user metadata

	idTokenString, ok := tok.Extra("id_token").(string)
	if !ok {
		return nil, nil
	}

	payload, err := idtoken.Validate(ctx, idTokenString, a.googleOAuth2Config.ClientID)
	if err != nil {
		return nil, nil
	}

	userMetadata := &AuthUserMetadata{}
	if accountIDVal, ok := payload.Claims["sub"].(string); ok {
		userMetadata.Identifier = accountIDVal
	}
	if userMetadata.Identifier == "" {
		return nil, errors.New("authorization with google: missing identifier")
	}

	if emailVal, ok := payload.Claims["email"].(string); ok {
		userMetadata.Email = emailVal
	}
	if userMetadata.Email == "" {
		return nil, errors.New("authorization with google: missing email")
	}

	if displayNameVal, ok := payload.Claims["given_name"].(string); ok {
		userMetadata.DisplayName = &displayNameVal
	}

	return userMetadata, nil
}

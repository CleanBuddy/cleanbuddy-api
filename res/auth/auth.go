package auth

import (
	"context"

	"github.com/golang-jwt/jwt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	AccessTokenLifespanInHours  = 24 * 3  // 3 days
	RefreshTokenLifespanInHours = 24 * 14 // 2 weeks
)

type AuthUserMetadata struct {
	Identifier  string
	Email       string
	DisplayName *string
}

type Auth interface {
	ValidateToken(token string, claims jwt.Claims) error

	GenerateAccessToken(userID string) (string, error)
	GenerateRefreshToken(userID, refreshTokenValue string) (string, error)

	AuthorizationWithGoogle(ctx context.Context, code string) (*AuthUserMetadata, error)
}

type authImpl struct {
	jwtPrivateKey string

	googleOAuth2Config oauth2.Config
}

func New(
	jwtSecret string,
	googleClientID, googleClientSecret, googleRedirectURL string,
) *authImpl {

	return &authImpl{
		jwtPrivateKey: jwtSecret,

		googleOAuth2Config: oauth2.Config{
			ClientID:     googleClientID,
			ClientSecret: googleClientSecret,
			RedirectURL:  googleRedirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.profile",
				"https://www.googleapis.com/auth/userinfo.email",
			},
			Endpoint: google.Endpoint,
		},
	}
}

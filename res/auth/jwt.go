package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt"
)

func (a *authImpl) ValidateToken(token string, claims jwt.Claims) error {
	t, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.jwtPrivateKey), nil
	})
	if err != nil {
		return err
	}
	if !t.Valid {
		return errors.New("invalid token")
	}

	return nil
}

type AccessTokenClaims struct {
	jwt.StandardClaims

	IsAccessToken bool   `json:"is_access_tok"`
	UserID        string `json:"user_id"`
}

func (a *authImpl) GenerateAccessToken(userID string) (string, error) {
	now := time.Now()
	token := jwt.New(jwt.SigningMethodHS256)

	token.Claims = AccessTokenClaims{
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  now.Unix(),
			ExpiresAt: now.Add(time.Duration(AccessTokenLifespanInHours) * time.Hour).Unix(),
		},
		IsAccessToken: true,
		UserID:        userID,
	}

	str, err := token.SignedString([]byte(a.jwtPrivateKey))
	if err != nil {
		return "", err
	}
	return str, nil
}

type RefreshTokenClaims struct {
	jwt.StandardClaims

	RefreshTokenValue string `json:"refresh_tok_val"`
	UserID            string `json:"user_id"`
}

func (a *authImpl) GenerateRefreshToken(userID, refreshTokenValue string) (string, error) {
	now := time.Now()
	token := jwt.New(jwt.SigningMethodHS256)

	token.Claims = RefreshTokenClaims{
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  now.Unix(),
			ExpiresAt: now.Add(time.Duration(RefreshTokenLifespanInHours) * time.Hour).Unix(),
		},
		RefreshTokenValue: refreshTokenValue,
		UserID:            userID,
	}

	str, err := token.SignedString([]byte(a.jwtPrivateKey))
	if err != nil {
		return "", err
	}
	return str, nil
}

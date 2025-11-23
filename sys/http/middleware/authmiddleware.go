package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"saas-starter-api/res/auth"
	"saas-starter-api/res/store"
)

// SESSION USER GETTER

type contextKey string

var contextKeyCurrentUser = contextKey("currentUser")

func GetCurrentUser(ctx context.Context) *store.User {
	if val := ctx.Value(contextKeyCurrentUser); val != nil {
		if currentUser, ok := val.(*store.User); ok {
			return currentUser
		}
	}

	return nil
}

func GetCurrentUserKey() contextKey {
	return contextKeyCurrentUser
}

// AUTH MIDDLEWARE

const authForbiddenCode = "FORBIDDEN"

func AuthMiddleware(logger *log.Logger, storeImpl store.Store, authImpl auth.Auth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headerVal := r.Header.Get("Authorization")

			if len(headerVal) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			headerValParts := strings.Split(headerVal, " ")
			if len(headerValParts) != 2 || !strings.EqualFold(headerValParts[0], "Bearer") {
				err := emitErrorResponse(w, "Malformed Authorization header", authForbiddenCode)
				if err != nil {
					logger.Printf("Error serializing graphQL response: %s", err)
				}
				return
			}

			var accessTokenClaims auth.AccessTokenClaims
			err := authImpl.ValidateToken(headerValParts[1], &accessTokenClaims)
			if err != nil {
				err := emitErrorResponse(w, "Invalid Authorization header", authForbiddenCode)
				if err != nil {
					logger.Printf("Error serializing graphQL response: %s", err)
				}
				return
			}

			currentUser, err := storeImpl.Users().Get(r.Context(), accessTokenClaims.UserID)
			if err != nil || currentUser == nil {
				err := emitErrorResponse(w, "Invalid Authorization header", authForbiddenCode)
				if err != nil {
					logger.Printf("Error serializing graphQL response: %s", err)
				}
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), contextKeyCurrentUser, currentUser))
			next.ServeHTTP(w, r)
		})
	}
}

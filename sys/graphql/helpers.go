package graphql

import (
	"context"
	"errors"
	"log"

	"saas-starter-api/res/store"
	"saas-starter-api/sys/http/middleware"
)

// requireAuth validates that a user is authenticated
// Returns the current user or an error if not authenticated
func requireAuth(ctx context.Context, logger *log.Logger) (*store.User, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		logger.Printf("Error: access forbidden, authorization required")
		return nil, errors.New("access forbidden, authorization required")
	}
	return currentUser, nil
}

// logAndReturnError logs an error message and returns a generic error to the client
// This prevents leaking internal error details to API consumers
func logAndReturnError(logger *log.Logger, logMsg string, err error, userMsg string) error {
	logger.Printf("%s: %s", logMsg, err)
	return errors.New(userMsg)
}

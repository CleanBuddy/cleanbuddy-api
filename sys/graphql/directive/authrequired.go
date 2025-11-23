package directive

import (
	"context"
	"errors"

	"saas-starter-api/sys/http/middleware"

	"github.com/99designs/gqlgen/graphql"
)

func AuthRequired(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser != nil {
		return next(ctx)
	}

	return nil, errors.New("access forbidden, auth required")
}

package playground

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/playground"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	h := playground.Handler("GraphQL playground", "/api")
	h.ServeHTTP(w, r)
}

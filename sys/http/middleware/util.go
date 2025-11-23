package middleware

import (
	"fmt"
	"net/http"
)

// Used to wrap error inside a GraphQL payload
func emitErrorResponse(w http.ResponseWriter, errorMsg, errorCode string) error {
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("X-Content-Type-Options", "nosniff")

	w.WriteHeader(http.StatusOK)
	size, err := fmt.Fprintf(
		w,
		"{\n\t\"errors\": [{\n\t\t\"message\": \"%s\",\n\t\t\"extensions\": { \"code\": \"%s\" }\n\t}],\n\t\"data\": null\n}",
		errorMsg, errorCode,
	)
	if err != nil {
		return err
	}
	if size == 0 {
		return fmt.Errorf("error writing response body")
	}

	return nil
}

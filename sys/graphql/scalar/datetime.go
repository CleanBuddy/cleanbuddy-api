package scalar

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

// MarshalDateTime marshals time.Time to RFC3339 string
func MarshalDateTime(t time.Time) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote(t.Format(time.RFC3339)))
	})
}

// UnmarshalDateTime unmarshals RFC3339 string to time.Time
func UnmarshalDateTime(v interface{}) (time.Time, error) {
	if tmpStr, ok := v.(string); ok {
		return time.Parse(time.RFC3339, tmpStr)
	}
	return time.Time{}, fmt.Errorf("DateTime should be RFC3339 formatted string")
}

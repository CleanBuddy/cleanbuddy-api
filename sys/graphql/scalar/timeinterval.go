package scalar

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

type TimeInterval [2]*time.Time

func MarshalTimeInterval(v TimeInterval) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		partA := "null"
		if v[0] != nil {
			partA = v[0].Format(time.RFC3339) // ISO 8601 format
		}

		partB := "null"
		if v[1] != nil {
			partB = v[1].Format(time.RFC3339) // ISO 8601 format
		}

		io.WriteString(w, "[")
		io.WriteString(w, strconv.Quote(partA))
		io.WriteString(w, ",")
		io.WriteString(w, strconv.Quote(partB))
		io.WriteString(w, "]")
	})
}

func UnmarshalTimeInterval(v interface{}) (TimeInterval, error) {
	parts, ok := v.([]interface{})
	if !ok || len(parts) != 2 {
		return TimeInterval{}, fmt.Errorf("TimeInterval: must have 2 parts")
	}

	var timeInterval TimeInterval
	var err error

	for i := range parts {
		if parts[i] != nil {
			if str, ok := parts[i].(string); ok {
				var t time.Time
				if t, err = time.Parse(time.RFC3339, str); err != nil {
					return TimeInterval{}, fmt.Errorf("TimeInterval (position %d): `%s` is not an ISO 8601 formatted string or null", i, str)
				}
				timeInterval[i] = &t
			} else {
				return TimeInterval{}, fmt.Errorf("TimeInterval (position %d): `%s` is not an ISO 8601 formatted string or null", i, str)
			}
		}
	}

	return timeInterval, nil
}

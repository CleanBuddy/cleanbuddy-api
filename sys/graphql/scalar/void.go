package scalar

import (
	"io"
	"strconv"

	"github.com/99designs/gqlgen/graphql"
)

type Void struct{}

func MarshalVoid(v Void) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote("@scalar.void"))
	})
}

// NOTE: scalar.Void should only be used for return signatures
func UnmarshalVoid(v interface{}) (Void, error) {
	return Void{}, nil
}

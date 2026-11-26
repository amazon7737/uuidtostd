package safe

import (
	"encoding"
	"fmt"

	"github.com/google/uuid" // want `github.com/google/uuid can be replaced by the standard library package uuid`
)

var _ fmt.Stringer = uuid.New()

var _ encoding.TextMarshaler = uuid.Nil

type record struct {
	id uuid.UUID
}

func generate() []uuid.UUID {
	newID := uuid.New
	return []uuid.UUID{
		uuid.New(),
		newID(),
		uuid.Must(uuid.NewRandom()),
		uuid.Must(uuid.NewV7()),
		uuid.Nil,
		uuid.Max,
		uuid.UUID{},
	}
}

func text(r record) string {
	b, _ := r.id.MarshalText()
	return fmt.Sprint(uuid.NewString(), r.id.String(), string(b))
}

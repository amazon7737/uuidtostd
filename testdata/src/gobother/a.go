package gobother

import "github.com/google/uuid"

type record struct {
	ID uuid.UUID
}

func newRecord() record { return record{ID: uuid.New()} }

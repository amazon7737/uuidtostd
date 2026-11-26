package exported

import "github.com/google/uuid"

type Item struct {
	ID uuid.UUID
}

func newItem() Item { return Item{ID: uuid.New()} }

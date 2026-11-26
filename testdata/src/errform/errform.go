package errform

import "github.com/google/uuid"

func id() (uuid.UUID, error) { return uuid.NewRandom() }

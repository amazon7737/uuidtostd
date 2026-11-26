//go:build go1.26

package oldversion

import "github.com/google/uuid"

func id() uuid.UUID { return uuid.New() }

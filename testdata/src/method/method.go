package method

import "github.com/google/uuid"

func v() uuid.Version { return uuid.New().Version() }

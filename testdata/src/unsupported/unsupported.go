package unsupported

import "github.com/google/uuid"

func a() uuid.UUID { return uuid.New() }

func b() uuid.UUID { return uuid.NewSHA1(uuid.NameSpaceDNS, []byte("x")) }

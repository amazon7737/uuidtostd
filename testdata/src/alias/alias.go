package alias

import guuid "github.com/google/uuid" // want `can be replaced by the standard library package uuid`

func id() guuid.UUID { return guuid.Must(guuid.NewRandom()) }

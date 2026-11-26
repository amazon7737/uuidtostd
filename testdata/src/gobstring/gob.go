package gobstring

import (
	"bytes"
	"encoding/gob"

	"github.com/google/uuid" // want `can be replaced by the standard library package uuid`
)

func enc() error { return gob.NewEncoder(new(bytes.Buffer)).Encode(uuid.New().String()) }

package gobuse

import (
	"bytes"
	"encoding/gob"

	"github.com/google/uuid"
)

func enc() error { return gob.NewEncoder(new(bytes.Buffer)).Encode(uuid.New()) }

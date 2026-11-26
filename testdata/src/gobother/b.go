package gobother

import (
	"bytes"
	"encoding/gob"
)

func enc(r record) error { return gob.NewEncoder(new(bytes.Buffer)).Encode(&r) }

package iface

import (
	"database/sql/driver"
	"encoding"

	"github.com/google/uuid"
)

func valuer() driver.Valuer { return uuid.New() }

func binary(m encoding.BinaryMarshaler) {}

func call() { binary(uuid.New()) }

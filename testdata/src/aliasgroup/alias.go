package aliasgroup

import (
	"fmt"

	"example.com/lib"
	guuid "github.com/google/uuid" // want `can be replaced by the standard library package uuid`
)

func id() string { return fmt.Sprint(lib.Name, guuid.New()) }

package nostd

import (
	"github.com/google/uuid" // want `can be replaced by the standard library package uuid`
	"golang.org/x/tools/go/analysis"
)

var _ *analysis.Analyzer

func id() uuid.UUID { return uuid.New() }

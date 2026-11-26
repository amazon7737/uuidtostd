package uuidtostd_test

import (
	"testing"

	"github.com/amazon7737/uuidtostd"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), uuidtostd.Analyzer,
		"safe", "alias", "parseonly", "mainpkg",
		"unsupported", "method", "exported", "gobuse", "errform", "addr", "oldversion", "multifile", "iface", "gobother", "gobstring",
	)
}

// Command uuidtostd runs the uuidtostd analyzer.
package main

import (
	"github.com/amazon7737/uuidtostd"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(uuidtostd.Analyzer)
}

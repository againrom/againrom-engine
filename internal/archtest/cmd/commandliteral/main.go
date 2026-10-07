// commandliteral prints a fresh CommandLiteralBaseline for
// internal/archtest/commandliteral_baseline.go by measuring the live tree, and
// lists the test files that hold the count so the number can be argued with.
package main

import (
	"fmt"
	"os"

	"againrom/internal/archtest"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	root, err := archtest.FindModuleRoot(wd)
	if err != nil {
		panic(err)
	}
	report, err := archtest.LoadCommandLiterals(root)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Tests: %d\n", report.Tests)
	for _, site := range report.Production {
		fmt.Printf("  production %s\n", site)
	}
	for _, f := range report.TestFiles {
		fmt.Printf("  %5d %s\n", f.Count, f.File)
	}
}

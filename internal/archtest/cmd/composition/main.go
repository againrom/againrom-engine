// composition prints a fresh CompositionBaseline for
// internal/archtest/composition_baseline.go by measuring the live tree, and
// lists every coordination point it counted so the number can be argued with.
package main

import (
	"fmt"
	"os"
	"strings"

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
	report, err := archtest.LoadComposition(root)
	if err != nil {
		panic(err)
	}
	for _, c := range report.Components {
		fmt.Printf("%-20s %3d\n", c.Name, c.Fields)
	}
	fmt.Printf("Fields:       %d\n", report.Fields)
	fmt.Printf("Coordinators: %d\n", len(report.Coordination))
	for _, f := range report.Coordination {
		fmt.Printf("  %-44s %-28s %s\n", f.File, f.Func, strings.Join(f.Components, " "))
	}
}

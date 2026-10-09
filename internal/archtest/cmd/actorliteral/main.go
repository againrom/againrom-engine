// actorliteral prints a fresh actor-literal baseline for
// internal/archtest/actorliteral_baseline.go by measuring the live tree, and
// lists every site so each count can be argued with.
package main

import (
	"fmt"
	"os"
	"sort"

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
	report, err := archtest.LoadActorLiterals(root)
	if err != nil {
		panic(err)
	}
	files := make([]string, 0, len(report.Files))
	for f := range report.Files {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, f := range files {
		fmt.Printf("\t%q: %d,\n", f, report.Files[f])
	}
	for _, site := range report.Sites {
		fmt.Printf("  site %s\n", site)
	}
}

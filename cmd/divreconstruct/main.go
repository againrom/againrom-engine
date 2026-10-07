// Command divreconstruct proves that splitting docs/DIVERGENCES.md into
// docs/divergences/ lost no row.
//
// It reads every row's full nine-cell tuple from an OLD single-file ledger
// (the pre-split shape, e.g. `git show 07cddcf:docs/DIVERGENCES.md`) and from
// a NEW split directory (`docs/divergences/*.md`), through the shared
// internal/divledger package both cmd/divcensus and
// pipeline/check-div-claims.sh's own algorithm agree with, then reports ids
// present on only one side and ids present on both whose cells differ.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"againrom/internal/divledger"
)

func byID(rows []divledger.Row) map[string]divledger.Row {
	m := make(map[string]divledger.Row, len(rows))
	for _, r := range rows {
		m[r.ID()] = r
	}
	return m
}

func sortedKeys(m map[string]divledger.Row) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func main() {
	oldPath := flag.String("old", "", "path to the old, pre-split single-file ledger (required)")
	newDir := flag.String("new", "docs/divergences", "directory of the new split ledger area files")
	flag.Parse()

	if *oldPath == "" {
		fmt.Fprintln(os.Stderr, "divreconstruct: -old is required (path to the pre-split single-file ledger)")
		os.Exit(2)
	}

	oldRows, err := divledger.ParseFile(*oldPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "divreconstruct: reading -old %s: %v\n", *oldPath, err)
		os.Exit(2)
	}
	newRows, err := divledger.ParseGlob(*newDir + "/*.md")
	if err != nil {
		fmt.Fprintf(os.Stderr, "divreconstruct: reading -new %s: %v\n", *newDir, err)
		os.Exit(2)
	}

	oldByID := byID(oldRows)
	newByID := byID(newRows)

	fmt.Printf("divreconstruct: old=%s (%d rows), new=%s/*.md (%d rows)\n\n",
		*oldPath, len(oldRows), *newDir, len(newRows))

	var onlyOld, onlyNew []string
	for id := range oldByID {
		if _, ok := newByID[id]; !ok {
			onlyOld = append(onlyOld, id)
		}
	}
	for id := range newByID {
		if _, ok := oldByID[id]; !ok {
			onlyNew = append(onlyNew, id)
		}
	}
	sort.Strings(onlyOld)
	sort.Strings(onlyNew)

	fmt.Printf("only in OLD (%d): %s\n", len(onlyOld), strings.Join(onlyOld, ", "))
	fmt.Printf("only in NEW (%d): %s\n", len(onlyNew), strings.Join(onlyNew, ", "))
	fmt.Println()

	type diff struct {
		id      string
		columns []string
	}
	var diffs []diff
	for _, id := range sortedKeys(oldByID) {
		nr, ok := newByID[id]
		if !ok {
			continue
		}
		or := oldByID[id]
		var changed []string
		for _, col := range divledger.Columns {
			if or.Cell(col) != nr.Cell(col) {
				changed = append(changed, col)
			}
		}
		if len(changed) > 0 {
			diffs = append(diffs, diff{id: id, columns: changed})
		}
	}

	fmt.Printf("cell-level differences on ids present in both (%d):\n", len(diffs))
	for _, d := range diffs {
		fmt.Printf("  %s: %s\n", d.id, strings.Join(d.columns, ", "))
	}
	fmt.Println()
	fmt.Printf("summary: %d only-old, %d only-new, %d common ids with cell diffs, %d common ids identical\n",
		len(onlyOld), len(onlyNew), len(diffs), len(oldByID)-len(onlyOld)-len(diffs))
}

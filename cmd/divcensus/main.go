// Command divcensus reports the state of the divergence ledger as counts.
//
// WHY IT IS A COMMITTED COMMAND AND NOT A SCRIPT SOMEBODY RAN ONCE. That
// ruling turns the ledger into a work list with a number attached, and the
// number is the only thing that says whether the list is getting shorter. A
// census quoted from a throwaway program is a number nobody can reproduce;
// this one falls out of a command in the tree.
//
// WHY THE OWNER-DIRECTIVE COLUMN WAS TAKEN AS THE BOUNDARY, AND WHY THAT
// READING IS NOT SUFFICIENT. The ruling's own boundary looks like a column: a
// row whose Owner directive cell is filled records a difference the owner
// chose, and it stays. The mouse wheel is the clearest case - ROM1 predates it,
// he asked for it, and "bring everything to the original" does not take it
// away. The first version of this tool therefore reported "ROM1 stated and no
// directive" as the actionable population and called it measured. See the
// amendment below: that cell also holds directives to FIX, so the reading is a
// lower bound, not a partition.
//
// THE COLUMN INDICES ARE READ FROM EACH SECTION'S OWN HEADER. The file has two
// tables and they need not stay identical. Assuming a column position is how a
// census silently reports on the wrong cell, so every lookup here goes through
// the header row that preceded it, and a section whose header lacks a column
// contributes empty strings rather than a panic or a shifted read.
//
// WHAT THIS TOOL MEASURES, AND WHAT IT ONLY APPROXIMATES. Two cells were
// read as if they carried a value; both carry free prose, and prose does not
// partition.
//
//  1. The Owner directive column carries TWO different meanings. DIV-101's
//     mouse wheel is "the owner chose this difference, and it stays". DIV-168
//     and DIV-169 are hotfix directives to make a screen match the original -
//     a request to FIX, which is the most actionable kind of row there is.
//     Both fill the same cell, and no mechanical test separates them. 27 of
//     the 53 directed rows carry imperative prose ("must", "should", "hotfix")
//     at the time of this amendment.
//  2. A filled ROM1 behaviour cell does not mean a behaviour is decoded. Four
//     rows fill it with a sentence that opens by saying no claim addresses the
//     subject. Those are research questions; the ruling has nothing to move
//     them to.
//
// So the two numbers that are measured are ROWS PARSED and OPEN. Everything
// below them reads prose and is reported here as a subset with its own count,
// so that a later reader can see the size of each approximation rather than
// inherit a single figure that hides both.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"againrom/internal/divledger"
)

// negativeROM1 matches a ROM1 behaviour cell that opens by stating research has
// said nothing about the subject. Such a cell is filled, so a cell-emptiness
// test counts it as decoded behaviour; its own first words say otherwise. The
// test is anchored at the start deliberately: a cell that decodes part of a
// subject and then names what is still open ("What no claim fixes is ...", as
// DIV-131 does) is a decoded row and must not match.
var negativeROM1 = regexp.MustCompile(`(?i)^\s*(no |none |nothing |not decoded|unknown\b|research (has not|does not|is silent))`)

// imperativeDirective matches an Owner directive cell whose prose reads as an
// instruction to fix rather than a record of a difference the owner chose. It
// is a PROSE HEURISTIC over free text and is reported as such: it cannot be
// checked against a cell value, because the two meanings share one column.
var imperativeDirective = regexp.MustCompile(`(?i)(hotfix|\bmust\b|\bshould\b|asked for|\bfix\b)`)

type row struct {
	section   string
	id        string
	subsystem string
	directive string
	rom1      string
	typ       string
	status    string
}

// visiblePrefixes drives one reported split and nothing else.
//
// IT IS A PREFIX HEURISTIC OVER THE SUBSYSTEM CELL, NOT A PROPERTY OF THE ROW,
// and it over-captures in at least one direction that is easy to demonstrate:
// "magic / Meteor Storm presentation" is something the player watches and
// "magic / area movement cost" is a simulation rule, and both start "magic /".
// Widening or narrowing this list moves the two visible/other numbers and
// changes no other count in this tool. So quote those two as a rough split with
// the list named, and never as a measured population. It is one of three prose
// heuristics here; the header names the other two and the output separates all
// three from the counts that read a cell as filled or empty.
var visiblePrefixes = []string{
	"client", "ui", "town", "school", "shop", "tavern",
	"inventory", "world map", "game", "chargen", "magic /",
}

// filled reports whether a table cell carries content. The ledger writes an
// em dash for "nothing here", which is a value and not a blank.
func filled(cell string) bool {
	switch strings.TrimSpace(cell) {
	case "", "-", "--", "—", "–":
		return false
	}
	return true
}

// toRows converts internal/divledger's own Row shape to this command's
// narrower one (the six fields the counts below read) and collects, per
// section, the column order the first row declaring that section used, for
// the startup report.
func toRows(dr []divledger.Row) ([]row, map[string][]string) {
	var rows []row
	headers := map[string][]string{}
	for _, r := range dr {
		if _, ok := headers[r.Section]; !ok {
			cols := make([]string, len(divledger.Columns))
			copy(cols, divledger.Columns)
			headers[r.Section] = cols
		}
		rows = append(rows, row{
			section:   r.Section,
			id:        r.Cell("ID"),
			subsystem: r.Cell("Subsystem"),
			directive: r.Cell("Owner directive"),
			rom1:      r.Cell("ROM1 behaviour (claims)"),
			typ:       r.Cell("Type"),
			status:    r.Cell("Status"),
		})
	}
	return rows, headers
}

// parse reads every ledger row matched by glob (e.g. "docs/divergences/*.md")
// through the shared internal/divledger package, so this tool's own idea of a
// cell boundary can never drift from pipeline/check-div-claims.sh's -- both
// split on a `|` NOT preceded by `\`.
func parse(glob string) ([]row, map[string][]string, error) {
	dr, err := divledger.ParseGlob(glob)
	if err != nil {
		return nil, nil, err
	}
	rows, headers := toRows(dr)
	return rows, headers, nil
}

func parseOne(path string) ([]row, map[string][]string, error) {
	dr, err := divledger.ParseFile(path)
	if err != nil {
		return nil, nil, err
	}
	rows, headers := toRows(dr)
	return rows, headers, nil
}

func visible(r row) bool {
	s := strings.ToLower(r.subsystem)
	for _, p := range visiblePrefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// ids renders a subset inline. A subset small enough to name is named, because
// a count alone cannot be checked against the ledger by the person reading it.
func ids(rows []row) string {
	if len(rows) == 0 {
		return " (none)"
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.id)
	}
	sort.Strings(out)
	return " " + strings.Join(out, ", ")
}

func counted(rows []row, key func(row) string) []string {
	n := map[string]int{}
	for _, r := range rows {
		k := key(r)
		if k == "" {
			k = "<blank>"
		}
		n[k]++
	}
	keys := make([]string, 0, len(n))
	for k := range n {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if n[keys[i]] != n[keys[j]] {
			return n[keys[i]] > n[keys[j]]
		}
		return keys[i] < keys[j]
	})
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%-18s %d", k, n[k]))
	}
	return out
}

func main() {
	dir := flag.String("dir", "docs/divergences", "directory of split ledger area files (docs/divergences/*.md)")
	file := flag.String("file", "", "parse this single ledger file instead of -dir (e.g. docs/DIVERGENCES-CLOSED.md, or an old pre-split single-file ledger)")
	list := flag.Bool("list", false, "list the actionable rows rather than only counting them")
	flag.Parse()

	glob := filepath.Join(*dir, "*.md")
	src := glob
	var (
		rows    []row
		headers map[string][]string
		err     error
	)
	if *file != "" {
		src = *file
		rows, headers, err = parseOne(*file)
	} else {
		rows, headers, err = parse(glob)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "divcensus: %v\n", err)
		os.Exit(2)
	}
	if len(rows) == 0 {
		// A zero here is the shape of a blind selector, not of a clean ledger.
		fmt.Fprintf(os.Stderr, "divcensus: parsed 0 rows from %s - wrong path, or the table shape changed\n", src)
		os.Exit(2)
	}

	fmt.Printf("divcensus: %s\n", src)
	fmt.Printf("  rows parsed: %d\n", len(rows))
	sections := []string{}
	for s := range headers {
		sections = append(sections, s)
	}
	sort.Strings(sections)
	for _, s := range sections {
		fmt.Printf("  section %-42q columns: %s\n", s, strings.Join(headers[s], ", "))
	}
	fmt.Println()

	var open []row
	for _, r := range rows {
		if strings.Contains(strings.ToUpper(r.status), "OPEN") {
			open = append(open, r)
		}
	}

	var actionable, directed, noROM1 []row
	for _, r := range open {
		switch {
		case filled(r.directive):
			directed = append(directed, r)
		case filled(r.rom1):
			actionable = append(actionable, r)
		default:
			noROM1 = append(noROM1, r)
		}
	}

	fmt.Println("MEASURED (a cell is filled or it is not):")
	fmt.Printf("  OPEN rows, both sections: %d\n", len(open))
	fmt.Printf("    ROM1 cell filled, owner-directive cell empty: %d\n", len(actionable))
	fmt.Printf("    owner-directive cell filled:                  %d\n", len(directed))
	fmt.Printf("    ROM1 cell empty:                              %d\n", len(noROM1))
	fmt.Println()

	// Both subsets below read prose. They are printed with their counts so the
	// size of each approximation is visible, rather than folded into one figure.
	var silent []row
	for _, r := range actionable {
		if negativeROM1.MatchString(r.rom1) {
			silent = append(silent, r)
		}
	}
	var toFix []row
	for _, r := range directed {
		if imperativeDirective.MatchString(r.directive) {
			toFix = append(toFix, r)
		}
	}

	fmt.Println("APPROXIMATE (reads free prose; the Owner directive column carries two meanings):")
	fmt.Printf("  of the %d with no directive, the ROM1 cell OPENS by saying no claim addresses it: %d\n", len(actionable), len(silent))
	fmt.Printf("    -> research questions, not work the 2026-08-21 ruling can act on:%s\n", ids(silent))
	fmt.Printf("  of the %d directed, the directive reads as a request to FIX, not to deviate:      %d\n", len(directed), len(toFix))
	fmt.Println("    -> DIV-168 and DIV-169 are the demonstrated cases: both are hotfix directives")
	fmt.Println("       to make a screen match the original, and both sit in the 'directed' bucket.")
	fmt.Printf("  so the ruling's work list is bounded below by %d and above by %d, and no\n", len(actionable)-len(silent), len(open)-len(silent))
	fmt.Println("  mechanical test narrows it further. Read the rows.")
	fmt.Println()

	var vis int
	for _, r := range actionable {
		if visible(r) {
			vis++
		}
	}
	fmt.Printf("Of the %d with no directive, by the subsystem-prefix heuristic:\n", len(actionable))
	fmt.Printf("  visible on screen:            %d\n", vis)
	fmt.Printf("  simulation/persistence/other: %d\n", len(actionable)-vis)
	fmt.Println()

	fmt.Println("rows with no directive, by type:")
	for _, l := range counted(actionable, func(r row) string { return r.typ }) {
		fmt.Printf("   %s\n", l)
	}

	if *list {
		fmt.Println()
		fmt.Println("actionable rows:")
		sort.Slice(actionable, func(i, j int) bool { return actionable[i].id < actionable[j].id })
		for _, r := range actionable {
			mark := " "
			if visible(r) {
				mark = "*"
			}
			fmt.Printf("  %s %-9s %-14s %s\n", mark, r.id, r.typ, r.subsystem)
		}
		fmt.Println("  (* = visible on screen)")
	}
}

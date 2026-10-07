// Package divledger reads a divergence-ledger table -- the `| DIV-NNNN | ... |`
// row format `docs/DIVERGENCES.md` (now an index) defines and every file under
// `docs/divergences/` carries -- the same way `pipeline/check-div-claims.sh`
// does, so a Go reader and that seat-owned shell script never disagree about
// where a cell boundary falls.
//
// WHY A SHARED PACKAGE AND NOT A SECOND COPY OF THE PARSER.
// `pipeline/check-div-claims.sh` is a seat-owned file this repository cannot
// edit, so its own `esplit` awk function is the reference implementation;
// Esplit below is the same algorithm, not a reimplementation from the row
// format's written description, so the two cannot drift apart by one of them
// reading the rule differently.
//
// DIV-237
package divledger

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Columns lists the nine cells every ledger row carries, in the order
// `docs/DIVERGENCES.md`'s own "The row format" table declares them.
var Columns = []string{
	"ID",
	"Subsystem",
	"Owner directive",
	"ROM1 behaviour (claims)",
	"Implemented behaviour",
	"Type",
	"Reason",
	"Revisit condition",
	"Status",
}

// IDPattern matches a divergence id anywhere in free text, for scanning a
// citing document (a `.go` comment, another `.md` file) rather than a ledger
// row itself. It intentionally allows three or four digits: the ledger's own
// ids run from DIV-001 into four digits, and a citation is never zero-padded
// to a fixed width.
var IDPattern = regexp.MustCompile(`DIV-[0-9]{3,4}`)

// Row is one parsed divergence-ledger row plus where it came from.
type Row struct {
	File    string // path Row was read from, as given to ParseFile
	Section string // the "## " heading in force when the row appeared, or ""
	Line    int    // 1-based line number within File
	Cells   map[string]string
}

// ID returns the row's own ID cell (its "DIV-" plus digits identifier).
func (r Row) ID() string { return r.Cells["ID"] }

// Cell returns a named cell's value, or "" for a column the row's own section
// header did not declare (a header that dropped a column reports absence
// rather than reading the wrong cell for it).
func (r Row) Cell(name string) string { return r.Cells[name] }

// Esplit splits a markdown table row on a `|` NOT preceded by `\`, matching
// `pipeline/check-div-claims.sh`'s own `esplit` awk function: an escaped pipe
// is parked on a byte no ledger row contains (U+0001) before the split and
// restored in every resulting field afterward. A `| a | b |`-shaped row
// yields a leading and trailing empty field the same way awk's FS="|" split
// does; ParseFile drops both, matching that script's own `nf - 2` cell count.
func Esplit(line string) []string {
	protected := strings.ReplaceAll(line, `\|`, "\x01")
	parts := strings.Split(protected, "|")
	for i, p := range parts {
		parts[i] = strings.ReplaceAll(p, "\x01", `\|`)
	}
	return parts
}

// cell trims a split field the way check-div-claims.sh's own `cell()` does.
func cell(s string) string {
	return strings.TrimSpace(s)
}

// ParseFile reads one ledger-format markdown file: zero or more `## ` section
// headings, and zero or more `| DIV-NNNN | ... |` data rows, each mapped onto
// Columns BY FIXED POSITION rather than by a header row read from the file.
//
// THIS WAS ONCE HEADER-DRIVEN, AND THE OLD SINGLE-FILE LEDGER IS WHY IT IS
// NOT ANY MORE. A first version of this function required a literal
// `| ID | ... |` header line before it would recognize any `| DIV-` row in
// the section that followed, the same way a rendered markdown table needs
// its header first. Running it against `07cddcf`'s pre-split
// `docs/DIVERGENCES.md` for `cmd/divreconstruct`'s own proof silently
// dropped 15 of 510 rows: that file's own "Divergences" section opens with a
// short, blank-line-separated run of individually spotlighted rows --
// `DIV-1329`, `DIV-1295`, `DIV-805`, `DIV-804`, ... down to `DIV-778` and
// `DIV-779` -- BEFORE its own `| ID | Subsystem | ... |` header line, which
// does not appear until several lines later. Those 15 rows are real,
// un-duplicated content (each id occurs exactly once in the file); they are
// simply placed ahead of the header that would otherwise license reading
// them. `pipeline/check-div-claims.sh` never has this problem, because it
// was never header-driven either: its own awk reads `Type` and `Status` from
// hardcoded field positions 7 and 10 regardless of whether its `want`
// variable (set only once, the FIRST time an `| ID |` line is seen in the
// WHOLE file, not per section) has been set yet. Matching that reference
// behaviour -- a fixed column order, not a discovered one -- is what fixes
// this: `docs/DIVERGENCES.md`'s own "row format" table already commits every
// reader to the same nine columns in the same order, so a row does not need
// its own file to repeat that order before it can be read.
//
// A `| ID | ... |` line is therefore never treated as structural; it is just
// prose that happens to start with `|` and is skipped like any other
// non-`| DIV-` line. A row whose own cell count does not equal len(Columns)
// is still reported as an error naming the file, line and count -- the same
// defect `check-div-claims.sh` refuses to scan past silently -- a shifted
// cell is worse than a missing row, since Status (which many callers gate
// on) would otherwise silently read a neighbouring cell instead.
func ParseFile(path string) ([]Row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var (
		rows    []Row
		section string
		lineNo  int
	)
	want := len(Columns)

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<22)
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		if strings.HasPrefix(line, "## ") {
			section = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			continue
		}
		if !strings.HasPrefix(strings.TrimSpace(line), "| DIV-") {
			continue
		}
		fields := Esplit(line)
		got := len(fields) - 2
		if got != want {
			return nil, fmt.Errorf("divledger: %s:%d has %d cells, expected %d (Columns)",
				path, lineNo, got, want)
		}
		cells := make(map[string]string, want)
		for i, name := range Columns {
			cells[name] = cell(fields[i+1])
		}
		rows = append(rows, Row{File: path, Section: section, Line: lineNo, Cells: cells})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return rows, nil
}

// ParseGlob parses every file matching pattern (filepath.Glob), in sorted
// path order, and concatenates their rows. Every area file under
// `docs/divergences/` shares this row format, so scanning the whole
// directory this way is how `cmd/divcensus` and this package's own guard
// tests read the split ledger as one logical table.
func ParseGlob(pattern string) ([]Row, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	if len(matches) == 0 {
		return nil, fmt.Errorf("divledger: glob %q matched no file", pattern)
	}
	var all []Row
	for _, m := range matches {
		rs, err := ParseFile(m)
		if err != nil {
			return nil, err
		}
		all = append(all, rs...)
	}
	return all, nil
}

// FindIDs returns every distinct DIV-NNNN id IDPattern matches in text, in
// first-seen order.
func FindIDs(text string) []string {
	matches := IDPattern.FindAllString(text, -1)
	seen := make(map[string]bool, len(matches))
	var out []string
	for _, m := range matches {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}

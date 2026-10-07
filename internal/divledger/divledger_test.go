package divledger

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// repoRoot resolves the module root from the test binary's own working
// directory (`go test` runs a package's tests with that package's own
// directory as cwd), not from runtime.Caller: -trimpath -- which
// `go test -trimpath -count=1 ./...` always passes -- rewrites source paths
// embedded in debug info to their module-relative form, so
// runtime.Caller(0) does not return a real filesystem path to walk from
// under that flag.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("divledger: os.Getwd: %v", err)
	}
	// this package is internal/divledger
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

const splitAreaCeiling = 60000

// TestLedgerRowsUnique checks that every DIV-NNNN id across
// docs/divergences/*.md is unique.
func TestLedgerRowsUnique(t *testing.T) {
	root := repoRoot(t)
	rows, err := ParseGlob(filepath.Join(root, "docs", "divergences", "*.md"))
	if err != nil {
		t.Fatalf("ParseGlob: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("divledger: parsed 0 rows from docs/divergences/*.md")
	}
	seen := map[string]Row{}
	var dups []string
	for _, r := range rows {
		id := r.ID()
		if id == "" {
			t.Errorf("%s:%d: row has no ID cell", r.File, r.Line)
			continue
		}
		if prior, ok := seen[id]; ok {
			dups = append(dups, id+": "+prior.File+" and "+r.File)
			continue
		}
		seen[id] = r
	}
	if len(dups) > 0 {
		sort.Strings(dups)
		t.Fatalf("divledger: %d duplicate id(s) across docs/divergences/*.md:\n  %s",
			len(dups), strings.Join(dups, "\n  "))
	}
}

func TestLedgerAreaFileSizeCeiling(t *testing.T) {
	root := repoRoot(t)
	matches, err := filepath.Glob(filepath.Join(root, "docs", "divergences", "*.md"))
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("divledger: glob matched no file under docs/divergences/")
	}
	var over []string
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil {
			t.Fatalf("Stat %s: %v", m, err)
		}
		if info.Size() > splitAreaCeiling {
			over = append(over, filepath.Base(m))
		}
	}
	if len(over) > 0 {
		sort.Strings(over)
		t.Fatalf("divledger: %d file(s) exceed the %d-byte split ceiling: %s",
			len(over), splitAreaCeiling, strings.Join(over, ", "))
	}
}

// DIV-1261
var preExistingCitationGaps = map[string]bool{
	"DIV-067":  true,
	"DIV-101":  true,
	"DIV-223":  true,
	"DIV-1262": true,
}

// skipCitationDirs names top-level directories under the module root this
// test does not walk for .go files: .git carries no source, and knowledge/
// is a separate pinned submodule (the public research snapshot) whose own
// .go tools are not this repository's citations of this repository's own
// ledger.
var skipCitationDirs = map[string]bool{
	".git":      true,
	"knowledge": true,
}

// TestLedgerCitationsResolve checks that every DIV-NNNN id referenced in a
// .go file under the module resolves to exactly one row in the union of
// docs/divergences/*.md and docs/DIVERGENCES-CLOSED.md, except the
// documented preExistingCitationGaps above. This is the guard
// docs/DIVERGENCES.md's own "Escaping" section points to: a citation
// resolving to zero rows, or to more than one across the split files, is
// exactly the failure a naive, non-escape-aware split could introduce
// silently.
func TestLedgerCitationsResolve(t *testing.T) {
	root := repoRoot(t)

	live, err := ParseGlob(filepath.Join(root, "docs", "divergences", "*.md"))
	if err != nil {
		t.Fatalf("ParseGlob(docs/divergences): %v", err)
	}
	closedPath := filepath.Join(root, "docs", "DIVERGENCES-CLOSED.md")
	closed, err := ParseFile(closedPath)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", closedPath, err)
	}

	known := map[string]int{} // id -> count, to catch a split id landing in two files
	for _, r := range live {
		known[r.ID()]++
	}
	for _, r := range closed {
		known[r.ID()]++
	}
	var multi []string
	for id, n := range known {
		if n > 1 {
			multi = append(multi, id)
		}
	}
	if len(multi) > 0 {
		sort.Strings(multi)
		t.Fatalf("divledger: %d id(s) appear in more than one ledger row across the split files and DIVERGENCES-CLOSED.md: %s",
			len(multi), strings.Join(multi, ", "))
	}

	cited := map[string][]string{} // id -> citing file(s)
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipCitationDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		for _, id := range FindIDs(string(data)) {
			cited[id] = append(cited[id], rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}

	var unresolved []string
	for id, files := range cited {
		if known[id] > 0 {
			continue
		}
		if preExistingCitationGaps[id] {
			continue
		}
		unresolved = append(unresolved, id+" (cited by "+strings.Join(files, ", ")+")")
	}
	if len(unresolved) > 0 {
		sort.Strings(unresolved)
		t.Fatalf("divledger: %d .go citation(s) resolve to no ledger row and are not in preExistingCitationGaps:\n  %s",
			len(unresolved), strings.Join(unresolved, "\n  "))
	}

	// Every documented gap must still actually be a gap. If a later story adds
	// the missing row, the allowlist itself goes stale in the direction that
	// hides nothing (a resolved id sitting in the allowlist is inert), but a
	// stale entry naming an id no .go file cites any more is worth surfacing
	// so the allowlist does not silently outlive its own reason.
	for id := range preExistingCitationGaps {
		if _, ok := cited[id]; !ok {
			t.Logf("divledger: preExistingCitationGaps names %s, but no .go file cites it any more; consider removing the entry", id)
		}
	}
}

var experimentID = regexp.MustCompile(`EXP-[0-9]{4}`)

func TestLedgerRowsCiteNoExperiment(t *testing.T) {
	root := repoRoot(t)
	rows, err := ParseGlob(filepath.Join(root, "docs", "divergences", "*.md"))
	if err != nil {
		t.Fatalf("ParseGlob: %v", err)
	}
	data, err := os.ReadFile(filepath.Join("testdata", "experiment-citation-allowlist.txt"))
	if err != nil {
		t.Fatalf("allow-list: %v", err)
	}
	allowed := map[string]bool{}
	for _, f := range strings.Fields(string(data)) {
		allowed[f] = true
	}
	cites := map[string]bool{}
	for _, r := range rows {
		for _, name := range Columns {
			if experimentID.MatchString(r.Cell(name)) {
				cites[r.ID()] = true
			}
		}
	}
	for id := range cites {
		if !allowed[id] {
			t.Errorf("%s names an experiment id; cite the promoted claim instead", id)
		}
	}
	for id := range allowed {
		if !cites[id] {
			t.Errorf("%s no longer names an experiment id; remove it from the allow-list", id)
		}
	}
}

// TestIndexListsEveryAreaFile checks that the Area files table in
// docs/DIVERGENCES.md names exactly the files under docs/divergences/, so a
// file split at the size ceiling cannot ship without its index entry.
func TestIndexListsEveryAreaFile(t *testing.T) {
	root := repoRoot(t)
	matches, err := filepath.Glob(filepath.Join(root, "docs", "divergences", "*.md"))
	if err != nil || len(matches) == 0 {
		t.Fatalf("divledger: glob docs/divergences/*.md: %v (%d files)", err, len(matches))
	}
	data, err := os.ReadFile(filepath.Join(root, "docs", "DIVERGENCES.md"))
	if err != nil {
		t.Fatalf("divledger: %v", err)
	}
	listed := map[string]bool{}
	for _, m := range regexp.MustCompile(`docs/divergences/([a-z0-9-]+[.]md)`).FindAllStringSubmatch(string(data), -1) {
		listed[m[1]] = true
	}
	present := map[string]bool{}
	for _, m := range matches {
		name := filepath.Base(m)
		present[name] = true
		if !listed[name] {
			t.Errorf("%s is not listed in docs/DIVERGENCES.md Area files", name)
		}
	}
	for name := range listed {
		if !present[name] {
			t.Errorf("docs/DIVERGENCES.md lists %s, which does not exist", name)
		}
	}
}

package archtest

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// heldRowsHome holds heldRows and editRows, the one page step.
const heldRowsHome = "mapeditormodterrain.go"

// heldRowFindings reports each pkg/game source (by file name) that parses
// a held kind's file or reads or writes its row outside the one step.
func heldRowFindings(files map[string]string) []string {
	var out []string
	for _, name := range slices.Sorted(maps.Keys(files)) {
		src := files[name]
		for _, call := range []string{"mod.ParseTerrain(", "mod.ParseItems(", "mod.ParseCharacters(", "mod.ParseStrings("} {
			if strings.Contains(src, call) {
				out = append(out, name+" calls "+call+"; a page reads its kind through heldRows")
			}
		}
		if name == heldRowsHome {
			continue
		}
		for _, call := range []string{".ReadHeld(", "mod.WrittenKind("} {
			if strings.Contains(src, call) {
				out = append(out, name+" calls "+call+"; a page edits its kind through editRows")
			}
		}
	}
	if home, ok := files[heldRowsHome]; ok && (strings.Count(home, ".ReadHeld(") != 1 || strings.Count(home, "mod.WrittenKind(") != 2) {
		out = append(out, heldRowsHome+" holds other than one held read and one write step")
	}
	return out
}

func gameSources(t *testing.T) map[string]string {
	t.Helper()
	dir := filepath.Join(moduleRootForTest(t), "pkg", "game")
	names, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, path := range names {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		files[filepath.Base(path)] = string(b)
	}
	return files
}

func TestModPagesReadAndWriteTheirKindThroughOneStep(t *testing.T) {
	for _, f := range heldRowFindings(gameSources(t)) {
		t.Error(f)
	}
}

func TestModPageSecondStepIsFound(t *testing.T) {
	files := gameSources(t)
	files["mapeditormoditems.go"] += "\nfunc x() { mod.ParseItems(\"\", \"\", nil, nil, \"\"); mod.WrittenKind(\"\") }\n"
	got := strings.Join(heldRowFindings(files), "\n")
	for _, want := range []string{"mapeditormoditems.go calls mod.ParseItems(", "mapeditormoditems.go calls mod.WrittenKind("} {
		if !strings.Contains(got, want) {
			t.Errorf("findings %q lack %q", got, want)
		}
	}
}

package ui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"againrom/pkg/words"
)

// retiredAddRoutes are the names of the add routes the Add window replaced:
// one prompt or pick per kind and the per-list add keys.
var retiredAddRoutes = []string{"promptNewSack", "promptNewUnit", "promptNewUnitTier", "promptNewStructure",
	"promptNewScript", "pickSack", "pickUnit", "pickStructure", "keyPlayer", "addScriptKey", "askUnitTier",
	"openAdd", "openAddRun", "promptAdd", "addSpec", "AddStep", "newRecordField", "InspectionUnitTiers", "InspectionScriptAuthor", "InspectionPlayers"}

// retiredRoutesIn lists each non-test Go file of dir naming a retired route.
func retiredRoutesIn(t *testing.T, dir string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	var hits []string
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range retiredAddRoutes {
			if regexp.MustCompile(`(^|[^A-Za-z0-9_])` + name + `($|[^A-Za-z0-9_])`).Match(b) {
				hits = append(hits, filepath.Base(f)+": "+name)
			}
		}
	}
	return hits
}

// Every kind a palette or command adds is a data row of the one builder: its
// title and step labels are words of both tables, every step is one of the
// builder's controls, every list view that adds names its row, and no
// per-kind prompt or pick of the old add routes is left in the editor.
func TestMapEditorAddKindsAreRowsOfOneBuilder(t *testing.T) {
	ru, err := words.Load(words.Engine(), "ru")
	if err != nil {
		t.Fatal(err)
	}
	en := words.For("")
	has := func(id string) {
		t.Helper()
		if en.Text(id) == id || ru.Text(id) == id {
			t.Errorf("word %s is not in both tables", id)
		}
	}
	filters := map[int]string{}
	for _, k := range addKindForms {
		has(k.Word)
		for _, st := range k.Steps {
			has(st.Word)
			if st.Key == "" || st.Control > addMany {
				t.Errorf("%s step %+v is not a builder step", k.Kind, st)
			}
		}
		for _, f := range k.Filters {
			if filters[f] != "" {
				t.Errorf("list %d adds %s and %s", f, filters[f], k.Kind)
			}
			filters[f] = string(k.Kind)
		}
	}
	want := map[int]string{1: "Unit", 2: "Structure", 3: "Sack", editorPlayersFilter: "Player"}
	for f, kind := range want {
		if filters[f] != kind {
			t.Errorf("list %d adds %q, want %q", f, filters[f], kind)
		}
		e, _ := newModesEditor(t, "full")
		e.showRecords(f)
		_, add := e.FindControl(ControlQuery{Role: ControlListButtonKey, Key: "add"})
		if add != (f == editorPlayersFilter) {
			t.Errorf("list %d shows Add = %v", f, add)
		}
	}
	if hits := retiredRoutesIn(t, "."); len(hits) > 0 {
		t.Fatal("old add routes remain:", hits)
	}
	// Loss control: the scan finds a retired route where one stands.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "old.go"), []byte("package ui\nconst promptNewUnit = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if hits := retiredRoutesIn(t, dir); len(hits) != 1 {
		t.Fatal("the scan misses a retired route:", hits)
	}
}

func TestMapEditorKeysOpenPlacementPalettes(t *testing.T) {
	for _, route := range []struct {
		key    string
		filter int
	}{{"unit", 1}, {"structure", 2}, {"sack", 3}, {"casts", editorCellSpellsFilter}} {
		e, calls := newModesEditor(t, "full")
		if err := e.Key(route.key); err != nil {
			t.Fatal(err)
		}
		if e.filter != route.filter || !e.listShown || e.prompt.open() || len(calls.lines) != 0 {
			t.Fatalf("%s did not open its palette without writing", route.key)
		}
		_ = e.Key("player")
		if len(calls.lines) != 0 || e.prompt.open() {
			t.Fatalf("A on %s started an edit", route.key)
		}
	}
}

func addKindNow(e *MapEditor) string {
	if r, ok := e.addRunning(); ok {
		return string(r.kind)
	}
	return ""
}

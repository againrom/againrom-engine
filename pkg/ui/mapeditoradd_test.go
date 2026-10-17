package ui

import (
	"image"
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
	"newRecordField", "InspectionUnitTiers", "InspectionScriptAuthor", "InspectionPlayers"}

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

// Every kind the Add window adds is a data row of the one builder: its
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
	for _, k := range addKinds {
		has(k.Word)
		for _, st := range k.Steps {
			has(st.Word)
			if st.Key == "" || st.Control > addKind {
				t.Errorf("%s step %+v is not a builder step", k.Kind, st)
			}
			for _, next := range st.Kinds {
				row, ok := addKindOf(next)
				if !ok {
					t.Errorf("%s offers kind %s, no row", k.Kind, next)
				}
				has(row.ChoiceWord)
			}
		}
		for _, f := range k.Filters {
			if filters[f] != "" {
				t.Errorf("list %d adds %s and %s", f, filters[f], k.Kind)
			}
			filters[f] = k.Kind
		}
	}
	want := map[int]string{1: "Unit", 2: "Structure", 3: "Sack", 7: "Trigger", editorPlayersFilter: "Player", editorScriptsFilter: "Script", editorGroupsFilter: "Group"}
	for f, kind := range want {
		if filters[f] != kind {
			t.Errorf("list %d adds %q, want %q", f, filters[f], kind)
		}
		e, _ := newModesEditor(t, "full")
		e.showRecords(f)
		s, ok := e.listSpec()
		if !ok || len(s.Buttons) != 1 || s.Buttons[0] != e.word("editor.button.add") {
			t.Errorf("list %d shows no Add button: %v", f, s.Buttons)
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

// addKindNow is the kind the open Add window adds, "" for none.
func addKindNow(e *MapEditor) string {
	if r, ok := e.addRunning(); ok {
		return r.kind
	}
	return ""
}

// Every add route opens the one Add window: the old keys, the A key of each
// list, each list's Add button and the trigger popup's New button.
func TestMapEditorEveryAddRouteOpensTheAddWindow(t *testing.T) {
	for _, c := range []struct{ key, kind string }{{"unit", "Unit"}, {"structure", "Structure"}, {"sack", "Sack"}} {
		e, calls := newModesEditor(t, "full")
		_ = e.Key(c.key)
		if addKindNow(e) != c.kind || len(calls.lines) != 0 {
			t.Errorf("key %s opened %q, wrote %v", c.key, addKindNow(e), calls.lines)
		}
	}
	for f, kind := range map[int]string{1: "Unit", 2: "Structure", 3: "Sack", 7: "Trigger", editorPlayersFilter: "Player", editorScriptsFilter: "Script", editorGroupsFilter: "Group"} {
		e, _ := newModesEditor(t, "full")
		e.showRecords(f)
		_ = e.Key("player")
		if addKindNow(e) != kind {
			t.Errorf("A on list %d opened %q, want %s", f, addKindNow(e), kind)
		}
		e.leave(leavePrompt)
		e.objectAt = image.Pt(0, e.window.Y) // the selected player's popup stands clear of the list
		s, _ := e.listSpec()
		modesClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Buttons[0]))
		if addKindNow(e) != kind {
			t.Errorf("the Add button of list %d opened %q, want %s", f, addKindNow(e), kind)
		}
	}
	e, _ := newModesEditor(t, "full")
	e.newTrigger()
	if addKindNow(e) != "Trigger" {
		t.Error("the trigger popup's New opened", addKindNow(e))
	}
	plain, _ := newModesEditor(t, "plain")
	_ = plain.Key("unit")
	if addKindNow(plain) != "" {
		t.Error("a host without the Add seam opened the window")
	}
}

// addAdvance answers up to n steps of the open window with their defaults
// and reports whether the window is still open.
func addAdvance(t *testing.T, e *MapEditor, n int) bool {
	t.Helper()
	for i := 0; i < n; i++ {
		r, ok := e.addRunning()
		if !ok {
			return false
		}
		switch r.steps[r.at].Control {
		case addCell:
			e.prompt.path = "25,20"
		case addMany:
			e.addTake(0)
			_ = e.Key("enter")
		}
		_ = e.Key("enter")
		if r.note != "" {
			t.Fatalf("step %d refused: %s", i, r.note)
		}
	}
	_, ok := e.addRunning()
	return ok
}

// Leaving the Add window at any step, by Esc, Cancel or the close button,
// writes nothing; finishing writes once.
func TestMapEditorAddCancelAtEveryStepWritesNothing(t *testing.T) {
	ways := map[string]func(e *MapEditor){
		"escape": func(e *MapEditor) { _ = e.Key("escape") },
		"cancel": func(e *MapEditor) {
			s, _ := e.addSpec()
			modesClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Buttons[2]))
		},
		"close": func(e *MapEditor) {
			s, _ := e.addSpec()
			modesClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Close))
		},
	}
	for _, k := range addKinds {
		if len(k.Steps) == 0 {
			continue
		}
		for at := 0; at < 16; at++ {
			probe, _ := newModesEditor(t, "full")
			probe.openAdd(k.Kind)
			if !addAdvance(t, probe, at) {
				break // at is past the last step
			}
			for way, leave := range ways {
				e, calls := newModesEditor(t, "full")
				e.openAdd(k.Kind)
				addAdvance(t, e, at)
				leave(e)
				if addKindNow(e) != "" || e.prompt.open() || len(calls.lines) != 0 {
					t.Errorf("%s: %s at step %d left %q open or wrote %v", k.Kind, way, at, addKindNow(e), calls.lines)
				}
			}
		}
		e, calls := newModesEditor(t, "full")
		e.openAdd(k.Kind)
		for n := 0; n < 16 && addKindNow(e) != ""; n++ {
			addAdvance(t, e, 1)
		}
		if addKindNow(e) != "" || len(calls.lines) != 1 || !strings.HasPrefix(calls.lines[0], "AddRecord ") {
			t.Errorf("%s: finishing wrote %v", k.Kind, calls.lines)
		}
	}
}

// The cell step's pick button hands the next map click to the window; the
// clicked cell becomes the step's value and the window shows again. Esc while
// picking returns to the window.
func TestMapEditorAddCellPickedOnTheMap(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	_ = e.Key("unit")
	addAdvance(t, e, 2) // class, owner; the modes host offers no tier
	if key, _, _ := e.AddStep(); key != "cell" {
		t.Fatal("the window asks", key)
	}
	pick := func() {
		s, _ := e.addSpec()
		l := layoutPopup(s, e.font, e.bodyFont())
		for _, r := range l.Rows {
			if len(r.Buttons) > 0 {
				modesClick(e, center(r.Buttons[0]))
				return
			}
		}
		t.Fatal("the cell step has no pick button")
	}
	pick()
	if !e.addPicking() || e.helpScreen() != screenAddCell {
		t.Fatal("the pick button did not arm the cell pick")
	}
	if s, ok := e.promptSpec(); ok || len(s.Rows) != 0 {
		t.Fatal("the window covers the map while picking")
	}
	_ = e.Key("escape")
	if e.addPicking() || addKindNow(e) != "Unit" {
		t.Fatal("Esc while picking did not return to the window")
	}
	pick()
	e.Fit()
	modesClick(e, e.CellPoint(image.Pt(25, 20)))
	if e.addPicking() || e.prompt.path != "25,20" || len(calls.lines) != 0 {
		t.Fatal("the clicked cell is not the step's value", e.prompt.path, calls.lines)
	}
	_ = e.Key("enter")
	if addKindNow(e) != "" || len(calls.lines) != 1 || !strings.Contains(calls.lines[0], "AddRecord Unit (25,20) true") {
		t.Fatal("the picked cell did not reach the add", calls.lines)
	}
	if kind, index := e.selectedRecord(); kind != "Unit" || index < 0 {
		t.Fatal("the new record's popup is not open", kind, index)
	}
}

// tierHost offers a tier step with three tiers and refuses a cell off the
// left half of the map.
type tierHost struct{ *modesFull }

func (h *tierHost) NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep {
	if key == "tier" {
		return InspectionNewStep{Choices: []InspectionChoice{{Value: 1, Label: "Tier 1"}, {Value: 2, Label: "Tier 2"}, {Value: 3, Label: "Tier 3"}}, Value: 1}
	}
	return h.modesFull.NewRecordStep(r, key)
}

// The tier step shows when the host offers two or more tiers; Back walks the
// asked steps and keeps their values; an invalid value says why and stays.
func TestMapEditorAddTierBackAndRefusal(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	host := &tierHost{e.doc.Edits.(*modesFull)}
	host.self, e.doc.Edits = host, host
	_ = e.Key("unit")
	addAdvance(t, e, 1)
	if key, _, _ := e.AddStep(); key != "tier" {
		t.Fatal("the tier step is not asked:", key)
	}
	_ = e.Key("down")
	_ = e.Key("down")
	if e.prompt.path != "3" || addFocus(t, e).Value != "Tier 3" {
		t.Fatal("Down did not walk the tiers", e.prompt.path)
	}
	_ = e.Key("enter")
	s, _ := e.addSpec()
	modesClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Buttons[0]))
	if key, _, _ := e.AddStep(); key != "tier" || e.prompt.path != "3" {
		t.Fatal("Back did not return to the tier with its value", key, e.prompt.path)
	}
	addAdvance(t, e, 2)
	for _, bad := range []string{"", "x", "60,1", "-1,0"} {
		e.prompt.path = bad
		_ = e.Key("enter")
		if r, _ := e.addRunning(); r.note != e.word("editor.add.cellbad") || len(calls.lines) != 0 {
			t.Fatal("a bad cell was taken:", bad, r.note, calls.lines)
		}
	}
	e.prompt.path = "1,1"
	_ = e.Key("enter")
	if len(calls.lines) != 1 || !strings.Contains(calls.lines[0], "{tier 3 false  false}") {
		t.Fatal("the tier did not reach the add", calls.lines)
	}
}

// A many step adds each value once, removes one by its button, and does not
// go on empty. A group adds its members; a script record names its kind
// first and then asks that kind's steps.
func TestMapEditorAddManyGroupAndScriptKinds(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	_ = e.Key("sack")
	addAdvance(t, e, 1)
	_ = e.Key("enter")
	if r, _ := e.addRunning(); r.note != e.word("editor.add.atleast") {
		t.Fatal("an empty sack went on", r.note)
	}
	for _, v := range []string{"1", "2", "2"} {
		e.prompt.path = v
		_ = e.Key("enter")
	}
	r, _ := e.addRunning()
	if len(r.many[r.at]) != 2 {
		t.Fatal("the items are not each taken once", r.many[r.at])
	}
	s, acts := e.addSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	for i, a := range acts {
		if a.button == "remove" && a.item == 0 {
			modesClick(e, center(l.Rows[i].Buttons[0]))
			break
		}
	}
	if len(r.many[r.at]) != 1 || r.many[r.at][0] != 2 {
		t.Fatal("Remove did not drop the first item", r.many[r.at])
	}
	_ = e.Key("enter")
	if len(calls.lines) != 1 || !strings.Contains(calls.lines[0], "AddRecord Sack (25,20) true [{item 2 false  false}]") {
		t.Fatal("the sack add is wrong", calls.lines)
	}

	e, calls = newModesEditor(t, "full")
	e.showRecords(editorGroupsFilter)
	_ = e.Key("player")
	for n := 0; n < 8 && addKindNow(e) != ""; n++ {
		addAdvance(t, e, 1)
	}
	if len(calls.lines) != 1 || !strings.Contains(calls.lines[0], "AddRecord Group (0,0) false [{owner 1 false  false} {group 1 false  false} {member 1 false  false}]") {
		t.Fatal("the group add is wrong", calls.lines)
	}

	e, calls = newModesEditor(t, "full")
	e.showRecords(editorScriptsFilter)
	_ = e.Key("player")
	e.prompt.path = "2"
	_ = e.Key("enter")
	if r, _ := e.addRunning(); r.kind != "Trigger" {
		t.Fatal("the kind step did not go on as a trigger", r.kind)
	}
	for n := 0; n < 8 && addKindNow(e) != ""; n++ {
		addAdvance(t, e, 1)
	}
	if len(calls.lines) != 1 || !strings.Contains(calls.lines[0], `AddRecord Trigger (0,0) false [{name 0 true New false} {once 1 false  false}]`) {
		t.Fatal("the trigger add is wrong", calls.lines)
	}
}

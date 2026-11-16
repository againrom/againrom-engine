package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

// addInputs are the values each kind's Add window is given on one
// installed map.
type addInputs struct {
	key      string
	cell     image.Point
	class    int16
	tier     uint16
	identity int64
	kind     int64
	items    []int64
	members  []int
	group    int64
}

// addInputsOf picks the inputs: the first installed map with two players
// whose first player owns two units; the first cell, row by row from (4,4),
// that takes the first structure kind; the four-tier class and its third
// tier; the first two item codes; player 1's first two units and the lowest
// group number no unit of the map carries.
func addInputsOf(t *testing.T, x *MapInspector) addInputs {
	t.Helper()
	var in addInputs
	class, tiers := fourTierClass(t, x)
	in.class, in.tier = class, tiers[2].Face
	_, in.identity = creatureIdentity(tiers[2])
	keys, maps := installedMapBytes(t, x)
	for n, b := range maps {
		m, err := alm.Open(b)
		if err != nil || len(m.Groups) < 2 {
			continue
		}
		var members []int
		used := map[int64]bool{}
		for i, u := range m.Units {
			used[int64(u.GroupID)] = true
			if u.Owner == 1 && len(members) < 2 {
				members = append(members, i)
			}
		}
		free := int64(1)
		for used[free] {
			free++
		}
		if len(members) < 2 {
			continue
		}
		in.key, in.members, in.group = keys[n], members, free
		break
	}
	if in.key == "" {
		t.Fatal("no installed map fits")
	}
	d, err := x.Open(in.key)
	if err != nil {
		t.Fatal(err)
	}
	s := d.Edits.(*mapInspectionEdits)
	kinds := s.RecordChoices("Structure", -1, "kind")
	items := s.RecordChoices("Sack", -1, "item")
	in.kind = kinds[0].Value
	in.items = []int64{items[0].Value, items[1].Value}
	for y := 4; y < d.Height-4 && in.cell == (image.Point{}); y++ {
		for xx := 4; xx < d.Width-4; xx++ {
			probe, err := x.Open(in.key)
			if err != nil {
				t.Fatal(err)
			}
			if next, err := probe.Edits.(*mapInspectionEdits).AddStructure(image.Pt(xx, y), in.kind); err == nil && next != nil {
				in.cell = image.Pt(xx, y)
				break
			}
		}
	}
	if in.cell == (image.Point{}) {
		t.Fatal("no cell takes the structure")
	}
	return in
}

// addCase is one kind's Add window: the keys that open it, the values its
// steps are given, and the existing host calls that add the same record.
type addCase struct {
	name   string
	keys   []string
	values map[string]string
	route  func(t *testing.T, s *mapInspectionEdits)
}

// routeAdd runs add, then sets fields on the record it appended.
func routeAdd(t *testing.T, s *mapInspectionEdits, kind string, add func() error, fields ...ui.InspectionFieldValue) {
	t.Helper()
	if err := add(); err != nil {
		t.Fatal(kind, err)
	}
	if len(fields) == 0 {
		return
	}
	ek, err := editKind(ui.RecordKind(kind))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetFields(ui.RecordKind(kind), s.model.Count(ek)-1, fields); err != nil {
		t.Fatal(kind, err)
	}
}

func addCases(in addInputs) []addCase {
	cell := fmt.Sprintf("%d,%d", in.cell.X, in.cell.Y)
	return []addCase{
		{"unit", []string{"unit"}, map[string]string{"identity": fmt.Sprint(in.identity), "owner": "1", "cell": cell},
			func(t *testing.T, s *mapInspectionEdits) {
				// The window's group step starts the unit in the lowest free group.
				routeAdd(t, s, "Unit", func() error { _, err := s.AddUnitIdentity(in.cell, in.identity); return err },
					ui.InspectionFieldValue{Key: "owner", Value: 1}, ui.InspectionFieldValue{Key: "group", Value: in.group})
			}},
		{"structure", []string{"structure"}, map[string]string{"kind": fmt.Sprint(in.kind), "owner": "2", "cell": cell},
			func(t *testing.T, s *mapInspectionEdits) {
				routeAdd(t, s, "Structure", func() error { _, err := s.AddStructure(in.cell, in.kind); return err },
					ui.InspectionFieldValue{Key: "owner", Value: 2})
			}},
		{"sack", []string{"sack"}, map[string]string{"cell": cell, "item": fmt.Sprintf("%d|%d", in.items[0], in.items[1])},
			func(t *testing.T, s *mapInspectionEdits) {
				routeAdd(t, s, "Sack", func() error { _, err := s.AddSack(in.cell, in.items[0]); return err },
					ui.InspectionFieldValue{Key: "additem", Value: in.items[1]})
			}},
		{"player", []string{"players", "player"}, map[string]string{"name": "Wizard", "colour": "3", "control": "1"},
			func(t *testing.T, s *mapInspectionEdits) {
				routeAdd(t, s, "Player", func() error { _, err := s.AddPlayer(); return err },
					ui.InspectionFieldValue{Key: "name", Text: true, TextValue: "Wizard"}, ui.InspectionFieldValue{Key: "colour", Value: 3}, ui.InspectionFieldValue{Key: "control", Value: 1})
			}},
	}
}

// openAddMap opens the inputs' map with its Map panel closed.
func openAddMap(t *testing.T, x *MapInspector, in addInputs) *ui.MapEditor {
	t.Helper()
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open(in.key); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	return e
}

// openAddCase opens the map and presses the case's keys; "players" is a
// key, "groups" the bar entry.
func openAddCase(t *testing.T, x *MapInspector, in addInputs, c addCase) *ui.MapEditor {
	t.Helper()
	e := openAddMap(t, x, in)
	clicker := &editorClicker{t: t}
	for _, k := range c.keys {
		if k == "groups" {
			r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlBar, Key: k})
			clicker.click(e, r, ok, "the Groups entry")
			continue
		}
		if err := e.Key(k); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, ok := e.AddStep(); !ok {
		t.Fatalf("%s: %v opened no Add window", c.name, c.keys)
	}
	return e
}

// TestReleaseMapEditorAddWindowFrames writes the Add window, rendered
// headless from the install, as PNGs into AGAINROM_EDITOR_FRAMES: each
// list's Add button, each kind at each step with its value typed, a many
// step with its chosen values, the popup the new record opens, and the new
// trigger's wizard F4 opens, before and after its name is typed. Without
// that directory the same steps run. In every frame the installed fonts keep
// every glyph.
func TestReleaseMapEditorAddWindowFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	in := addInputsOf(t, x)
	dir := os.Getenv("AGAINROM_EDITOR_FRAMES")
	shot := func(e *ui.MapEditor, name string) {
		t.Helper()
		writeEditorFrame(t, e, filepath.Join(dir, name+".png"), dir != "", image.Rectangle{})
	}
	clicker := &editorClicker{t: t}
	for _, bar := range []string{"units", "structures", "sacks", "groups", "players", "scripts"} {
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(in.key); err != nil {
			t.Fatal(err)
		}
		closeMapPanel(t, e)
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlBar, Key: bar})
		clicker.click(e, r, ok, "the "+bar+" entry")
		shot(e, "list-"+bar)
	}
	for _, c := range addCases(in) {
		e := openAddCase(t, x, in, c)
		for at := 0; at < 16; at++ {
			key, _, ok := e.AddStep()
			if !ok {
				break
			}
			v, given := c.values[key]
			parts := strings.Split(v, "|")
			if given && (key == "item" || key == "member") {
				for _, p := range parts {
					_ = e.Key("clear")
					e.TextInput(p)
					_ = e.Key("enter")
				}
				shot(e, fmt.Sprintf("add-%s-%d-%s", c.name, at, key))
				addEndGroups(e)
				_ = e.Key("enter")
				continue
			}
			if given {
				_ = e.Key("clear")
				e.TextInput(v)
			}
			shot(e, fmt.Sprintf("add-%s-%d-%s", c.name, at, key))
			addStepsN(t, e, c.values, 1)
		}
		if _, _, ok := e.AddStep(); ok {
			t.Fatalf("%s: the Add window did not end", c.name)
		}
		shot(e, "add-"+c.name+"-end")
	}
	// A script record is added in one click: F4 opens the new trigger's
	// wizard with its name selected, and typing names it.
	e := openAddMap(t, x, in)
	pressF4(t, e)
	if !e.WizardShown() {
		t.Fatal("F4 opened no wizard")
	}
	shot(e, "add-trigger-end")
	e.TextInput("Gate")
	shot(e, "add-trigger-named")
}

package game

import (
	"bytes"
	"crypto/sha256"
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

// newTriggerRoutes add a script record in one click and open the new
// trigger's wizard: F4 on the map, the Scenario palette's New trigger, and
// the wizard's New on the map's first trigger.
var newTriggerRoutes = []struct {
	name string
	run  func(t *testing.T, e *ui.MapEditor)
}{
	{"f4", pressF4},
	{"palette", newTriggerClick},
	{"wizard", func(t *testing.T, e *ui.MapEditor) {
		t.Helper()
		sc := e.Document().Script
		if sc == nil || len(sc.Triggers) == 0 {
			t.Fatal("the map holds no trigger whose wizard offers New")
		}
		e.Select(sc.Triggers[0].Record, true)
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 2})
		trigClick(t, e, r, ok, "New")
	}},
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

// TestReleaseMapEditorAddWindowWritesTheRouteBytes adds a unit, a
// structure, a sack and a player through the Add window on an installed
// map: the map's bytes are those of the existing host calls with the same
// values, one Undo restores the source bytes, and leaving the window at any
// step changes nothing. A window that loses one of the sack's items does not
// match: the comparison sees a dropped step value. Each one-click route adds
// the trigger the host's script add writes, and a trigger named with the
// wizard's Always check writes the host's trigger write; one Undo restores
// the source bytes. AGAINROM_WITNESS_OUT receives each case's SHA-256.
func TestReleaseMapEditorAddWindowWritesTheRouteBytes(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	in := addInputsOf(t, x)
	var lines []string
	for _, c := range addCases(in) {
		t.Run(c.name, func(t *testing.T) {
			e := openAddCase(t, x, in, c)
			source := bytes.Clone(objectEdits(t, e).model.Bytes())
			addSteps(t, e, c.values)
			got := objectEdits(t, e).model.Bytes()

			d, err := x.Open(in.key)
			if err != nil {
				t.Fatal(err)
			}
			s := d.Edits.(*mapInspectionEdits)
			c.route(t, s)
			if want := s.model.Bytes(); !bytes.Equal(got, want) {
				t.Fatalf("the Add window wrote %x (%d bytes), the host calls %x (%d bytes)", sha256.Sum256(got), len(got), sha256.Sum256(want), len(want))
			}
			lines = append(lines, fmt.Sprintf("%s %x %d", c.name, sha256.Sum256(got), len(got)))
			if !e.Dirty() {
				t.Fatal("the add left the map clean")
			}
			if err := e.Key("undo"); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(objectEdits(t, e).model.Bytes(), source) {
				t.Fatal("one Undo did not restore the source bytes")
			}

			// Leaving at each step writes nothing.
			for at := 0; ; at++ {
				e := openAddCase(t, x, in, c)
				if !addStepsN(t, e, c.values, at) {
					break
				}
				r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: 2})
				trigClick(t, e, r, ok, "Cancel Add")
				if _, _, open := e.AddStep(); open || e.Dirty() || !bytes.Equal(objectEdits(t, e).model.Bytes(), source) {
					t.Fatalf("Cancel at step %d left the window open or changed the map", at)
				}
			}
		})
	}
	t.Run("lost-step", func(t *testing.T) {
		c := addCases(in)[2]
		c.values = map[string]string{"cell": c.values["cell"], "item": fmt.Sprint(in.items[0])}
		e := openAddCase(t, x, in, c)
		addSteps(t, e, c.values)
		d, err := x.Open(in.key)
		if err != nil {
			t.Fatal(err)
		}
		s := d.Edits.(*mapInspectionEdits)
		addCases(in)[2].route(t, s)
		if bytes.Equal(objectEdits(t, e).model.Bytes(), s.model.Bytes()) {
			t.Fatal("a sack missing its second item matches the route: the comparison cannot see a lost step")
		}
	})
	t.Run("trigger", func(t *testing.T) {
		d, err := x.Open(in.key)
		if err != nil {
			t.Fatal(err)
		}
		s := d.Edits.(*mapInspectionEdits)
		if _, err := s.AddScript(ui.KindTrigger); err != nil {
			t.Fatal(err)
		}
		added := bytes.Clone(s.model.Bytes())
		undone := func(e *ui.MapEditor, source []byte) {
			t.Helper()
			if err := e.Key("undo"); err != nil || !bytes.Equal(objectEdits(t, e).model.Bytes(), source) {
				t.Fatalf("one Undo did not restore the source bytes: %v", err)
			}
		}
		for _, route := range newTriggerRoutes {
			e := openAddMap(t, x, in)
			source := bytes.Clone(objectEdits(t, e).model.Bytes())
			route.run(t, e)
			got := bytes.Clone(objectEdits(t, e).model.Bytes())
			if !bytes.Equal(got, added) {
				t.Fatalf("%s wrote %x (%d bytes), the script add %x (%d bytes)", route.name, sha256.Sum256(got), len(got), sha256.Sum256(added), len(added))
			}
			if !e.WizardShown() || !e.Dirty() {
				t.Fatalf("%s: wizard %v, dirty %v", route.name, e.WizardShown(), e.Dirty())
			}
			lines = append(lines, fmt.Sprintf("trigger-%s %x %d", route.name, sha256.Sum256(got), len(got)))
			r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 1})
			trigClick(t, e, r, ok, "Cancel")
			if !bytes.Equal(objectEdits(t, e).model.Bytes(), got) {
				t.Fatalf("%s: Cancel took the new trigger out", route.name)
			}
			undone(e, source)
		}

		// The new trigger named and given the wizard's Always check: its
		// Apply joins the add, so one Undo takes both out.
		e := openAddMap(t, x, in)
		source := bytes.Clone(objectEdits(t, e).model.Bytes())
		pressF4(t, e)
		e.TextInput("Gate")
		wizPress(t, e, "", 0)
		trigChoose(t, e, wizField(t, e, "c0.item"), x.inspectionUTF8(x.word("editor.wizard.c.always")))
		wizApply(t, e)
		got := bytes.Clone(objectEdits(t, e).model.Bytes())
		triggers := wizScript(t, added).Triggers
		one := ui.WizardNode{Opcode: 0x10002}
		one.Types[0], one.Values[0] = 1, 1
		if _, err := s.WriteTrigger(ui.WizardTrigger{Index: len(triggers) - 1, Name: "Gate", Once: triggers[len(triggers)-1].Once, Pairs: []ui.WizardPair{{Left: one, Right: one}}}); err != nil {
			t.Fatal(err)
		}
		if want := s.model.Bytes(); !bytes.Equal(got, want) {
			t.Fatalf("the wizard wrote %x (%d bytes), the trigger write %x (%d bytes)", sha256.Sum256(got), len(got), sha256.Sum256(want), len(want))
		}
		lines = append(lines, fmt.Sprintf("trigger-always %x %d", sha256.Sum256(got), len(got)))
		undone(e, source)
	})
	text := fmt.Sprintf("map %s cell %v class %d tier %d kind %d items %v members %v group %d\n%s\n", in.key, in.cell, in.class, in.tier, in.kind, in.items, in.members, in.group, strings.Join(lines, "\n"))
	t.Log(text)
	if out := os.Getenv("AGAINROM_WITNESS_OUT"); out != "" {
		if err := os.WriteFile(out, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
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

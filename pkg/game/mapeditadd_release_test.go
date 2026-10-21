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

	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

// addInputs are the values each kind's Add window is given on one
// installed map.
type addInputs struct {
	key     string
	cell    image.Point
	class   int16
	tier    uint16
	kind    int64
	items   []int64
	members []int
	group   int64
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
	ek, err := editKind(kind)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetFields(kind, s.model.Count(ek)-1, fields); err != nil {
		t.Fatal(kind, err)
	}
}

func addCases(in addInputs) []addCase {
	cell := fmt.Sprintf("%d,%d", in.cell.X, in.cell.Y)
	script := func(n int, kind string, values map[string]string, fields ...ui.InspectionFieldValue) addCase {
		v := map[string]string{"kind": fmt.Sprint(n)}
		for k, x := range values {
			v[k] = x
		}
		return addCase{name: strings.ToLower(kind), keys: []string{"scripts", "player"}, values: v,
			route: func(t *testing.T, s *mapInspectionEdits) {
				routeAdd(t, s, kind, func() error { _, err := s.AddScript(kind); return err }, fields...)
			}}
	}
	return []addCase{
		{"unit", []string{"unit"}, map[string]string{"class": fmt.Sprint(in.class), "tier": fmt.Sprint(in.tier), "owner": "1", "cell": cell},
			func(t *testing.T, s *mapInspectionEdits) {
				routeAdd(t, s, "Unit", func() error { _, err := s.AddUnitTier(in.cell, int64(in.class), int64(in.tier)); return err },
					ui.InspectionFieldValue{Key: "owner", Value: 1})
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
		script(0, "Action", nil),
		script(1, "Condition", nil),
		script(2, "Trigger", map[string]string{"name": "Gate", "once": "0"},
			ui.InspectionFieldValue{Key: "name", Text: true, TextValue: "Gate"}, ui.InspectionFieldValue{Key: "once", Value: 0}),
	}
}

// openAddCase opens the map and presses the case's keys; "scripts" and
// "players" are keys, "groups" the bar entry.
func openAddCase(t *testing.T, x *MapInspector, in addInputs, c addCase) *ui.MapEditor {
	t.Helper()
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open(in.key); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	clicker := &editorClicker{t: t}
	for _, k := range c.keys {
		if k == "groups" {
			r, ok := e.BarRect(k)
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
// structure, a sack, a player, an action, a condition and a trigger
// through the Add window on an installed map: the map's bytes are those of
// the existing host calls with the same values, one Undo restores the source
// bytes, and leaving the window at any step changes nothing. A window that
// loses one of the sack's items does not match: the comparison sees a
// dropped step value. AGAINROM_WITNESS_OUT receives each case's SHA-256.
func TestReleaseMapEditorAddWindowWritesTheRouteBytes(t *testing.T) {
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
				r, ok := e.PromptButtonRect(2)
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
// step with its chosen values, and the popup the new record opens. Without
// that directory the same steps run. In every frame the installed fonts keep
// every glyph.
func TestReleaseMapEditorAddWindowFrames(t *testing.T) {
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
		r, ok := e.BarRect(bar)
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
}

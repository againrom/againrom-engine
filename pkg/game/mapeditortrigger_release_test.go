package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// installedMapBytes are the bytes of every installed map, by catalogue key.
func installedMapBytes(t *testing.T, x *MapInspector) (keys []string, maps [][]byte) {
	t.Helper()
	keys = installedMapKeys(x)
	for _, key := range keys {
		var b []byte
		var err error
		if name, ok := strings.CutPrefix(key, "loose/"); ok {
			b, err = x.archives.Loose.ReadFile(name)
		} else {
			b, err = x.archives.Containers.ReadFile(key)
		}
		if err != nil {
			t.Fatal(key, err)
		}
		maps = append(maps, b)
	}
	return keys, maps
}

// editableSlot is the popup field and the slot of a node's first number,
// constant or cell parameter, false when it has none; a group command's
// command is a drop-down.
func editableSlot(condition bool, n ui.InspectionScriptNode) (field, slot int, ok bool) {
	field = 2 // its label and operation come first
	for s, t := range n.Types {
		if t == 0 {
			continue
		}
		if !condition && (n.Opcode == 6 && s == 0 || n.Opcode == 34 && s == 1) { // a choice: a command, a property
			field++
			continue
		}
		if cell := t == 5 && s+1 < len(n.Types) && n.Types[s+1] == 6; cell || t == 1 || t == 7 {
			return field, s, true
		}
		field++
	}
	return 0, 0, false
}

// scriptNodeOf is the one node of a family with id.
func scriptNodeOf(sc *ui.InspectionScriptModel, condition bool, id uint32) (ui.InspectionScriptNode, bool) {
	nodes := sc.Actions
	if condition {
		nodes = sc.Conditions
	}
	var found ui.InspectionScriptNode
	n := 0
	for _, node := range nodes {
		if node.ID == id {
			found, n = node, n+1
		}
	}
	return found, n == 1 && id != 0
}

// TestReleaseMapEditorTriggerAreaCensus lists over every installed map how
// many triggers draw an area on the map, which places are drawn and which
// are not with the reason; AGAINROM_TRIGGER_AREAS names a file to write it
// to. Every drawn area lies inside its map, and the editor's model carries
// the decoder's areas unchanged.
func TestReleaseMapEditorTriggerAreaCensus(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	keys, maps := installedMapBytes(t, x)
	type tally struct{ maps, triggers, drawing, areas, notes int }
	var all tally
	shapes := map[string]int{}
	reasons := map[string]int{}
	var lines []string
	for i, b := range maps {
		m, err := alm.Open(b)
		if err != nil {
			t.Fatal(keys[i], err)
		}
		sc, err := m.Script()
		if err != nil || len(sc.Triggers) == 0 {
			continue
		}
		areas, notes := mapload.ScriptAreas(m, sc)
		drawing := map[int]bool{}
		for _, a := range areas {
			drawing[a.Trigger] = true
			kind := "action"
			if a.Condition {
				kind = "check"
			}
			shapes[fmt.Sprintf("%s %d: %s", kind, a.Opcode, []string{"cell", "box", "square"}[a.Shape])]++
			if a.Rect.Empty() || !a.Rect.In(image.Rect(0, 0, int(m.Width), int(m.Height))) {
				t.Errorf("%s: trigger %d draws %v outside its %dx%d map", keys[i], a.Trigger, a.Rect, m.Width, m.Height)
			}
		}
		for _, n := range notes {
			if !n.Drawn {
				kind := "action"
				if n.Condition {
					kind = "check"
				}
				reasons[fmt.Sprintf("%s %d: %s", kind, n.Opcode, n.Reason)]++
			}
		}
		all.maps++
		all.triggers += len(sc.Triggers)
		all.drawing += len(drawing)
		all.areas += len(areas)
		lines = append(lines, fmt.Sprintf("| %s | %d | %d | %d |", keys[i], len(sc.Triggers), len(drawing), len(areas)))
		if len(lines) == 1 {
			e := x.Editor()
			e.Layout(1280, 800)
			if err := e.Open(keys[i]); err != nil {
				t.Fatal(err)
			}
			if got := len(e.Document().Script.Areas); got != len(areas) {
				t.Errorf("%s: the editor carries %d areas, the decoder finds %d", keys[i], got, len(areas))
			}
		}
	}
	var out []string
	out = append(out, fmt.Sprintf("maps with triggers %d, triggers %d, triggers drawing an area %d, areas %d", all.maps, all.triggers, all.drawing, all.areas))
	var keysShape []string
	for k := range shapes {
		keysShape = append(keysShape, k)
	}
	sort.Strings(keysShape)
	out = append(out, "", "| place drawn | areas |", "|---|---|")
	for _, k := range keysShape {
		out = append(out, fmt.Sprintf("| %s | %d |", k, shapes[k]))
	}
	var keysReason []string
	for k := range reasons {
		keysReason = append(keysReason, k)
	}
	sort.Strings(keysReason)
	out = append(out, "", "| place not drawn | count |", "|---|---|")
	for _, k := range keysReason {
		out = append(out, fmt.Sprintf("| %s | %d |", k, reasons[k]))
	}
	out = append(out, "", "| map | triggers | drawing | areas |", "|---|---|---|---|")
	out = append(out, lines...)
	t.Log(out[0])
	if all.drawing == 0 {
		t.Fatal("no installed trigger draws an area")
	}
	if path := os.Getenv("AGAINROM_TRIGGER_AREAS"); path != "" {
		if err := os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestReleaseMapEditorTriggerFrames writes the trigger editor's states,
// rendered headless from the install, into AGAINROM_EDITOR_FRAMES: the
// trigger list, a trigger's popup, an action with its operation's drop-down
// open, a cell pick, and the areas with one trigger highlighted. In each the
// installed fonts keep every glyph.
func TestReleaseMapEditorTriggerFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir := os.Getenv("AGAINROM_EDITOR_FRAMES")
	keys := installedMapKeys(x)
	open := func(key string) *ui.MapEditor {
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		return e
	}
	// The first map with a trigger drawing an area, whose first action has a
	// cell parameter somewhere among its triggers.
	key, ti, cellAct := "", -1, -1
	for _, k := range keys {
		e := open(k)
		sc := e.Document().Script
		if sc == nil {
			continue
		}
		for i, tr := range sc.Triggers {
			if len(e.TriggerAreas(i)) == 0 {
				continue
			}
			for a, id := range tr.Acts {
				if n, ok := scriptNodeOf(sc, false, id); ok && n.Opcode < 0x10000 {
					if _, s, ok := editableSlot(false, n); ok && n.Types[s] == 5 {
						key, ti, cellAct = k, i, a
						break
					}
				}
			}
			if key != "" {
				break
			}
		}
		if key != "" {
			break
		}
	}
	if key == "" {
		t.Fatal("no installed trigger draws an area and moves a cell")
	}
	t.Logf("%s trigger %d action %d", key, ti+1, cellAct+1)
	shot := func(e *ui.MapEditor, name string, cover image.Rectangle) {
		t.Helper()
		editorShot(t, e, dir, name, cover)
	}

	e := open(key)
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlBar, Key: "scripts"})
	trigClick(t, e, r, ok, "the Scripts entry")
	shot(e, "trigger-list", image.Rectangle{})

	e = open(key)
	e.Select(e.Document().Script.Triggers[ti].Record, true)
	shot(e, "trigger-popup", image.Rectangle{})

	if err := e.Key("escape"); err != nil {
		t.Fatal(err)
	}
	for _, a := range e.TriggerAreas(ti) {
		p := e.CellPoint(a.Rect.Min)
		clearPopup(e, p, p.Add(image.Pt(-30, -40)), p.Add(image.Pt(60, 20)))
	}
	if _, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlAreaLabel, N: ti}); !ok {
		t.Fatal("the highlighted trigger has no label on the map")
	}
	shot(e, "trigger-areas", image.Rectangle{})

	e = open(key)
	e.Select(e.Document().Script.Triggers[ti].Record, true)
	trigDetails(t, e)
	r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: 11 + cellAct, K: 0, Flag: false})
	trigClick(t, e, r, ok, "Edit the action")
	r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: 1})
	trigClick(t, e, r, ok, "the operation")
	shot(e, "action-dropdown", image.Rect(r.Min.X, r.Max.Y, 1280, 800))
	for _, c := range e.DropdownChoices() {
		if strings.Contains(c, "?") {
			t.Errorf("the operation choice %q lost letters to the font's encoding", c)
		}
	}

	e = open(key)
	e.Select(e.Document().Script.Triggers[ti].Record, true)
	trigDetails(t, e)
	r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: 11 + cellAct, K: 0, Flag: false})
	trigClick(t, e, r, ok, "Edit the action")
	n, _ := scriptNodeOf(e.Document().Script, false, e.Document().Script.Triggers[ti].Acts[cellAct])
	field, _, _ := editableSlot(false, n)
	r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: field, K: 0, Flag: false})
	trigClick(t, e, r, ok, "On map")
	shot(e, "cell-pick", image.Rectangle{})
}

// editorShot draws the editor headless, checks that the installed fonts keep
// every glyph outside cover, and writes the frame into dir when it is set.
func editorShot(t *testing.T, e *ui.MapEditor, dir, name string, cover image.Rectangle) {
	t.Helper()
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeEditorFrame(t, e, filepath.Join(dir, name+".png"), dir != "", cover)
}

// editorShotSized is editorShot for an editor laid out at size.
func editorShotSized(t *testing.T, e *ui.MapEditor, dir, name string, cover image.Rectangle, size image.Point) {
	t.Helper()
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeEditorFrameSized(t, e, filepath.Join(dir, name+".png"), dir != "", cover, size)
}

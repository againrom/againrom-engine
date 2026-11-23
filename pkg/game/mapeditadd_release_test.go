package game

import (
	"image"
	"os"
	"path/filepath"
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
// openAddMap opens the inputs' map with its Map panel closed.
func openAddMap(t *testing.T, x *MapInspector, in addInputs) *ui.MapEditor {
	t.Helper()
	e := x.Editor()
	e.Layout(1920, 1080)
	if err := e.Open(in.key); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	return e
}

// TestReleaseMapEditorPlacementPaletteFrames renders the installed palettes at the editor target size.
func TestReleaseMapEditorPlacementPaletteFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	in := addInputsOf(t, x)
	dir := os.Getenv("AGAINROM_EDITOR_FRAMES")
	shot := func(e *ui.MapEditor, name string) {
		t.Helper()
		writeEditorFrameSized(t, e, filepath.Join(dir, name+".png"), dir != "", image.Rectangle{}, image.Pt(1920, 1080))
	}
	clicker := &editorClicker{t: t}
	for _, bar := range []string{"units", "structures", "objects", "sacks", "cellspells", "groups", "players", "scripts"} {
		e := x.Editor()
		e.Layout(1920, 1080)
		if err := e.Open(in.key); err != nil {
			t.Fatal(err)
		}
		closeMapPanel(t, e)
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlBar, Key: bar})
		clicker.click(e, r, ok, "the "+bar+" entry")
		shot(e, "list-"+bar)
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

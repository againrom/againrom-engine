package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/modrt"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// groupPlayAction is the node ID of the guest map's start trigger action the
// test turns into "take group off the map" (instant 32): the second patrol
// order of the trigger that fires at mission start.
const groupPlayAction = 23

// unnamedPair are two units of owner whose groups no script node names, the
// nearest such pair whose second unit a Shift click on its cell selects in
// the editor.
func unnamedPair(t *testing.T, x *MapInspector, m *alm.Map, owner uint32) (int, int) {
	t.Helper()
	sc, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	named := map[uint32]bool{}
	for _, nodes := range [][]alm.ScriptNode{sc.Actions, sc.Conditions} {
		for _, n := range nodes {
			for p, tag := range n.Type {
				if tag == 2 {
					named[n.Value[p]] = true
				}
			}
		}
	}
	var free []int
	for i, u := range m.Units {
		if u.Owner == owner && !named[u.GroupID] {
			free = append(free, i)
		}
	}
	type pair struct{ a, b, d int }
	var pairs []pair
	for _, a := range free {
		for _, b := range free {
			dx, dy := int(m.Units[b].X>>8)-int(m.Units[a].X>>8), int(m.Units[b].Y>>8)-int(m.Units[a].Y>>8)
			if a < b && dx*dx+dy*dy > 0 {
				pairs = append(pairs, pair{a, b, dx*dx + dy*dy})
			}
		}
	}
	sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].d < pairs[j].d })
	for _, p := range pairs {
		e := openGroupMap(t, x)
		if trySelectUnits(t, e, p.a, p.b) {
			return p.a, p.b
		}
	}
	t.Fatalf("no pair of player %d's units outside the script's groups selects by the pointer", owner)
	return 0, 0
}

// groupThroughWindow puts the selected units in a new group through the
// Groups window: To group..., then OK on New.
func groupThroughWindow(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	c := &editorClicker{t: t, now: time.Unix(20, 0)}
	r, ok := e.BarRect("groups")
	c.click(e, r, ok, "the Groups entry")
	r, ok = e.GroupsButtonRect(0)
	c.click(e, r, ok, "To group...")
	r, ok = e.PromptButtonRect(0)
	c.click(e, r, ok, "OK on New")
}

// actOnGroup turns the start trigger's action groupPlayAction into instant 32
// on group: one script edit through the host, as the node popup writes one.
func actOnGroup(t *testing.T, e *ui.MapEditor, group uint32) {
	t.Helper()
	s := objectEdits(t, e)
	view, err := s.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	sc, err := view.Script()
	if err != nil {
		t.Fatal(err)
	}
	at := -1
	for i, n := range sc.Actions {
		if n.ID == groupPlayAction {
			at = i
		}
	}
	if at < 0 {
		t.Fatalf("no action %d", groupPlayAction)
	}
	if _, err := s.change(func(m *mapedit.Editor) (bool, error) {
		for slot := 1; slot < 10; slot++ {
			if err := m.SetField(mapedit.KindAction, at, fmt.Sprintf("type%d", slot), 0); err != nil {
				return false, err
			}
		}
		for _, f := range []mapedit.FieldSet{{Key: "type0", Value: 2}, {Key: "value0", Value: int64(group)}, {Key: "opcode", Value: int64(sim.ScriptInstantGroupOffMap)}} {
			if err := m.SetField(mapedit.KindAction, at, f.Key, f.Value); err != nil {
				return false, err
			}
		}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
}

// groupedGuestMap writes the guest map with units grouped through the window
// and the start trigger acting on that group, and returns the new group's
// number and the saved file.
func groupedGuestMap(t *testing.T, x *MapInspector, dir string, units ...int) (uint32, string) {
	t.Helper()
	var group uint32
	saveEditedGuestMap(t, x, dir, func(e *ui.MapEditor) {
		selectUnits(t, e, units...)
		groupThroughWindow(t, e)
		after, err := objectEdits(t, e).model.Map()
		if err != nil {
			t.Fatal(err)
		}
		group = after.Units[units[0]].GroupID
		for _, u := range units {
			if after.Units[u].GroupID != group || group == 0 {
				t.Fatalf("unit %d carries group %d, want the new group %d", u, after.Units[u].GroupID, group)
			}
		}
		actOnGroup(t, e, group)
	})
	lang := modrt.LanguageFor(BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))))
	return group, filepath.Join(dir, guestModID, "mission", lang, "map.alm")
}

// offMapAfterStart runs the guest mission until the start trigger has run and
// reports, by map unit id, whether each entity carrying group or listed in
// ids is off the map.
func offMapAfterStart(t *testing.T, dir string, group uint32, ids []uint16) map[uint16]bool {
	t.Helper()
	f := guestFrontWorld(t, dir)
	w := f.live.world
	for i := 0; i < 40; i++ {
		sim.Step(w, nil)
	}
	out := map[uint16]bool{}
	for _, ent := range w.Entities() {
		for _, id := range ids {
			if ent.MapUnitID == id {
				out[id] = ent.OffMap
			}
		}
		if ent.Group == group {
			if _, ok := out[ent.MapUnitID]; !ok {
				t.Errorf("entity %d (map unit %d) carries group %d without being grouped in the editor", ent.ID, ent.MapUnitID, group)
			}
		}
	}
	return out
}

// TestReleaseMapEditorGroupIsTheGroupATriggerActsOn groups two units of the
// guest mission through the Groups window and turns the start trigger's
// patrol order into "take group off the map". The saved map reopens in the
// editor with the group, the game's reader gives the two entities that group,
// and the trigger takes exactly those two off the map. Loss control: the same
// map with the second unit left out of the group leaves that unit on the map.
func TestReleaseMapEditorGroupIsTheGroupATriggerActsOn(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	grouped := guestMods(t, guestModID)
	control := guestMods(t, guestModID)
	x := installedInspector(t)
	raw, err := x.archives.Containers.ReadFile(groupIDScenario)
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, b := unnamedPair(t, x, m, 4)
	ids := []uint16{m.Units[a].UnitID, m.Units[b].UnitID}
	group, target := groupedGuestMap(t, x, grouped, a, b)
	controlGroup, _ := groupedGuestMap(t, x, control, a)
	if controlGroup != group {
		t.Fatalf("the control's new group is %d, the grouped map's %d", controlGroup, group)
	}

	// The saved map reopens in the editor and reads back through the game's
	// reader with the group on both units.
	saved, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Open(target); err != nil || !bytes.Equal(e.Document().SourceBytes, saved) {
		t.Fatalf("the saved map does not reopen: %v", err)
	}
	members := 0
	for _, r := range e.Document().Records {
		if r.Kind == "Unit" && r.Group == int(group) {
			members++
		}
	}
	if members != 2 {
		t.Fatalf("the reopened map shows %d members of group %d", members, group)
	}
	back, err := alm.Open(saved)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range []int{a, b} {
		if back.Units[u].GroupID != group {
			t.Fatalf("the game's reader gives unit %d group %d, want %d", u, back.Units[u].GroupID, group)
		}
	}

	got := offMapAfterStart(t, grouped, group, ids)
	if !got[ids[0]] || !got[ids[1]] {
		t.Fatalf("grouped map: off the map %v, want both grouped units", got)
	}
	lost := offMapAfterStart(t, control, group, ids)
	if !lost[ids[0]] || lost[ids[1]] {
		t.Fatalf("control map: off the map %v, want only the unit left in the group", lost)
	}
	t.Logf("group %d: grouped run off map %v; control run off map %v", group, got, lost)
}

// TestReleaseMapEditorPlacingAndGroupsWithoutAnEditSaveUnchanged opens shipped
// maps, uses the picture grid, the selection box and the Groups window without
// an edit, and saves: the file is the installed bytes.
func TestReleaseMapEditorPlacingAndGroupsWithoutAnEditSaveUnchanged(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir := t.TempDir()
	keys := installedMapKeys(x)
	for n, key := range []string{keys[0], groupIDScenario, keys[len(keys)/2], keys[len(keys)-1]} {
		want, err := x.Open(key)
		if err != nil {
			t.Fatal(err)
		}
		e := x.Editor()
		if err := e.Open(key); err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		e.Layout(1280, 800)
		c := &editorClicker{t: t, now: time.Unix(30, 0)}
		r, ok := e.BarRect("units")
		c.click(e, r, ok, "the Units entry")
		if e.PlaceGridLen() == 0 {
			t.Fatalf("%s: the Units tab shows no picture grid", key)
		}
		for i := 0; i < 2; i++ { // choose a class, then press it again to stop
			r, ok = e.PlaceGridCellRect(0)
			c.click(e, r, ok, "grid cell 0")
		}
		from := bareCell(t, e.Document(), image.Pt(e.Document().Width/2, e.Document().Height/2))
		e.ShowCell(from)
		pointerStroke(e, []image.Point{from, from.Add(image.Pt(6, 6))})
		if e.Dirty() {
			t.Fatalf("%s: the box from %v edited the map", key, from)
		}
		r, ok = e.BarRect("groups")
		c.click(e, r, ok, "the Groups entry")
		_ = e.Key("group1")
		target := filepath.Join(dir, fmt.Sprintf("m%d.alm", n))
		_ = e.Key("saveas")
		e.TextInput(target)
		if err := e.Key("enter"); err != nil {
			t.Fatalf("%s: Save As: %v", key, err)
		}
		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want.SourceBytes) {
			t.Errorf("%s: saved %d bytes differ from the installed %d", key, len(got), len(want.SourceBytes))
		}
	}
}

// bareCell is a cell near from with no record within two cells: a press there
// starts a selection box, not a record drag.
func bareCell(t *testing.T, d *ui.InspectionDocument, from image.Point) image.Point {
	t.Helper()
	for r := 0; r < 40; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				c := from.Add(image.Pt(dx, dy))
				if !c.In(image.Rect(2, 2, d.Width-8, d.Height-8)) {
					continue
				}
				near := image.Rect(c.X-2, c.Y-2, c.X+3, c.Y+3)
				clear := true
				for _, rec := range d.Records {
					if rec.Spatial && near.Overlaps(image.Rectangle{Min: rec.Cell, Max: rec.Cell.Add(rec.Size.Add(image.Pt(1, 1)))}) {
						clear = false
						break
					}
				}
				if clear {
					return c
				}
			}
		}
	}
	t.Fatal("no bare cell")
	return image.Point{}
}

// TestReleaseMapEditorObjectsAndGroupsFrames writes the owner's frames of the
// guest mission into AGAINROM_EDITOR_FRAMES, rendered headless from the
// install: the Units grid with a chosen class, a selection box being drawn,
// the selection's silhouettes during a move, and the Groups window with the
// chosen group's members outlined and badged. Without the directory it only
// drives the same steps.
func TestReleaseMapEditorObjectsAndGroupsFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir, lang := os.Getenv("AGAINROM_EDITOR_FRAMES"), installLanguage(x)
	frame := func(e *ui.MapEditor, name string) {
		writeEditorFrame(t, e, filepath.Join(dir, "objects-"+name+"-"+lang+".png"), dir != "", image.Rectangle{})
	}
	e := openGroupMap(t, x)
	closeMapPanel(t, e)
	c := &editorClicker{t: t, now: time.Unix(40, 0)}
	r, ok := e.BarRect("units")
	c.click(e, r, ok, "the Units entry")
	r, ok = e.PlaceGridCellRect(0)
	c.click(e, r, ok, "grid cell 0")
	frame(e, "grid")
	r, ok = e.PlaceGridCellRect(0)
	c.click(e, r, ok, "grid cell 0 again")

	// A box held over the unit with the most units within three
	// cells, then released.
	var unit ui.InspectionRecord
	best := 0
	for _, rec := range e.Document().Records {
		near := 0
		for _, o := range e.Document().Records {
			if d := o.Cell.Sub(rec.Cell); o.Kind == "Unit" && max(d.X, -d.X, d.Y, -d.Y) <= 3 {
				near++
			}
		}
		if rec.Kind == "Unit" && near > best {
			unit, best = rec, near
		}
	}
	e.ShowCell(unit.Cell)
	from, to := boxCornerCell(t, e, unit.Cell, -1), boxCornerCell(t, e, unit.Cell, 1)
	now := time.Unix(41, 0)
	at := func(cell image.Point, down bool) {
		p := e.CellPoint(cell)
		e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: down}, now)
	}
	at(from, true)
	at(to, true)
	frame(e, "box")
	at(to, false)
	if len(e.SelectedRecords()) < 2 {
		t.Fatalf("the box selected %v", e.SelectedRecords())
	}

	// The selection moves under the pointer as silhouettes until the release.
	at(unit.Cell, true)
	at(unit.Cell.Add(image.Pt(4, 3)), true)
	frame(e, "move")
	at(unit.Cell.Add(image.Pt(4, 3)), false)
	_ = e.Key("undo")

	r, ok = e.BarRect("groups")
	c.click(e, r, ok, "the Groups entry")
	r, ok = e.GroupsRowRect(2, 2, false)
	c.click(e, r, ok, "group 2's row")
	if e.GroupBadges() == 0 || len(e.SelectedRecords()) == 0 {
		t.Fatalf("group 2 shows %d badges over %d selected units", e.GroupBadges(), len(e.SelectedRecords()))
	}
	frame(e, "groups")
}

// boxCornerCell is a cell on screen, up to five cells from at in direction dir on
// both axes, that no record covers: a press there starts a selection box.
func boxCornerCell(t *testing.T, e *ui.MapEditor, at image.Point, dir int) image.Point {
	t.Helper()
	screen := image.Rect(420, 40, 1260, 760)
	for k := 5; k >= 2; k-- {
		c := at.Add(image.Pt(dir*k, dir*k))
		covered := false
		for _, rec := range e.Document().Records {
			if rec.Spatial && c.In(image.Rectangle{Min: rec.Cell, Max: rec.Cell.Add(rec.Size.Add(image.Pt(1, 1)))}) {
				covered = true
			}
		}
		if !covered && e.CellPoint(c).In(screen) {
			return c
		}
	}
	t.Fatalf("no open cell on screen beside %v", at)
	return image.Point{}
}

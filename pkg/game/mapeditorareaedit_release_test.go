package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// installedArea is an area of an installed map with room for the drag the
// release tests make: the map, the trigger and the area.
type installedArea struct {
	key     string
	trigger int
	area    ui.InspectionArea
}

// areaLimit is the last cell an X or Y slot of e's map can name, plus one.
func areaLimit(e *ui.MapEditor) int {
	m, err := alm.Open(e.Document().SourceBytes)
	if err != nil {
		return 0
	}
	return min(int(m.Width), int(m.Height), 256)
}

// findInstalledArea is the first area of shape on an installed map other
// than skip that fit accepts, opened in a new editor.
func findInstalledArea(t *testing.T, x *MapInspector, shape int, skip string, fit func(ui.InspectionArea, int) bool) (installedArea, *ui.MapEditor) {
	t.Helper()
	for _, key := range editorKeys(x) {
		if key == skip {
			continue
		}
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		if e.Document().Script == nil {
			continue
		}
		lim := areaLimit(e)
		for n := range e.Document().Script.Triggers {
			for _, a := range e.TriggerAreas(n) {
				if a.Shape == shape && fit(a, lim) {
					return installedArea{key, n, a}, e
				}
			}
		}
	}
	t.Fatalf("no installed map has an area of shape %d with room for the drag", shape)
	return installedArea{}, nil
}

// boxWithRoom: a box whose four slots are cells and whose far corner can
// move two cells right and down.
func boxWithRoom(a ui.InspectionArea, lim int) bool {
	return a.Slots[0] >= 0 && a.Slots[2] >= 0 && a.Rect.Max.X+2 <= lim && a.Rect.Max.Y+2 <= lim &&
		a.Rect.Dx() <= 12 && a.Rect.Dy() <= 12
}

// squareWithRoom: a square whose radius a parameter holds and whose right
// rim can move one cell right.
func squareWithRoom(a ui.InspectionArea, lim int) bool {
	return a.Slots[0] >= 0 && a.RadiusNode >= 0 && a.Centre.X+a.Radius+2 <= lim && a.Radius < 8
}

// areaSaveAs writes e's map to a new file and returns its bytes, decoded.
func areaSaveAs(t *testing.T, e *ui.MapEditor, name string) (*alm.Map, alm.Script, []byte) {
	t.Helper()
	target := filepath.Join(t.TempDir(), name)
	_ = e.Key("saveas")
	e.TextInput(target)
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(b)
	if err != nil {
		t.Fatal(err)
	}
	sc, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	return m, sc, b
}

// areaCompiled requires the game's script compiler to accept m and check
// reg, condition reg of sc, to hold want in its parameter slot: the
// compiler packs the int, X, Y and constant slots in order.
func areaCompiled(t *testing.T, m *alm.Map, sc alm.Script, reg, slot int, want int32) {
	t.Helper()
	prog, _, err := mapload.CompileScript(m, mapload.ScriptRefs{})
	if err != nil {
		t.Fatalf("the game's decoder refused the saved map: %v", err)
	}
	arg := 0
	for _, typ := range sc.Conditions[reg].Type[:slot] {
		if typ == mapload.ScriptInt || typ == mapload.ScriptX || typ == mapload.ScriptY || typ == mapload.ScriptConst {
			arg++
		}
	}
	for _, c := range prog.Checks() {
		if int(c.Register) == reg {
			if c.Op != int32(sc.Conditions[reg].Opcode) || c.Args[arg] != want {
				t.Fatalf("check %d op %d argument %d: the game reads %d, want %d", reg, c.Op, arg, c.Args[arg], want)
			}
			return
		}
	}
	t.Fatalf("the compiled script has no check in register %d", reg)
}

// areaFind is the area of trigger n made by condition node with shape, from
// mapload's own reading of a decoded map.
func areaFind(t *testing.T, m *alm.Map, sc alm.Script, n, node int, shape mapload.ScriptAreaShape) mapload.ScriptArea {
	t.Helper()
	areas, _ := mapload.ScriptAreas(m, sc)
	for _, a := range areas {
		if a.Trigger == n && a.Condition && a.Node == node && a.Shape == shape {
			return a
		}
	}
	t.Fatalf("the reopened map has no area of trigger %d node %d", n+1, node)
	return mapload.ScriptArea{}
}

// areaReady selects trigger n with its jump and closes its popup.
func areaReady(t *testing.T, e *ui.MapEditor, n int) {
	t.Helper()
	e.Select(trigRecord(t, e, "Trigger", n), true)
	_ = e.Key("escape")
}

// areaConfirm answers the shared-node notice with Change all when it opens.
func areaConfirm(e *ui.MapEditor) bool {
	if _, open := e.PromptRect(); open {
		_ = e.Key("enter")
		return true
	}
	return false
}

// On two installed maps a drag on a box's far corner and on a square's rim,
// saved and reopened, shows the moved area, and the game's compiler reads
// the new parameters; a box drawn on the first map and given a new trigger
// is read back as that trigger's area.
func TestReleaseMapEditorAreaDragsSaveAndReopen(t *testing.T) {
	x := installedInspector(t)

	box, e := findInstalledArea(t, x, ui.AreaBox, "", boxWithRoom)
	a := box.area
	node := e.Document().Records[a.Node].Index
	areaReady(t, e, box.trigger)
	areaPress(e, e.AreaCornerPoint(a.Rect.Max), e.CellPoint(a.Rect.Max.Add(image.Pt(1, 1))), true)
	shared := areaConfirm(e)
	if !e.Dirty() {
		t.Fatalf("%s T%d: the corner drag wrote nothing; message %q", box.key, box.trigger+1, e.Message())
	}
	m, sc, _ := areaSaveAs(t, e, "box.alm")
	got := areaFind(t, m, sc, box.trigger, node, mapload.AreaBox)
	want := image.Rect(a.Rect.Min.X, a.Rect.Min.Y, a.Rect.Max.X+2, a.Rect.Max.Y+2)
	if got.Rect != want {
		t.Fatalf("%s T%d: reopened box %v, want %v", box.key, box.trigger+1, got.Rect, want)
	}
	areaCompiled(t, m, sc, node, a.Slots[2], int32(a.Rect.Max.X+1))
	areaCompiled(t, m, sc, node, a.Slots[3], int32(a.Rect.Max.Y+1))
	t.Logf("%s T%d box %v -> %v (shared notice %v)", box.key, box.trigger+1, a.Rect, got.Rect, shared)

	triggers := len(sc.Triggers)
	c := areaLimit(e) / 2
	areaDraw(t, e, image.Pt(c, c), image.Pt(c+3, c+2))
	r, ok := e.PromptButtonRect(0)
	trigClick(t, e, r, ok, "Create")
	m, sc, _ = areaSaveAs(t, e, "drawn.alm")
	if len(sc.Triggers) != triggers+1 {
		t.Fatalf("%s: %d triggers after the drawn box, want %d", box.key, len(sc.Triggers), triggers+1)
	}
	areas, _ := mapload.ScriptAreas(m, sc)
	drawn := -1
	for _, d := range areas {
		if d.Trigger == triggers && d.Shape == mapload.AreaBox && d.Rect == image.Rect(c, c, c+4, c+3) {
			drawn = d.Node
		}
	}
	if drawn < 0 {
		t.Fatalf("%s: the new trigger has no box %d,%d to %d,%d", box.key, c, c, c+3, c+2)
	}
	areaCompiled(t, m, sc, drawn, 1, int32(c))
	areaCompiled(t, m, sc, drawn, 4, int32(c+2))

	sq, e := findInstalledArea(t, x, ui.AreaSquare, box.key, squareWithRoom)
	a = sq.area
	node = e.Document().Records[a.Node].Index
	radiusNode := e.Document().Records[a.RadiusNode].Index
	areaReady(t, e, sq.trigger)
	areaPress(e, e.AreaCornerPoint(image.Pt(a.Rect.Max.X, a.Centre.Y)), e.CellPoint(image.Pt(a.Centre.X+a.Radius+1, a.Centre.Y)), true)
	shared = areaConfirm(e)
	if !e.Dirty() {
		t.Fatalf("%s T%d: the rim drag wrote nothing; message %q", sq.key, sq.trigger+1, e.Message())
	}
	m, sc, _ = areaSaveAs(t, e, "square.alm")
	gotSq := areaFind(t, m, sc, sq.trigger, node, mapload.AreaSquare)
	if gotSq.Radius != a.Radius+1 || gotSq.Centre != a.Centre {
		t.Fatalf("%s T%d: reopened square centre %v radius %d, want %v radius %d", sq.key, sq.trigger+1, gotSq.Centre, gotSq.Radius, a.Centre, a.Radius+1)
	}
	areaCompiled(t, m, sc, radiusNode, a.RadiusSlot, int32(a.Radius+1+a.RadiusBias))
	t.Logf("%s T%d square at %v radius %d -> %d, register %d slot %d bias %d (shared notice %v)",
		sq.key, sq.trigger+1, a.Centre, a.Radius, gotSq.Radius, radiusNode, a.RadiusSlot, a.RadiusBias, shared)
}

// Every installed map, opened with its areas shown, its first trigger
// selected and its handles drawn, saves the bytes it was opened with.
func TestReleaseMapEditorAreasUntouchedMapsSaveTheirBytes(t *testing.T) {
	x := installedInspector(t)
	dir := t.TempDir()
	screen := ebiten.NewImage(1280, 800)
	defer screen.Dispose()
	shown := 0
	for i, key := range editorKeys(x) {
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		before := bytes.Clone(e.Document().SourceBytes)
		if sc := e.Document().Script; sc != nil && len(sc.Triggers) > 0 {
			areaReady(t, e, 0)
			for n := range sc.Triggers {
				shown += len(e.TriggerAreas(n))
			}
		}
		e.Draw(screen)
		if e.Dirty() || !bytes.Equal(e.Document().SourceBytes, before) {
			t.Fatalf("%s: showing the areas changed the map", key)
		}
		target := filepath.Join(dir, fmt.Sprintf("%02d.alm", i))
		_ = e.Key("saveas")
		e.TextInput(target)
		if err := e.Key("enter"); err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		if got, err := os.ReadFile(target); err != nil || !bytes.Equal(got, before) {
			t.Errorf("%s: Save As wrote other bytes than the installed map's (%v)", key, err)
		}
	}
	if shown == 0 {
		t.Fatal("no installed map showed an area")
	}
	t.Logf("%d maps, %d areas shown", len(editorKeys(x)), shown)
}

// TestReleaseMapEditorAreaFrames writes the handles of a selected trigger, a
// corner drag in progress, the armed draw tool, the shared-node notice and
// the trigger choice with its drop-down open, rendered headless from the
// install, as PNGs into AGAINROM_EDITOR_FRAMES; without it the same steps
// run. Each frame keeps every glyph of the installed fonts outside an open
// prompt or list.
func TestReleaseMapEditorAreaFrames(t *testing.T) {
	x := installedInspector(t)
	dir := os.Getenv("AGAINROM_EDITOR_FRAMES")
	lang := installLanguage(x)
	shot := func(e *ui.MapEditor, name string, cover image.Rectangle) {
		t.Helper()
		if r, open := e.PromptRect(); open {
			cover = cover.Union(r)
		}
		writeEditorFrame(t, e, filepath.Join(dir, name+"-"+lang+".png"), dir != "", cover)
	}

	box, e := findInstalledArea(t, x, ui.AreaBox, "", boxWithRoom)
	areaReady(t, e, box.trigger)
	shot(e, "area-handles", image.Rectangle{})
	a := box.area
	areaPress(e, e.AreaCornerPoint(a.Rect.Max), e.CellPoint(a.Rect.Max.Add(image.Pt(1, 1))), false)
	shot(e, "area-drag", image.Rectangle{})
	_ = e.Key("escape")
	e.Pointer(ui.Input{}, time.Unix(100, 0))
	if e.Dirty() {
		t.Fatal("a cancelled drag wrote the map")
	}

	e.Fit()
	if err := e.Key("drawarea"); err != nil || !e.AreaDrawArmed() {
		t.Fatalf("L did not arm the draw tool: %v", err)
	}
	shot(e, "area-draw", image.Rectangle{})
	_ = e.Key("drawarea")
	c := areaLimit(e) / 2
	areaDraw(t, e, image.Pt(c, c), image.Pt(c+3, c+2))
	shot(e, "area-trigger", image.Rectangle{})
	r, ok := e.AreaTriggerRect()
	trigClick(t, e, r, ok, "the trigger choice")
	if len(e.DropdownChoices()) < 2 {
		t.Fatalf("the trigger choice lists %q", e.DropdownChoices())
	}
	shot(e, "area-trigger-list", image.Rect(r.Min.X, r.Max.Y, 1280, 800))
	_ = e.Key("escape")
	_ = e.Key("escape")

	for _, key := range editorKeys(x) {
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		sc := e.Document().Script
		if sc == nil {
			continue
		}
		lim := areaLimit(e)
		for n := range sc.Triggers {
			for _, a := range e.TriggerAreas(n) {
				if a.Shape != ui.AreaSquare || !squareWithRoom(a, lim) {
					continue
				}
				areaReady(t, e, n)
				areaPress(e, e.AreaCornerPoint(image.Pt(a.Rect.Max.X, a.Centre.Y)), e.CellPoint(image.Pt(a.Centre.X+a.Radius+1, a.Centre.Y)), true)
				if _, open := e.PromptRect(); open {
					shot(e, "area-shared", image.Rectangle{})
					t.Logf("shared notice: %s T%d: %q", key, n+1, e.PromptNotes())
					return
				}
				if e.Dirty() {
					_ = e.Key("undo")
				}
			}
		}
	}
	t.Fatal("no installed square area shares its radius or its node")
}

// TestReleaseMapEditorAreaDrawRefusesOnMultiplayerMaps tries the draw tool, by
// L and by the View menu item, on the installed multiplayer maps Beast and
// LuMoir, as the root has them: it refuses with the multiplayer word and the
// map's bytes are unchanged.
func TestReleaseMapEditorAreaDrawRefusesOnMultiplayerMaps(t *testing.T) {
	x := installedInspector(t)
	tried := 0
	for _, key := range editorKeys(x) {
		if k := strings.ToLower(key); k != "loose/beast.alm" && k != "loose/lumoir.alm" {
			continue
		}
		for _, route := range []string{"key", "menu"} {
			e := x.Editor()
			e.Layout(1280, 800)
			if err := e.Open(key); err != nil {
				t.Fatal(err)
			}
			if m, err := alm.Open(e.Document().SourceBytes); err != nil || m.Meta.Word70 <= 1 {
				t.Fatalf("%s is not a multiplayer map (%v)", key, err)
			}
			before := bytes.Clone(e.Document().SourceBytes)
			e.Fit()
			if route == "key" {
				_ = e.Key("drawarea")
			} else {
				r, ok := e.BarRect("view")
				trigClick(t, e, r, ok, "View")
				r, ok = e.MenuRowRect(7)
				trigClick(t, e, r, ok, "Draw trigger area")
			}
			if e.AreaDrawArmed() || e.Message() == "" || e.Dirty() || !bytes.Equal(e.Document().SourceBytes, before) {
				t.Fatalf("%s by %s: armed %v, message %q, dirty %v", key, route, e.AreaDrawArmed(), e.Message(), e.Dirty())
			}
			t.Logf("%s by %s: %q", key, route, e.Message())
		}
		tried++
	}
	if tried == 0 {
		t.Fatal("no installed multiplayer map of the two named was found")
	}
}

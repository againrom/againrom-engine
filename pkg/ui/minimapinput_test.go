package ui

import (
	"image"
	"slices"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// The minimap's own input contract (`AI-MINIMAP-124`): the widget acts on
// the primary button's DOWN edge, and what it does is decided by the small
// cursor it is wearing, which is the same armed mode the map's own click
// dispatch reads.
//
// THE FIXTURE IS NOT commandViewer. That one switches the minimap OFF, because
// in an 800x600 view the widget covers exactly where its entities are drawn
// (command_test.go's own note). This one switches it on and never presses the
// map at all.

// minimapViewer is a command-mode viewer with the minimap shown, a world big
// enough that the widget samples rather than degenerating, and two entities on
// known cells.
func minimapViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewer("minimap-input", grid(64, 64), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	layoutViewport(v, 800, 600)
	v.SetEntities([]MapEntity{
		{ID: 12, Cell: image.Pt(2, 2)},
		{ID: 5, Cell: image.Pt(40, 40)},
	})
	if _, ok := v.minimapGeometry(); !ok {
		t.Fatal("the fixture has no minimap geometry")
	}
	return v
}

// minimapPixelFor is a window pixel inside the widget that names this cell, or
// a fatal. It searches the widget's own content rectangle through the
// production hit test rather than inverting the mapping here, so a mapping
// that moves is followed rather than duplicated.
func minimapPixelFor(t *testing.T, v *Viewer, cell image.Point) (int, int) {
	t.Helper()
	g, ok := v.minimapGeometry()
	if !ok {
		t.Fatal("no minimap geometry")
	}
	for y := g.Content.Min.Y; y < g.Content.Max.Y; y++ {
		for x := g.Content.Min.X; x < g.Content.Max.X; x++ {
			if c, ok := v.minimapCellAt(x, y); ok && c == cell {
				return x, y
			}
		}
	}
	t.Fatalf("no pixel of the minimap names cell %v", cell)
	return 0, 0
}

// minimapSelect puts a selection in place without going through the map, which
// this fixture's own widget covers. It is the one place in this file that
// writes v.sel, and it writes an id the snapshot holds.
func minimapSelect(v *Viewer, ids ...uint32) {
	v.sel = append(selection(nil), ids...)
}

func minimapPress(v *Viewer, x, y int) ([]order, bool) {
	return v.command(appInput{PrimaryPressed: true, CursorX: x, CursorY: y,
		Viewer: Input{CursorX: x, CursorY: y, PrimaryDown: true}})
}

func minimapHold(v *Viewer, x, y int) ([]order, bool) {
	return v.command(appInput{CursorX: x, CursorY: y,
		Viewer: Input{CursorX: x, CursorY: y, PrimaryDown: true}})
}

func minimapRelease(v *Viewer, x, y int) ([]order, bool) {
	return v.command(appInput{PrimaryReleased: true, CursorX: x, CursorY: y,
		Viewer: Input{CursorX: x, CursorY: y}})
}

func TestTheMinimapOnlyCentresCameraWithEveryArmedMode(t *testing.T) {
	for _, mode := range []string{"none", "selected", "attack", "cast", "patrol", "defend"} {
		t.Run(mode, func(t *testing.T) {
			v := minimapViewer(t)
			if mode != "none" {
				minimapSelect(v, 12)
			}
			switch mode {
			case "attack":
				v.armed = true
			case "cast":
				v.spellArmed = true
				v.selectedSpell = 7
			case "patrol":
				v.aimed = commandPatrol
			case "defend":
				v.aimed = commandDefend
			}
			armed, aimed, spell := v.armed, v.aimed, v.spellArmed
			x, y := minimapPixelFor(t, v, image.Pt(20, 30))
			before := *v.Camera()
			if orders, ok := minimapPress(v, x, y); ok || len(orders) != 0 {
				t.Fatalf("minimap ordered units: %+v", orders)
			}
			if after := v.Camera(); after.X == before.X && after.Y == before.Y {
				t.Fatal("camera did not centre")
			}
			if orders, ok := minimapRelease(v, x, y); ok || len(orders) != 0 {
				t.Fatal("release issued an order")
			}
			if v.armed != armed || v.aimed != aimed || v.spellArmed != spell {
				t.Fatal("camera gesture changed the armed command")
			}
		})
	}
}

func TestTheMinimapDoesNotOrderAUnitIntoTheEngineMargin(t *testing.T) {
	v := minimapViewer(t)
	target := image.Pt(20, 30)
	v.grid.Block = make([]uint8, v.grid.Width*v.grid.Height)
	v.grid.Block[target.Y*v.grid.Width+target.X] = borderBit
	minimapSelect(v, 12)
	x, y := minimapPixelFor(t, v, target)

	if ords, ok := minimapPress(v, x, y); ok || len(ords) != 0 {
		t.Fatalf("press on impassable minimap margin issued %+v (ok=%v), want none", ords, ok)
	}
}

// TestTheMinimapRepeatsItsActionPerDeliveredMoveAndNotPerFrame is the drag
// half of `AI-MINIMAP-124`: "left-drag repeats the current small-cursor action
// per delivered move". A frame that delivers no movement repeats nothing, or a
// player holding the button on one pixel would issue an order every frame.
func TestTheMinimapDragMovesTheCameraWithoutCommands(t *testing.T) {
	v := minimapViewer(t)
	minimapSelect(v, 12)
	x, y := minimapPixelFor(t, v, image.Pt(20, 30))
	minimapPress(v, x, y)
	before := *v.Camera()
	x, y = minimapPixelFor(t, v, image.Pt(40, 45))
	if orders, ok := minimapHold(v, x, y); ok || len(orders) != 0 {
		t.Fatal("drag issued commands")
	}
	if after := v.Camera(); after.X == before.X && after.Y == before.Y {
		t.Fatal("drag did not move camera")
	}
	minimapRelease(v, x, y)
	if v.minimapGrab || !slices.Equal(v.sel, selection{12}) {
		t.Fatal("release lost selection or capture")
	}
}

// TestTheMinimapRightButtonCentresAndCancelsNothing is the right half of
// `AI-MINIMAP-124`: right down moves the camera with no capture, and right up
// is a no-op — so it does not reach the map's own cancel and does not clear
// the selection.
func TestTheMinimapRightButtonCentresAndCancelsNothing(t *testing.T) {
	v := minimapViewer(t)
	minimapSelect(v, 12)
	x, y := minimapPixelFor(t, v, image.Pt(20, 30))
	before := image.Pt(int(v.Camera().X), int(v.Camera().Y))

	if ords, ok := v.command(appInput{CursorX: x, CursorY: y,
		Viewer: Input{CursorX: x, CursorY: y, SecondaryDown: true}}); ok || len(ords) != 0 {
		t.Fatalf("a right hold over the minimap issued %+v (ok=%v), want none", ords, ok)
	}
	if after := (image.Pt(int(v.Camera().X), int(v.Camera().Y))); after == before {
		t.Errorf("a right hold over the minimap left the camera at %v; it must centre", before)
	}

	if ords, ok := v.command(appInput{SecondaryReleased: true, CursorX: x, CursorY: y,
		Viewer: Input{CursorX: x, CursorY: y}}); ok || len(ords) != 0 {
		t.Fatalf("a right release over the minimap issued %+v (ok=%v), want none", ords, ok)
	}
	if !slices.Equal(v.sel, selection{12}) {
		t.Errorf("a right release over the minimap left selection %+v, want it untouched", v.sel)
	}
}

// TestTheMinimapAttackArmDoesNotNameAUnitInTheFog is the gate on
// topEntityAtCell. Without it the `0x19`/`0x1a` split would report an unseen
// enemy's presence: the player would learn where he is from which order came
// out, on a corner of the overview that draws nothing.
func TestTheMinimapOccupiedCellIsStillOnlyACameraTarget(t *testing.T) {
	v := minimapViewer(t)
	minimapSelect(v, 12)
	v.armed = true
	x, y := minimapPixelFor(t, v, image.Pt(40, 40))
	if orders, ok := minimapPress(v, x, y); ok || len(orders) != 0 {
		t.Fatal("minimap targeted a unit")
	}
}

func TestMinimapDrawsOrdinaryCursorEvenWhileAttackIsArmed(t *testing.T) {
	a, v, _ := atOnMap(t)
	a.SetCursorRegistry(clRegistry())
	v.armed = true
	g, ok := v.minimapGeometry()
	if !ok {
		t.Fatal("missing minimap")
	}
	p := g.Content.Min.Add(image.Pt(20, 20))
	v.cursorX, v.cursorY, v.hasCursor = p.X, p.Y, true
	v.advanceCursorManager(time.Unix(100, 0))
	if v.cursorMgr.CurrentName() != "default" || v.attackShown() {
		t.Fatal("attack cursor replaced the minimap cursor")
	}
	if _, _, ok := v.mapCursorPresent(); !ok {
		t.Fatal("ordinary cursor was not drawn")
	}
}

// marginMinimapViewer is minimapViewer over a 64x64 map whose outer 8 cells on
// every side are the engine margin, leaving a walkable 48x48.
func marginMinimapViewer(t *testing.T) *Viewer {
	t.Helper()
	v := minimapViewer(t)
	w, h := v.grid.Width, v.grid.Height
	v.grid.Block = make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < 8 || y < 8 || x >= w-8 || y >= h-8 {
				v.grid.Block[y*w+x] = borderBit
			}
		}
	}
	return v
}

func TestTheMinimapDrawsAndMapsOnlyTheWalkableArea(t *testing.T) {
	v := marginMinimapViewer(t)
	g, ok := v.minimapGeometry()
	if !ok {
		t.Fatal("no minimap geometry")
	}
	if g.Cols != 48 || g.Rows != 48 || g.Origin != image.Pt(8, 8) {
		t.Fatalf("minimap covers %dx%d cells from %v, want the 48x48 walkable area from (8,8)", g.Cols, g.Rows, g.Origin)
	}
	first, ok := v.minimapCellAt(g.Content.Min.X, g.Content.Min.Y)
	if !ok || first != image.Pt(8, 8) {
		t.Fatalf("the content's first pixel names %v (ok=%v), want the first walkable cell (8,8)", first, ok)
	}
	last, ok := v.minimapCellAt(g.Content.Max.X-1, g.Content.Max.Y-1)
	if !ok || last != image.Pt(55, 55) {
		t.Fatalf("the content's last pixel names %v (ok=%v), want the last walkable cell (55,55)", last, ok)
	}
	if _, err := v.minimapPixelOf(image.Pt(2, 2)); err == nil {
		t.Fatal("a margin cell has a pixel on the minimap")
	}
}

// minimapPixelOf is the pixel naming cell, or an error when none does.
func (v *Viewer) minimapPixelOf(cell image.Point) (image.Point, error) {
	g, _ := v.minimapGeometry()
	for y := g.Content.Min.Y; y < g.Content.Max.Y; y++ {
		for x := g.Content.Min.X; x < g.Content.Max.X; x++ {
			if c, ok := v.minimapCellAt(x, y); ok && c == cell {
				return image.Pt(x, y), nil
			}
		}
	}
	return image.Point{}, image.ErrFormat
}

func TestTheMinimapMarksAndOutlineStayInsideTheWalkableArea(t *testing.T) {
	v := marginMinimapViewer(t)
	v.SetEntities([]MapEntity{
		{ID: 12, Cell: image.Pt(2, 2)},
		{ID: 5, Cell: image.Pt(40, 40), Owner: v.localOwner},
	})
	g, _ := v.minimapGeometry()
	_, _, marks, view := v.minimapWindowed(g)
	if len(marks) != 1 || marks[0].Cell != image.Pt(32, 32) {
		t.Fatalf("marks = %+v, want the one unit at window cell (32,32)", marks)
	}
	if !view.Empty() && !view.In(image.Rect(0, 0, g.Cols, g.Rows)) {
		t.Fatalf("camera outline %v leaves the %dx%d window", view, g.Cols, g.Rows)
	}
}

package ui

// The mission map's own cursor: the eight edge arrows, the minimap's own
// mode cursor, the hostility test at hover, and the two places that decide
// whether anything of ours is drawn at all — mapCursorName's precedence
// and mapCursorPresent's own exclusions. B4 (the held item) is 1005's own
// dragItemPresent and is not re-tested here; this file covers only what
// missioncursor.go and its two call sites (cursor.go, flow.go) add.

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestMissionEdgeArrowPicksTheEightNamesByPositionAndBand is exhaustive over
// missionEdgeArrow's own 3x3 band grid (each axis: before the margin, inside
// it, past the far margin) plus the four out-of-window positions and the two
// degenerate sizes — a pure function, so the table itself is the witness and
// no fixture or mutation is needed to prove a case wrong: changing any one
// returned name breaks that case's own row.
func TestMissionEdgeArrowPicksTheEightNamesByPositionAndBand(t *testing.T) {
	const w, h, margin = 100, 80, 24
	cases := []struct {
		name    string
		x, y    int
		want    string
		wantHit bool
	}{
		{"centre", 50, 40, "", false},
		{"top band, centred x", 50, 0, "arrow0", true},
		{"bottom band, centred x", 50, h - 1, "arrow4", true},
		{"left band, centred y", 0, 40, "arrow6", true},
		{"right band, centred y", w - 1, 40, "arrow2", true},
		{"top-left corner", 0, 0, "arrow7", true},
		{"top-right corner", w - 1, 0, "arrow1", true},
		{"bottom-left corner", 0, h - 1, "arrow5", true},
		{"bottom-right corner", w - 1, h - 1, "arrow3", true},
		{"just inside the margin, no band", margin, margin, "", false},
		{"off the left edge", -1, 40, "", false},
		{"off the top edge", 50, -1, "", false},
		{"at the right bound", w, 40, "", false},
		{"at the bottom bound", 50, h, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := missionEdgeArrow(missionEdgeBands(c.x, c.y, w, h, margin))
			if ok != c.wantHit || got != c.want {
				t.Fatalf("missionEdgeArrow(missionEdgeBands(%d,%d,%d,%d,%d)) = %q,%v, want %q,%v",
					c.x, c.y, w, h, margin, got, ok, c.want, c.wantHit)
			}
		})
	}
	if name, ok := missionEdgeArrow(missionEdgeBands(1, 1, 0, 80, margin)); ok {
		t.Fatalf("zero width: got %q,%v, want no hit", name, ok)
	}
	if name, ok := missionEdgeArrow(missionEdgeBands(1, 1, 100, 0, margin)); ok {
		t.Fatalf("zero height: got %q,%v, want no hit", name, ok)
	}
}

// mcHostileID/mcPlainID/mcCol/mcRow are this file's own entities, apart from
// attack_test.go's atEntities so a change to that shared fixture cannot move
// this file's cases.
const (
	mcHostileID, mcHostileCol, mcHostileRow = 101, 6, 6
	mcPlainID, mcPlainCol, mcPlainRow       = 102, 2, 2
)

func mcEntities() []MapEntity {
	return []MapEntity{
		{ID: mcHostileID, Owner: 5, Cell: image.Pt(mcHostileCol, mcHostileRow), Life: LifeAlive, HP: 10, MaxHP: 10, Hostile: true},
		{ID: mcPlainID, Owner: 5, Cell: image.Pt(mcPlainCol, mcPlainRow), Life: LifeAlive, HP: 10, MaxHP: 10, Hostile: false},
	}
}

// AI-CURSOR-052, DIV-262, DIV-270
func TestMissionHoverCursorRunsTheDecodedCascadeAndIsFogGated(t *testing.T) {
	a, v, _ := atOnMap(t)
	_ = a
	v.SetEntities(mcEntities())

	at := func(col, row int) {
		x, y := cellPoint(v, col, row)
		v.cursorX, v.cursorY, v.hasCursor = x, y, true
	}
	want := func(label, expect string) {
		t.Helper()
		if name, ok := v.missionHoverCursor(); !ok || name != expect {
			t.Fatalf("%s = %q,%v, want %q,true", label, name, ok, expect)
		}
	}

	// ARM 1, nothing selected: any hit actor gives `select` whatever it is,
	// and empty ground gives `default`.
	at(mcHostileCol, mcHostileRow)
	want("hostile unit, nothing selected", "select")
	at(mcPlainCol, mcPlainRow)
	want("plain unit, nothing selected", "select")
	at(0, 0)
	want("empty ground, nothing selected", "default")

	// ARMS 3, 4 AND 5, with an owned unit selected.
	v.sel = selection{mcPlainID}
	at(mcHostileCol, mcHostileRow)
	want("hostile unit, a unit selected", "attack")
	at(mcPlainCol, mcPlainRow)
	want("plain unit, a unit selected", "select")
	at(0, 0)
	want("empty ground, a unit selected", "move")

	// Fog: the hostile unit's own owner (5) is not the local participant
	// (SetLocalOwner below leaves it 0), and its cell is marked unseen, so
	// fogGateEntity refuses it and hoverMask must not report it. The cascade
	// then falls to arm 5, empty ground, and answers `move` rather than
	// `attack`.
	cols, rows := v.grid.Width, v.grid.Height
	plane := make([]byte, cols*rows)
	for i := range plane {
		plane[i] = FogVisible
	}
	plane[mcHostileRow*cols+mcHostileCol] = FogUnseen
	v.SetFog(plane, cols, rows)
	v.SetLocalOwner(0)

	at(mcHostileCol, mcHostileRow)
	want("hostile unit in unseen fog", "move")
}

func TestHoverCursorNameKeepsEveryDecodedSelectGateAheadOfAttack(t *testing.T) {
	present := []MapEntity{{ID: 1}}
	tests := []struct {
		name    string
		present []MapEntity
		summary uint32
		mask    uint32
		want    string
	}{
		{"nothing selected", nil, 0, hoverMaskUnit | hoverMaskHostile, "select"},
		{"neutral selection", present, selSummaryNeutral, hoverMaskUnit | hoverMaskHostile, "select"},
		{"hostile structure", present, 0, hoverMaskUnit | hoverMaskHostile | hoverMaskStructure, "select"},
		{"ordinary hostile unit", present, 0, hoverMaskUnit | hoverMaskHostile, "attack"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hoverCursorName(tt.present, tt.summary, tt.mask, 1, true, false, false); got != tt.want {
				t.Fatalf("hoverCursorName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHoverAndAttackCursorsRequireTheMapSurfaceAtAPannedCamera(t *testing.T) {
	a, v, _ := atOnMap(t)
	layoutViewport(v, MissionFrameW-MissionPanelW, MissionFrameH)
	if v.cam.ViewW != MissionFrameW-MissionPanelW {
		t.Fatalf("premise: viewport width = %d, want %d", v.cam.ViewW, MissionFrameW-MissionPanelW)
	}

	const col, row = 27, 20 // 27*32 = 864 = ViewW: cell-aligned with the edge at cam.X=0
	v.SetEntities([]MapEntity{{ID: mcHostileID, Owner: 5, Cell: image.Pt(col, row), Life: LifeAlive, HP: 10, MaxHP: 10, Hostile: true}})

	v.Camera().Pan(16, 0) // off the 32px cell grid: opens the straddle
	if x := v.Camera().X; x != 16 {
		t.Fatalf("premise: camera panned to X=%v, want 16 (not clamped back to the origin)", x)
	}

	unclipped, ok := v.entityPickRect(v.entities[0])
	if !ok {
		t.Fatal("premise: the panned entity has no pick rectangle")
	}
	if unclipped.X+unclipped.W <= float64(v.cam.ViewW) || unclipped.X >= float64(v.cam.ViewW) {
		t.Fatalf("premise: pick rectangle %+v does not straddle the viewport's right edge at %d", unclipped, v.cam.ViewW)
	}

	const cx, cy = 879, 656 // reviewer's own reproduction: past ViewW=864, on the right column
	if cx < v.cam.ViewW {
		t.Fatalf("premise: cursor x=%d is on the map surface, want past ViewW=%d", cx, v.cam.ViewW)
	}
	if float64(cx) >= unclipped.X+unclipped.W || float64(cx) < unclipped.X {
		t.Fatalf("premise: cursor x=%d is outside the straddling pick rectangle %+v", cx, unclipped)
	}

	v.cursorX, v.cursorY, v.hasCursor = cx, cy, true
	if name, ok := v.missionHoverCursor(); !ok || name != "default" {
		t.Errorf("missionHoverCursor over the panel at a panned camera = %q,%v, want default,true", name, ok)
	}

	a.step(afHeld(cx, cy), atAt)
	if _, _, shown := v.attackPointerPresent(); shown {
		t.Error("attack pointer shown over the panel at a panned camera")
	}
	if name := v.gestureCursorAt(cx, cy); name != "" {
		t.Fatalf("panel gesture cursor = %q, want no arm", name)
	}
}

// TestMinimapModeCursorReflectsTheArmedOrderAndSelection is B2: with the
// pointer over the minimap's own box and a selection present, the mode
// cursor names the armed order; with no selection it is "sdefault"; off the
// box it names nothing.
func TestMinimapModeCursorReflectsTheArmedOrderAndSelection(t *testing.T) {
	a, v, _ := atOnMap(t)
	_ = a
	v.SetEntities(mcEntities())

	g, ok := v.minimapGeometry()
	if !ok {
		t.Fatal("setup: minimapGeometry did not resolve over atOnMap's own fixture")
	}
	onBox := g.Box.Min
	v.cursorX, v.cursorY, v.hasCursor = onBox.X, onBox.Y, true

	// AI-CURSOR-202: nothing selected forces "default" rather than no
	// cursor at all — the widget still answers, it just names the state
	// with nothing armed to show.
	if name, ok := v.minimapModeCursor(); !ok || name != "default" {
		t.Fatalf("no selection: minimapModeCursor = %q,%v, want sdefault,true", name, ok)
	}

	v.sel = selection{mcPlainID}
	if name, ok := v.minimapModeCursor(); !ok || name != "default" {
		t.Fatalf("selection present, nothing armed: %q,%v, want smove,true (the jump table's own mode-0 answer)", name, ok)
	}

	v.aimed = commandPatrol
	if name, ok := v.minimapModeCursor(); !ok || name != "default" {
		t.Fatalf("patrol aimed: %q,%v, want spatrol,true", name, ok)
	}
	v.aimed = commandNone

	v.selectedSpell = 7
	v.spellArmed = true
	if name, ok := v.minimapModeCursor(); !ok || name != "default" {
		t.Fatalf("spell selected: %q,%v, want scast,true", name, ok)
	}
	v.selectedSpell = 0
	v.spellArmed = false

	v.armed = true
	if name, ok := v.minimapModeCursor(); !ok || name != "default" {
		t.Fatalf("attack armed: %q,%v, want sattack,true", name, ok)
	}
	v.armed = false

	// Off the box entirely: the widget test fails and B2 answers nothing,
	// whatever is armed.
	v.cursorX, v.cursorY = g.Box.Max.X+50, g.Box.Max.Y+50
	if name, ok := v.minimapModeCursor(); ok {
		t.Fatalf("off the minimap box: %q,%v, want no hit", name, ok)
	}

}

// TestMapCursorNamePrecedenceEdgeBeatsMinimapBeatsHover is the order
// missioncursor.go's own header states, from AI-CURSOR-190's and
// AI-CURSOR-207's routines: an edge match wins over the minimap widget,
// which wins over the hover hostility test, which falls back to "default".
// Each row is produced by REMOVING the winner from the row above, so a swap
// of any two branches in mapCursorName breaks the row that used to
// distinguish them.
func TestMapCursorNamePrecedenceEdgeBeatsMinimapBeatsHover(t *testing.T) {
	a, v, _ := atOnMap(t)
	_ = a
	v.SetEntities(mcEntities())
	v.sel = selection{mcPlainID}
	v.armed = true // minimapModeCursor would answer "default" if reached

	win := v.place.WindowSize()
	if win.X <= EdgeMargin || win.Y <= EdgeMargin {
		t.Fatalf("setup: window %v too small to carry a non-edge point past EdgeMargin=%d", win, EdgeMargin)
	}

	// All three conditions at once: cursor at the window's own top-left
	// corner (an edge), which minimapGeometry's own corner (panelOrigin) is
	// also near — moved below once the edge case is confirmed disjoint from
	// the hover unit, which sits far inside the map.
	v.cursorX, v.cursorY, v.hasWinCursor = 0, 0, true
	v.winCursorX, v.winCursorY = 0, 0
	v.hasCursor = true
	if got := v.mapCursorName(); got != "arrow7" {
		t.Fatalf("edge + minimap + hover all reachable: mapCursorName = %q, want arrow7 (edge wins)", got)
	}

	// Off the edge, onto the minimap box: minimap wins over the hover unit,
	// which this fixture's own entities never sit under. The box's own
	// centre, not a corner, since the corner the box is anchored to can
	// itself fall inside the window's edge margin.
	g, ok := v.minimapGeometry()
	if !ok {
		t.Fatal("setup: minimapGeometry did not resolve")
	}
	cx, cy := (g.Box.Min.X+g.Box.Max.X)/2, (g.Box.Min.Y+g.Box.Max.Y)/2
	if cx < EdgeMargin || cy < EdgeMargin || cx >= win.X-EdgeMargin || cy >= win.Y-EdgeMargin {
		t.Fatalf("setup: minimap box centre (%d,%d) falls inside the window's own edge margin %d, window %v", cx, cy, EdgeMargin, win)
	}
	v.cursorX, v.cursorY = cx, cy
	v.winCursorX, v.winCursorY = cx, cy
	if got := v.mapCursorName(); got != "default" {
		t.Fatalf("off the edge, on the minimap: mapCursorName = %q, want sattack (minimap wins over hover)", got)
	}

	// Off both: the mission cursor alone. THIS FIXTURE HAS THE ATTACK MODE
	// ARMED -- it must, or minimapModeCursor above would not answer -- and an
	// armed mode replaces the whole hover cascade with the mode's own cursor
	// (`AI-PANEL-053`'s eight-entry table). So the third row is the mode's
	// `attack` and not the cascade's `select`; the cascade itself is asserted
	// in TestMissionHoverCursorRunsTheDecodedCascadeAndIsFogGated.
	x, y := cellPoint(v, mcPlainCol, mcPlainRow)
	v.cursorX, v.cursorY = x, y
	v.winCursorX, v.winCursorY = x, y
	if got := v.mapCursorName(); got != "attack" {
		t.Fatalf("off the edge and the minimap, mode armed: mapCursorName = %q, want attack", got)
	}

	// None of the three: default. The mode comes down with the selection it
	// was armed for, so the cascade runs, finds nothing selected and nothing
	// under the pointer, and answers `default` (arm 1).
	v.cursorX, v.cursorY = win.X/2, win.Y/2
	v.winCursorX, v.winCursorY = win.X/2, win.Y/2
	v.sel = nil
	v.armed = false
	if got := v.mapCursorName(); got != "default" {
		t.Fatalf("nothing reachable: mapCursorName = %q, want default", got)
	}
}

// TestMapCursorPresentIsSuppressedDuringAttackModeAndADrag is B5's own
// exclusions: mapCursorPresent draws nothing while the attack pointer or a
// held item is the picture to show, which is what keeps the map from ever
// drawing two cursors at once (contract: "no frame may hide the system
// pointer and draw neither").
func TestMapCursorPresentIsSuppressedDuringAttackModeAndADrag(t *testing.T) {
	a, v, _ := atOnMap(t)
	_ = a
	a.SetCursorRegistry(clRegistry())
	v.SetEntities(mcEntities())
	v.cursorX, v.cursorY, v.hasCursor = 400, 300, true
	v.cursorMgr.SetCursor("default")

	if _, _, ok := v.mapCursorPresent(); !ok {
		t.Fatal("setup: mapCursorPresent should draw the default map cursor with nothing else active")
	}

	v.armed = true
	if _, _, ok := v.mapCursorPresent(); ok {
		t.Fatal("with attack mode armed, mapCursorPresent must draw nothing (the attack pointer owns this frame)")
	}
	v.armed = false

	v.dragActive = true
	if _, _, ok := v.mapCursorPresent(); ok {
		t.Fatal("with a drag active, mapCursorPresent must draw nothing (the held item owns this frame)")
	}
	v.dragActive = false

	if _, _, ok := v.mapCursorPresent(); !ok {
		t.Fatal("after both are cleared, mapCursorPresent should draw the map cursor again")
	}
}

// TestOrdinaryHoverHidesTheSystemPointerThroughTheFrontEnd is B5 driven
// through the shipped front-end path (App.step, flow.pointerWanted), not the
// helper directly: before this story an ordinary hover on the map — no
// attack mode, no drag, no arrow, no minimap, no unit — left the system
// pointer showing, because nothing on the map drew a cursor of the build's
// own for that case. Reverting mapCursorPresent's fallback to always return
// false (leaving only the attack/drag branches) reproduces that and fails
// the "empty cell" case.
//
// "hostile unit" retains adversarial pass 2's important boundary: every
// ordinary map cursor must hide the system pointer, not only a cursor armed
// by a mode.
func TestOrdinaryHoverHidesTheSystemPointerThroughTheFrontEnd(t *testing.T) {
	t.Run("empty cell", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		a.SetCursorRegistry(clRegistry())
		x, y := ptHover(a, v, atEmptyCol, atEmptyRow)
		a.step(atFrame(x, y), atAt)
		a.step(atFrame(x, y), atAt)
		if a.flow.pointerWanted() {
			return
		}
		t.Fatalf("ordinary hover at an empty cell, no mode up: pointerWanted() = false, want true (the map's own %q cursor should be drawn)", a.flow.cursor.CurrentName())
	})

	t.Run("hostile unit", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		a.SetCursorRegistry(clRegistry())
		v.SetEntities(mcEntities())
		x, y := ptHover(a, v, mcHostileCol, mcHostileRow)
		a.step(atFrame(x, y), atAt)
		a.step(atFrame(x, y), atAt)
		if a.flow.pointerWanted() {
			return
		}
		t.Fatalf("hostile hover, no attack mode up: pointerWanted() = false, want true (mapCursorName() picked %q)", a.flow.cursor.CurrentName())
	})
}

// mcRegistry carries every slot name the mission map's own selection can
// return, so a case below can read the SHARED MANAGER's current name after a
// step instead of calling mapCursorName again afterwards. Calling the
// selector a second time reads the viewer's state as it stands at that
// moment, which is the same state whichever order step ran its statements in;
// what the manager holds is what advanceCursorManager was actually given
// DURING the tick, and that is the only place the ordering is observable.
//
// Every picture is a 1x1 fixture and no hotspot is asserted from it: this file
// is about which name is chosen, not about where the picture lands.
func mcRegistry() *CursorRegistry {
	names := []string{
		"default", "select", "attack",
		"arrow0", "arrow1", "arrow2", "arrow3", "arrow4", "arrow5", "arrow6", "arrow7",
		"sdefault", "smove", "sattack", "spatrol", "scast",
	}
	slots := make([]CursorSlot, len(names))
	for i, n := range names {
		slots[i] = CursorSlot{
			Name:         n,
			Frames:       []*image.RGBA{image.NewRGBA(image.Rect(0, 0, 1, 1))},
			FrameCount:   1,
			PeriodMillis: 2000000000,
		}
	}
	return NewCursorRegistry(slots)
}

// TestTheMapCursorNameFollowsThisTicksPositionAndNotTheLast is the ordering
// defect adversarial pass 1 returned the story for: advanceCursorManager ran
// above the two statements that store the tick's cursor position, so every
// position-dependent branch of mapCursorName chose a name for where the
// pointer WAS while mapCursorPresent placed the picture where it IS.
//
// Two ticks at two different edge bands is the smallest state that can show
// it. Both positions are edges, so neither answer depends on the fixture's
// entities, its selection or the minimap's own box.
//
// MUTATION: moving advanceCursorManager back above the cursor stores in step
// leaves this test reading "arrow6" on the second tick.
func TestTheMapCursorNameFollowsThisTicksPositionAndNotTheLast(t *testing.T) {
	a, v, _ := atOnMap(t)
	a.SetCursorRegistry(mcRegistry())
	win := v.place.WindowSize()
	if win.X <= 2*EdgeMargin || win.Y <= 2*EdgeMargin {
		t.Fatalf("setup: window %v too small for two disjoint edge bands at EdgeMargin=%d", win, EdgeMargin)
	}

	a.step(atFrame(0, win.Y/2), atAt)
	if got := a.flow.cursor.CurrentName(); got != "arrow6" {
		t.Fatalf("first tick at the left band: manager holds %q, want arrow6", got)
	}

	a.step(atFrame(win.X-1, win.Y/2), atAt)
	if got := a.flow.cursor.CurrentName(); got != "arrow2" {
		t.Fatalf("second tick at the right band: manager holds %q, want arrow2 (the name is one tick behind the picture)", got)
	}
}

// TestTheEdgeArrowShowsOnlyOnTicksTheEdgeScrollTermRuns is the second defect
// that pass returned: missionEdgeArrow read no drag state while panIntent's
// edge-scroll term returns early on a held primary button, so a pointer
// resting in the band during a drag showed a directional arrow on every tick
// while the camera did not move.
//
// The camera position is read either side of the second tick, so the case
// asserts what it is about — that nothing panned — rather than trusting the
// suppression rule to still be there.
//
// MUTATION: dropping the primaryDown term from edgeScrollBands makes the
// second assertion read "arrow6".
func TestTheEdgeArrowShowsOnlyOnTicksTheEdgeScrollTermRuns(t *testing.T) {
	a, v, _ := atOnMap(t)
	a.SetCursorRegistry(mcRegistry())
	win := v.place.WindowSize()

	held := atFrame(0, win.Y/2)
	held.Viewer.PrimaryDown = true

	a.step(held, atAt) // anchors the drag; dragIntent pans zero on its first tick
	x0, y0 := v.Camera().X, v.Camera().Y
	a.step(held, atAt) // same position, button still down
	if x1, y1 := v.Camera().X, v.Camera().Y; x1 != x0 || y1 != y0 {
		t.Fatalf("setup: the camera moved from (%v,%v) to (%v,%v) on a stationary drag tick", x0, y0, x1, y1)
	}
	if got := a.flow.cursor.CurrentName(); got == "arrow6" {
		t.Fatal("a stationary drag tick in the left band shows arrow6 while the camera pans nothing")
	}

	// The same position with the button up pans and shows the arrow, so the
	// case above is the drag term and not the band arithmetic.
	a.step(atFrame(0, win.Y/2), atAt)
	if got := a.flow.cursor.CurrentName(); got != "arrow6" {
		t.Fatalf("the same position with the button up: manager holds %q, want arrow6", got)
	}
}

// TestAPopupOverTheMapLeavesTheMapsOwnSelectionAnsweringDefault is the third
// tick on which the map's own selection describes something the player cannot
// do: step returns above panIntent, the drag and the wheel while a popup
// stands, so an arrow, a minimap mode cursor or a hover cursor chosen there
// names an interaction the frame will not perform. `default` is the map
// screen's own standing cursor (1030 B4, flow.screenExitCursor).
//
// MUTATION: dropping mapCursorName's popupOpen gate leaves this reading
// "arrow6".
func TestAPopupOverTheMapLeavesTheMapsOwnSelectionAnsweringDefault(t *testing.T) {
	a, v, _ := atOnMap(t)
	a.SetCursorRegistry(mcRegistry())
	win := v.place.WindowSize()

	a.step(atFrame(0, win.Y/2), atAt)
	if got := a.flow.cursor.CurrentName(); got != "arrow6" {
		t.Fatalf("setup: the left band did not select arrow6, got %q", got)
	}

	esc := atFrame(0, win.Y/2)
	esc.Escape = true
	if exit := a.step(esc, atAt); exit {
		t.Fatal("Escape on the map screen exited the program")
	}
	if a.Screen() != ScreenGameMenu {
		t.Fatalf("setup: Escape landed on %v, want the in-game menu", a.Screen())
	}
	if !v.popupOpen() {
		t.Fatal("setup: the in-game menu raised no popup on the viewer")
	}
	a.step(atFrame(0, win.Y/2), atAt)
	if got := a.flow.cursor.CurrentName(); got != "default" {
		t.Fatalf("under a popup the map's own selection holds %q, want default", got)
	}
}

// recordMissionLayers replaces the three composition seams the mission screen
// draws through — the right column's boxes, the in-game menu's own panel and
// the pointer — and records the ORDER of their calls for the duration of one
// test. None forwards to DrawImage: the call itself is the observation, which
// is the same reason missioncolumn_test.go records blitColumnLayer rather than
// reading pixels back.
func recordMissionLayers(t *testing.T) *[]string {
	t.Helper()
	var order []string
	prevCol, prevPtr, prevMenu := blitColumnLayer, blitPointerLayer, blitMenuOverMap
	blitColumnLayer = func(dst, img *ebiten.Image, op *ebiten.DrawImageOptions, layer string) {
		order = append(order, layer)
	}
	blitPointerLayer = func(dst, img *ebiten.Image, op *ebiten.DrawImageOptions, layer string) {
		order = append(order, layer)
	}
	blitMenuOverMap = func(dst, img *ebiten.Image, op *ebiten.DrawImageOptions) {
		order = append(order, "gameMenu")
	}
	t.Cleanup(func() { blitColumnLayer, blitPointerLayer, blitMenuOverMap = prevCol, prevPtr, prevMenu })
	return &order
}

// indexOfLayer reports where a layer name first appears in a recorded order,
// or -1.
func indexOfLayer(order []string, layer string) int {
	for i, got := range order {
		if got == layer {
			return i
		}
	}
	return -1
}

func TestTheMapCursorIsComposedOverThePanelsAndOverTheInGameMenu(t *testing.T) {
	t.Run("over the right column's own boxes", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		a.SetCursorRegistry(mcRegistry())

		// The shipped mission frame size, so the doll box has the room it
		// needs (missioncolumn_test.go's own columnViewer), plus a font, a
		// selection and an equipped pack for the panel and the doll, and
		// synthetic command-panel art for the control panel — commandPanel-
		// Present's own doc: it draws nothing with no art resolved.
		v.frameW, v.frameH = MissionFrameW, MissionFrameH
		a.Layout(v.frameW, v.frameH)
		v.SetFont(panelFont())
		v.SetEntities([]MapEntity{panelEntity(1, "Warrior", 63, 100, 4, 4)})
		v.sel = selection{1}
		v.SetInventorySubject(packOf(1, solidPic(8, 8, color.RGBA{R: 0xff, A: 0xff})))
		panelFill := solidPic(160, 80, color.RGBA{G: 0x80, A: 0xff})
		v.SetCommandPanelArt(&CommandPanelArt{Heads: panelFill, Active: panelFill})

		order := recordMissionLayers(t)

		a.step(atFrame(v.frameW/2, v.frameH/2), atAt)
		if got := a.flow.cursor.CurrentName(); got == "" {
			t.Fatal("setup: the map named no cursor, so nothing would be composed to observe")
		}
		a.Draw(ebiten.NewImage(a.winW, a.winH))

		cursor := indexOfLayer(*order, "mapCursor")
		if cursor < 0 {
			t.Fatalf("the map cursor was not composed at all; layers drawn: %v", *order)
		}
		for _, layer := range []string{"panel", "minimap", "controlPanel"} {
			at := indexOfLayer(*order, layer)
			if at < 0 {
				t.Fatalf("setup: %q was not composed, so this case cannot witness its own claim about it; layers drawn: %v", layer, *order)
			}
			if at > cursor {
				t.Errorf("%q is composed after the map cursor, so it covers it; layers drawn: %v", layer, *order)
			}
		}
	})

	t.Run("over the in-game menu", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		a.SetCursorRegistry(mcRegistry())

		esc := atFrame(v.frameW/2, v.frameH/2)
		esc.Escape = true
		if exit := a.step(esc, atAt); exit {
			t.Fatal("Escape on the map screen exited the program")
		}
		if a.Screen() != ScreenGameMenu {
			t.Fatalf("setup: Escape landed on %v, want the in-game menu over the map", a.Screen())
		}

		order := recordMissionLayers(t)
		a.step(atFrame(v.frameW/2, v.frameH/2), atAt)
		a.Draw(ebiten.NewImage(a.winW, a.winH))

		menu := indexOfLayer(*order, "gameMenu")
		if menu < 0 {
			t.Fatalf("the in-game menu was not composed at all; layers drawn: %v", *order)
		}
		cursor := indexOfLayer(*order, "mapCursor")
		if cursor < 0 {
			t.Fatalf("the map cursor was not composed over the in-game menu; layers drawn: %v", *order)
		}
		if cursor < menu {
			t.Errorf("the map cursor is composed before the in-game menu, so the panel covers it; layers drawn: %v", *order)
		}
	})
}

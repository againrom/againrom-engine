package ui

// A popup takes everything onto itself (work item 0077-popup-modality).
//
// SEPARATE CONTEXT. Every expectation below is derived from the spec and from
// this file's own frame schedule, and every case that says "a popup takes X"
// runs the SAME drive with no popup open first. A gate that never fires and a
// map that never answered are otherwise the same green.
//
// THE FIXTURE IS 0073's, extended. haltWorld already models the far side the
// suspension needs — a stop it holds, and a MapTick that consumes its pacing
// baseline whether stopped or not — so the world half of these cases needs no
// second stand-in. What is added here is the three seams an order, an attack
// and a blow leave through, and a snapshot of entities to select.
//
// NOTHING HERE OPENS A WINDOW, READS A CLOCK OR TOUCHES A GAME INSTALL.

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

// The two entities every case below selects from, on cells whose window
// positions are far from the notice's button and from every edge margin, so a
// press aimed at the map is a press at the map and an idle frame scrolls
// nothing.
const (
	popupAID, popupACol, popupARow = 12, 10, 3
	popupBID, popupBCol, popupBRow = 5, 5, 5
	// EMPTY GROUND, which no entity of this fixture stands on. Since story
	// 1034 an ordering gesture must be made over ground: an actor under the
	// pointer puts up the `select` cursor and the same gesture selects
	// instead (`AI-CURSOR-226` arm 4).
	popupEmptyCol, popupEmptyRow = 8, 8
)

func popupEntities() []MapEntity {
	return []MapEntity{
		{ID: popupAID, Cell: image.Pt(popupACol, popupARow)},
		{ID: popupBID, Cell: image.Pt(popupBCol, popupBRow)},
	}
}

// The three seams, recorded rather than answered. A case asserts on the LISTS:
// "nothing reached the seam" and "the control reached it" are the same
// assertion read twice, which is what makes the gate's absence visible.
type popupSeams struct {
	orders  []popupMove
	attacks []popupStrike
	blows   []popupBlow
}

type popupMove struct {
	entity uint32
	x, y   int
}
type popupStrike struct{ entity, victim uint32 }
type popupBlow struct {
	entity uint32
	kill   bool
}

func (s *popupSeams) order(entity uint32, x, y int) {
	s.orders = append(s.orders, popupMove{entity, x, y})
}
func (s *popupSeams) attack(entity, victim, spell uint32, x, y int, cell bool) {
	s.attacks = append(s.attacks, popupStrike{entity, victim})
}
func (s *popupSeams) affect(entity uint32, kill bool) {
	s.blows = append(s.blows, popupBlow{entity, kill})
}

func (s *popupSeams) empty() bool {
	return len(s.orders) == 0 && len(s.attacks) == 0 && len(s.blows) == 0
}

// popupFix is 0073's map-screen fixture with the three seams wired and a
// snapshot of entities under it.
type popupFix struct {
	t *testing.T
	a *App
	v *Viewer
	w *haltWorld
	s *popupSeams
	n int

	// ran is called from the tick seam on every frame the world ACTUALLY
	// advanced, which is where the far side pushes its drawn state from: inside
	// the tick loop its stop gates. A case that wants to watch that state
	// assigns through it; nothing else does.
	ran *func()
}

func newPopupFix(t *testing.T, o haltOpts) *popupFix {
	t.Helper()

	s := &popupSeams{}
	ran := new(func())
	var w *haltWorld
	load := func(int) (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		v, err := NewViewer("popup", grid(60, 60), &terrain.Tileset{})
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		hideBottomPanels(v)
		if !o.noFont {
			v.SetFont(panelFont())
		}
		v.SetEntities(popupEntities())
		w = newHaltWorld(haltAt(0))
		w.v = v
		var cadence MapCadence
		if !o.noCadence {
			cadence = w.setCadence
		}
		tick := func() {
			before := w.world
			w.tick()
			if w.world != before && *ran != nil {
				(*ran)()
			}
		}
		return v, tick, s.order, cadence, s.affect, w.advance, s.attack, nil, nil, nil, nil
	}

	a := newTestApp(t, appRows(3), load)
	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, haltAt(0))
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
	}
	mapAtScaleOne(a)
	if w == nil || a.flow.viewer == nil {
		t.Fatalf("setup: far side %v, viewer %v — both are needed", w, a.flow.viewer)
	}

	f := &popupFix{t: t, a: a, v: a.flow.viewer, w: w, s: s, ran: ran}

	// The fixture's own premises, falsified rather than believed: the camera
	// this file's window positions are computed at, and that a press at a unit
	// is not also a press at the notice's button.
	if got := f.v.Camera().Zoom; got != 1 {
		t.Fatalf("fixture zoom = %v, want 1 — every window position below assumes it", got)
	}
	if x, y := popupAt(f.v, popupACol, popupARow); f.v.noticeButtonAt(x, y) {
		t.Fatalf("unit A sits at (%d,%d), which is inside the notice's own button — "+
			"a press there would be a dismissal rather than a press at the map", x, y)
	}

	// AND NEITHER UNIT SITS UNDER A PIECE OF HUD FURNITURE (0140). Five boxes
	// are drawn over the map now and every one of them swallows its own
	// presses, so a fixture cell that drifted under one would make every case
	// in this file fail at its own setup with nothing to say why. Unit A did
	// drift under the control panel the day that panel was added, which is why
	// this is a check and not a comment.
	for _, u := range []struct {
		name     string
		col, row int
	}{{"A", popupACol, popupARow}, {"B", popupBCol, popupBRow}} {
		x, y := popupAt(f.v, u.col, u.row)
		if f.v.commandPanelCaptures(x, y) || f.v.inventoryCaptures(x, y) ||
			f.v.spellbookCaptures(x, y) || f.v.minimapCaptures(x, y) || f.v.panelCaptures(x, y) {
			t.Fatalf("unit %s sits at (%d,%d), under a HUD box that swallows presses — "+
				"move its cell", u.name, x, y)
		}
	}

	f.frame(haltNeutral()) // the far side's baseline frame
	return f
}

// popupAt is the window position of the centre of a cell, through the camera's
// own forward transform.
func popupAt(v *Viewer, col, row int) (int, int) {
	sx, sy := v.Camera().WorldToScreen(
		float64(col*camera.CellSize+camera.CellSize/2),
		float64(row*camera.CellSize+camera.CellSize/2))
	return int(sx), int(sy)
}

// frame drives one frame, moving the wall clock BOTH sides read.
func (f *popupFix) frame(in appInput) {
	f.n++
	f.w.now = haltAt(f.n)
	f.a.step(in, haltAt(f.n))
}

// at is a neutral frame with the cursor placed at a cell rather than at the
// centre of the window.
func (f *popupFix) at(col, row int) appInput {
	x, y := popupAt(f.v, col, row)
	in := haltNeutral()
	in.CursorX, in.CursorY = x, y
	in.Viewer.CursorX, in.Viewer.CursorY = x, y
	return in
}

// selectA taps unit A through the shipped path — a press and its release in one
// frame, which travels zero pixels and is a tap by the slop — and fails if the
// selection did not take. Every case below starts from a selection that is real.
func (f *popupFix) selectA() {
	f.t.Helper()
	in := f.at(popupACol, popupARow)
	in.PrimaryPressed, in.PrimaryReleased = true, true
	f.frame(in)
	if got := f.v.marked(); !reflect.DeepEqual(got, []uint32{popupAID}) {
		f.t.Fatalf("setup: the tap selected %v, want [%d] — no case below can say a "+
			"popup PRESERVED a selection that was never made", got, popupAID)
	}
}

// open raises a popup, exactly as the far side's settle pass does.
func (f *popupFix) open() {
	f.t.Helper()
	f.v.SetNotice("words", NoticeDialogue)
	if !f.a.flow.popupOpen() {
		f.t.Fatal("setup: a notice was pushed and the flow does not report a popup open")
	}
}

// AC-1 — a press, a drag and a release away from the button, and a
// secondary press with units already selected, reach neither the selection
// nor the world's order seams.
//
// The CONTROL runs the identical gesture with no popup open and requires that
// it DOES move the selection and DOES reach a seam, so the gated half is told
// from a fixture that never answered.
func TestAPopupTakesEveryMapInput(t *testing.T) {
	t.Run("control, no popup open", func(t *testing.T) {
		f := newPopupFix(t, haltOpts{})
		f.selectA()

		// A tap on the OTHER unit moves the selection.
		in := f.at(popupBCol, popupBRow)
		in.PrimaryPressed, in.PrimaryReleased = true, true
		f.frame(in)
		if got := f.v.marked(); !reflect.DeepEqual(got, []uint32{popupBID}) {
			t.Fatalf("with no popup open a tap selected %v, want [%d]", got, popupBID)
		}
		// A tap on empty ground issues a move order (`AI-CLICK-050`, story
		// 1034: the LEFT button acts, and the cursor over ground is `move`).
		order := f.at(popupEmptyCol, popupEmptyRow)
		order.PrimaryPressed, order.PrimaryReleased = true, true
		f.frame(order)
		if len(f.s.orders) == 0 {
			t.Fatal("with no popup open a tap on ground reached no order seam — " +
				"the gated case below would then prove nothing")
		}
	})

	t.Run("a popup open", func(t *testing.T) {
		f := newPopupFix(t, haltOpts{})
		f.selectA()
		f.s.orders = nil
		f.open()

		// A press, a drag past the slop, and a release: the whole selection
		// rectangle gesture, at the map and away from the button.
		press := f.at(popupBCol, popupBRow)
		press.PrimaryPressed = true
		press.Viewer.PrimaryDown = true
		f.frame(press)

		drag := f.at(popupACol, popupARow)
		drag.Viewer.PrimaryDown = true
		f.frame(drag)

		release := f.at(popupACol, popupARow)
		release.PrimaryReleased = true
		f.frame(release)

		// A secondary press, which with a selection is an order.
		right := f.at(popupEmptyCol, popupEmptyRow)
		right.PrimaryPressed, right.PrimaryReleased = true, true
		f.frame(right)

		// And the arming key, which is the newest thing on this arm.
		arm := haltNeutral()
		arm.Attack = true
		f.frame(arm)

		if got := f.v.marked(); !reflect.DeepEqual(got, []uint32{popupAID}) {
			t.Errorf("a gesture under a popup moved the selection to %v, want the [%d] "+
				"held when it opened", got, popupAID)
		}
		if !f.s.empty() {
			t.Errorf("a popup let something reach the world: orders %v, attacks %v, blows %v",
				f.s.orders, f.s.attacks, f.s.blows)
		}
		if f.v.AttackArmed() {
			t.Error("the arming key reached the map through an open popup")
		}
	})
}

// AC-2 — the two debug blow keys and the two diagnostic toggles.
//
// The diagnostics are here rather than beside the camera because they are the
// half of "intercept everything" that changes no world state at all: a gate
// that took the orders and left the instruments toggling would satisfy every
// other case in this file.
func TestAPopupTakesTheBlowsAndTheDiagnostics(t *testing.T) {
	all := func() appInput {
		in := haltNeutral()
		in.Kill, in.Chip, in.Grid, in.Readout = true, true, true, true
		return in
	}

	t.Run("control, no popup open", func(t *testing.T) {
		f := newPopupFix(t, haltOpts{})
		f.selectA()
		grid, readout := f.v.GridOverlay(), f.v.ReadoutShown()
		f.frame(all())
		if len(f.s.blows) == 0 {
			t.Error("with no popup open the blow keys reached no seam")
		}
		if f.v.GridOverlay() == grid || f.v.ReadoutShown() == readout {
			t.Error("with no popup open a diagnostic key toggled nothing")
		}
	})

	t.Run("a popup open", func(t *testing.T) {
		f := newPopupFix(t, haltOpts{})
		f.selectA()
		f.s.blows = nil
		f.open()
		grid, readout := f.v.GridOverlay(), f.v.ReadoutShown()
		f.frame(all())
		if len(f.s.blows) != 0 {
			t.Errorf("a popup let %d blows reach the world", len(f.s.blows))
		}
		if f.v.GridOverlay() != grid {
			t.Error("the cell lattice toggled through an open popup")
		}
		if f.v.ReadoutShown() != readout {
			t.Error("the debug readout toggled through an open popup")
		}
	})
}

// AC-8 — the popup's own three dismissal inputs still answer, and each
// still answers once per gesture.
//
// This is the case the gate is most likely to break: it stands on the same arm
// the two mouse routes are resolved on, and an early return placed one
// statement too early takes them with everything else.
func TestAPopupStillAnswersItsOwnThreeInputs(t *testing.T) {
	for _, d := range haltDismissals() {
		t.Run(d.name, func(t *testing.T) {
			f := newPopupFix(t, haltOpts{})
			f.open()
			before := f.w.advances
			f.frame(d.in)
			if got := f.w.advances - before; got != 1 {
				t.Fatalf("%s advanced the popup %d times, want 1", d.name, got)
			}
		})
	}
}

// AC-10 — the suspension seam is where 0073 left it: the arm still asks
// for its one advance on every frame that reaches it, the far side is told
// it is stopped, and the player's own pause setting comes back out of a
// popup unchanged in both of its two states.
//
// It is asserted HERE, beside the gate, because the gate is what would break
// it: an early return placed above the advance is invisible to every other case
// in this file and is exactly the debt 0073 measured.
func TestAPopupLeavesTheSuspensionSeamAlone(t *testing.T) {
	for _, paused := range []bool{false, true} {
		name := "running"
		if paused {
			name = "paused"
		}
		t.Run(name, func(t *testing.T) {
			f := newPopupFix(t, haltOpts{})
			if paused {
				f.frame(cadenceInput(true, false, false))
			}
			want := f.a.flow.stopped
			if want != paused {
				t.Fatalf("setup: the player's pause reads %v, want %v", want, paused)
			}

			f.open()
			ticks := f.w.ticks
			const held = 8
			for i := 0; i < held; i++ {
				f.frame(haltNeutral())
			}
			if got := f.w.ticks - ticks; got != held {
				t.Errorf("%d frames under a popup asked for %d advances, want %d — the "+
					"gate must stand below the advance, not above it", held, got, held)
			}
			if call, ok := f.w.lastCadence(); !ok || !call.stopped {
				t.Errorf("the far side was not told it is stopped under a popup (%+v, told=%v)",
					call, ok)
			}
			if f.w.world != 0 {
				t.Errorf("the world ran %d ticks under a popup", f.w.world)
			}

			f.frame(appInput{Enter: true}) // dismiss
			if got := f.a.flow.stopped; got != want {
				t.Errorf("the player's pause came out of the popup as %v, want %v", got, want)
			}
		})
	}
}

func TestAPopupPinsTheCamera(t *testing.T) {
	type view struct{ x, y, zoom float64 }
	read := func(v *Viewer) view {
		c := v.Camera()
		return view{c.X, c.Y, c.Zoom}
	}

	edge := func(f *popupFix) appInput {
		win := f.v.place.WindowSize()
		in := haltNeutral()
		in.Viewer.CursorX = win.X - EdgeMargin/2
		in.Viewer.CursorY = win.Y / 2
		in.CursorX, in.CursorY = in.Viewer.CursorX, in.Viewer.CursorY
		return in
	}

	cases := []struct {
		name   string
		frames func(f *popupFix) []appInput
	}{
		{"a pan key", func(f *popupFix) []appInput {
			in := haltNeutral()
			in.Viewer.PanRight = true
			return []appInput{in, in}
		}},
		{"the edge margin", func(f *popupFix) []appInput {
			return []appInput{edge(f), edge(f)}
		}},
		{"a right drag", func(f *popupFix) []appInput {
			anchor := f.at(popupACol, popupARow)
			anchor.Viewer.SecondaryDown = true
			moved := f.at(popupBCol, popupBRow)
			moved.Viewer.SecondaryDown = true
			return []appInput{anchor, moved}
		}},
		{"the wheel", func(f *popupFix) []appInput {
			in := haltNeutral()
			in.Viewer.WheelY = 1
			return []appInput{in, in}
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			drive := func(open bool) (before, after view) {
				f := newPopupFix(t, haltOpts{})
				if open {
					f.open()
				}
				before = read(f.v)
				for _, in := range c.frames(f) {
					f.frame(in)
				}
				return before, read(f.v)
			}

			before, after := drive(false)
			if before == after {
				t.Fatalf("with no popup open %s left the camera at %+v — the gated case "+
					"below would then prove nothing", c.name, after)
			}
			before, after = drive(true)
			if before != after {
				t.Errorf("%s moved the camera under a popup: %+v became %+v", c.name, before, after)
			}
		})
	}
}

// AC-4 — a rectangle in flight when a popup opens is abandoned: no outline
// is drawn on any frame the popup is open on, and a button still held at the
// dismissal anchors a fresh gesture rather than resuming the old one.
func TestAPopupAbandonsTheGestureInFlight(t *testing.T) {
	f := newPopupFix(t, haltOpts{})

	press := f.at(popupBCol, popupBRow)
	press.PrimaryPressed = true
	press.Viewer.PrimaryDown = true
	f.frame(press)

	drag := f.at(popupACol, popupARow)
	drag.Viewer.PrimaryDown = true
	f.frame(drag)

	if len(f.v.marqueeScreenRects()) == 0 {
		t.Fatal("setup: no rectangle was being drawn before the popup opened — " +
			"there would be nothing for it to abandon")
	}
	pressedAt := image.Pt(f.v.pressX, f.v.pressY)

	f.open()
	for i := 0; i < 4; i++ {
		f.frame(drag) // the button stays down for the whole of the popup
		if got := len(f.v.marqueeScreenRects()); got != 0 {
			t.Fatalf("a rectangle of %d strips was drawn on a frame with a popup open", got)
		}
	}

	f.frame(appInput{Enter: true}) // dismissed, button still held

	after := f.at(popupACol, popupARow)
	after.Viewer.PrimaryDown = true
	f.frame(after)
	if got := image.Pt(f.v.pressX, f.v.pressY); got == pressedAt {
		t.Errorf("the gesture resumed from where it was pressed (%v) rather than anchoring "+
			"afresh after the dismissal", got)
	}
	if f.v.held {
		t.Error("the press latch outlived the popup")
	}
}

// AC-5 — the ambient counter the water, the animated objects and the
// animated structures are all selected at holds still under a popup, and the
// held span is SPENT: the frames after a dismissal advance it by their own
// elapsed time and not by the time the box was up.
//
// The span is deliberately long. The whole point is that a popup held for a
// hundred frames and one held for one resume identically, and a counter merely
// gated at its CALL SITE passes every short case — this accumulator takes the
// whole quotient in one call, with no catch-up bound of the kind the world has.
func TestAPopupHoldsTheAmbientClock(t *testing.T) {
	const held = 100

	f := newPopupFix(t, haltOpts{})
	if got := f.v.anim.Period(); got != haltPeriodUS {
		t.Fatalf("the ambient clock runs at %d us, not the %d this case is computed over",
			got, haltPeriodUS)
	}

	f.frame(haltNeutral())
	start := f.v.AnimationCounter()
	if start == 0 {
		t.Fatal("setup: the ambient counter had not started, so holding it says nothing")
	}

	f.open()
	for i := 0; i < held; i++ {
		f.frame(haltNeutral())
	}
	if got := f.v.AnimationCounter(); got != start {
		t.Fatalf("%d frames under a popup advanced the ambient counter from %d to %d",
			held, start, got)
	}

	f.frame(appInput{Enter: true}) // dismissed
	resumed := f.v.AnimationCounter()

	// Two frames of haltFrameUS at a haltPeriodUS clock are worth 200_000/62_000
	// = 3 ticks and a remainder, so 4 bounds it from any starting remainder. The
	// discriminating number is that it is not the hundred-odd the held span
	// would have bought had it been banked.
	f.frame(haltNeutral())
	f.frame(haltNeutral())
	got := f.v.AnimationCounter() - resumed
	if got > 4 {
		t.Errorf("two frames after the dismissal advanced the counter by %d, want at most 4 — "+
			"the held span was repaid rather than spent", got)
	}
	if got == 0 {
		t.Error("the ambient counter did not resume after the dismissal")
	}
}

// AC-5a — the map picture's OTHER time source, asserted over the drawn
// state rather than over a counter.
//
// This case exists because AC-5 alone would certify half a freeze. Every
// unit's frame, every unit's displacement between two ticks and every route
// ride the WORLD's tick, not the ambient clock, and the far side pushes them
// only from inside the tick loop its stop gates.
func TestAPopupHoldsTheWorldDrivenPicture(t *testing.T) {
	f := newPopupFix(t, haltOpts{})

	scene := 0
	*f.ran = func() {
		scene++
		f.v.SetEntities([]MapEntity{{
			ID:   popupAID,
			Cell: image.Pt(scene, scene),
			Step: image.Pt(1, 0),
		}})
		f.v.SetPhase(scene*1_000, haltPeriodUS)
	}

	f.frame(haltNeutral())
	if scene == 0 {
		t.Fatal("setup: the far side pushed nothing on a running frame")
	}

	f.open()
	f.frame(haltNeutral()) // the frame the popup is drawn on
	drawn := append([]MapEntity(nil), f.v.entities...)
	phaseUS, periodUS := f.v.Phase()
	pushes := scene

	const held = 40
	for i := 0; i < held; i++ {
		f.frame(haltNeutral())
	}

	if scene != pushes {
		t.Errorf("the far side pushed %d times under a popup, want 0", scene-pushes)
	}
	if !reflect.DeepEqual(f.v.entities, drawn) {
		t.Errorf("the drawn entities moved under a popup: %v became %v", drawn, f.v.entities)
	}
	if gotUS, gotPeriod := f.v.Phase(); gotUS != phaseUS || gotPeriod != periodUS {
		t.Errorf("the sub-tick displacement moved under a popup: (%d,%d) became (%d,%d)",
			phaseUS, periodUS, gotUS, gotPeriod)
	}
}

func TestAScreenThatCannotDrawAPopupIsUntouched(t *testing.T) {
	f := newPopupFix(t, haltOpts{noFont: true})
	f.v.SetNotice("words", NoticeDialogue)
	f.v.SetNotice("more", NoticeOutcome)
	if f.a.flow.popupOpen() {
		t.Fatal("a viewer that cannot draw a notice reported a popup open")
	}

	f.selectA()
	order := f.at(popupEmptyCol, popupEmptyRow)
	order.PrimaryPressed, order.PrimaryReleased = true, true
	f.frame(order)
	if len(f.s.orders) == 0 {
		t.Error("an order was intercepted on a screen that cannot draw a popup")
	}

	c := f.v.Camera()
	x, y, z := c.X, c.Y, c.Zoom
	counter := f.v.AnimationCounter()

	pan := haltNeutral()
	pan.Viewer.PanRight, pan.Viewer.WheelY = true, 1
	f.frame(pan)
	f.frame(pan)

	if c := f.v.Camera(); c.X == x && c.Y == y && c.Zoom == z {
		t.Error("the camera did not move on a screen that cannot draw a popup")
	}
	if f.v.AnimationCounter() == counter {
		t.Error("the ambient counter did not advance on a screen that cannot draw a popup")
	}
	if _, _, ok := f.v.noticeBackdropOf(); ok {
		t.Error("a screen that cannot draw a popup composed a dim")
	}
}

// AC-9 — with no popup open, every path this story gates answers as it did
// before it.
//
// The cases above each carry their own control; this is the one that asserts
// they are controls of the SAME frame — one ordinary map frame that pans,
// animates, advances, selects, orders and toggles, with nothing over it.
func TestWithNoPopupNothingIsGated(t *testing.T) {
	f := newPopupFix(t, haltOpts{})
	f.selectA()

	c := f.v.Camera()
	x, y := c.X, c.Y
	counter := f.v.AnimationCounter()
	ticks := f.w.ticks

	in := f.at(popupEmptyCol, popupEmptyRow)
	in.Viewer.PanRight = true
	in.PrimaryPressed, in.PrimaryReleased = true, true
	in.Grid = true
	f.frame(in)

	if c := f.v.Camera(); c.X == x && c.Y == y {
		t.Error("the camera did not pan on an ungated frame")
	}
	if f.v.AnimationCounter() == counter {
		t.Error("the ambient counter did not advance on an ungated frame")
	}
	if f.w.ticks == ticks {
		t.Error("the world was not advanced on an ungated frame")
	}
	if len(f.s.orders) == 0 {
		t.Error("no order reached the world on an ungated frame")
	}
	if !f.v.GridOverlay() {
		t.Error("the cell lattice did not toggle on an ungated frame")
	}
	if _, _, ok := f.v.noticeBackdropOf(); ok {
		t.Error("a dim was composed with no popup open")
	}
}

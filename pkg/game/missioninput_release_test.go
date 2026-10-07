package game

import (
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// releaseMissionInputApp opens a real campaign mission through the production
// front end and returns the App, the live world and one live entity the local
// participant owns.
//
// The mission is 10, the campaign's first, chosen because every asset root
// ships it and `pipeline/check-milestone.sh` already drives it: a failure here
// is a failure on content the rest of the pipeline already measures.
func releaseMissionInputApp(t *testing.T) (*ui.App, *mapWorld, sim.EntityID) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1034-mission-input")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatalf("open mission 10: %v", err)
	}
	if f.live == nil {
		t.Fatal("mission 10 opened with no live world")
	}
	if len(f.live.mission.ids) == 0 {
		t.Fatal("mission 10 opened with no party entity")
	}
	return app, f.live, f.live.mission.ids[0]
}

// TestReleaseMissionMapLeftTapSelectsThenOrdersOnRealMissionContent drives the
// left button's whole decoded contract on mission 10: a tap on an owned unit
// selects it (`AI-CLICK-050`'s `select` arm), and a tap on plain ground with
// that selection standing issues one move order per selected unit
// (`AI-CLICK-050`'s `move` arm, order `0x16`).
//
// The order is observed at the simulation's own queue, which is the value the
// production path produces, rather than at any intermediate return: enqueue
// appends a sim.Command and marks the entity commanded in one statement, so the
// queue is where the whole chain's result actually lands.
func TestReleaseMissionMapLeftTapSelectsThenOrdersOnRealMissionContent(t *testing.T) {
	app, live, id := releaseMissionInputApp(t)

	// A production left press and release on the unit's own drawn pixel. The
	// point comes from the production hit test, not from arithmetic here.
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatalf("select %d through production input: %v", id, err)
	}
	got, ok := live.view.SelectedUnit()
	if !ok || got != uint32(id) {
		t.Fatalf("after a left tap on the unit, SelectedUnit = (%d,%v), want (%d,true)", got, ok, id)
	}

	before := len(live.pending)
	if live.commanded[id] {
		t.Fatalf("entity %d was already commanded before this test issued anything", id)
	}

	gx, gy, err := app.HeadlessGroundPoint()
	if err != nil {
		t.Fatalf("ground point: %v", err)
	}
	// A tap: press and release at the same pixel, so the gesture travels zero
	// screen pixels and is under the marquee threshold whatever the frame is.
	if err := app.HeadlessPointer("press", gx, gy); err != nil {
		t.Fatalf("press on ground: %v", err)
	}
	if err := app.HeadlessPointer("release", gx, gy); err != nil {
		t.Fatalf("release on ground: %v", err)
	}
	// THE ORACLE IS READ AFTER THE RELEASE, not before the press. Both
	// HeadlessDropCell and the order path ask the same v.groundCellAt, so the
	// only thing that can put them out of step is the camera, and the camera
	// moves on a frame the pointer spends near the view edge -- which is where
	// HeadlessGroundPoint's own search starts. Reading it after the release
	// reads it at the camera the release frame decided with; reading it before
	// the press disagreed by one column on mission 10.
	cx, cy, err := app.HeadlessDropCell(gx, gy)
	if err != nil {
		t.Fatalf("drop cell at (%d,%d): %v", gx, gy, err)
	}

	if len(live.pending) != before+1 {
		t.Fatalf("a left tap on ground with one unit selected queued %d commands, want 1 — the left button does not order",
			len(live.pending)-before)
	}
	cmd := live.pending[len(live.pending)-1]
	if cmd.Entity != id {
		t.Errorf("queued command names entity %d, want %d", cmd.Entity, id)
	}
	if cmd.Kind != sim.KindGroupMoveTo {
		t.Errorf("queued command kind = %d, want KindGroupMoveTo (%d)", cmd.Kind, sim.KindGroupMoveTo)
	}
	if int(cmd.X) != cx || int(cmd.Y) != cy {
		t.Errorf("queued command cell = (%d,%d), want the tapped cell (%d,%d)", cmd.X, cmd.Y, cx, cy)
	}
	if !live.commanded[id] {
		t.Errorf("entity %d is not marked commanded after its order was queued", id)
	}
	// The selection survives the order. `AI-SELECT-122` gives no path by which
	// issuing an order clears a selection, and a build that dropped it would
	// make every second order impossible.
	if got, ok := live.view.SelectedUnit(); !ok || got != uint32(id) {
		t.Errorf("after ordering, SelectedUnit = (%d,%v), want (%d,true)", got, ok, id)
	}
}

// TestReleaseMissionMapRightUpCancelsOnRealMissionContent drives the right
// button's decoded contract on mission 10: a press and release with no drag
// between them cancels — with nothing armed, it clears the selection
// (`AI-INPUT-127`) — and it issues no order.
//
// The no-order half is the assertion that discriminates against the build this
// story replaced, where exactly this gesture was how a player ordered.
func TestReleaseMissionMapRightUpCancelsOnRealMissionContent(t *testing.T) {
	app, live, id := releaseMissionInputApp(t)

	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatalf("select %d through production input: %v", id, err)
	}
	if got, ok := live.view.SelectedUnit(); !ok || got != uint32(id) {
		t.Fatalf("after a left tap on the unit, SelectedUnit = (%d,%v), want (%d,true)", got, ok, id)
	}

	before := len(live.pending)
	gx, gy, err := app.HeadlessGroundPoint()
	if err != nil {
		t.Fatalf("ground point: %v", err)
	}
	if err := app.HeadlessPointer("right-press", gx, gy); err != nil {
		t.Fatalf("right press on ground: %v", err)
	}
	if err := app.HeadlessPointer("right-release", gx, gy); err != nil {
		t.Fatalf("right release on ground: %v", err)
	}

	if len(live.pending) != before {
		t.Errorf("a right press and release queued %d commands, want 0 — the right button must not order",
			len(live.pending)-before)
	}
	if got, ok := live.view.SelectedUnit(); ok {
		t.Errorf("after a right press and release with nothing armed, SelectedUnit = (%d,true), want no selection", got)
	}
	if live.commanded[id] {
		t.Errorf("entity %d is marked commanded after a right press and release", id)
	}
}

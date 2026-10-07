package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// attackMoveArrival is how near its commanded cell a unit counts as arrived.
// The witness states it here rather than reading the production distance.
const attackMoveArrival = 2

func attackMoveDist(e sim.Entity, x, y int32) int32 {
	dx, dy := e.X-x, e.Y-y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return max(dx, dy)
}

// attackMoveStep answers any open notice the way a player does, then advances
// the App one step.
func attackMoveStep(app *ui.App) error {
	for i := 0; i < 16 && app.HeadlessNoticeOpen(); i++ {
		if err := app.HeadlessKey("enter"); err != nil {
			return err
		}
	}
	return app.HeadlessStep()
}

func attackMoveEntity(w *sim.World, id sim.EntityID) sim.Entity {
	for _, e := range w.Entities() {
		if e.ID == id {
			return e
		}
	}
	return sim.Entity{}
}

// attackMoveReload saves the mission as it stands, restores it through the
// ordinary LOAD path and steps the restored world beside the saved bytes'
// own control until the unit stands on its commanded cell.
func attackMoveReload(t *testing.T, f *FrontEnd, hero sim.EntityID, dx, dy int32) {
	t.Helper()
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := EncodeSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(raw)
	if err != nil {
		t.Fatal(err)
	}
	back := releaseFront(t)
	opener, town, err := back.Restore(decoded)
	if err != nil || town || opener == nil {
		t.Fatal("restore", err, town)
	}
	app := back.App("attack-move-restore")
	app.Layout(1024, 768)
	if err := app.OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	var control sim.World
	if err := control.UnmarshalBinary(snapshot.World); err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(&control)
	for n := 0; n < 4000; n++ {
		sim.Step(&control, nil)
		sim.Step(back.live.world, nil)
		if n%8 != 0 {
			continue
		}
		if control.Hash() != back.live.world.Hash() {
			t.Fatalf("restored world differs from its saved bytes at tick %d", n)
		}
		if e := attackMoveEntity(back.live.world, hero); e.Alive() && !e.HasTarget && attackMoveDist(e, dx, dy) <= attackMoveArrival {
			t.Logf("after LOAD mid-fight: unit on its commanded cell (%d,%d) at tick %d, control identical", dx, dy, n)
			return
		}
	}
	e := attackMoveEntity(back.live.world, hero)
	t.Fatalf("after LOAD mid-fight the unit never reached its commanded cell (%d,%d); it stands at (%d,%d), HasTarget=%v", dx, dy, e.X, e.Y, e.HasTarget)
}

// The attack move ("Swarm" panel cell, RU "Идти в боевой готовности") is
// pressed on the command panel and aimed with a ground click, on an installed
// mission's hero with the weakest fighting hostile of the mission placed on
// its way. The unit fights it and, once nothing is in sight, walks on to the
// clicked cell and stays there; a save taken mid-fight and restored does the
// same (AI-SWARM2GATE-107, AI-MOVE-023).
func TestReleaseAttackMoveWalksOnAfterTheFight(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("attack-move")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	key := func(k string) {
		t.Helper()
		if err := app.HeadlessKey(k); err != nil {
			t.Fatal(err)
		}
	}
	pointer := func(action string, x, y int) {
		t.Helper()
		if err := app.HeadlessPointer(action, x, y); err != nil {
			t.Fatal(err)
		}
	}
	key("0")
	live := f.live
	hero := live.mission.ids[0]
	start, _ := live.entity(hero)
	var foe sim.EntityID
	weakest := int32(-1)
	for _, e := range live.world.Entities() {
		if e.Alive() && !e.OffMap && e.DamageBase > 0 && live.world.Relations().Hostile(start.Owner, e.Owner) &&
			(weakest < 0 || e.MaxHP < weakest) {
			foe, weakest = e.ID, e.MaxHP
		}
	}
	if weakest < 0 {
		t.Fatal("installed mission has no fighting hostile")
	}
	// Fixture placement and presentation fog only; identities, stats,
	// diplomacy, the group queue, the fight and the walk stay production-owned.
	for _, p := range []struct {
		id   sim.EntityID
		x, y int32
	}{{hero, 20, 60}, {foe, 25, 60}} {
		if err := live.world.HeadlessPlace(p.id, p.x, p.y); err != nil {
			t.Fatal(err)
		}
	}
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()

	// The commanded cell: the first of these the lone hero walks to in a copy
	// of the mission, at least 8 cells beyond the hostile.
	var dx, dy int32
	for _, c := range [][2]int32{{38, 60}, {36, 58}, {36, 62}, {34, 60}, {38, 56}, {38, 64}, {35, 60}} {
		form, err := live.world.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var scratch sim.World
		if err := scratch.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
		mapload.BindSourceDerive(&scratch)
		sim.Step(&scratch, []sim.Command{sim.MoveTo(hero, sim.CellPoint{X: c[0], Y: c[1]})})
		for n := 0; n < 400 && dx == 0; n++ {
			sim.Step(&scratch, nil)
			if n%8 == 0 && attackMoveDist(attackMoveEntity(&scratch, hero), c[0], c[1]) <= attackMoveArrival {
				dx, dy = c[0], c[1]
			}
		}
		if dx != 0 {
			break
		}
	}
	if dx == 0 {
		t.Fatal("no candidate cell east of the hostile is reachable for the installed hero")
	}

	live.view.Camera().CenterOn(22*32, 60*32)
	if err := app.HeadlessSelectEntity(uint32(hero)); err != nil {
		t.Fatal(err)
	}
	px, py, err := app.HeadlessCommandPoint(5)
	if err != nil {
		t.Fatal(err)
	}
	pointer("press", px, py)
	pointer("release", px, py)

	// Centre the camera on the chosen cell and click the window pixel nearest
	// the middle of the view that resolves to it or a neighbour.
	live.view.Camera().CenterOn(float64(dx)*32+16, float64(dy)*32+16)
	gx, gy, aimed := 0, 0, false
	for r := 0; r <= 240 && !aimed; r += 4 {
		for oy := -r; oy <= r && !aimed; oy += 4 {
			for ox := -r; ox <= r; ox += 4 {
				if max(max(ox, -ox), max(oy, -oy)) != r {
					continue
				}
				cellX, cellY, err := app.HeadlessDropCell(512+ox, 384+oy)
				if err == nil && attackMoveDist(sim.Entity{X: int32(cellX), Y: int32(cellY)}, dx, dy) <= 1 {
					gx, gy, aimed = 512+ox, 384+oy, true
					break
				}
			}
		}
	}
	if !aimed {
		t.Fatalf("no window pixel near the middle of the view resolves to cell (%d,%d)", dx, dy)
	}
	n := len(live.pending)
	pointer("press", gx, gy)
	pointer("release", gx, gy)
	if len(live.pending) != n+1 || live.pending[n].Kind != sim.KindGroupSwarmTo || live.pending[n].Entity != hero {
		t.Fatalf("panel Swarm and ground click queued %+v, want one KindGroupSwarmTo for hero %d", live.pending[n:], hero)
	}
	clicked := sim.Entity{X: int32(live.pending[n].X), Y: int32(live.pending[n].Y)}
	if attackMoveDist(clicked, dx, dy) > 2 {
		t.Fatalf("the click at window (%d,%d) commanded (%d,%d), not near the chosen cell (%d,%d)", gx, gy, clicked.X, clicked.Y, dx, dy)
	}
	dx, dy = clicked.X, clicked.Y
	t.Logf("attack move commanded to (%d,%d) by a click at window (%d,%d)", dx, dy, gx, gy)
	key("0")

	engaged, ended, reloaded, arrived := false, false, false, false
	var shortBy int32
	for step := 0; step < 4000 && !arrived; step++ {
		if err := attackMoveStep(app); err != nil {
			t.Fatal(err)
		}
		e, _ := live.entity(hero)
		if !e.Alive() {
			t.Fatalf("step %d: the hero died at (%d,%d) before reaching (%d,%d)", step, e.X, e.Y, dx, dy)
		}
		if step%100 == 0 {
			t.Logf("step %d hero (%d,%d) attack=%v target=%v", step, e.X, e.Y, e.HasAttackTarget, e.HasTarget)
		}
		if e.HasAttackTarget && !engaged {
			engaged = true
			t.Logf("step %d: hero engages at (%d,%d)", step, e.X, e.Y)
		}
		if engaged && !reloaded {
			reloaded = true
			attackMoveReload(t, f, hero, dx, dy)
		}
		if engaged && !e.HasAttackTarget && !ended {
			ended, shortBy = true, attackMoveDist(e, dx, dy)
			t.Logf("step %d: fight over at (%d,%d), %d cells from (%d,%d)", step, e.X, e.Y, shortBy, dx, dy)
		}
		arrived = ended && !e.HasTarget && attackMoveDist(e, dx, dy) <= attackMoveArrival
	}
	if !engaged {
		t.Fatal("the hero never engaged the hostile on its way")
	}
	if !ended || shortBy <= attackMoveArrival {
		t.Fatalf("witness needs the fight to end short of the commanded cell: ended=%v shortBy=%d", ended, shortBy)
	}
	e, _ := live.entity(hero)
	if !arrived {
		t.Fatalf("after the fight the hero stands at (%d,%d), %d cells from the commanded cell (%d,%d), HasTarget=%v",
			e.X, e.Y, attackMoveDist(e, dx, dy), dx, dy, e.HasTarget)
	}
	held := e
	for step := 0; step < 96; step++ {
		if err := attackMoveStep(app); err != nil {
			t.Fatal(err)
		}
		if e, _ = live.entity(hero); e.X != held.X || e.Y != held.Y || e.HasTarget {
			t.Fatalf("step %d after arrival: the hero moved from (%d,%d) to (%d,%d), HasTarget=%v", step, held.X, held.Y, e.X, e.Y, e.HasTarget)
		}
	}
	t.Logf("attack move to (%d,%d): fight ended %d cells short, the hero walked on and stayed at (%d,%d)", dx, dy, shortBy, held.X, held.Y)
}

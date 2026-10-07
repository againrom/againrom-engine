package game

import (
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// cornerSlinger opens mission 41 in the live App and stands its goblin slinger
// 0 on a cell of the playable rectangle's west edge with the party's first hero
// two cells east of it, so that the decoded flee cell clamps onto the slinger's
// own cell. Only the two placements are fixture; the withdrawal arm, the flee
// cell and every later tick are production.
func cornerSlinger(t *testing.T) (*FrontEnd, *ui.App, sim.EntityID, sim.EntityID, int32, int32) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	app := f.App("corner seed")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
	party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	if err := app.OpenMission(f.MissionOpenerWith(releaseWithdrawalMission, party)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	for k := 0; k < 16 && live.mission.open; k++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	hero := live.mission.ids[0]
	const slinger = sim.EntityID(0)
	if s, ok := live.entity(slinger); !ok || s.Withdraw <= 0 || s.Reach <= 1 || s.Owner == 0 {
		t.Fatalf("entity 0 is not a ranged withdrawer: %+v", s)
	}
	b := live.world.Bounds()
	type edge struct{ x, y, dx, dy int32 }
	var edges []edge
	for y := int32(8); y <= b.Height-9; y++ {
		edges = append(edges, edge{8, y, 2, 0}, edge{b.Width - 9, y, -2, 0})
	}
	for x := int32(8); x <= b.Width-9; x++ {
		edges = append(edges, edge{x, 8, 0, 2}, edge{x, b.Height - 9, 0, -2})
	}
	for _, c := range edges {
		if live.world.HeadlessPlace(slinger, c.x, c.y) != nil {
			continue
		}
		if s, _ := live.entity(slinger); s.X != c.x || s.Y != c.y {
			continue
		}
		if live.world.HeadlessPlace(hero, c.x+c.dx, c.y+c.dy) != nil {
			continue
		}
		if p, _ := live.entity(hero); p.X != c.x+c.dx || p.Y != c.y+c.dy {
			continue
		}
		live.push()
		return f, app, slinger, hero, c.x, c.y
	}
	t.Fatal("no edge cell fits the slinger with the hero two cells inward")
	return nil, nil, 0, 0, 0, 0
}

// TestReleaseCorneredSlingerCentredSeedRequestSAVColdLoadAndNextAction holds
// slinger 0 of mission 41 against the playable edge with the hero two cells
// inward. The Withdraw arm's flee cell clamps onto the slinger's own cell and,
// the slinger being centred, the request returns before search and raises no
// failure (MOVE-081, MOVE-083): the reacquisition of a refused flee does not run
// and the slinger holds no victim while the hero stays beside it. The state is
// saved through F2, restored from a cold LOAD, and the next production ticks
// must keep the two worlds equal.
func TestReleaseCorneredSlingerCentredSeedRequestSAVColdLoadAndNextAction(t *testing.T) {
	f, app, slinger, hero, x, y := cornerSlinger(t)
	live := f.live
	const settle = 40
	for n := 0; n < settle; n++ {
		live.tick()
		s, _ := live.entity(slinger)
		h, _ := live.entity(hero)
		if s.X != x || s.Y != y || h.X != 0 && max(releaseAbs32(h.X-x), releaseAbs32(h.Y-y)) != 2 {
			t.Fatalf("tick %d: slinger (%d,%d) hero (%d,%d), want both held on their cells", live.world.Tick(), s.X, s.Y, h.X, h.Y)
		}
		if s.HasAttackTarget || s.HasTarget {
			t.Fatalf("tick %d: slinger holds attack %t move %t after a request for its own centred cell, want neither", live.world.Tick(), s.HasAttackTarget, s.HasTarget)
		}
	}
	for k := 0; k < 8; k++ {
		if _, _, open := f.LiveNotice(); !open {
			break
		}
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	w := live.world
	dir, raw := castOrderF2Save(t, f, app, "Corner seed")
	if len(raw) == 0 {
		t.Fatal("empty SAV")
	}
	if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenMap {
		t.Fatal("SAVE return to GAME", err, app.Screen())
	}
	cold, _ := castOrderSession(t, dir)
	if cold.live.world.Hash() != w.Hash() {
		t.Fatalf("cold LOAD changed state: %x != %x", cold.live.world.Hash(), w.Hash())
	}
	var blow bool
	for n := 0; n < 48; n++ {
		live.tick()
		cold.live.tick()
		if w.Hash() != cold.live.world.Hash() {
			t.Fatalf("cold LOAD changed the next action at tick %d", w.Tick())
		}
		if s, _ := live.entity(slinger); s.X != x || s.Y != y {
			t.Fatalf("slinger left its cell at tick %d", w.Tick())
		} else if s.HasAttackTarget {
			blow = true
		}
	}
	t.Logf("slinger %d at (%d,%d): held no victim for %d ticks, later acquisition by the ordinary decision: %t", slinger, x, y, settle, blow)
}

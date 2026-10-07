package game

import (
	"sort"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// refusedFleeSpan is the number of ticks, counted from a unit's first blow, over
// which the two witnesses below tally what the unit does next.
const refusedFleeSpan = 128

// refusedFleeTally is what one witness reads off a unit that stands beside its
// hostile with a flee cell no route serves.
type refusedFleeTally struct {
	ticks       int      // counted ticks after the first blow
	blows       []uint64 // ticks of the attack cycles resolved, the first blow included
	idle        int      // counted ticks with no order, a ready cycle and no crossing
	longestIdle int      // longest run of such ticks
	ended       string   // why the count stopped
}

// tallyRefusedFlee ticks the live world and counts the unit's blows and its idle
// ticks over the ticks on which counts holds, from the first blow it strikes on
// such a tick until the span is full. It stops early when over holds, the unit
// dies or the horizon passes. A tick on which counts is false is skipped: the
// flee cell is closed only while the two units stay where the witness placed
// them, so no other tick says anything about it.
func tallyRefusedFlee(live *mapWorld, id sim.EntityID, horizon int, state func() (counts, over bool)) refusedFleeTally {
	c := refusedFleeTally{ended: "horizon"}
	var prev sim.Entity
	run := 0
	for n := 0; n < horizon; n++ {
		live.tick()
		u, ok := live.entity(id)
		if !ok || !u.Alive() {
			c.ended = "the unit died"
			break
		}
		counts, over := state()
		if over {
			c.ended = "the other unit died"
			break
		}
		struck := prev.AttackPhase == sim.AttackCharging && u.AttackPhase == sim.AttackRelaxing
		prev = u
		if !counts {
			continue
		}
		if len(c.blows) == 0 {
			if struck {
				c.blows = append(c.blows, live.world.Tick())
			}
			continue
		}
		c.ticks++
		if struck {
			c.blows = append(c.blows, live.world.Tick())
		}
		if !u.HasAttackTarget && !u.HasTarget && u.AttackPhase == sim.AttackReady && u.Transit == 0 {
			c.idle++
			run++
			c.longestIdle = max(c.longestIdle, run)
		} else {
			run = 0
		}
		if c.ticks == refusedFleeSpan {
			c.ended = "the span filled"
			break
		}
	}
	return c
}

// openRefusedFleeMission opens the shipped mission both witnesses use with the
// party the front end builds and leaves the opening notices behind.
func openRefusedFleeMission(t *testing.T, label string) (*ui.App, *mapWorld, sim.EntityID) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	app := f.App(label)
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(releaseRetreatRefusedMission, party)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	for k := 0; k < 16 && live.mission.open; k++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	return app, live, live.mission.ids[0]
}

// TestReleaseRetreatWithNoRouteKeepsItsPickAcrossCycles stands the party's first
// hero on the wall side of the shipped hostile at (13,49) of mission 41 and
// orders Retreat through the App's own input, the R key. A wall closes every
// route from the flee cell (9,45), so the original's order machine clears the
// refused move and reacquires a victim within reach (AI-RETREAT-273,
// AI-ROUTE-045, AI-327). The pick is a pursuit order, which nothing ends until
// the state arm's next dispatch rewrites it (AI-PURSUE-040, AI-BREAK-041): the
// hero starts its next cycle as soon as one resolves, and no tick after its
// first blow finds it without an order. Before, each resolved cycle ended the
// order and the hero waited for the next decision period.
func TestReleaseRetreatWithNoRouteKeepsItsPickAcrossCycles(t *testing.T) {
	app, live, heroID := openRefusedFleeMission(t, "retreat pick")
	hero, ok := live.entity(heroID)
	if !ok {
		t.Fatal("no party hero")
	}
	const foeX, foeY, heroX, heroY = 13, 49, 12, 48
	rel := live.world.Relations()
	var foe sim.EntityID
	found := false
	for _, e := range live.world.Entities() {
		if e.Alive() && !e.OffMap && e.X == foeX && e.Y == foeY && e.Owner != 0 && rel.Hostile(e.Owner, hero.Owner) {
			foe, found = e.ID, true
		}
	}
	if !found {
		t.Fatalf("mission %d holds no hostile at (%d,%d)", releaseRetreatRefusedMission, foeX, foeY)
	}
	// The script's placement puts the hero on a random fitting cell within one
	// of the anchor and draws the world's random stream each time, so it is
	// repeated until the hero stands on the cell the flee cell is closed from.
	for k := 0; k < 200; k++ {
		if err := live.world.HeadlessPlace(heroID, heroX, heroY); err != nil {
			t.Fatal(err)
		}
		if h, _ := live.entity(heroID); h.X == heroX && h.Y == heroY {
			break
		}
	}
	if h, _ := live.entity(heroID); h.X != heroX || h.Y != heroY {
		t.Fatalf("the hero stands on (%d,%d), want (%d,%d) beside the hostile", h.X, h.Y, heroX, heroY)
	}
	live.push()
	inspectionCentre(live, heroX, heroY)
	if err := app.HeadlessSelectEntity(uint32(heroID)); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("r"); err != nil {
		t.Fatal(err)
	}
	if len(live.pending) != 1 || live.pending[0].Kind != sim.KindGroupRetreat || live.pending[0].Entity != heroID ||
		live.pending[0].Player != sim.SelfSlot {
		t.Fatalf("the R key queued %+v, want Retreat for hero %d", live.pending, heroID)
	}

	const (
		horizon    = 900
		minBlows   = 8
		retreating = 0x16
	)
	var leftRetreat uint64
	c := tallyRefusedFlee(live, heroID, horizon, func() (counts, over bool) {
		h, _ := live.entity(heroID)
		s, _ := live.entity(foe)
		if !s.Alive() {
			return false, true
		}
		if h.ActorState != retreating && leftRetreat == 0 {
			leftRetreat = live.world.Tick()
		}
		return h.X == heroX && h.Y == heroY && s.X == foeX && s.Y == foeY, false
	})
	t.Logf("mission %d hero %d: blows at ticks %v in %d counted ticks; %d tick(s) without an order (longest run %d); count ended: %s",
		releaseRetreatRefusedMission, heroID, c.blows, c.ticks, c.idle, c.longestIdle, c.ended)
	if leftRetreat != 0 {
		t.Fatalf("the hero left Retreat at tick %d", leftRetreat)
	}
	if c.ticks != refusedFleeSpan {
		t.Fatalf("the two units held their cells for %d counted ticks, want %d: the drive does not reach the state under test", c.ticks, refusedFleeSpan)
	}
	if c.idle != 0 {
		t.Errorf("the hero held no order on %d tick(s) after its first blow (longest run %d), want none: its pick persists across cycles", c.idle, c.longestIdle)
	}
	if got := len(c.blows) - 1; got < minBlows-1 {
		t.Errorf("the hero struck %d time(s) after its first blow in %d ticks, want at least %d: each cycle follows the last at once", got, c.ticks, minBlows-1)
	}
}

// TestReleaseRangedWithdrawalWithNoRouteKeepsItsPickAcrossCycles orders the
// party's first hero onto the nearest shipped ranged withdrawer of mission 41
// through the App's own input: the hero is selected, the attack key pressed
// and the creature clicked. The creature backs away until a flee cell nothing
// leads to leaves it beside the hero, and the original's reacquisition writes a
// pursuit order that persists across its cycles (AI-ROUTE-045, AI-327,
// AI-PURSUE-040). Counted from its first shot beside the hero, the creature is
// never without an order and its shots are one cycle apart.
func TestReleaseRangedWithdrawalWithNoRouteKeepsItsPickAcrossCycles(t *testing.T) {
	app, live, heroID := openRefusedFleeMission(t, "ranged pick")
	hero, ok := live.entity(heroID)
	if !ok {
		t.Fatal("no party hero")
	}
	type candidate struct {
		id   sim.EntityID
		dist int32
	}
	var candidates []candidate
	rel := live.world.Relations()
	for _, e := range live.world.Entities() {
		if !e.Alive() || e.OffMap || e.Owner == 0 || e.Withdraw <= 0 || e.Reach <= 1 || !rel.Hostile(e.Owner, hero.Owner) {
			continue
		}
		candidates = append(candidates, candidate{e.ID, max(releaseAbs32(e.X-hero.X), releaseAbs32(e.Y-hero.Y))})
	}
	if len(candidates) == 0 {
		t.Fatalf("mission %d holds no hostile ranged withdrawer", releaseRetreatRefusedMission)
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].dist < candidates[j].dist })
	foe := candidates[0].id

	inspectionCentre(live, int(hero.X), int(hero.Y))
	target, _ := live.entity(foe)
	inspectionCentre(live, int(target.X), int(target.Y))
	if err := app.HeadlessSelectEntity(uint32(heroID)); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("attack"); err != nil {
		t.Fatal(err)
	}
	x, y, err := app.HeadlessEntityPoint(uint32(foe))
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if len(live.pending) != 1 || live.pending[0].Kind != sim.KindAttack || live.pending[0].Entity != heroID ||
		live.pending[0].X != int32(foe) {
		t.Fatalf("the click queued %+v, want the hero's attack on creature %d", live.pending, foe)
	}

	const (
		horizon  = 2200
		minBlows = 2
		minTicks = 64
	)
	c := tallyRefusedFlee(live, foe, horizon, func() (counts, over bool) {
		s, _ := live.entity(foe)
		h, _ := live.entity(heroID)
		if !h.Alive() {
			return false, true
		}
		return max(releaseAbs32(s.X-h.X), releaseAbs32(s.Y-h.Y)) <= 2, false
	})
	t.Logf("mission %d creature %d: blows beside the hero at ticks %v in %d counted ticks; %d tick(s) without an order (longest run %d); count ended: %s",
		releaseRetreatRefusedMission, foe, c.blows, c.ticks, c.idle, c.longestIdle, c.ended)
	if len(c.blows) < minBlows || c.ticks < minTicks {
		t.Fatalf("the creature struck %d time(s) beside the hero over %d counted ticks, want %d over %d: the drive does not reach the state under test", len(c.blows), c.ticks, minBlows, minTicks)
	}
	if c.idle != 0 {
		t.Errorf("the creature held no order on %d tick(s) beside the hero after its first blow (longest run %d), want none", c.idle, c.longestIdle)
	}
}

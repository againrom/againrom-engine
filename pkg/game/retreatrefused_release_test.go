package game

import (
	"fmt"
	"testing"

	"againrom/pkg/sim"
)

// releaseRetreatRefusedMission is the campaign mission whose shipped west
// wing places a hostile beside a wall thick enough to close every cell within
// two of a unit's flee cell. Its layout is the same on both installs.
const releaseRetreatRefusedMission = 41

// TestReleaseRetreatWhoseFleeCellHasNoRouteKeepsFighting stands the party's first
// hero on the wall side of the shipped hostile at (13,49) of mission 41 and
// orders Retreat through the App's own input: the hero is selected and the R key
// pressed. The hostile is south-east of the hero, so the decoded flee cell is
// (9,45) inside the west wall and no route serves it. The original clears the
// refused move and reacquires a victim within reach (AI-RETREAT-273,
// AI-ROUTE-045, AI-335, MOVE-072), and Retreat stays the hero's state, so the
// hero must never hold no order for longer than one AI period while the hostile
// is beside it, and it must resolve at least three attack cycles. Before, it
// stood until the hostile left.
//
// Only the hero's first cell is fixture placement: the mission script's own
// placement routine puts it within one cell of the hostile, repeated until it
// stands on the wall side. Retreat, the flee cell, the refusal and every later
// tick are production.
func TestReleaseRetreatWhoseFleeCellHasNoRouteKeepsFighting(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	app := f.App("retreat refused")
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
	heroID := live.mission.ids[0]
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
		horizon    = 600
		aiPeriod   = 16
		maxIdle    = aiPeriod + 4
		minTicks   = 4 * aiPeriod
		minBlows   = 3
		shown      = 8
		retreating = 0x16
	)
	var (
		lines         []string
		held, run     int
		longestIdle   int
		idleStart     uint64
		cycles, spans int
		prev          sim.Entity
		diedAt        uint64
		movedAt       uint64
		notRetreating uint64
	)
	endSpan := func() {
		if run > aiPeriod/2 {
			spans++
			if spans <= shown {
				lines = append(lines, fmt.Sprintf("without an order from t=%d for %d ticks", idleStart, run))
			}
		}
		run = 0
	}
	// The window is the run of ticks on which both units stay on their cells, the
	// only ticks on which the flee cell is the closed one. It ends when either
	// unit moves, when the hostile dies and when the hero does.
	for n := 0; n < horizon; n++ {
		live.tick()
		h, _ := live.entity(heroID)
		s, _ := live.entity(foe)
		tick := live.world.Tick()
		if !s.Alive() {
			diedAt = tick
			break
		}
		if !h.Alive() {
			break
		}
		if h.X != heroX || h.Y != heroY || s.X != foeX || s.Y != foeY {
			movedAt = tick
			break
		}
		held++
		if h.ActorState != retreating && notRetreating == 0 {
			notRetreating = tick
		}
		if prev.AttackPhase == sim.AttackCharging && h.AttackPhase == sim.AttackRelaxing {
			cycles++
			if cycles <= shown {
				lines = append(lines, fmt.Sprintf("t=%d cycle resolved, hostile hp %d", tick, s.HP))
			}
		}
		prev = h
		if !h.HasAttackTarget && !h.HasTarget && h.AttackPhase == sim.AttackReady && h.Transit == 0 {
			if run == 0 {
				idleStart = tick
			}
			run++
			longestIdle = max(longestIdle, run)
		} else {
			endSpan()
		}
	}
	endSpan()
	for _, line := range lines {
		t.Logf("mission %d hero %d: %s", releaseRetreatRefusedMission, heroID, line)
	}
	t.Logf("mission %d hero %d: %d cycle(s) and %d run(s) without an order longer than %d ticks in all; the first %d of each are listed",
		releaseRetreatRefusedMission, heroID, cycles, spans, aiPeriod/2, shown)
	t.Logf("mission %d hero %d: both units held their cells for %d ticks, hostile died at tick %d, a unit moved at tick %d, %d cycle(s), longest run without an order %d ticks",
		releaseRetreatRefusedMission, heroID, held, diedAt, movedAt, cycles, longestIdle)
	if notRetreating != 0 {
		t.Fatalf("the hero left Retreat at tick %d", notRetreating)
	}
	if diedAt == 0 && held < minTicks {
		t.Fatalf("the two units held their cells for %d ticks, want %d: the drive does not reach the state under test", held, minTicks)
	}
	if longestIdle > maxIdle {
		t.Fatalf("the hero held no order for %d consecutive ticks beside the hostile, want at most %d", longestIdle, maxIdle)
	}
	if cycles < minBlows {
		t.Fatalf("the hero resolved %d attack cycle(s) beside the hostile, want at least %d", cycles, minBlows)
	}
}

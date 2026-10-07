package game

import (
	"fmt"
	"sort"
	"testing"

	"againrom/pkg/sim"
)

// TestReleaseRangedWithdrawalWhoseFleeCellHasNoRouteKeepsFighting orders the
// party's first hero onto the nearest shipped ranged withdrawer of mission 41
// through the App's own input: the hero is selected, the attack key pressed
// and the creature clicked. The creature backs away until a flee cell nothing
// leads to leaves it beside the hero. The original clears such a move and
// reacquires a victim within reach (AI-ROUTE-045, AI-335, MOVE-072), so the
// creature must never stand without an order for longer than one AI period
// while the hero is beside it, and it must land at least two blows. Before, it
// stood until it died.
func TestReleaseRangedWithdrawalWhoseFleeCellHasNoRouteKeepsFighting(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	app := f.App("ranged flee")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(releaseWithdrawalMission, party)); err != nil {
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
		d := releaseAbs32(e.X - hero.X)
		if dy := releaseAbs32(e.Y - hero.Y); dy > d {
			d = dy
		}
		candidates = append(candidates, candidate{e.ID, d})
	}
	if len(candidates) == 0 {
		t.Fatalf("mission %d holds no hostile ranged withdrawer", releaseWithdrawalMission)
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
		horizon     = 2200
		aiPeriod    = 16
		maxIdle     = aiPeriod + 4
		minTogether = 4 * aiPeriod
		minBlows    = 2
	)
	const shown = 8
	var (
		lines             []string
		together, idleRun int
		longestTogether   int
		longestIdle       int
		idleStart         uint64
		blows, spans      int
		prev              sim.Entity
		diedAt            uint64
	)
	// endSpan closes a run without an order; only a run longer than half an AI
	// period is worth a line.
	endSpan := func() {
		if idleRun > aiPeriod/2 {
			spans++
			if spans <= shown {
				lines = append(lines, fmt.Sprintf("without an order from t=%d for %d ticks", idleStart, idleRun))
			}
		}
		idleRun = 0
	}
	for n := 0; n < horizon; n++ {
		live.tick()
		s, _ := live.entity(foe)
		h, _ := live.entity(heroID)
		tick := live.world.Tick()
		if !s.Alive() {
			diedAt = tick
			break
		}
		if prev.AttackPhase == sim.AttackCharging && s.AttackPhase == sim.AttackRelaxing {
			blows++
			if blows <= shown {
				lines = append(lines, fmt.Sprintf("t=%d blow at (%d,%d), hero (%d,%d) hp %d", tick, s.X, s.Y, h.X, h.Y, h.HP))
			}
		}
		prev = s
		near := h.Alive() && max(releaseAbs32(s.X-h.X), releaseAbs32(s.Y-h.Y)) <= 2
		if near {
			together++
			longestTogether = max(longestTogether, together)
		} else {
			together = 0
		}
		if near && !s.HasAttackTarget && !s.HasTarget && s.AttackPhase == sim.AttackReady && s.Transit == 0 {
			if idleRun == 0 {
				idleStart = tick
			}
			idleRun++
			longestIdle = max(longestIdle, idleRun)
		} else {
			endSpan()
		}
		if !h.Alive() {
			break
		}
	}
	endSpan()
	for _, line := range lines {
		t.Logf("mission %d creature %d: %s", releaseWithdrawalMission, foe, line)
	}
	t.Logf("mission %d creature %d: %d blow(s) and %d run(s) without an order longer than %d ticks in all; the first %d of each are listed",
		releaseWithdrawalMission, foe, blows, spans, aiPeriod/2, shown)
	t.Logf("mission %d creature %d: died at tick %d, %d blow(s), longest run beside the hero %d ticks, longest run without an order %d ticks",
		releaseWithdrawalMission, foe, diedAt, blows, longestTogether, longestIdle)
	if longestTogether < minTogether {
		t.Fatalf("the creature was never held beside the hero for %d ticks (longest %d): the drive does not reach the state under test", minTogether, longestTogether)
	}
	if longestIdle > maxIdle {
		t.Fatalf("the creature stood without an order for %d consecutive ticks beside the hero, want at most %d", longestIdle, maxIdle)
	}
	if blows < minBlows {
		t.Fatalf("the creature landed %d blow(s) before it died, want at least %d", blows, minBlows)
	}
}

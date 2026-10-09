package game

import (
	"sort"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func pursuitAppClick(t *testing.T, app *ui.App, x, y int) {
	t.Helper()
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReleaseAcquiredVictimLeavingReachTurnsWithoutWalking(t *testing.T) {
	total, controls := 0, 0
	for candidate := 0; candidate < 3 && total == 0; candidate++ {
		// Defend's acquisition takes the adjacent victim only on the actor
		// pass, every 16 ticks, so whether a Defend given at contact acquires
		// depends on the tick it lands on. Each start delay shifts the
		// contact against that pass.
		for delay := 0; delay < 16 && total == 0; delay++ {
			app, live, heroID := openRefusedFleeMission(t, "acquisition reach")
			hero, _ := live.entity(heroID)
			var foes []sim.Entity
			rel := live.world.Relations()
			for _, e := range live.world.Entities() {
				if e.Alive() && !e.OffMap && e.Withdraw > 0 && e.Reach > 1 && rel.Hostile(e.Owner, hero.Owner) {
					foes = append(foes, e)
				}
			}
			sort.SliceStable(foes, func(i, j int) bool {
				return max(releaseAbs32(foes[i].X-hero.X), releaseAbs32(foes[i].Y-hero.Y)) < max(releaseAbs32(foes[j].X-hero.X), releaseAbs32(foes[j].Y-hero.Y))
			})
			if candidate >= len(foes) {
				break
			}
			for range delay {
				live.tick()
			}
			checked, control := acquiredVictimRound(t, app, live, heroID, foes[candidate].ID, candidate, delay)
			if control {
				controls++
			}
			total += checked
		}
	}
	if total == 0 {
		t.Fatal("bounded AGAINROM_ASSETS mission 41 drive: three ranged withdrawers, sixteen start delays each, 2200 contact ticks and 240 Defend ticks; no acquisition beyond reach reached")
	}
	if controls == 0 {
		t.Fatal("player attack never exposed a path control")
	}
}

// acquiredVictimRound orders the hero onto the foe, gives Defend at contact
// and checks every ready acquisition tick beyond reach over 240 ticks. It
// returns the ticks checked and whether the attack exposed a path control.
func acquiredVictimRound(t *testing.T, app *ui.App, live *mapWorld, heroID, foeID sim.EntityID, candidate, delay int) (int, bool) {
	t.Helper()
	foe, ok := live.entity(foeID)
	if h, alive := live.entity(heroID); !ok || !foe.Alive() || foe.OffMap || !alive || !h.Alive() {
		return 0, false
	}
	if err := app.HeadlessSelectEntity(uint32(heroID)); err != nil {
		t.Fatal(err)
	}
	inspectionCentre(live, int(foe.X), int(foe.Y))
	if err := app.HeadlessKey("attack"); err != nil {
		t.Fatal(err)
	}
	x, y, err := app.HeadlessEntityPoint(uint32(foeID))
	if err != nil {
		t.Fatal(err)
	}
	pursuitAppClick(t, app, x, y)
	if len(live.pending) != 1 || live.pending[0].Kind != sim.KindAttack {
		t.Fatal("App did not queue player attack", live.pending)
	}
	contact, control := false, false
	for tick := 0; tick < 2200; tick++ {
		live.tick()
		foe, ok := live.entity(foeID)
		h, _ := live.entity(heroID)
		if !ok || !foe.Alive() || !h.Alive() {
			break
		}
		if h.HasAttackTarget && !h.AcquirePursuit && h.HasTarget {
			control = true
		}
		if max(releaseAbs32(foe.X-h.X), releaseAbs32(foe.Y-h.Y)) <= int32(h.Reach) && h.Transit == 0 {
			contact = true
			t.Logf("candidate %d foe %d delay %d contact tick %d: foe (%d,%d) hero (%d,%d) phase %d", candidate, foeID, delay, live.world.Tick(), foe.X, foe.Y, h.X, h.Y, h.AttackPhase)
			break
		}
	}
	if !contact {
		t.Logf("candidate %d foe %d delay %d: no surviving contact in 2200 ticks", candidate, foeID, delay)
		return 0, false
	}
	h, _ := live.entity(heroID)
	inspectionCentre(live, int(h.X), int(h.Y))
	if err := app.HeadlessSelectEntity(uint32(heroID)); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("d"); err != nil {
		t.Fatal(err)
	}
	x, y, err = app.HeadlessEntityPoint(uint32(heroID))
	if err != nil {
		t.Fatal(err)
	}
	pursuitAppClick(t, app, x, y)
	// A click on the hero's point can land on a body standing over it; that
	// delay takes no Defend.
	if len(live.pending) != 1 || live.pending[0].Kind != sim.KindGroupDefend {
		t.Logf("candidate %d foe %d delay %d: the click queued %v, not Defend", candidate, foeID, delay, live.pending)
		return 0, control
	}
	// The Defend arm's scorer refuses a victim past reach, so the victim it
	// acquired is held beyond reach only until the arm's next pass releases
	// it (AI-REACH-072); a victim that leaves during a loaded cycle is often
	// released before the cycle ends. Either way the hero does not walk: a
	// loaded cycle keeps its cell, and a ready hero turns where it stands.
	markers, loaded, checked := 0, 0, 0
	for tick := 0; tick < 240; tick++ {
		live.tick()
		h, ok := live.entity(heroID)
		if !ok || !h.Alive() {
			break
		}
		foe, ok := live.entity(h.AttackTarget)
		if !h.AcquirePursuit || !h.HasAttackTarget || !ok {
			continue
		}
		markers++
		distance := max(releaseAbs32(foe.X-h.X), releaseAbs32(foe.Y-h.Y))
		if h.AttackPhase == sim.AttackReady && distance > int32(h.Reach) && h.HasTarget {
			t.Fatalf("tick %d: acquisition stored a walk beyond reach", live.world.Tick())
		}
		if h.Turning() || h.Transit != 0 || distance <= int32(h.Reach) {
			continue
		}
		before := h
		live.tick()
		h, _ = live.entity(heroID)
		if h.X != before.X || h.Y != before.Y || h.HasTarget && (h.TargetX != h.X || h.TargetY != h.Y) {
			t.Fatalf("tick %d: acquired victim beyond reach, the hero walked (%d,%d)->(%d,%d), target %t (%d,%d)", live.world.Tick(), before.X, before.Y, h.X, h.Y, h.HasTarget, h.TargetX, h.TargetY)
		}
		if before.AttackPhase != sim.AttackReady {
			loaded++
			continue
		}
		foe, _ = live.entity(before.AttackTarget)
		if !h.AcquirePursuit || h.AttackTarget != before.AttackTarget {
			continue
		}
		checked++
		if h.HasTarget {
			t.Fatalf("tick %d: acquired pursuit stored a target (%d,%d)", live.world.Tick(), h.TargetX, h.TargetY)
		}
		want := pursuitReleaseFacing(foe.X-before.X, foe.Y-before.Y)
		if h.Facing != want && (!h.Turning() || h.DesiredFacing != want) {
			t.Fatalf("tick %d: facing %d desired %d, want %d", live.world.Tick(), h.Facing, h.DesiredFacing, want)
		}
	}
	t.Logf("candidate %d foe %d delay %d: %d acquisition ticks, %d loaded and %d ready beyond-reach ticks, player path control %t", candidate, foeID, delay, markers, loaded, checked, control)
	return loaded + checked, control
}

func pursuitReleaseFacing(dx, dy int32) uint8 {
	if dy < 0 {
		if dx < 0 {
			return 224
		}
		if dx > 0 {
			return 32
		}
		return 0
	}
	if dy > 0 {
		if dx < 0 {
			return 160
		}
		if dx > 0 {
			return 96
		}
		return 128
	}
	if dx < 0 {
		return 192
	}
	return 64
}

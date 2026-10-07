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
	for candidate := 0; candidate < 3; candidate++ {
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
		foeID := foes[candidate].ID
		if err := app.HeadlessSelectEntity(uint32(heroID)); err != nil {
			t.Fatal(err)
		}
		inspectionCentre(live, int(foes[candidate].X), int(foes[candidate].Y))
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
				t.Logf("candidate %d foe %d contact tick %d: foe (%d,%d) hero (%d,%d) phase %d", candidate, foeID, live.world.Tick(), foe.X, foe.Y, h.X, h.Y, h.AttackPhase)
				break
			}
		}
		if !contact {
			t.Logf("candidate %d foe %d: no surviving contact in 2200 ticks", candidate, foeID)
			continue
		}
		if control {
			controls++
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
		if len(live.pending) != 1 || live.pending[0].Kind != sim.KindGroupDefend {
			t.Fatal("App did not queue Defend subject", live.pending)
		}
		markers, checked := 0, 0
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
			if h.AttackPhase != sim.AttackReady || h.Turning() || h.Transit != 0 || distance <= int32(h.Reach) {
				continue
			}
			before := h
			live.tick()
			h, _ = live.entity(heroID)
			foe, _ = live.entity(before.AttackTarget)
			if !h.AcquirePursuit || h.AttackTarget != before.AttackTarget {
				continue
			}
			checked++
			if h.X != before.X || h.Y != before.Y || h.HasTarget {
				t.Fatalf("tick %d: acquired pursuit walked (%d,%d)->(%d,%d), target %t", live.world.Tick(), before.X, before.Y, h.X, h.Y, h.HasTarget)
			}
			want := pursuitReleaseFacing(foe.X-before.X, foe.Y-before.Y)
			if h.Facing != want && (!h.Turning() || h.DesiredFacing != want) {
				t.Fatalf("tick %d: facing %d desired %d, want %d", live.world.Tick(), h.Facing, h.DesiredFacing, want)
			}
		}
		t.Logf("candidate %d foe %d: %d acquisition ticks, %d ready beyond-reach ticks, player path control %t", candidate, foeID, markers, checked, control)
		total += checked
		if total > 0 {
			break
		}
	}
	if total == 0 {
		t.Fatal("bounded AGAINROM_ASSETS mission 41 drive: three ranged withdrawers, 2200 contact ticks and 240 Defend ticks each; no ready acquisition beyond reach reached")
	}
	if controls == 0 {
		t.Fatal("player attack never exposed a path control")
	}
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

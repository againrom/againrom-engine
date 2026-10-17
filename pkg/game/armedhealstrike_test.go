package game

import (
	"testing"

	"againrom/pkg/sim"
)

func armedHealBegin(w *sim.World, n int) uint64 {
	for range n {
		sim.Step(w, nil)
		if len(w.Actions().Books) != 0 {
			return w.Tick()
		}
	}
	return 0
}

func TestArmedHealAfterAMidStrikeSAVEBeginsOnTheSameTick(t *testing.T) {
	f := midStrikeCastFront(t)
	w := f.live.world
	f.live.pending = []sim.Command{sim.Autocast(1, 6)}
	f.live.tick()
	f.live.tick()
	e := pendingVictimEntity(t, w, 1)
	if e.AutoSpell != 6 || e.AttackPhase != sim.AttackCharging || e.AttackCountdown < 2 || len(w.Actions().Books) != 0 {
		t.Fatalf("fixture: the mage does not charge a strike with Heal armed and no cast: %+v", e)
	}
	raw, doc, _ := saveCurrentEffect(t, f)
	cold := openCurrentEffectSave(t, f, raw)
	assertCurrentWorldEqual(t, w, cold.live.world, "armed heal mid-strike SAVE cut")
	unarmed := midStrikeSAVControl(t, f, doc, func(row *sim.ActorContinuation) { row.AutoSpell = 0 })
	early := midStrikeSAVControl(t, f, doc, func(row *sim.ActorContinuation) { row.AttackCountdown = 1 })

	begin, exit := uint64(0), uint64(0)
	for range 200 {
		ready := pendingVictimEntity(t, w, 1).AttackPhase == sim.AttackReady
		sim.Step(w, nil)
		sim.Step(cold.live.world, nil)
		assertCurrentWorldEqual(t, w, cold.live.world, "armed heal mid-strike cold continuation")
		if len(w.Actions().Books) != 0 {
			begin = w.Tick()
			if !ready || exit == 0 {
				t.Fatalf("tick %d: the heal began before the strike's exit", begin)
			}
			break
		}
		if exit == 0 && pendingVictimEntity(t, w, 1).AttackPhase == sim.AttackReady {
			exit = w.Tick()
		}
	}
	if begin == 0 || begin != exit+1 {
		t.Fatalf("the heal began on tick %d, want the first tick after the strike's exit on %d", begin, exit)
	}
	if pendingVictimEntity(t, w, 3).HP >= 200 {
		t.Fatal("the strike's blow did not land before the heal")
	}
	if got := armedHealBegin(unarmed.live.world, int(begin-unarmed.live.world.Tick())); got != 0 {
		t.Fatalf("loss control: a SAV without the armed spell began a cast on tick %d", got)
	}
	if got := armedHealBegin(early.live.world, int(begin-early.live.world.Tick())); got == 0 || got >= begin {
		t.Fatalf("loss control: a SAV with the strike's countdown cut began the heal on tick %d, want before %d", got, begin)
	}
}

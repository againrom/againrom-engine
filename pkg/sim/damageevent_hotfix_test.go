package sim

import (
	"reflect"
	"testing"
)

func TestDamageReportNamesCausalBodyDamageButNotCorpseDecay(t *testing.T) {
	body := Entity{ID: 7, X: 2, Y: 2, HP: -1, MaxHP: 20, Decay: DecayFallen}
	w := mustWorld(t, 1, Bounds{Width: 8, Height: 8}, []Entity{body})

	report := StepReported(w, []Command{{Kind: KindDamage, Entity: 7, X: 3}})
	want := []DamageEvent{{Target: 7, BeforeHP: -1, AfterHP: -4}}
	if !reflect.DeepEqual(report.Damages, want) {
		t.Fatalf("body damage report = %+v, want %+v", report.Damages, want)
	}

	// Put the next step exactly on the passive ladder's decrement phase. The
	// health level changes, but no cause emitted damage, so the report is empty.
	w.tick = decayPhase
	before := occEntity(t, w, 7).HP
	report = StepReported(w, nil)
	after := occEntity(t, w, 7).HP
	if after != before-1 {
		t.Fatalf("decay fixture moved HP %d -> %d, want exactly one passive decrement", before, after)
	}
	if len(report.Damages) != 0 {
		t.Fatalf("passive corpse decay reported as damage: %+v", report.Damages)
	}
}

func TestFinishableFallenUnitStillRefusesMovement(t *testing.T) {
	w := mustWorld(t, 1, Bounds{Width: 8, Height: 8}, []Entity{
		{ID: 7, X: 2, Y: 2, HP: -5, MaxHP: 20},
	})

	Step(w, []Command{{Kind: KindMoveTo, Entity: 7, X: 6, Y: 6}})
	got := occEntity(t, w, 7)
	if got.X != 2 || got.Y != 2 || got.HasTarget {
		t.Fatalf("finishable fallen unit moved/kept an order: cell=(%d,%d) target=%v (%d,%d)",
			got.X, got.Y, got.HasTarget, got.TargetX, got.TargetY)
	}
}

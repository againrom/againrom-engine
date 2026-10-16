package sim

import (
	"reflect"
	"strings"
	"testing"
)

// A restored order at the attack boundary owes no count: the constructor and
// the decoder refuse one that does, and the one that does not spends the two
// boundary turns and starts its next cycle (HERO-CADENCE-112).
func TestOriginalBoundaryOwesNoCountAndRearms(t *testing.T) {
	w := targetOrderWorld(t)
	before := w.entities[0]
	owed := OriginalActorAction{Entity: 1, HasTarget: true, Target: 2, Phase: AttackBoundaryOne, Countdown: 1, ActorState: actorStateEngage, PostX: 2, PostY: 2}
	if err := w.ImportOriginalActorActions([]OriginalActorAction{owed}); err == nil || !strings.Contains(err.Error(), "boundary phase 4 owes 1") {
		t.Fatalf("a boundary owing a count was admitted: %v", err)
	}
	if !reflect.DeepEqual(w.entities[0], before) {
		t.Fatal("a refused import changed the actor")
	}
	owed.Countdown = 0
	if err := w.ImportOriginalActorActions([]OriginalActorAction{owed}); err != nil {
		t.Fatal(err)
	}
	if e := w.entities[0]; e.AttackPhase != AttackBoundaryOne || e.AttackCountdown != 0 || !e.HasAttackTarget || e.AttackTarget != 2 {
		t.Fatalf("restored boundary actor %+v", e)
	}
	var phases []AttackPhase
	for range 3 {
		Step(w, nil)
		phases = append(phases, w.entities[0].AttackPhase)
	}
	// The third turn loads the next charge; this fixture's charge is short
	// enough to strike on the same turn and enter recovery.
	if phases[0] != AttackBoundaryTwo || phases[1] != AttackReady || phases[2] != AttackCharging && phases[2] != AttackRelaxing {
		t.Fatalf("boundary turns %v, want boundary two, ready, then a new cycle", phases)
	}
}

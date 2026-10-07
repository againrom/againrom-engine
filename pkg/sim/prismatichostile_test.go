package sim

import (
	"reflect"
	"testing"
)

// DIV-072 (owner): «маг судя по всему радужной
// молнией еще задевает себя и союзников.
// только врагов должен (нейтралов тоже не
// должен трогать)». applyPrismatic (celleffect.go) now filters
// a caster-directed cast to actors hostileTo the caster, the same question
// autoCastTarget already asks of every other offensive row.
//
// FOUR ACTORS STAND IN RADIUS: the caster, an actor the caster's own owner
// is allied to (relationLocked), a neutral actor the relation matrix names
// nothing about, and an actor the caster's own owner is hostile to. Only the
// hostile actor's health is expected to move.
//
// TO CONFIRM IT WITNESSES THE FIX, delete the `ci >= 0 && (...)` filter line
// at the head of applyPrismatic's loop (pkg/sim/celleffect.go) and rerun.
func TestPrismaticSprayTouchesOnlyTheHostile(t *testing.T) {
	rule := SpellRule{ID: 14, ManaCost: 4, School: 1, MaxRange: 8, TargetsUnit: true,
		Damaging: true, DamageMin: 20, DamageMax: 20}

	caster := effectMage(1, 2, 2, 1<<14) // Owner 1 (SelfSlot), effectMage's own default
	ally := spEnt(2, 3, 2)
	ally.TokenSize, ally.Owner = 1, 2
	neutral := spEnt(3, 2, 3)
	neutral.TokenSize, neutral.Owner = 1, 3
	hostile := spEnt(4, 3, 3)
	hostile.TokenSize, hostile.Owner = 1, 4

	rel := engRel(t,
		[3]uint32{1, 2, relationLocked}, // ally: locked, never hostile
		// neutral (owner 3) gets no row at all — Byte answers 0, Hostile false.
		[3]uint32{1, 4, relationHostile}, // hostile: bit 0 set
	)
	w := hlWorld(t, 0x1009, rel, []SpellRule{rule}, caster, ally, neutral, hostile)

	// Aimed at the hostile actor, so the radius is centred on its cell; power
	// 1 gives radius min(1/20+2,7) = 2, which reaches all three others from
	// (3,3).
	spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 4, Y: 14})

	if got := spAt(t, w, 4).HP; got == 100 {
		t.Fatal("the hostile actor took no damage — the assertions below would witness nothing")
	}
	if got := spAt(t, w, 1).HP; got != 100 {
		t.Errorf("prismatic spray hit its own caster: health %d, want 100", got)
	}
	if got := spAt(t, w, 2).HP; got != 100 {
		t.Errorf("prismatic spray hit the caster's ally: health %d, want 100", got)
	}
	if got := spAt(t, w, 3).HP; got != 100 {
		t.Errorf("prismatic spray hit a neutral actor: health %d, want 100", got)
	}
}

func TestPrismaticSprayReportsEveryActorItReached(t *testing.T) {
	rule := SpellRule{ID: 14, ManaCost: 4, School: 1, MaxRange: 8, TargetsUnit: true,
		Damaging: true, DamageMin: 20, DamageMax: 20}

	caster := effectMage(1, 2, 2, 1<<14)
	ally := spEnt(2, 3, 2)
	ally.TokenSize, ally.Owner = 1, 2
	first := spEnt(4, 3, 3)
	first.TokenSize, first.Owner = 1, 4
	second := spEnt(5, 4, 4)
	second.TokenSize, second.Owner = 1, 4

	rel := engRel(t,
		[3]uint32{1, 2, relationLocked},
		[3]uint32{1, 4, relationHostile},
	)
	w := hlWorld(t, 0x1009, rel, []SpellRule{rule}, caster, ally, first, second)

	events := spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 4, Y: 14})
	if len(events) != 1 {
		t.Fatalf("the cast was observed %d times, want once", len(events))
	}
	want := []CellPoint{{X: 3, Y: 3}, {X: 4, Y: 4}}
	if !reflect.DeepEqual(events[0].Victims, want) {
		t.Errorf("the observation reports victims %v, want the two hostile cells %v", events[0].Victims, want)
	}
	if got := spAt(t, w, 5).HP; got == 100 {
		t.Error("the second hostile actor took no damage — the radius did not reach it and the list says nothing")
	}
}

func TestPrismaticSprayDoesNotReachCorpses(t *testing.T) {
	rule := SpellRule{ID: 14, ManaCost: 4, School: 1, MaxRange: 8, TargetsUnit: true,
		Damaging: true, DamageMin: 20, DamageMax: 20}

	caster := effectMage(1, 2, 2, 1<<14)
	ally := spEnt(2, 3, 2)
	ally.TokenSize, ally.Owner = 1, 2
	first := spEnt(4, 3, 3)
	first.TokenSize, first.Owner = 1, 4
	corpse := spEnt(5, 4, 4)
	corpse.TokenSize, corpse.Owner = 1, 4
	corpse.HP = -10

	rel := engRel(t,
		[3]uint32{1, 2, relationLocked},
		[3]uint32{1, 4, relationHostile},
	)
	w := hlWorld(t, 0x1009, rel, []SpellRule{rule}, caster, ally, first, corpse)

	events := spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 4, Y: 14})
	if len(events) != 1 {
		t.Fatalf("the cast was observed %d times, want once", len(events))
	}
	want := []CellPoint{{X: 3, Y: 3}}
	if !reflect.DeepEqual(events[0].Victims, want) {
		t.Errorf("the victim list is %v, want only the live hostile's cell %v — "+
			"a corpse's cell on it is the defect the owner reported", events[0].Victims, want)
	}
}

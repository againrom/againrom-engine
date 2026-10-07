package sim

import "testing"

// TestDepartedCompanionIdentityReservationSurvivesColdLoadAndFreshSummon
// covers two of the story's four requirements together, in the order the
// defect actually occurred: the reservation surviving a native cold load, and
// a later summon receiving a fresh id rather than the departed one.
func TestDepartedCompanionIdentityReservationSurvivesColdLoadAndFreshSummon(t *testing.T) {
	caster := effectMage(1, 1, 1, 1<<25)
	corpse := spEnt(2, 2, 1)
	departed := spEnt(3, 10, 10)
	tmpl := hlGhostTemplate()
	w := hlGhostWorld(t, 1177, []SpellRule{{ID: 25, ManaCost: 2, TargetsUnit: true, MaxRange: 4}},
		tmpl, caster, corpse, departed)
	w.entities[1].HP, w.entities[1].Decay = -10, DecayBones // the corpse, ready for the raise
	// The departed companion has no SourceBinding. Its terminal tuple must
	// remain separate from OriginalDeadActors, which represents source-backed
	// provenance, while reserving the departed native identity across LOAD.
	w.entities[2].HP, w.entities[2].Decay = decayGoneHP-1, DecayBones

	Step(w, nil)
	if indexOfEntity(w.entities, 3) >= 0 {
		t.Fatal("fixture did not remove the departed companion")
	}
	if len(w.OriginalDeadActors()) != 0 {
		t.Fatal("native terminal actor fabricated original dead provenance")
	}
	if got := w.CurrentTerminalActors(); len(got) != 1 || got[0].ID != 3 || got[0].Cell != 10<<8|10 {
		t.Fatalf("current terminal actors = %+v, want exact departed actor 3", got)
	}
	if w.entityIDFloor != 4 {
		t.Fatalf("entityIDFloor after removal = %d, want 4", w.entityIDFloor)
	}

	// A native AGS save and cold load: the reservation has to be a byte-form
	// fact, not in-memory bookkeeping a fresh process never sees.
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var cold World
	if err := cold.UnmarshalBinary(b); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	// Declared rule tables (spells, the ghost template) are not part of the
	// wire form — they come from installed map data, redeclared by the
	// loader on every resume — so a bare cold World needs them back before it
	// can resolve a cast at all. Nothing about entity identity lives here.
	cold.spells, cold.ghost = w.spells, w.ghost
	if cold.entityIDFloor != 4 {
		t.Fatalf("cold-loaded entityIDFloor = %d, want 4", cold.entityIDFloor)
	}
	if next, ok := cold.NextEntityID(); !ok || next != 4 {
		t.Fatalf("cold-loaded NextEntityID = %d,%t, want 4,true — id 3 would otherwise be free "+
			"to hand back out", next, ok)
	}

	// The ordinary Control Spirit release, cast against the cold-loaded world.
	spRunCast(&cold, spCast(1, 2, 25))
	if indexOfEntity(cold.entities, 3) >= 0 {
		t.Fatal("the departed companion's id was reoccupied by the new summon")
	}
	i := indexOfEntity(cold.entities, 4)
	if i < 0 {
		t.Fatal("the raise did not mint the reserved floor's own next id")
	}
	if cold.entities[i].TypeID == HumanTypeID {
		t.Fatal("fixture's Ghost carries a persist-band TypeID; it would no longer exercise " +
			"guardedCharacterLost's own reused-id distinction")
	}
}

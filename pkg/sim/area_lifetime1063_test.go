package sim

import (
	"testing"
)

// TestCloudLandingAndInstantTwentyNineShareTheRawCounter proves the two writers
// of effect+0x4c now feed the same countdown. A fresh AreaDuration 1 cloud opens
// at V0=16 rather than 17. Instant 29 then writes 17 raw, so the next tick
// reaches the positive multiple 16 and pulses. A raw one reaches zero on its
// next tick, remains canonical at zero, and is removed on the following tick.
func TestCloudLandingAndInstantTwentyNineShareTheRawCounter(t *testing.T) {
	rule := SpellRule{ID: 7, Area: true, Distribution: distributionDiamond,
		Radius: 0, AreaDuration: 1, DamageMin: 5, DamageMax: 5,
		Damaging: true, TargetsUnit: true}
	victim := spEnt(1, 5, 5)
	w := hlWorld(t, 1, Relations{}, []SpellRule{rule}, victim)
	if !w.landArea(rule, 0, 0, false, 0, 0, 5, 5, nil) {
		t.Fatal("fresh cloud landing was refused")
	}
	if got := w.CellEffects(); len(got) != 1 || got[0].Remaining != 16 {
		t.Fatalf("fresh cloud = %+v, want raw V0 counter 16", got)
	}

	beforeWrite := w.Hash()
	laNode(w, ScriptInstant{Op: ScriptInstantCellEffectAge,
		Args: [scriptParams]int32{5, 5, int32(rule.ID), 17}})
	if got := w.CellEffects(); len(got) != 1 || got[0].Remaining != 17 {
		t.Fatalf("instant 29 left %+v, want the raw authored word 17", got)
	}
	if w.Hash() == beforeWrite {
		t.Fatal("the raw instant-29 counter write did not move the world hash")
	}

	beforeHP := w.entities[0].HP
	Step(w, nil)
	if got := w.CellEffects(); len(got) != 1 || got[0].Remaining != 16 {
		t.Fatalf("first retimed tick left %+v, want raw counter 16", got)
	}
	if w.entities[0].HP >= beforeHP {
		t.Fatalf("raw 17 did not pulse on its next value 16: health %d -> %d", beforeHP, w.entities[0].HP)
	}

	laNode(w, ScriptInstant{Op: ScriptInstantCellEffectAge,
		Args: [scriptParams]int32{5, 5, int32(rule.ID), 1}})
	Step(w, nil)
	if got := w.CellEffects(); len(got) != 1 || got[0].Remaining != 0 {
		t.Fatalf("duration one after one tick = %+v, want one canonical zero-counter record", got)
	}
	zeroForm, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary at raw zero: %v", err)
	}
	var zeroBack World
	if err := zeroBack.UnmarshalBinary(zeroForm); err != nil {
		t.Fatalf("UnmarshalBinary at raw zero: %v", err)
	}
	if zeroBack.Hash() != w.Hash() {
		t.Fatalf("raw-zero round trip changed hash from %#x to %#x", w.Hash(), zeroBack.Hash())
	}
	Step(&zeroBack, nil)
	if got := zeroBack.CellEffects(); len(got) != 0 {
		t.Fatalf("zero counter survived its removal tick: %+v", got)
	}
}

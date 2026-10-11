package sim

import "testing"

// heldDriverRows counts the rows a World's world-effect driver table holds. A
// first-game World holds the table from construction, so a retired effect
// leaves it empty rather than absent.
func heldDriverRows(w *World) int {
	if w.savedWorldEffects == nil {
		return 0
	}
	return len(w.savedWorldEffects.Areas) + len(w.savedWorldEffects.Projectiles)
}

func TestAFirstGameWorldIsConstructedWithTheEffectDriverTable(t *testing.T) {
	b, ents := sample()
	w := mustWorld(t, 1, b, ents)
	if w.savedWorldEffects == nil || heldDriverRows(w) != 0 {
		t.Fatalf("driver table at construction: %+v", w.savedWorldEffects)
	}
	var cold World
	if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	if cold.savedWorldEffects == nil || heldDriverRows(&cold) != 0 {
		t.Fatal("a decoded first-game World lost its driver table")
	}
	if err := cold.ImportOriginalWorldEffectDrivers(nil); err != nil || cold.savedWorldEffects == nil {
		t.Fatal("an import with no driver row dropped the table", err)
	}
	p := cold.CurrentPolicy()
	no := false
	p.EffectDriverCarrier = &no
	if err := cold.RestoreCurrentContinuation(&p, nil, cold.Actions(), nil); err != nil {
		t.Fatal(err)
	}
	if cold.savedWorldEffects == nil {
		t.Fatal("an older engine SAV that marked the table absent left the World without it")
	}
}

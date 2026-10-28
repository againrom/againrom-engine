package game

import (
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

// TestNativeAreasSurviveTheWorldByteCycle pins the native resume audit's
// comparison for roots bound as native areas, and that it sees a lost area.
func TestNativeAreasSurviveTheWorldByteCycle(t *testing.T) {
	script, err := sim.NewScript(nil, []sim.ScriptInstant{{Op: sim.ScriptInstantCastAtCell, Args: [10]int32{10, 10, 16, 16, 3, 10}}},
		[]sim.ScriptTrigger{{Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}, Once: true, Latch: 999}})
	if err != nil {
		t.Fatal(err)
	}
	grid := make([]byte, 32*32)
	grid[16*32+16] = 3
	w, err := sim.NewSpelledWorld(7, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, grid, nil, script,
		[]sim.SpellRule{{ID: 3, Area: true, Distribution: 3, Radius: 1, AreaDuration: 15, Damaging: true, DamageMin: 1, DamageMax: 2}})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; !w.HasNativeAreaEffects() && i < 32; i++ {
		sim.Step(w, nil)
	}
	before, err := w.NativeAreaSaveStates()
	if err != nil || len(before) != 1 {
		t.Fatalf("script did not create one native area: %d, %v", len(before), err)
	}
	encoded, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var resumed sim.World
	if err := resumed.UnmarshalBinary(encoded); err != nil {
		t.Fatal(err)
	}
	after, err := resumed.NativeAreaSaveStates()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("the cycle changed the native areas:\n%+v\n%+v", before, after)
	}
	if !reflect.DeepEqual(w.SavedSpellEffects(), resumed.SavedSpellEffects()) {
		t.Fatal("the cycle changed the carried SpellEffect graph")
	}

	// Loss control: a World past the area's life compares unequal.
	for i := 0; w.HasNativeAreaEffects() && i < 256; i++ {
		sim.Step(w, nil)
	}
	gone, err := w.NativeAreaSaveStates()
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(before, gone) {
		t.Fatalf("a World without the area compares equal to one with it (%d states)", len(gone))
	}
}

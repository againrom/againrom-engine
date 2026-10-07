package sim

import "testing"

func TestOriginalActionClock1170AtomicExactDeadlineAndNativeContinuation(t *testing.T) {
	w := mustWorld(t, 1170, Bounds{8, 8}, []Entity{{ID: 7, X: 1, Y: 1, HP: 50, MaxHP: 100, HealthRegenPeriod: 100}})
	before := w.Hash()
	if err := w.ImportOriginalActionClocks(map[EntityID]uint32{7: 0xfffffff0, 9: 0}); err == nil || before != w.Hash() {
		t.Fatal("unbound deadline partially changed the candidate")
	}
	if err := w.ImportOriginalActionClocks(map[EntityID]uint32{7: 0xfffffff0}); err != nil {
		t.Fatal(err)
	}
	e := w.Entities()[0]
	if e.ActionClock != (ActionClock{Known: true, End: 0xfffffff0}) || e.regenerationRate(64) != 1 || e.regenerationRate(65) != 3 {
		t.Fatal("literal wrapping deadline or strict idle boundary changed", e.ActionClock)
	}
	before = w.Hash()
	if err := w.ImportOriginalActionClocks(map[EntityID]uint32{7: 0}); err == nil || before != w.Hash() {
		t.Fatal("a later import overwrote the known action clock")
	}
	var cold World
	if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	for range 161 {
		if w.Hash() != cold.Hash() {
			t.Fatal("native checkpoint lost imported idle age")
		}
		Step(w, nil)
		Step(&cold, nil)
	}
}

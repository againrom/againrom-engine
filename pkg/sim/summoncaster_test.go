package sim

import (
	"slices"
	"testing"
)

// R2-ENGINE-1549 (Medium): no Summon key site tests the caster, so a creature
// outlives its caster. With the caster dead, a lone keyed creature keeps its
// key and stays alive through ten minutes of World ticks: the cap never takes
// the last actor of a key. A first-game World loses no actor either.
func TestSecondSummonedCreatureOutlivesItsCaster(t *testing.T) {
	for _, second := range []bool{false, true} {
		w := keyedWorld(t, 7, second, true, 1)
		key := w.entities[slices.IndexFunc(w.entities, func(e Entity) bool { return e.ID == 2 })].summonKey
		ci := slices.IndexFunc(w.entities, func(e Entity) bool { return e.ID == 1 })
		w.entities[ci].setCurrentHealth(0)
		w.clearFelled(ci)
		start := w.tick
		for range 16 * 60 * 10 {
			Step(w, nil)
		}
		if w.tick-start != 16*60*10 {
			t.Fatalf("second=%v: %d ticks ran, want %d", second, w.tick-start, 16*60*10)
		}
		i := slices.IndexFunc(w.entities, func(e Entity) bool { return e.ID == 2 })
		if i < 0 || !w.entities[i].Alive() || w.entities[i].summonKey != key {
			t.Fatalf("second=%v: the creature after its caster's death: index %d, key %d, want alive with key %d", second, i, w.entities[max(i, 0)].summonKey, key)
		}
		if ids := keyedIDs(w, key); !slices.Equal(ids, []EntityID{2}) {
			t.Fatalf("second=%v: keyed actors %v, want the creature", second, ids)
		}
	}
}

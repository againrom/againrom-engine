package game

import (
	"testing"

	"againrom/pkg/sim"
)

// A loaded spell graph that has retired leaves its nodes in the document. A
// projectile driver written later has its own store, so it does not need the
// graph and the SAVE goes on; an area driver is bound through the graph's
// records, so it still refuses.
func TestRetiredSpellGraphRefusesOnlyForAnAreaDriver(t *testing.T) {
	for _, c := range []struct {
		name    string
		drivers sim.SavedWorldEffects
		refuse  bool
	}{
		{"projectile", sim.SavedWorldEffects{Projectiles: []sim.SavedProjectileDriver{{ID: 267, Retired: true}}}, false},
		{"area", sim.SavedWorldEffects{Areas: []sim.SavedAreaDriver{{ID: 1, Root: -1, Mode: sim.AreaModeBlast, Layer: 255, Spell: 1}}}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, make([]byte, 64), nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := w.ImportOriginalWorldEffectDrivers(&c.drivers); err != nil {
				t.Fatal(err)
			}
			state := &SnapshotSAVDocument{WorldEffects: &SnapshotSAVWorldEffects{SpellNodes: []SnapshotSAVSpellNode{{ObjectIndex: 1}}}}
			if got := validateSavedSpellGraph(state, w) != nil; got != c.refuse {
				t.Errorf("validate refused=%v, want %v", got, c.refuse)
			}
			if got := spellGraphAreasRemain(w); got != c.refuse {
				t.Errorf("areas remain=%v, want %v", got, c.refuse)
			}
		})
	}
}

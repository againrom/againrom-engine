package sim

import "fmt"

// SavedProjectile is one Prj<id> section's sixteen leaves (sav.Projectile),
// carried opaquely: this build has no live projectile registry —
// pkg/game/projectiles.go is a cosmetic art loader and cast-event renderer,
// not a persistent entity — so a restored section is never bound to a live
// object and never simulated. docs/DIVERGENCES.md names the gap.
type SavedProjectile struct {
	ID uint16

	X, Y, Z        int32
	Picture        int32
	Dir            int32
	Phase          int32
	LastAction     int32
	Action         int32
	ActionDir      int32
	ActionTarget   int32
	ActionX        int32
	ActionY        int32
	ActionZ        int32
	ActionPhase    int32
	ActionSegments int32
	ActionSpell    int32
}

// SavedProjectiles is one save's whole Projectiles subtree, the sim-layer
// mirror of sav.ProjectileStore: the manager's allocator word, the wire IDs
// array's own order and multiplicity, and one Items entry per distinct id.
type SavedProjectiles struct {
	FreeIndex uint16
	IDs       []uint16
	Items     []SavedProjectile
}

func cloneSavedProjectiles(store SavedProjectiles) SavedProjectiles {
	return SavedProjectiles{
		FreeIndex: store.FreeIndex,
		IDs:       append([]uint16(nil), store.IDs...),
		Items:     append([]SavedProjectile(nil), store.Items...),
	}
}

// SavedProjectiles returns this world's carried projectile store. It is a
// fresh copy: mutating the result cannot reach the world it came from.
func (w *World) SavedProjectiles() SavedProjectiles {
	return cloneSavedProjectiles(w.savedProjectiles)
}

// SetSavedProjectiles replaces the carried store outright, with no
// validation. resumeWorld (pkg/game/resume.go) never needed a call here even
// then: it decodes onto the SAME receiver rather than a fresh one, so the
// field already survived
// (TestUnmarshalBinaryOntoTheSameReceiverNeedsNoProjectilePriming). Form85
// gives the field a wire position (carriedresumebinary.go), so both staging
// round trips now carry it through UnmarshalBinary like every other field
// and neither calls this any more; it remains for a caller that wants to set
// the field directly without a byte-form round trip.
func (w *World) SetSavedProjectiles(store SavedProjectiles) {
	w.savedProjectiles = cloneSavedProjectiles(store)
}

// ImportOriginalProjectiles overlays the saved projection after map
// construction, on ImportOriginalCellRecords' own standing: it validates
// rather than trusts its argument. Items must carry no two entries repeating
// an id — sav.File.Projectiles already returns that shape (one entry per
// distinct id named by IDs), so a caller passing anything else built the
// argument wrong, not this save.
func (w *World) ImportOriginalProjectiles(store SavedProjectiles) error {
	seen := make(map[uint16]bool, len(store.Items))
	for _, p := range store.Items {
		if seen[p.ID] {
			return fmt.Errorf("sim: saved projectiles have duplicate id %04x", p.ID)
		}
		seen[p.ID] = true
	}
	w.savedProjectiles = cloneSavedProjectiles(store)
	return nil
}

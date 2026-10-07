package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// originalProjectiles decodes the complete supported Projectiles-subtree
// projection before a mission is prepared, mirroring originalCellRecords'
// own shape for a different section of the same state store.
func originalProjectiles(f *sav.File) (sav.ProjectileStore, bool, error) {
	return f.Projectiles()
}

func savProjectileToSaved(p sav.Projectile) sim.SavedProjectile {
	return sim.SavedProjectile{
		ID: p.ID, X: p.X, Y: p.Y, Z: p.Z, Picture: p.Picture, Dir: p.Dir, Phase: p.Phase,
		LastAction: p.LastAction, Action: p.Action, ActionDir: p.ActionDir, ActionTarget: p.ActionTarget,
		ActionX: p.ActionX, ActionY: p.ActionY, ActionZ: p.ActionZ, ActionPhase: p.ActionPhase,
		ActionSegments: p.ActionSegments, ActionSpell: p.ActionSpell,
	}
}

func savedProjectileToSav(p sim.SavedProjectile) sav.Projectile {
	return sav.Projectile{
		ID: p.ID, X: p.X, Y: p.Y, Z: p.Z, Picture: p.Picture, Dir: p.Dir, Phase: p.Phase,
		LastAction: p.LastAction, Action: p.Action, ActionDir: p.ActionDir, ActionTarget: p.ActionTarget,
		ActionX: p.ActionX, ActionY: p.ActionY, ActionZ: p.ActionZ, ActionPhase: p.ActionPhase,
		ActionSegments: p.ActionSegments, ActionSpell: p.ActionSpell,
	}
}

// applyOriginalProjectiles is shared by both original LOAD doors, on
// applyOriginalCellRecords' own standing (originalcellrecords.go): an
// independent reader of the same state store's own Projectiles subtree.
// importOriginalWorldEffects subsequently binds the known current-coordinate
// and phase/countdown consumers after actor and dead identities are available.
// Unbound arms remain explicit in the complete Document metadata.
func applyOriginalProjectiles(ms *Mission, store sav.ProjectileStore, present bool, r *OriginalSaveResume) error {
	if !present {
		return nil
	}
	if ms == nil || ms.World == nil {
		return fmt.Errorf("original projectiles: mission has no world")
	}
	items := make([]sim.SavedProjectile, len(store.Items))
	for i, p := range store.Items {
		items[i] = savProjectileToSaved(p)
	}
	saved := sim.SavedProjectiles{FreeIndex: store.FreeIndex, IDs: append([]uint16(nil), store.IDs...), Items: items}
	if err := ms.World.ImportOriginalProjectiles(saved); err != nil {
		return fmt.Errorf("original projectiles: %w", err)
	}
	if r != nil {
		r.Projectiles, r.ProjectilesApplied = len(store.Items), true
	}
	return nil
}

// exportOriginalProjectiles writes a world's carried Projectiles store back
// into a decoded original save's own world half — the counterpart write to
// applyOriginalProjectiles, exportOriginalCellRecords' own building-block
// role for a different section of the same state store. Unlike
// exportOriginalCellRecords it does not iterate the file's own existing rows
// first: the Projectiles subtree carries its own id set rather than a fixed
// per-cell table, so the carried store's own FreeIndex/IDs/Items REPLACE the
// file's Projectiles subtree outright, on sav.SetProjectiles' own contract
// (always the complete canonical shape, SAV-PROJSTORE-428).
//
// NO PRODUCTION CALLER INVOKES THIS. This project has no production mission
// SAVE path yet (owner rule); it is an unshipped round-trip building block,
// checked only by the corpus audit
// (originalprojectiles1133_corpus_test.go).
//
// IT NEVER TOUCHES THE BETWEEN-MISSION FORM, on exportOriginalCellRecords'
// own rule: a city save has no world half, so f.World == nil there refuses.
func exportOriginalProjectiles(f *sav.File, w *sim.World) error {
	if f == nil || f.World == nil {
		return fmt.Errorf("original projectiles export: this save has no world session")
	}
	if w == nil {
		return fmt.Errorf("original projectiles export: nil world")
	}
	saved := w.SavedProjectiles()
	store := sav.ProjectileStore{FreeIndex: saved.FreeIndex, IDs: append([]uint16(nil), saved.IDs...)}
	store.Items = make([]sav.Projectile, len(saved.Items))
	for i, p := range saved.Items {
		store.Items[i] = savedProjectileToSav(p)
	}
	return f.SetProjectiles(store)
}

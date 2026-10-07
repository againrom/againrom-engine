package game

import (
	"errors"
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// exportOriginalMoverRoutes writes a world's carried mover block and both
// saved route lists back into a decoded original save's own actors — the
// mover/route counterpart to exportOriginalSpellEffects and
// exportOriginalCellRecords, the same building-block role for this table.
// It targets the SAME open *sav.File the mission was resumed from, joining
// each SavedActorMotion to its own archive record by
// Entity.SourceBinding.ArchiveIndex — never position, which every tick can
// move, and SourceBinding carries no MapUnitID at all. sourceBindingsFault
// (pkg/sim/sourcebinding.go) is the invariant that keeps ArchiveIndex unique
// across every bound entity, which is what makes it a safe map key here. The
// function refuses rather than silently reshaping a route whose element
// count no longer matches the file's own declared count
// (sav.SetActorMoverRoute).
//
// IT NEVER TOUCHES THE BETWEEN-MISSION FORM: a city save has no world half,
// so f.World == nil there refuses, on exportOriginalCellRecords' own rule.
//
// This is a round-trip building block, not a production SAVE entry point:
// this project ships no original-format SAVE command, the same standing gap
// exportOriginalSpellEffects and exportOriginalCellRecords already carry
// (docs/1130/story.md, docs/1132/story.md).
//
// Every motion is attempted (F-5): one actor's binding or route-length
// failure must not leave every actor after it in file iteration order
// unexamined and unpatched, silently keeping their file bytes stale rather
// than reporting them too. The caller sees every failing record from one
// call, not one per retry.
func exportOriginalMoverRoutes(f *sav.File, w *sim.World) error {
	if f == nil || f.World == nil {
		return fmt.Errorf("original mover/route export: this save has no world session")
	}
	if w == nil {
		return fmt.Errorf("original mover/route export: nil world")
	}
	motions, _, _, present := w.SavedActorMotions()
	if !present {
		return fmt.Errorf("original mover/route export: this world carries no saved motion state")
	}
	byEntity := make(map[sim.EntityID]uint16, len(w.Entities()))
	for _, e := range w.Entities() {
		if e.SourceBinding.Class != 0 {
			byEntity[e.ID] = e.SourceBinding.ArchiveIndex
		}
	}
	var errs []error
	for _, m := range motions {
		archiveIndex, ok := byEntity[m.Entity]
		if !ok {
			errs = append(errs, fmt.Errorf("original mover/route export: actor %d has no archive binding", m.Entity))
			continue
		}
		if err := f.SetActorMoverRoute(archiveIndex, m.Mover, m.StaticRoute, m.DynamicRoute); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

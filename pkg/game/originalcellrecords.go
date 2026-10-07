package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// originalCellRecords decodes the complete supported cell-record projection
// before a mission is prepared, mirroring originalCellTails's own shape for
// the narrower trigger-only projection of the same table.
func originalCellRecords(f *sav.File) ([]sav.Cell, bool, error) {
	return f.Cells()
}

// applyOriginalCellRecords is shared by both original LOAD doors, once the
// actor registry plan exists: registry.actors already carries every living
// actor's own archive Identity beside the EntityID it was assigned, a plan
// fixed by prepareOriginalActorRegistry independently of whether
// admitOriginalActorRegistry has actually run yet.
//
// It is an independent reader of the same source table applyOriginalCellTails
// (the trigger tail) and applyOriginalStructures (baselines and the Building
// key) already overlay: this carries the fields neither of those claims —
// layer count, both residue spans, the Sack and six SpellEffect identity
// keys, kept opaque because this build has no live identity registry for
// either class yet — and the two movement-domain actor keys, each rebound to
// a live EntityID per SAV-CELLLOAD-110 wherever this build's own actor
// registry names one, with the raw key surviving unresolved otherwise.
//
// Records are deduplicated to last-write-wins by cell key first, the same
// way applyOriginalStructures already dedupes StructureCells before its own
// overlay call: ImportOriginalCellRecords refuses an input list carrying a
// duplicate key outright, on validateSavedStructures' own precedent, rather
// than silently choosing one itself.
func applyOriginalCellRecords(ms *Mission, cells []sav.Cell, present bool, registry *originalActorRegistry, r *OriginalSaveResume) error {
	if !present {
		return nil
	}
	if ms == nil || ms.World == nil {
		return fmt.Errorf("original cell records: mission has no world")
	}
	byIdentity := map[uint32]sim.EntityID{}
	if registry != nil {
		for _, binding := range registry.actors {
			if binding.Source.Identity != 0 {
				byIdentity[binding.Source.Identity] = binding.ID
			}
		}
	}
	rebind := func(key uint32) sim.SavedCellActorSlot {
		slot := sim.SavedCellActorSlot{Key: key}
		if key != 0 {
			if id, ok := byIdentity[key]; ok {
				slot.Entity, slot.Bound = id, true
			}
		}
		return slot
	}
	last := make(map[uint16]int, len(cells))
	for i, c := range cells {
		last[c.Key] = i
	}
	records := make([]sim.SavedCellRecord, 0, len(last))
	for i, c := range cells {
		if last[c.Key] != i {
			continue // Archive order, last-write-wins: an earlier duplicate is shadowed.
		}
		records = append(records, sim.SavedCellRecord{
			Cell: c.Key, LayerCount: c.LayerCount, Residue0: c.Residue0, Residue1: c.Residue1,
			Ground: rebind(c.Ground), Air: rebind(c.Air), Sack: c.Sack, SpellEffects: c.SpellEffects,
		})
	}
	if err := ms.World.ImportOriginalCellRecords(records); err != nil {
		return fmt.Errorf("original cell records: %w", err)
	}
	if r != nil {
		r.CellRecords, r.CellRecordsApplied = len(cells), true
	}
	return nil
}

// exportOriginalCellRecords writes a world's complete live cell-record table
// back into a decoded original save's own world half — the counterpart write
// to applyOriginalCellRecords, applyOriginalCellTails and applyOriginalStructures
// together, the same building-block role exportOriginalMissionSession plays
// for the session block (docs/1130/story.md). It targets the SAME open
// *sav.File the mission was resumed from, row for row: SAV-CELLLOAD-109
// neither adds nor removes records on LOAD, so this build's own record count
// and archive order already equal the file's, and each row keeps its own
// saved key rather than one re-derived from any live map order.
//
// BaselineCost/BaselineStatic and the Building link come from
// SavedStructureCell; the Building key itself is not carried there as a raw
// dword, so it is re-derived from the matching SavedStructure's own
// SourceKey. The trigger tail comes from CellTails. Ground, Air, Sack and
// SpellEffects come from SavedCellRecord and are written back as their own
// carried raw key, never reconstructed from Bound/Entity: this field has no
// rebuild rule (savedcellrecord.go doc comment), so the raw key already IS
// the byte-fidelity path, the same choice rawSessionHead/rawSessionMid made.
//
// A row whose key this build's own carriers no longer hold (a class of
// gameplay-driven removal none of the three overlay's own import functions
// perform, so unreached today) writes its zero baseline/residue/key values
// rather than refusing outright, on exportOriginalMissionSession's own
// preference for an explicit partial answer over an aborted export.
//
// IT NEVER TOUCHES THE BETWEEN-MISSION FORM, on exportOriginalMissionSession's
// own rule: a city save has no world half, so f.World == nil there refuses.
func exportOriginalCellRecords(f *sav.File, w *sim.World) error {
	if f == nil || f.World == nil {
		return fmt.Errorf("original cell records export: this save has no world session")
	}
	if w == nil {
		return fmt.Errorf("original cell records export: nil world")
	}
	existing, present, err := f.Cells()
	if err != nil {
		return err
	}
	if !present {
		return fmt.Errorf("original cell records export: this save has no world session")
	}
	structures, structCells, _ := w.SavedStructures()
	sourceKey := make(map[sim.StructureID]uint32, len(structures))
	for _, s := range structures {
		sourceKey[s.ID] = s.SourceKey
	}
	cellByKey := make(map[uint16]sim.SavedStructureCell, len(structCells))
	for _, c := range structCells {
		cellByKey[c.Cell] = c
	}
	tails := w.CellTails()
	tailByKey := make(map[uint16][6]byte, len(tails))
	for _, t := range tails {
		tailByKey[uint16(uint8(t.Y))<<8|uint16(uint8(t.X))] = t.Bytes
	}
	records := w.SavedCellRecords()
	recByKey := make(map[uint16]sim.SavedCellRecord, len(records))
	for _, rec := range records {
		recByKey[rec.Cell] = rec
	}
	for i, old := range existing {
		c := sav.Cell{Key: old.Key}
		if sc, ok := cellByKey[old.Key]; ok {
			c.BaselineCost, c.BaselineStatic = sc.BaselineCost, sc.BaselineStatic
			if sc.HasStructure {
				c.Building = sourceKey[sc.ID]
			}
		}
		if rec, ok := recByKey[old.Key]; ok {
			c.LayerCount, c.Residue0, c.Residue1 = rec.LayerCount, rec.Residue0, rec.Residue1
			c.Ground, c.Air, c.Sack, c.SpellEffects = rec.Ground.Key, rec.Air.Key, rec.Sack, rec.SpellEffects
		}
		if tail, ok := tailByKey[old.Key]; ok {
			c.Trigger = tail
		}
		if err := f.SetCell(i, c); err != nil {
			return err
		}
	}
	return nil
}

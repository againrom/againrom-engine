package sim

import (
	"fmt"
	"slices"
)

// CurrentTerminalActor is a departed native actor's tuple; not OriginalDeadActor.
// MapUnitID is the authored placement it had, 0 for none; a LOAD that no longer
// constructs the placement rebinds script references to the row through it.
// Worn has bit k set when equipment slot k still held an item at departure; a
// SAVE writes those items of the placement on the departed actor's record.
type CurrentTerminalActor struct {
	ID        EntityID
	Cell      uint16
	HP        int32
	Stage     uint8
	MapUnitID uint16 `json:",omitempty"`
	Worn      uint16 `json:",omitempty"`
}

// CurrentTerminalActors returns terminal rows in ID order.
func (w *World) CurrentTerminalActors() []CurrentTerminalActor {
	return slices.Clone(w.currentTerminalActors)
}

// ImportCurrentTerminalActors restores rows once materialized.
func (w *World) ImportCurrentTerminalActors(batch []CurrentTerminalActor) error {
	return w.importCurrentTerminalActors(batch, false)
}

// RestoreCurrentTerminalActors reuses a SAV ID; caller binds DeadActors.
func (w *World) RestoreCurrentTerminalActors(batch []CurrentTerminalActor) error {
	return w.importCurrentTerminalActors(batch, true)
}

func (w *World) importCurrentTerminalActors(batch []CurrentTerminalActor, replaceConstruction bool) error {
	if w == nil || len(batch) == 0 || len(w.currentTerminalActors) != 0 {
		return fmt.Errorf("current terminal actor import requires a fresh terminal set")
	}
	rows := slices.Clone(batch)
	slices.SortFunc(rows, func(a, b CurrentTerminalActor) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	if err := currentTerminalActorsFault(rows, w.bounds); err != nil {
		return err
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		return err
	}
	next := *w
	if err := next.UnmarshalBinary(raw); err != nil {
		return err
	}
	var remove []EntityID
	for _, row := range rows {
		if i := indexOfEntity(next.entities, row.ID); i >= 0 {
			e := next.entities[i]
			if replaceConstruction {
				if e.SourceBinding.Class != 0 {
					return fmt.Errorf("current terminal actor %d collides with a bound actor", row.ID)
				}
			} else {
				cell := uint16(e.Y)<<8 | uint16(e.X)
				if e.Alive() || e.HP != row.HP || uint8(e.Decay) != row.Stage || cell != row.Cell {
					return fmt.Errorf("current terminal actor %d differs from its exact ordinary body", row.ID)
				}
			}
			// Discard any LOAD-attached SourceBinding; stays native, not OriginalDead.
			next.entities[i].SourceBinding = SourceBinding{}
			if replaceConstruction {
				next.entities[i].NativeBasis = NativeActorBasis{}
			}
			remove = append(remove, row.ID)
		}
		for _, dead := range next.originalDead {
			if dead.ID == row.ID {
				return fmt.Errorf("current terminal actor %d collides with a retained dead actor", row.ID)
			}
		}
	}
	if len(remove) != 0 && !next.remove(remove) {
		return fmt.Errorf("current terminal actor import could not retire exact ordinary bodies")
	}
	if replaceConstruction {
		ids := make(map[EntityID]bool, len(rows))
		for _, row := range rows {
			ids[row.ID] = true
		}
		next.currentTerminalActors = slices.DeleteFunc(next.currentTerminalActors, func(row CurrentTerminalActor) bool {
			return ids[row.ID]
		})
	}
	for _, row := range rows {
		if !next.hasCurrentTerminalActor(row) {
			next.currentTerminalActors = append(next.currentTerminalActors, row)
			continue
		}
		for i := range next.currentTerminalActors {
			if held := &next.currentTerminalActors[i]; held.ID == row.ID && row.Worn != 0 {
				held.Worn = row.Worn
			}
		}
	}
	slices.SortFunc(next.currentTerminalActors, func(a, b CurrentTerminalActor) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	if err := currentTerminalActorsFault(next.currentTerminalActors, next.bounds); err != nil {
		return err
	}
	for _, row := range next.currentTerminalActors {
		if indexOfEntity(next.entities, row.ID) >= 0 {
			return fmt.Errorf("current terminal actor %d collides with a live entity", row.ID)
		}
	}
	*w = next
	return nil
}

func currentTerminalActorsFault(rows []CurrentTerminalActor, bounds Bounds) error {
	seen := map[EntityID]bool{}
	for i, row := range rows {
		if seen[row.ID] || i > 0 && rows[i-1].ID >= row.ID ||
			row.HP >= decayGoneHP || row.Stage > consumedCorpseStage || row.Worn >= 1<<EquipSlots ||
			int32(row.Cell&255) >= bounds.Width || int32(row.Cell>>8) >= bounds.Height {
			return fmt.Errorf("sim: invalid current terminal actor %d", row.ID)
		}
		seen[row.ID] = true
	}
	return nil
}

func (w *World) hasCurrentTerminalActor(want CurrentTerminalActor) bool {
	for _, row := range w.currentTerminalActors {
		if row.ID == want.ID {
			row.Worn, want.Worn = 0, 0
			return row == want
		}
	}
	return false
}

// RetireUnboundConstructors removes LOAD constructors no SAV actor binds; each
// named entity must carry no SourceBinding. It changes nothing on refusal.
func (w *World) RetireUnboundConstructors(ids []EntityID) error {
	if w == nil || len(ids) == 0 {
		return nil
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		return err
	}
	next := *w
	if err := next.UnmarshalBinary(raw); err != nil {
		return err
	}
	for _, id := range ids {
		i := indexOfEntity(next.entities, id)
		if i < 0 || next.entities[i].SourceBinding.Class != 0 {
			return fmt.Errorf("sim: entity %d is not an unbound constructor", id)
		}
		next.entities[i].NativeBasis = NativeActorBasis{}
	}
	if !next.remove(slices.Clone(ids)) {
		return fmt.Errorf("sim: could not retire unbound constructors")
	}
	*w = next
	return nil
}

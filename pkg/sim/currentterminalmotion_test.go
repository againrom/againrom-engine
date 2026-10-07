package sim

import (
	"slices"
	"testing"
)

func currentDetachedMotionCandidate(t *testing.T) (*World, *World, CurrentTerminalMotion) {
	t.Helper()
	w := importedRelocationWorld1115(t)
	w.SetSavedCellRecords([]SavedCellRecord{{Cell: 0x0203, Ground: SavedCellActorSlot{Key: 0x1004, Entity: 1, Bound: true}}})
	if !w.remove([]EntityID{1}) || len(w.OriginalDeadActors()) != 0 {
		t.Fatal("control requires a removed actor without retained corpse")
	}
	row := CurrentTerminalMotion{Motion: *w.motionFor(1), Detached: true, Slots: []CurrentTerminalMotionSlot{{Cell: 0x0203, Key: 0x1004, Motion: true, Record: true}}}
	var cold World
	if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	cold.savedMotion.Motions = nil
	cold.motionCell(0x0203).Ground = SavedActorSlot{Key: 0x1004}
	cold.savedCellRecords[0].Ground = SavedCellActorSlot{Key: 0x1004}
	return w, &cold, row
}

func TestCurrentDetachedMotionKeepsTypedOccupancy(t *testing.T) {
	w, cold, row := currentDetachedMotionCandidate(t)
	at, _ := w.cellIndex(3, 2)
	if newRouteScratch(w).at(0, at) != 0 || newRouteScratch(cold).at(0, at) != 1 {
		t.Fatal("control failed to distinguish a superseded binding from an unresolved key")
	}
	if err := cold.RestoreCurrentContinuation(nil, nil, cold.Actions(), nil, row); err != nil {
		t.Fatal(err)
	}
	if cold.Hash() != w.Hash() || newRouteScratch(cold).at(0, at) != 0 {
		t.Fatal("detached motion changed World or left a phantom occupied cell")
	}
	for range 20 {
		Step(w, nil)
		Step(cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("removed motion continuation differs")
		}
	}
}

func TestCurrentDetachedMotionRejectsMalformedInputAtomically(t *testing.T) {
	for _, name := range []string{"duplicate", "live", "current", "active", "empty-issue", "wrong-key", "duplicate-site", "empty-site", "retained-owner"} {
		t.Run(name, func(t *testing.T) {
			_, cold, row := currentDetachedMotionCandidate(t)
			row.Slots = slices.Clone(row.Slots)
			rows := []CurrentTerminalMotion{row}
			switch name {
			case "duplicate":
				rows = append(rows, row)
			case "live":
				rows[0].Motion.Entity = cold.entities[0].ID
			case "current":
				rows[0].Motion.Current = true
			case "active":
				rows[0].Motion.Active = true
			case "empty-issue":
				rows[0].Motion.Issue = ""
			case "wrong-key":
				rows[0].Slots[0].Key++
			case "duplicate-site":
				rows[0].Slots = append(rows[0].Slots, rows[0].Slots[0])
			case "empty-site":
				rows[0].Slots[0].Motion, rows[0].Slots[0].Record = false, false
			case "retained-owner":
				rows[0].Detached, rows[0].Key = false, 0x1004
			}
			before := cold.Hash()
			if err := cold.RestoreCurrentContinuation(nil, nil, cold.Actions(), nil, rows...); err == nil || cold.Hash() != before {
				t.Fatal("malformed motion accepted or changed World", err)
			}
		})
	}
}

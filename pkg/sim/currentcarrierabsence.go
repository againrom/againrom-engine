package sim

import (
	"encoding/binary"
	"fmt"
	"slices"
)

type CurrentSackKeyAbsence struct {
	Cell     uint16
	Wire     uint32
	Carriers uint8
}

const (
	CurrentSackKeyRecord uint8 = 1 << iota
	CurrentSackKeyMotion
)

// A current native cell may lack an archive key for an extant Sack. Restore
// that absence only after the adapter matched the exact ordinary cell/key.
func (w *World) RestoreCurrentSackKeyAbsence(rows []CurrentSackKeyAbsence) error {
	if len(rows) == 0 {
		return nil
	}
	indices := map[uint16]int{}
	for i, row := range w.savedCellRecords {
		indices[row.Cell] = i
	}
	seen := map[uint16]bool{}
	for _, row := range rows {
		if row.Carriers == 0 || row.Carriers > CurrentSackKeyRecord|CurrentSackKeyMotion || row.Wire == 0 || seen[row.Cell] {
			return fmt.Errorf("sim: invalid current cell Sack absence")
		}
		if row.Carriers&CurrentSackKeyRecord != 0 {
			i, ok := indices[row.Cell]
			if !ok || w.savedCellRecords[i].Sack != row.Wire {
				return fmt.Errorf("sim: current record Sack absence lost its ordinary key")
			}
		}
		if row.Carriers&CurrentSackKeyMotion != 0 {
			cell := w.motionCell(row.Cell)
			if cell == nil || binary.LittleEndian.Uint32(cell.Payload[16:]) != row.Wire {
				return fmt.Errorf("sim: current motion Sack absence lost its ordinary key")
			}
		}
		seen[row.Cell] = true
	}
	w.savedCellRecords = slices.Clone(w.savedCellRecords)
	w.savedMotion = cloneActorMotions(w.savedMotion)
	for _, row := range rows {
		if row.Carriers&CurrentSackKeyRecord != 0 {
			w.savedCellRecords[indices[row.Cell]].Sack = 0
		}
		if row.Carriers&CurrentSackKeyMotion != 0 {
			binary.LittleEndian.PutUint32(w.motionCell(row.Cell).Payload[16:], 0)
		}
	}
	return nil
}

// RestoreCurrentCarrierAbsence removes only carrier rows whose ordinary
// representation was checked by the current LOAD adapter. Motion, terrain and
// every other projection of the same cell remain independent current state.
func (w *World) RestoreCurrentCarrierAbsence(tails, records []uint16, diaries []SavedDiaryOwner, spatialRows ...[]uint16) error {
	if len(spatialRows) > 2 {
		return fmt.Errorf("sim: excessive current spatial absence populations")
	}
	var blocks, cells []uint16
	if len(spatialRows) >= 1 {
		blocks = spatialRows[0]
	}
	if len(spatialRows) == 2 {
		cells = spatialRows[1]
	}
	tailKeys, recordKeys := map[uint16]bool{}, map[uint16]bool{}
	for _, row := range w.cellTails {
		tailKeys[row.Key] = true
	}
	for _, row := range w.savedCellRecords {
		recordKeys[row.Cell] = true
	}
	validateCells := func(keys map[uint16]bool, remove []uint16) error {
		for _, key := range remove {
			if !keys[key] {
				return fmt.Errorf("sim: current carrier absence has a missing or repeated cell")
			}
			keys[key] = false
		}
		return nil
	}
	if err := validateCells(tailKeys, tails); err != nil {
		return err
	}
	if err := validateCells(recordKeys, records); err != nil {
		return err
	}
	blockKeys := map[uint16]bool{}
	if w.savedMotion != nil {
		for _, row := range w.savedMotion.Blocks {
			blockKeys[row.Cell] = true
		}
	}
	if err := validateCells(blockKeys, blocks); err != nil {
		return err
	}
	motionKeys := map[uint16]bool{}
	if w.savedMotion != nil {
		for _, row := range w.savedMotion.Cells {
			motionKeys[row.Cell] = true
		}
	}
	if err := validateCells(motionKeys, cells); err != nil {
		return err
	}
	owners := map[SavedDiaryOwner]bool{}
	for _, row := range w.savedDiaries {
		owners[row.Owner] = true
	}
	for _, owner := range diaries {
		if owner.Player && owner.Actor != 0 || !owners[owner] {
			return fmt.Errorf("sim: current carrier absence has a missing or repeated Diary owner")
		}
		owners[owner] = false
	}
	if len(tails) != 0 {
		w.cellTails = slices.DeleteFunc(slices.Clone(w.cellTails), func(row cellTail) bool { return !tailKeys[row.Key] })
	}
	if len(records) != 0 {
		w.savedCellRecords = slices.DeleteFunc(slices.Clone(w.savedCellRecords), func(row SavedCellRecord) bool { return !recordKeys[row.Cell] })
	}
	if len(diaries) != 0 {
		w.savedDiaries = slices.DeleteFunc(cloneSavedDiaries(w.savedDiaries), func(row SavedDiary) bool { return !owners[row.Owner] })
	}
	if len(blocks)+len(cells) != 0 {
		w.savedMotion = cloneActorMotions(w.savedMotion)
	}
	if len(blocks) != 0 {
		w.savedMotion.Blocks = slices.DeleteFunc(w.savedMotion.Blocks, func(row SavedActorBlock) bool { return !blockKeys[row.Cell] })
	}
	if len(cells) != 0 {
		w.savedMotion.Cells = slices.DeleteFunc(w.savedMotion.Cells, func(row SavedActorCell) bool { return !motionKeys[row.Cell] })
	}
	return nil
}

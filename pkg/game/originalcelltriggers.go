package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// originalCellTails decodes the complete supported projection before a mission
// is prepared. It does not reconstruct occupancy or interpret operation bytes.
func originalCellTails(f *sav.File) ([]sim.CellTail, bool, error) {
	records, present, err := f.CellTriggers()
	if err != nil || !present {
		return nil, present, err
	}
	tails := make([]sim.CellTail, len(records))
	for i, record := range records {
		tails[i] = sim.CellTail{X: int32(record.Cell & 255), Y: int32(record.Cell >> 8), Bytes: record.Bytes}
	}
	return tails, true, nil
}

// applyOriginalCellTails is shared by both original LOAD doors, after ALM
// construction and before publication. It deliberately never attaches actors.
func applyOriginalCellTails(ms *Mission, tails []sim.CellTail, present bool, r *OriginalSaveResume) error {
	if !present {
		return nil
	}
	if ms == nil || ms.World == nil {
		return fmt.Errorf("original cell triggers: mission has no world")
	}
	if err := ms.World.ImportOriginalCellTails(tails); err != nil {
		return fmt.Errorf("original cell triggers: %w", err)
	}
	if r != nil {
		r.CellTriggerRecords, r.CellTriggersApplied = len(tails), true
	}
	return nil
}

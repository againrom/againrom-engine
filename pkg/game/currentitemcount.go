package game

import "fmt"

// Ordinary F42 is a word. The native width operand preserves only the excess;
// changing the ordinary word discards it without restoring any Item value.
type currentItemCountLift struct {
	Wire uint16
	Lift uint32
}

func validateCurrentItemCount(row currentOwnedObject) error {
	if lift := row.CountLift; lift != nil {
		count := uint64(lift.Wire) + uint64(lift.Lift)
		if row.Kind != 1 || row.ID == 0 || row.Object == 0 || row.Item != nil || lift.Wire == 0 || lift.Lift == 0 || count <= 65535 || count > uint64(^uint32(0)) {
			return fmt.Errorf("invalid current Item count width operand")
		}
	}
	return nil
}

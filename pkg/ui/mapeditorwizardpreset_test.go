package ui

import (
	"testing"

	"againrom/pkg/scriptprofile"
)

// Every preset row of a game joins an operation row of that game, and the
// game's offered rows are exactly its presets.
func TestWizardPresetsJoinOperationRows(t *testing.T) {
	for ops, presets := range wizPresets {
		for k, condition := range []bool{false, true} {
			for _, it := range presets[k] {
				if _, ok := scriptprofile.OpRow(ops, condition, it.opcode); !ok {
					t.Errorf("preset %s has no operation row %d (condition %v)", it.labelWord, it.opcode, condition)
				}
			}
			if got := len(wizardTables[ops][k]); got != len(presets[k]) {
				t.Errorf("game %v condition %v offers %d rows for %d presets", ops, condition, got, len(presets[k]))
			}
		}
	}
}

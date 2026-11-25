package sim

import "testing"

// TestScriptOpRowsAreTheSecondGameOps pins the op set the second game's rows
// cover, which is the one dispatch's whole second-game route, and that the
// first game holds no rows.
func TestScriptOpRowsAreTheSecondGameOps(t *testing.T) {
	rows := newGameProfile(ScriptROM2).scriptRows
	checkOps := []int32{
		ScriptCheckPlayerStructures, ScriptCheckCellObject,
		ScriptCheckScenario, ScriptCheckObjective, ScriptCheckCellEffect,
		ScriptCheckUnitEffect, ScriptCheckCentered,
	}
	for _, op := range checkOps {
		if rows.check[op] == nil {
			t.Errorf("check op %d has no second-game row", op)
		}
	}
	if len(rows.check) != len(checkOps) {
		t.Errorf("check rows = %d, want %d", len(rows.check), len(checkOps))
	}
	instantOps := []int32{
		ScriptInstantCounter, ScriptInstantMessage, ScriptInstantSetScenario,
		ScriptInstantObjective, ScriptInstantDiagnostic, ScriptInstantTakeItemFromMap,
		ScriptInstantClearGroupActivity, ScriptInstantLose, ScriptInstantMoveItem,
		ScriptInstantRelocateUnit, ScriptInstantOutpostCountdown,
	}
	for _, op := range instantOps {
		if rows.instant[op] == nil {
			t.Errorf("instant op %d has no second-game row", op)
		}
	}
	if len(rows.instant) != len(instantOps) {
		t.Errorf("instant rows = %d, want %d", len(rows.instant), len(instantOps))
	}
	first := newGameProfile(ScriptROM1).scriptRows
	if len(first.check) != 0 || len(first.instant) != 0 {
		t.Errorf("first game holds script rows: %d checks, %d instants", len(first.check), len(first.instant))
	}
}

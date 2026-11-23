package mapload

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/scriptprofile"
)

// ScriptOperation is one operation a game's editor offers, by id and name.
type ScriptOperation struct {
	Opcode uint32
	Name   string
}

// ScriptOperationNameIn is the name of an operation in the game's
// vocabulary: an id with no row, or a row no claim names, is "Unknown
// operation".
func ScriptOperationNameIn(ops scriptprofile.Ops, condition bool, opcode uint32) string {
	if row, ok := scriptprofile.OpRow(ops, condition, opcode); ok && row.Name != "" {
		return row.Name
	}
	return "Unknown operation"
}

// ScriptOperationName is the first game's name of an operation.
func ScriptOperationName(condition bool, opcode uint32) string {
	return ScriptOperationNameIn(scriptprofile.OpsFirst, condition, opcode)
}

// ScriptOperationDecoded reports that a claim reads what the operation does
// in the game's vocabulary.
func ScriptOperationDecoded(ops scriptprofile.Ops, condition bool, opcode uint32) bool {
	row, ok := scriptprofile.OpRow(ops, condition, opcode)
	return ok && row.Decoded
}

// ScriptOperationsIn lists the operations the game's editor offers, in the
// row table's order.
func ScriptOperationsIn(ops scriptprofile.Ops, condition bool) []ScriptOperation {
	var rows []ScriptOperation
	for _, row := range scriptprofile.OpRows(ops, condition) {
		if row.Offered {
			rows = append(rows, ScriptOperation{row.ID, ScriptOperationNameIn(ops, condition, row.ID)})
		}
	}
	return rows
}

// ScriptOperations are the first game's offered operations.
func ScriptOperations(condition bool) []ScriptOperation {
	return ScriptOperationsIn(scriptprofile.OpsFirst, condition)
}

// AgainromScriptOperation is the one Againrom operation of each node array
// (DIV-4009): the health-below condition and the undying action.
func AgainromScriptOperation(condition bool) uint32 {
	if condition {
		return opHealthBelow
	}
	return opUndying
}

// ScriptAgainromOnly reports that a script holds an Againrom operation, a
// row of the op table marked Again, which is what marks its map as
// Againrom-only: the original builds such a node and its dispatch does
// nothing with it (TRIG-COND-003, TRIG-ACT-004).
func ScriptAgainromOnly(sc alm.Script) bool {
	again := func(condition bool, nodes []alm.ScriptNode) bool {
		for _, n := range nodes {
			if row, ok := scriptprofile.OpRow(scriptprofile.OpsFirst, condition, n.Opcode); ok && row.Again {
				return true
			}
		}
		return false
	}
	return again(true, sc.Conditions) || again(false, sc.Actions)
}

var ScriptParameterNames = [...]string{"None", "Int/Enum", "Group", "Player", "Unit", "X", "Y", "Const", "Item", "Structure"}
var ScriptComparisons = [...]string{"==", "!=", ">", "<", ">=", "<="}

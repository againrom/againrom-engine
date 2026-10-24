package mapload

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

type ScriptOperation struct {
	Opcode uint32
	Name   string
}

var scriptCheckNames = map[uint32]string{
	1: "Group count", 2: "Unit in box", 3: "Unit in distance square", 4: "Unit field (selector 6)", 5: "Unit alive", 6: "Distance between units", 7: "Point to unit distance", 8: "Player population", 9: "Unit target ID", 10: "Player relation", 11: "No-write arm", 12: "Item presence", 13: "No-write arm", 14: "Sack at cell", 15: "Nearest player unit distance", 16: "Item to point distance", 17: "Item presence", 18: "Lose if unit dies", 19: "Read variable", 21: "Structure field", 22: "Move object check", 0x10002: "Constant / variable preset",
	opHealthBelow: "Health below (Againrom)",
}
var scriptActionNames = map[uint32]string{
	2: "Send message", 3: "Set variable", 4: "Win", 5: "Lose", 6: "Group command", 7: "Set formation", 8: "Increment variable", 9: "No-op arm", 10: "Set relation", 12: "Create item", 13: "Take item", 16: "Take unit off map", 17: "Return unit to map", 18: "Replace unit on map", 19: "Give unit to player", 20: "Drop inventory", 21: "Cast at cell", 22: "Give group to player", 23: "Give money to player", 24: "Cast at unit", 25: "Set cell tail", 26: "Set structure field", 27: "Move unit", 28: "Give entire inventory", 29: "Set cell effect duration", 30: "Set unit effect duration", 32: "Take group off map", 33: "Return group to map", 34: "Set unit field", 0x10002: "Drop location (build time)", 0x10003: "Object form (build time)",
	opUndying: "Set undying (Againrom)",
}

func ScriptOperationName(condition bool, opcode uint32) string {
	names := scriptActionNames
	if condition {
		names = scriptCheckNames
	}
	if name := names[opcode]; name != "" {
		return name
	}
	return "Unknown operation"
}

func ScriptOperations(condition bool) []ScriptOperation {
	var rows []ScriptOperation
	for op := uint32(1); op <= 34; op++ {
		if sim.ScriptOperationSupported(condition, int32(op)) {
			rows = append(rows, ScriptOperation{op, ScriptOperationName(condition, op)})
		}
	}
	rows = append(rows, ScriptOperation{0x10002, ScriptOperationName(condition, 0x10002)})
	if !condition {
		rows = append(rows, ScriptOperation{0x10003, ScriptOperationName(false, 0x10003)})
	}
	op := AgainromScriptOperation(condition)
	return append(rows, ScriptOperation{op, ScriptOperationName(condition, op)})
}

// AgainromScriptOperation is the one Againrom operation of each node array
// (DIV-4009): the health-below condition and the undying action.
func AgainromScriptOperation(condition bool) uint32 {
	if condition {
		return opHealthBelow
	}
	return opUndying
}

// ScriptAgainromOnly reports that a script holds an Againrom operation, which
// is what marks its map as Againrom-only: the original builds such a node and
// its dispatch does nothing with it (TRIG-COND-003, TRIG-ACT-004).
func ScriptAgainromOnly(sc alm.Script) bool {
	for _, n := range sc.Conditions {
		if n.Opcode == AgainromScriptOperation(true) {
			return true
		}
	}
	for _, n := range sc.Actions {
		if n.Opcode == AgainromScriptOperation(false) {
			return true
		}
	}
	return false
}

var ScriptParameterNames = [...]string{"None", "Int/Enum", "Group", "Player", "Unit", "X", "Y", "Const", "Item", "Structure"}
var ScriptComparisons = [...]string{"==", "!=", ">", "<", ">=", "<="}

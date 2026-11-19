package mapload

import (
	"sort"

	"againrom/pkg/scriptprofile"
)

// The second game's operations a claim reads the effect of (R2-ENGINE-1313 to
// R2-ENGINE-1315, R2-ENGINE-046, R2-ENGINE-053 to R2-ENGINE-057). An id
// absent here is shown by its raw number, not decoded.
var secondCheckNames = map[uint32]string{
	5: "Unit state is not 16", 18: "Failure counter while unit is in state 16", 19: "Map register",
	23: "Scenario variable", 24: "Scenario variable 752+n", 25: "Cell effect layer present",
	26: "Unit effect active", 27: "Unit tile position", ScriptConstant: "Compiler parameter",
}
var secondActionNames = map[uint32]string{
	2: "Send event", 3: "Set map register", 4: "Add to win counter", 5: "Store failure reason",
	8: "Increment map register", 10: "Set relation", 35: "Set scenario variable",
	36: "Set scenario state 752+n", 38: "Take item from units", 39: "Clear group forced activity",
	ScriptConstant: "Compiler parameter",
}

// scriptNames is a game's operation names of one node family.
func scriptNames(ops scriptprofile.Ops, condition bool) map[uint32]string {
	switch {
	case ops == scriptprofile.OpsSecond && condition:
		return secondCheckNames
	case ops == scriptprofile.OpsSecond:
		return secondActionNames
	case condition:
		return scriptCheckNames
	}
	return scriptActionNames
}

// ScriptOperationNameIn is the name of an operation in the game's
// vocabulary: a second-game id no claim reads is "Unknown operation".
func ScriptOperationNameIn(ops scriptprofile.Ops, condition bool, opcode uint32) string {
	if name := scriptNames(ops, condition)[opcode]; name != "" {
		return name
	}
	return "Unknown operation"
}

// ScriptOperationDecoded reports that a claim reads what the operation does
// in the game's vocabulary; the first game's answer is ScriptOperationNamed.
func ScriptOperationDecoded(ops scriptprofile.Ops, condition bool, opcode uint32) bool {
	if ops == scriptprofile.OpsSecond {
		return secondCheckNames[opcode] != "" && condition || secondActionNames[opcode] != "" && !condition
	}
	return ScriptOperationNamed(condition, opcode)
}

// ScriptOperationsIn lists the operations the game's editor offers or shows,
// by id: the first game's are the simulation's supported arms.
func ScriptOperationsIn(ops scriptprofile.Ops, condition bool) []ScriptOperation {
	if ops != scriptprofile.OpsSecond {
		return ScriptOperations(condition)
	}
	var rows []ScriptOperation
	for op, name := range scriptNames(ops, condition) {
		rows = append(rows, ScriptOperation{op, name})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Opcode < rows[j].Opcode })
	return rows
}

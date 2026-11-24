package mapload

import (
	"againrom/pkg/scriptprofile"
	"againrom/pkg/sim"
)

// The parameters an operation's arm reads, in the node's file slot order
// (ALM-TRIG-045: parameters are stored by slot). The type codes are those of
// ALM-TRIG-046; the role names what the arm does with the slot, from the
// claim cited beside each row of the operation table. The builder packs plain
// slots into p0, p1... and gives reference slots their own fields
// (TRIG-PARAM-030), so the order below is the order the arm reads its plain
// parameters in.

// The parameter type codes of ALM-TRIG-046.
const (
	ScriptNone      = 0
	ScriptInt       = 1
	ScriptGroup     = 2
	ScriptPlayer    = 3
	ScriptUnit      = 4
	ScriptX         = 5
	ScriptY         = 6
	ScriptConst     = 7
	ScriptItem      = 8
	ScriptStructure = 9
)

// ScriptSlot is one parameter slot of a signature: its type code and the role
// word of what the arm reads it as. A cell is an X slot followed by a Y slot,
// and the X slot carries the role.
type ScriptSlot = scriptprofile.Slot

// ScriptGroupCommand is the action whose first parameter selects a group
// command (TRIG-GROUP-005); its signature follows that command.
const ScriptGroupCommand = 6

// scriptFollowCommand is the group command whose shipped nodes store the
// target group in the slot after the command, not in the last slot.
const scriptFollowCommand = 15

// The Againrom operations' opcodes (DIV-4009).
const (
	opHealthBelow = uint32(sim.ScriptCheckHealthBelow)
	opUndying     = uint32(sim.ScriptInstantUndying)
)

// ScriptConstant is the build-time operation: a condition preset to its value
// (TRIG-COND-003) or the action that is the drop cell (TRIG-DROP-013).
const ScriptConstant = 0x10002

type slots []ScriptSlot

var group = ScriptSlot{Type: ScriptGroup, Role: "group"}

// ScriptGroupCommands are the commands action 6 can be authored with
// (TRIG-GROUP-005), in order.
var ScriptGroupCommands = func() []uint32 {
	row, _ := scriptprofile.OpRow(scriptprofile.OpsFirst, false, ScriptGroupCommand)
	return row.Commands
}()

// ScriptSignatureIn is the ten parameter slots of operation opcode in the
// game's rows, and false when no claim states them. For an operation with
// group commands, command is its first parameter.
func ScriptSignatureIn(ops scriptprofile.Ops, condition bool, opcode, command uint32) ([10]ScriptSlot, bool) {
	var out [10]ScriptSlot
	row, ok := scriptprofile.OpRow(ops, condition, opcode)
	if !ok {
		return out, false
	}
	s := slots(row.Signature)
	ok = row.HasSignature()
	if len(row.Commands) > 0 {
		// The command is the Const first parameter; the target group stands
		// in the last slot, and for Follow in the slot after the command.
		s, ok = row.CommandSignatures[command]
		if ok && command == scriptFollowCommand {
			s = append(slots{{Type: ScriptConst, Role: "command"}, group}, s...)
		} else if ok {
			s = append(slots{{Type: ScriptConst, Role: "command"}}, s...)
			out[9] = group
		}
	}
	copy(out[:], s)
	return out, ok
}

// ScriptSignature is the first game's signature of an operation.
func ScriptSignature(condition bool, opcode, command uint32) ([10]ScriptSlot, bool) {
	return ScriptSignatureIn(scriptprofile.OpsFirst, condition, opcode, command)
}

// ScriptOperationNamed reports that a claim names what the first game's
// operation opcode does.
func ScriptOperationNamed(condition bool, opcode uint32) bool {
	return ScriptOperationDecoded(scriptprofile.OpsFirst, condition, opcode)
}

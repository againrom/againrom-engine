package mapload

// The parameters an operation's arm reads, in the node's file slot order
// (ALM-TRIG-045: parameters are stored by slot). The type codes are those of
// ALM-TRIG-046; the role names what the arm does with the slot, from the
// claim cited beside each row. The builder packs plain slots into p0, p1...
// and gives reference slots their own fields (TRIG-PARAM-030), so the order
// below is the order the arm reads its plain parameters in.

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
type ScriptSlot struct {
	Type uint32
	Role string
}

// ScriptGroupCommand is the action whose first parameter selects a group
// command (TRIG-GROUP-005); its signature follows that command.
const ScriptGroupCommand = 6

// scriptFollowCommand is the group command whose shipped nodes store the
// target group in the slot after the command, not in the last slot.
const scriptFollowCommand = 15

// ScriptConstant is the build-time operation: a condition preset to its value
// (TRIG-COND-003) or the action that is the drop cell (TRIG-DROP-013).
const ScriptConstant = 0x10002

type slots []ScriptSlot

func unit(role string) ScriptSlot   { return ScriptSlot{ScriptUnit, role} }
func number(role string) ScriptSlot { return ScriptSlot{ScriptInt, role} }
func cell(role string) slots        { return slots{{ScriptX, role}, {ScriptY, ""}} }
func join(parts ...any) slots {
	var out slots
	for _, p := range parts {
		switch v := p.(type) {
		case ScriptSlot:
			out = append(out, v)
		case slots:
			out = append(out, v...)
		}
	}
	return out
}

var (
	group     = ScriptSlot{ScriptGroup, "group"}
	player    = ScriptSlot{ScriptPlayer, "player"}
	item      = ScriptSlot{ScriptItem, "item"}
	structure = ScriptSlot{ScriptStructure, "structure"}
)

// checkSignatures are the conditions' parameter slots.
var checkSignatures = map[uint32]slots{
	1:              {group},                                             // TRIG-COUNT-015
	2:              join(unit("unit"), cell("corner"), cell("corner2")), // TRIG-CHECKGEOM-130
	3:              join(unit("unit"), cell("cell"), number("radius")),  // TRIG-DIST-014
	4:              {unit("unit"), number("selector")},                  // TRIG-CHECK-051
	5:              {unit("unit")},                                      // TRIG-CHECKGEOM-130
	6:              {unit("unit"), unit("unit2")},                       // TRIG-DIST-014
	7:              join(unit("unit"), cell("cell")),                    // TRIG-DIST-014
	8:              {player},                                            // TRIG-LISTREAD-131
	9:              {unit("unit")},                                      // TRIG-TARGETID-032
	10:             {player, {ScriptPlayer, "player2"}},                 // TRIG-DIPLO-020
	11:             {},                                                  // TRIG-COND-003, a dead arm
	12:             {unit("unit"), item},                                // TRIG-ITEMTEST-040
	13:             {},                                                  // TRIG-COND-003, a dead arm
	14:             cell("cell"),                                        // TRIG-SACK-022
	15:             join(player, cell("cell")),                          // TRIG-NEAREST-029
	16:             join(unit("unit"), item, cell("cell")),              // TRIG-CHECK-052
	17:             {unit("unit"), item},                                // TRIG-ITEMTEST-040
	18:             {unit("unit")},                                      // MISSION-VIP-004
	19:             {number("variable")},                                // TRIG-COND-003
	20:             {player},                                            // TRIG-CHECKCOUNT-132
	21:             {structure},                                         // TRIG-CHECK-053
	22:             join(cell("cell"), cell("cell2")),                   // TRIG-CELLRESULT-133
	ScriptConstant: {number("value")},                                   // TRIG-COND-003
}

// actionSignatures are the actions' parameter slots; action 6 is in
// groupCommandSignatures.
var actionSignatures = map[uint32]slots{
	1:              {},                                                                  // TRIG-ACTIONWRITE-135
	2:              {number("message")},                                                 // TRIG-MSG-023
	3:              {number("variable"), number("value")},                               // TRIG-ACT-004
	4:              {},                                                                  // TRIG-ACT-004
	5:              {},                                                                  // TRIG-ACT-004
	8:              {number("variable")},                                                // TRIG-ACT-004
	9:              {},                                                                  // TRIG-ACT-004, a dead arm
	10:             {number("player"), number("player2"), number("value")},              // TRIG-DIPLO-019
	11:             {unit("unit"), unit("unit2"), item},                                 // TRIG-XFERITEM-039
	12:             {unit("unit"), item},                                                // TRIG-ADDITEM-027
	13:             {unit("unit"), item},                                                // TRIG-TAKEITEM-038
	16:             {unit("unit")},                                                      // TRIG-OFFMAP-041
	17:             {unit("unit")},                                                      // TRIG-RETURN-042
	18:             {unit("unit"), unit("unit2")},                                       // TRIG-MAPGROUP-043
	19:             {unit("unit"), player},                                              // PARTY-JOIN-025
	20:             {unit("unit")},                                                      // TRIG-DROPALL-024
	21:             join(cell("cell"), cell("cell2"), number("spell"), number("power")), // TRIG-CAST-033
	22:             {group, player},                                                     // PARTY-JOIN-025
	23:             {player, number("money")},                                           // TRIG-MONEY-028
	24:             join(cell("cell"), unit("unit"), number("spell"), number("power")),  // TRIG-CAST-033
	25:             join(number("spell"), number("power"), cell("cell")),                // TRIG-CELLTAIL-035
	26:             {structure, number("value")},                                        // TRIG-CHECK-053
	27:             join(unit("unit"), cell("cell")),                                    // AI-SCRIPTGUARD-625
	28:             {unit("unit"), unit("unit2")},                                       // TRIG-GIVEALL-025
	29:             join(cell("cell"), number("spell"), number("time")),                 // TRIG-EFFECTTIME-034
	30:             {unit("unit"), number("spell"), number("time")},                     // TRIG-EFFECTTIME-034
	32:             {group},                                                             // TRIG-MAPGROUP-043
	33:             {group},                                                             // TRIG-MAPGROUP-043
	34:             {unit("unit"), number("selector"), number("value")},                 // TRIG-PROPERTY-036
	ScriptConstant: cell("cell"),                                                        // TRIG-DROP-013
}

// groupCommandSignatures are action 6's slots by its command, the Const
// first parameter; the target group stands in the last slot, and for Follow
// in the slot after the command (ALM-TRIG-045, TRIG-GROUPBOUNDS-136).
var groupCommandSignatures = map[uint32]slots{
	1:  {number("value")},
	2:  cell("cell"),
	3:  {},
	4:  cell("cell"),
	5:  cell("cell"),
	10: {unit("unit")},
	11: {unit("unit"), number("range")},
	14: cell("cell"),
	15: {unit("unit"), number("range")},
	17: {},
	18: {},
}

// ScriptGroupCommands are the commands action 6 can be authored with
// (TRIG-GROUP-005), in order.
var ScriptGroupCommands = []uint32{1, 2, 3, 4, 5, 10, 11, 14, 15, 17, 18}

// ScriptSignature is the ten parameter slots of operation opcode, and false
// when no claim states them. For the group command, command is its first
// parameter.
func ScriptSignature(condition bool, opcode, command uint32) ([10]ScriptSlot, bool) {
	var out [10]ScriptSlot
	var s slots
	var ok bool
	switch {
	case condition:
		s, ok = checkSignatures[opcode]
	case opcode == ScriptGroupCommand:
		s, ok = groupCommandSignatures[command]
		if ok && command == scriptFollowCommand {
			s = append(slots{{ScriptConst, "command"}, group}, s...)
		} else if ok {
			s = append(slots{{ScriptConst, "command"}}, s...)
			out[9] = group
		}
	default:
		s, ok = actionSignatures[opcode]
	}
	copy(out[:], s)
	return out, ok
}

// scriptUnnamed are the operations no claim names: their arm is located, but
// what it does to the game is Unknown (TRIG-CHECKCOUNT-132,
// TRIG-ACTIONWRITE-135), or no claim reads it.
var scriptUnnamed = map[bool]map[uint32]bool{
	true:  {20: true},
	false: {1: true, 7: true, 14: true, 15: true, 31: true, 0x10003: true},
}

// ScriptOperationNamed reports that a claim names what operation opcode does:
// a check of 1..22 or the constant, an action of 1..34 or the drop cell, and
// not one of the unnamed arms.
func ScriptOperationNamed(condition bool, opcode uint32) bool {
	limit := uint32(34)
	if condition {
		limit = 22
	}
	if opcode != ScriptConstant && (opcode < 1 || opcode > limit) {
		return false
	}
	return !scriptUnnamed[condition][opcode]
}

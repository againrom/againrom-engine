package scriptprofile

// The opcodes of Againrom's operations only the extended script record
// runs, named here alone: the operation rows, the editor and the
// simulation read these names.
const (
	ReadGlobalOp   = 0x4102 // DIV-5108
	SetGlobalOp    = 0x4103
	AddGlobalOp    = 0x4104
	UnitLeftCellOp = 0x4105 // DIV-5204
	TeleportCellOp = 0x4106 // DIV-5200
)

// messageOp is the original's message action. One that asks for an
// interactive dialogue runs only on the extended record.
const messageOp = 2

// OpID is one opcode of an operation, a check (Condition) or an instant.
type OpID struct {
	Condition bool
	ID        uint32
}

// ExtendedOp is one operation only the extended script record runs. Name
// keys the row; Ops are its opcodes. Interactive narrows the row to a
// message node that asks for an interactive dialogue. Held is how many of
// its nodes the original record holds. Word is the key of the refusal's
// words, which take the node count where Counted. Original is that
// Againrom's simulation runs it on a script of the original record too.
type ExtendedOp struct {
	Name        string
	Ops         []OpID
	Interactive bool
	Held        int
	Word        string
	Counted     bool
	Original    bool
}

// ExtendedOps is the one table of extended-record operations, in the order
// the editor's refusal names them. A new such operation is one row here.
var ExtendedOps = []ExtendedOp{
	{Name: "dialogue", Ops: []OpID{{false, messageOp}}, Interactive: true, Word: "editor.script.limit.dialogue", Counted: true},
	{Name: "leftcell", Ops: []OpID{{true, UnitLeftCellOp}}, Word: "editor.script.limit.leftcell"},
	{Name: "global", Ops: []OpID{{true, ReadGlobalOp}, {false, SetGlobalOp}, {false, AddGlobalOp}}, Word: "editor.script.limit.global", Counted: true, Original: true},
	{Name: "teleport", Ops: []OpID{{false, TeleportCellOp}}, Word: "editor.script.limit.teleport", Counted: true, Original: true},
}

// ExtendedOpAt is the index in ExtendedOps of the row of a node of the
// family with opcode id, -1 where the original record runs it.
// interactive is that the node asks for an interactive dialogue.
func ExtendedOpAt(condition bool, id uint32, interactive bool) int {
	for i, o := range ExtendedOps {
		if o.Interactive && !interactive {
			continue
		}
		for _, op := range o.Ops {
			if op == (OpID{condition, id}) {
				return i
			}
		}
	}
	return -1
}

// ExtendedRuns reports whether Againrom's simulation runs a node of the
// family with opcode id that the row of an extended-record operation
// names, on the extended machine where extended, else on the original.
// False for an opcode no row names.
func ExtendedRuns(condition bool, id uint32, extended bool) bool {
	i := ExtendedOpAt(condition, id, false)
	return i >= 0 && (extended || ExtendedOps[i].Original)
}

package ui

import "image"

// InspectionScriptModel is the map's script as the trigger editor reads it:
// its nodes and triggers with the document record each one is, the places
// the triggers name, and what each kind of node parameter can name.
type InspectionScriptModel struct {
	Conditions, Actions []InspectionScriptNode
	Triggers            []InspectionTrigger
	Areas               []InspectionArea
	// Targets are the values a parameter of a type code can take: units,
	// groups, players and structures the map holds.
	Targets map[uint32][]InspectionScriptTarget
	// Catalog is what the host's decoder knows of the operations.
	Catalog InspectionScriptCatalog
}

// InspectionScriptCatalog is the operations a node can run: the opcodes of
// each family with whether a claim names them, the group commands, and each
// operation's parameter signature where a claim states one.
type InspectionScriptCatalog struct {
	Checks, Actions []InspectionScriptOperation
	GroupCommands   []uint32
	Signature       func(condition bool, opcode, command uint32) ([10]InspectionScriptSlot, bool)
}

// InspectionScriptOperation is one opcode and whether a claim names it.
type InspectionScriptOperation struct {
	Opcode uint32
	Named  bool
}

// InspectionScriptSlot is one parameter slot of a signature: its type code
// and the role a claim gives it; a cell's Y slot has no role.
type InspectionScriptSlot struct {
	Type uint32
	Role string
}

// The parameter type codes of a node slot (ALM-TRIG-046).
const (
	scriptNone uint32 = iota
	scriptInt
	scriptGroup
	scriptPlayer
	scriptUnit
	scriptX
	scriptY
	scriptConst
	scriptItem
	scriptStructure
)

// scriptGroupCommand is the action whose first parameter is a group command.
const scriptGroupCommand = 6

// operations are the catalog's operations of a family.
func (c InspectionScriptCatalog) operations(condition bool) []InspectionScriptOperation {
	if condition {
		return c.Checks
	}
	return c.Actions
}

// named reports that a claim names the opcode.
func (c InspectionScriptCatalog) named(condition bool, opcode uint32) bool {
	for _, op := range c.operations(condition) {
		if op.Opcode == opcode {
			return op.Named
		}
	}
	return false
}

// signature is the operation's signature, false where none is stated.
func (c InspectionScriptCatalog) signature(condition bool, opcode, command uint32) ([10]InspectionScriptSlot, bool) {
	if c.Signature == nil {
		return [10]InspectionScriptSlot{}, false
	}
	return c.Signature(condition, opcode, command)
}

// InspectionScriptNode is one condition or action node.
type InspectionScriptNode struct {
	Record        int // its document record
	ID, Opcode    uint32
	Label         string // UTF-8, the text its name field edits
	Types, Values [10]uint32
}

// InspectionTrigger is one trigger: its condition pairs, comparisons,
// actions and once flag.
type InspectionTrigger struct {
	Record           int
	Name             string // UTF-8, the text its name field edits
	Left, Right, Cmp [3]uint32
	Acts             [4]uint32
	Once             uint32
}

// The shapes of an area.
const (
	AreaCell   = iota // one cell
	AreaBox           // a box between two corners
	AreaSquare        // the cells within a distance of a centre
)

// InspectionArea is one place a trigger names: the cells Rect covers (Max
// exclusive), drawn as Shape.
type InspectionArea struct {
	Trigger int // the index in Triggers
	Node    int // the document record of the node that names it
	Shape   int
	Rect    image.Rectangle
	Centre  image.Point
	Radius  int
	// Slots are the node's parameter slots the cells come from: a box's
	// X, Y, X2, Y2, a cell's or a square's centre X, Y in the first two;
	// -1 where none. An area with no slots has no handles.
	Slots [4]int
	// RadiusNode is the document record of the condition whose slot
	// RadiusSlot holds a square's radius plus RadiusBias; -1 for none.
	RadiusNode, RadiusSlot, RadiusBias int
}

// InspectionScriptTarget is one value a parameter can name, the document
// record it stands for (-1 none), the members it counts and its own name.
type InspectionScriptTarget struct {
	Value   uint32
	Record  int
	Members int
	Name    string
}

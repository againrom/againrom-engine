package ui

import "image"

// InspectionScriptModel is the map's script as the trigger editor reads it:
// its nodes and triggers with the document record each one is, the places
// the triggers name, and what each kind of node parameter can name.
type InspectionScriptModel struct {
	Conditions, Actions []InspectionScriptNode
	Triggers            []InspectionTrigger
	Areas               []InspectionArea
	// Targets are named values by host parameter kind.
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
	ParameterKind   func(condition bool, opcode uint32, types [10]uint32, slot int) uint32
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

// ScriptVariableParameter is a view kind, not an ALM type.
const ScriptVariableParameter uint32 = 10

const ScriptMessageParameter uint32 = 11

const NewMessageChoice int64 = 1 << 32

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
	Record            int // its document record
	ID, Opcode        uint32
	Label             string // UTF-8, the text its name field edits
	Types, Values     [10]uint32
	AddressesVariable bool
	VariableRegister  uint32
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
// record link (-1 none, <=-2 a variable row), member count and display name.
type InspectionScriptTarget struct {
	Value      uint32
	Record     int
	Members    int
	Name       string
	Choice     string
	NewMessage bool
	Uses       []InspectionScriptUse
}

func (t InspectionScriptTarget) ChoiceValue() int64 {
	if t.NewMessage {
		return NewMessageChoice
	}
	return int64(t.Value)
}

// InspectionScriptUse is one parameter and the triggers naming its node.
type InspectionScriptUse struct {
	Node, Slot int
	Triggers   []int
}

// OpenTarget admits exactly the host choices that parameter Open can follow.
func (m *InspectionScriptModel) OpenTarget(kind, value uint32) *InspectionScriptTarget {
	if m == nil || kind == scriptItem || kind == scriptUnit && value >= 10001 {
		return nil
	}
	for i := range m.Targets[kind] {
		t := &m.Targets[kind][i]
		if !t.NewMessage && t.Value == value && (t.Record >= 0 || kind == ScriptVariableParameter && t.Record <= -2 || kind == scriptGroup && value != 0 && t.Members > 0 || kind == ScriptMessageParameter) {
			return t
		}
	}
	return nil
}

func (t InspectionScriptTarget) ChoiceText() string {
	if t.Choice != "" {
		return t.Choice
	}
	return t.Name
}

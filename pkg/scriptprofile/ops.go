package scriptprofile

import (
	"fmt"
	"slices"
	"strconv"
)

// The one table of script operations: a row per (game profile, family, id)
// holding the operation's name, its editor word, its parameter signature and
// whether it addresses a variable register. The decoder's names, the editor's
// operation words, the catalog and the inspection's operation text all read
// these rows; a difference between the games is a row or a Vocabulary field.

// The parameter type codes of a node slot.
const (
	slotInt       = 1
	slotGroup     = 2
	slotPlayer    = 3
	slotUnit      = 4
	slotX         = 5
	slotY         = 6
	slotConst     = 7
	slotItem      = 8
	slotStructure = 9
)

// Slot is one parameter slot of a signature: its type code and the role word
// of what the operation reads it as. A cell is an X slot followed by a Y
// slot, and the X slot carries the role.
type Slot struct {
	Type uint32
	Role string
}

// Op is one operation row. Name is the decoder's text ("" where no claim
// names it), Word the editor's word key ("" where the editor shows the
// fallback of the game's Vocabulary). Decoded is that a claim reads what the
// operation does; Offered that the editor lists it, in table order. Again is
// an Againrom operation. Register is that the operation addresses a variable
// register in its first parameter. Signature is the parameter slots, nil
// where none is stated.
type Op struct {
	Ops       Ops
	Condition bool
	ID        uint32
	Name      string
	Word      string
	Decoded   bool
	Offered   bool
	Again     bool
	Register  bool
	Signature []Slot
	// CopyRefused is the word of the refusal of a trigger copy holding a
	// node of the operation, which takes the node's id; "" where it copies.
	CopyRefused string
	// CopyNamed is that a trigger copy names the operation's node itself
	// rather than a copy of it.
	CopyNamed bool
	// EffectUnread is that a claim reads the operation's slots but not what
	// it does: the trigger wizard offers no ready row of it.
	EffectUnread bool
	// Variants are the readings by the value of the first parameter. With
	// Grouped they are group commands: slot 0 is the command, a signature
	// holds the slots after it, and the target group takes the last slot
	// unless the signature places it. Otherwise a variant holds every slot.
	Variants        []Variant
	Grouped         bool
	Presets         []WizardPreset
	WizardSignature []Slot // editor-authored slots when the raw signature is unknown
}

// WizardPreset describes one reading of an operation in the trigger wizard.
// Fields selects a control by slot role. Compared names an editable constant,
// PeerWord a second node, and Fixed the values of trailing, uneditable slots.
type WizardPreset struct {
	Order                   int
	Word, Hint, SubjectWord string
	Command                 uint32
	GroupFirst, Variant     bool
	Fields                  map[string]string
	ExtraSlots              []Slot
	Fixed                   []uint32
	Cmp                     int
	ChooseCmp, Holds        bool
	Right                   int64
	RightType               uint32
	Also                    [][2]int64
	Compared, PeerWord      string
	DialogSource            string
}

// ReadyPresets returns the authored readings, or the ordinary reading of an
// offered, decoded operation. An explicit empty list omits an operation.
func (o Op) ReadyPresets() []WizardPreset {
	if o.Presets != nil || !o.Offered || !o.Decoded || o.Again || o.EffectUnread || o.ID >= constantOp {
		return o.Presets
	}
	if o.Condition {
		return []WizardPreset{{ChooseCmp: true, Compared: "value", Right: 1}}
	}
	return []WizardPreset{{}}
}

// Slots combines the operation's signature with any additional authored
// slots. Fixed values take their types from the end of that layout.
func (p WizardPreset) Slots(op Op, command uint32) ([10]Slot, [][2]uint32, bool) {
	if !op.HasSignature() {
		op.Signature = op.WizardSignature
	}
	slots, ok := op.Slots(command)
	end := len(slots)
	for end > 0 && slots[end-1].Type == 0 {
		end--
	}
	if !ok || end+len(p.ExtraSlots) > len(slots) {
		return slots, nil, false
	}
	end += copy(slots[end:], p.ExtraSlots)
	if len(p.Fixed) > end {
		return slots, nil, false
	}
	var fixed [][2]uint32
	for i, value := range p.Fixed {
		at := end - len(p.Fixed) + i
		fixed = append(fixed, [2]uint32{slots[at].Type, value})
		slots[at] = Slot{}
	}
	return slots, fixed, true
}

// Variant is the reading of a node whose first parameter holds Value: its
// name, word key ("" where the command panel names it) and slots. Follows maps
// a cell's X slot to the X slot of the cell it copies while they hold one cell.
type Variant struct {
	Value     uint32
	Name      string
	Word      string
	Signature []Slot
	Defaults  map[int]uint32
	Follows   map[int]int
}

// Variant is the variant for first parameter value, false where none is.
func (o Op) Variant(value uint32) (Variant, bool) {
	for _, v := range o.Variants {
		if v.Value == value {
			return v, true
		}
	}
	return Variant{}, false
}

// Values are the variants' first-parameter values, in order.
func (o Op) Values() []uint32 {
	var out []uint32
	for _, v := range o.Variants {
		out = append(out, v.Value)
	}
	return out
}

// HasSignature reports that a claim states the operation's parameter slots.
func (o Op) HasSignature() bool { return o.Signature != nil }

// Slots are the ten parameter slots of the operation, false where no claim
// states them; value is the first parameter of an operation with variants.
func (o Op) Slots(value uint32) ([10]Slot, bool) {
	var out [10]Slot
	s, ok := o.Signature, o.HasSignature()
	v, found := o.Variant(value)
	if o.Grouped {
		s, ok = v.Signature, found
		if ok && !slices.ContainsFunc(s, func(x Slot) bool { return x.Type == slotGroup }) {
			out[9] = group[0]
		}
		s = append([]Slot{{slotConst, "command"}}, s...)
	} else if found {
		s = v.Signature
	}
	if ok {
		copy(out[:], s)
	}
	return out, ok
}

// Vocabulary is what differs between the games' operation vocabularies
// beyond the rows: the fallback word of an operation with no word, whether
// the inspection names an undecoded operation by its number, whether its
// parameters carry the first game's message, spell, property and variable
// kinds, and whether an undecoded operation's fields stay read-only.
type Vocabulary struct {
	CheckWord, ActionWord string
	Naming                NameRule
	Kinds                 KindRule
	Fields                FieldRule
	// ObjectType is the parameter type whose value is the identity of a
	// type-4 object of the same map, 0 where no type is.
	ObjectType uint32
	// CellMove is the cell event that moves a unit reaching the cell's
	// centre to the event's target cell, 0 where no event does.
	CellMove uint32
	// Unknown, Region and Edges are the inspection's notes (InspectRule).
	// Unknown: an operation with no word is an issue.
	Unknown InspectRule
	// Region: a distance condition's signature bounds a Chebyshev square.
	Region InspectRule
	// Edges: a box condition's edge rules are noted as unknown.
	Edges InspectRule
}

// InspectRule is whether the inspection makes one of its notes.
type InspectRule uint8

const (
	// NoNote makes no note.
	NoNote InspectRule = iota
	// Noted makes the note.
	Noted
)

// CellMoveEvent is the second game's cell event 23, a move to the target
// cell, whether action 25 or a type-9 record writes it (R2-ENGINE-1776,
// R2-ENGINE-1778).
const CellMoveEvent = 23

// NameRule is how an operation's name is written where no claim reads it.
type NameRule uint8

const (
	// NameWord writes the name a claim reads, or the raw id where none does.
	NameWord NameRule = iota
	// NumberedUndecoded writes an undecoded operation by its number.
	NumberedUndecoded
)

// KindRule is whether parameters carry the first game's message, spell,
// property and variable kinds.
type KindRule uint8

const (
	// NoKinds types a parameter by the type its node stores.
	NoKinds KindRule = iota
	// FirstKinds types a parameter by the first game's kinds.
	FirstKinds
)

// FieldRule is whether an undecoded operation's fields stay read-only.
type FieldRule uint8

const (
	// FieldsEditable leaves every field of a node editable.
	FieldsEditable FieldRule = iota
	// UndecodedReadOnly keeps the fields of an undecoded operation read-only.
	UndecodedReadOnly
)

// Undecoded is the fallback word key of a condition or an action.
func (v Vocabulary) Undecoded(condition bool) string {
	if condition {
		return v.CheckWord
	}
	return v.ActionWord
}

var vocabularies = [...]Vocabulary{
	OpsFirst: {CheckWord: "editor.check.unnamed", ActionWord: "editor.action.unnamed", Kinds: FirstKinds,
		Unknown: Noted, Region: Noted, Edges: Noted},
	// A type-9 value is a type-4 id of the map (R2-ENGINE-1725). The loader
	// tests no node or trigger word and the compiler reads a node's word
	// +0x48 nowhere found, so a new node holds zero there and in its texts
	// (R2-ENGINE-1728).
	OpsSecond: {CheckWord: "editor.op.undecoded", ActionWord: "editor.op.undecoded", Naming: NumberedUndecoded, Fields: UndecodedReadOnly, ObjectType: 9,
		CellMove: CellMoveEvent},
}

// VocabularyOf is the vocabulary of a game's operations.
func VocabularyOf(ops Ops) Vocabulary { return vocabularies[ops] }

type opKey struct {
	ops       Ops
	condition bool
	id        uint32
}

var (
	opIndex = map[opKey]Op{}
	opLists = map[opKey][]Op{} // by (ops, condition), id 0
)

func init() {
	for _, group := range [][]Op{firstChecks, firstActions, secondChecks, secondActions} {
		for _, o := range group {
			opIndex[opKey{o.Ops, o.Condition, o.ID}] = o
			k := opKey{o.Ops, o.Condition, 0}
			opLists[k] = append(opLists[k], o)
		}
	}
}

// OpRow is the row of an operation, false where the game has none.
func OpRow(ops Ops, condition bool, id uint32) (Op, bool) {
	o, ok := opIndex[opKey{ops, condition, id}]
	return o, ok
}

// OpRows are every row of a game's family, in table order.
func OpRows(ops Ops, condition bool) []Op { return opLists[opKey{ops, condition, 0}] }

// AddressesRegister reports that operation id of the family addresses a
// variable register; both games' rows name the same ids.
func AddressesRegister(condition bool, id uint32) bool {
	for _, ops := range []Ops{OpsFirst, OpsSecond} {
		if o, ok := OpRow(ops, condition, id); ok && o.Register {
			return true
		}
	}
	return false
}

// globalRole is the role of the slot that names a mod global.
const globalRole = "global"

// AddressesGlobal reports that first-game operation id of the family names
// a mod global in its first parameter (DIV-5108).
func AddressesGlobal(condition bool, id uint32) bool {
	o, ok := OpRow(OpsFirst, condition, id)
	return ok && len(o.Signature) > 0 && o.Signature[0].Role == globalRole
}

func unit(role string) []Slot   { return []Slot{{slotUnit, role}} }
func number(role string) []Slot { return []Slot{{slotInt, role}} }
func cell(role string) []Slot   { return []Slot{{slotX, role}, {slotY, ""}} }
func player(role string) []Slot { return []Slot{{slotPlayer, role}} }

var (
	group     = []Slot{{slotGroup, "group"}}
	item      = []Slot{{slotItem, "item"}}
	structure = []Slot{{slotStructure, "structure"}}
)

// sig joins slot lists into one non-nil signature.
func sig(parts ...[]Slot) []Slot {
	out := []Slot{}
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// The Againrom operations' opcodes and the build-time constant operation.
const (
	constantOp  = 0x10002
	formOp      = 0x10003
	healthBelow = 0x4101
	undying     = 0x4100
	readGlobal  = 0x4102
	setGlobal   = 0x4103
	addGlobal   = 0x4104
)

// The global operations, as the wizard lists them.
const (
	ReadGlobalOp = readGlobal
	SetGlobalOp  = setGlobal
	AddGlobalOp  = addGlobal
)

// firstCommandVariants and secondCommandVariants are each game's action 6
// commands (ALM-TRIG-045, TRIG-GROUPBOUNDS-136; the second game's vectors are
// R2-ASSET-363's, its operands R2-ENGINE-1726's, selector 2's cell Medium).
// Follow (15) places the target group right after the command.
var firstCommandVariants = []Variant{
	{Value: 1, Signature: number("value")}, {Value: 2, Signature: cell("cell")}, {Value: 3, Signature: sig()},
	{Value: 4, Signature: cell("cell")}, {Value: 5, Signature: cell("cell")}, {Value: 10, Signature: unit("unit")},
	{Value: 11, Signature: sig(unit("unit"), number("range"))}, {Value: 14, Signature: cell("cell")},
	{Value: 15, Signature: sig(group, unit("unit"), number("range"))}, {Value: 17, Signature: sig()}, {Value: 18, Signature: sig()},
}

var secondCommandVariants = []Variant{
	{Value: 1, Signature: number("value")}, {Value: 2, Signature: cell("cell")}, {Value: 3, Signature: sig()},
	{Value: 4, Signature: cell("cell")}, {Value: 5, Signature: cell("cell")}, {Value: 10, Signature: unit("unit")},
	{Value: 11, Signature: sig(unit("unit"), number("value"))}, {Value: 14, Signature: cell("cell")},
	{Value: 15, Signature: sig(group, unit("unit"), number("value"))}, {Value: 17, Signature: sig()}, {Value: 18, Signature: number("value")},
}

// worded is variants with the word key format gives each value.
func worded(variants []Variant, format string) []Variant {
	out := slices.Clone(variants)
	for i := range out {
		out[i].Word = fmt.Sprintf(format, out[i].Value)
	}
	return out
}

// The first game's rows. An operation with no row is read by its raw number.
var firstChecks = []Op{
	{Condition: true, ID: 1, Name: "Group count", Word: "editor.check.op1", Decoded: true, Offered: true, Signature: sig(group), // TRIG-COUNT-015
		Presets: []WizardPreset{{Order: 6, Word: "editor.wizard.c.groupcount", ChooseCmp: true, Compared: "count"}}},
	{Condition: true, ID: 2, Name: "Unit in box", Word: "editor.check.op2", Decoded: true, Offered: true, Signature: sig(unit("unit"), cell("corner"), cell("corner2")), // TRIG-CHECKGEOM-130
		Presets: []WizardPreset{{Order: 1, Word: "editor.wizard.c.heroarea", Right: 1, Also: [][2]int64{{1, 0}}}}},
	{Condition: true, ID: 3, Name: "Unit in distance square", Word: "editor.check.op3", Decoded: true, Offered: true, Signature: sig(unit("unit"), cell("cell"), number("radius")), // TRIG-DIST-014
		Presets: []WizardPreset{{Order: 3, Word: "editor.wizard.c.unitnear", Right: 1, Fields: map[string]string{"radius": "radius"}}}},
	{Condition: true, ID: 4, Name: "Unit field (selector 6)", Word: "editor.check.op4", Decoded: true, Offered: true, Signature: sig(unit("unit"), number("selector")), // TRIG-CHECK-051
		Presets: []WizardPreset{{Order: 12, Word: "editor.wizard.c.health", Fixed: []uint32{6}, Cmp: 2, ChooseCmp: true, Compared: "health"}}},
	{Condition: true, ID: 5, Name: "Unit alive", Word: "editor.check.op5", Decoded: true, Offered: true, Signature: sig(unit("unit")), // TRIG-CHECKGEOM-130
		Presets: []WizardPreset{
			{Order: 5, Word: "editor.wizard.c.unitdead", Also: [][2]int64{{1, 1}}},
			{Order: 10, Word: "editor.wizard.c.unitalive", Cmp: 1, Also: [][2]int64{{0, 1}}},
		}},
	{Condition: true, ID: 6, Name: "Distance between units", Word: "editor.check.op6", Decoded: true, Offered: true, Signature: sig(unit("unit"), unit("unit2")), // TRIG-DIST-014
		Presets: []WizardPreset{{Order: 4, Word: "editor.wizard.c.unitnearunit", Hint: "editor.wizard.c.unitnearunit.hint", Cmp: 5, ChooseCmp: true, Compared: "radius", Right: 3}}},
	{Condition: true, ID: 7, Name: "Point to unit distance", Word: "editor.check.op7", Decoded: true, Offered: true, Signature: sig(unit("unit"), cell("cell")), // TRIG-DIST-014
		Presets: []WizardPreset{{Order: 11, Word: "editor.wizard.c.unitcell", Cmp: 5, ChooseCmp: true, Compared: "radius", Right: 3}}},
	{Condition: true, ID: 8, Name: "Player population", Word: "editor.check.op8", Decoded: true, Offered: true, Signature: sig(player("player")), // TRIG-LISTREAD-131
		Presets: []WizardPreset{{Order: 7, Word: "editor.wizard.c.playercount", ChooseCmp: true, Compared: "count"}}},
	{Condition: true, ID: 9, Name: "Unit target ID", Word: "editor.check.op9", Decoded: true, Offered: true, Signature: sig(unit("unit")), // TRIG-TARGETID-032
		Presets: []WizardPreset{{Order: 13, Word: "editor.wizard.c.target", Hint: "editor.wizard.c.target.hint", ChooseCmp: true, Compared: "targetunit", Right: 1}}},
	{Condition: true, ID: 10, Name: "Player relation", Word: "editor.check.op10", Decoded: true, Offered: true, Signature: sig(player("player"), player("player2")), // TRIG-DIPLO-020
		Presets: []WizardPreset{{Order: 8, Word: "editor.wizard.c.relation", Hint: "editor.wizard.c.relation.hint", ChooseCmp: true, Compared: "relation"}}},
	{Condition: true, ID: 11, Name: "No-write arm", Word: "editor.check.op11", Decoded: true, Offered: true, Signature: sig(), // TRIG-COND-003, a dead arm
		Presets: []WizardPreset{}},
	{Condition: true, ID: 12, Name: "Item presence", Word: "editor.check.op12", Decoded: true, Offered: true, Signature: sig(unit("unit"), item), // TRIG-ITEMTEST-040
		Presets: []WizardPreset{}},
	{Condition: true, ID: 13, Name: "No-write arm", Word: "editor.check.op13", Decoded: true, Offered: true, Signature: sig(), // TRIG-COND-003, a dead arm
		Presets: []WizardPreset{}},
	{Condition: true, ID: 14, Name: "Sack at cell", Word: "editor.check.op14", Decoded: true, Offered: true, Signature: sig(cell("cell")), // TRIG-SACK-022
		Presets: []WizardPreset{
			{Order: 16, Word: "editor.wizard.c.sackgone", RightType: slotConst, Also: [][2]int64{{1, 1}}},
			{Order: 17, Word: "editor.wizard.c.sacklies", Right: 1, RightType: slotConst, Also: [][2]int64{{1, 0}}},
		}},
	{Condition: true, ID: 15, Name: "Nearest player unit distance", Word: "editor.check.op15", Decoded: true, Offered: true, Signature: sig(player("player"), cell("cell")), // TRIG-NEAREST-029
		Presets: []WizardPreset{{Order: 2, Word: "editor.wizard.c.playernear", Cmp: 5, ChooseCmp: true, Compared: "radius", Right: 3}}},
	{Condition: true, ID: 16, Name: "Item to point distance", Word: "editor.check.op16", Decoded: true, Offered: true, Signature: sig(unit("unit"), item, cell("cell")), // TRIG-CHECK-052
		Presets: []WizardPreset{{Order: 20, Word: "editor.wizard.c.itemnear", Hint: "editor.wizard.c.itemnear.hint", Cmp: 5, ChooseCmp: true, Compared: "radius", Right: 3}}},
	{Condition: true, ID: 17, Name: "Item presence", Word: "editor.check.op17", Decoded: true, Offered: true, Signature: sig(unit("unit"), item), // TRIG-ITEMTEST-040
		Presets: []WizardPreset{
			{Order: 18, Word: "editor.wizard.c.hasitem", Right: 1, RightType: slotConst, Also: [][2]int64{{1, 0}}},
			{Order: 19, Word: "editor.wizard.c.lacksitem", RightType: slotConst, Also: [][2]int64{{1, 1}}},
		}},
	{Condition: true, ID: 18, Name: "Lose if unit dies", Word: "editor.check.op18", Decoded: true, Offered: true, Signature: sig(unit("unit")), // MISSION-VIP-004
		Presets: []WizardPreset{
			{Order: 14, Word: "editor.wizard.c.lose", SubjectWord: "editor.wizard.s.mission", Hint: "editor.wizard.c.lose.hint", Cmp: 2, ChooseCmp: true, Compared: "value", Right: 1},
			{Order: 15, Word: "editor.wizard.c.losetwo", SubjectWord: "editor.wizard.s.mission", Hint: "editor.wizard.c.lose.hint", Cmp: 1, ChooseCmp: true, PeerWord: "editor.wizard.p.unit2"},
		}},
	{Condition: true, ID: 19, Name: "Read variable", Word: "editor.check.op19", Decoded: true, Offered: true, Register: true, Signature: sig(number("variable")), // TRIG-COND-003
		Presets: []WizardPreset{
			{Order: 22, Word: "editor.wizard.c.variable", Fields: map[string]string{"variable": "variable"}, ChooseCmp: true, Compared: "value"},
			{Order: 23, Word: "editor.wizard.c.variables", Fields: map[string]string{"variable": "variable"}, ChooseCmp: true, PeerWord: "editor.wizard.p.variable2"},
		}},
	{Condition: true, ID: 20, Signature: sig(player("player"))}, // TRIG-CHECKCOUNT-132
	{Condition: true, ID: 21, Name: "Structure field", Word: "editor.check.op21", Decoded: true, Offered: true, Signature: sig(structure), // TRIG-CHECK-053
		Presets: []WizardPreset{{Order: 21, Word: "editor.wizard.c.structure", Cmp: 3, ChooseCmp: true, Compared: "value", Right: 1}}},
	{Condition: true, ID: 22, Name: "Move object check", Word: "editor.check.op22", Decoded: true, Signature: sig(cell("cell"), cell("cell2"))}, // TRIG-CELLRESULT-133
	{Condition: true, ID: constantOp, Name: "Constant / variable preset", Word: "editor.check.op65538", Decoded: true, Offered: true, Signature: sig(number("value")), // TRIG-COND-003
		Presets: []WizardPreset{{Order: 9, Word: "editor.wizard.c.always", Right: 1, Fixed: []uint32{1}, Holds: true}}},
	{Condition: true, ID: healthBelow, Name: "Health below (Againrom)", Word: "editor.check.op16641", Decoded: true, Offered: true, Again: true, Signature: sig(unit("unit"), number("health"))}, // DIV-4009
	{Condition: true, ID: readGlobal, Name: "Read global (Againrom)", Word: "editor.check.op16642", Decoded: true, Again: true, Signature: sig(number("global")), // DIV-5108
		Presets: []WizardPreset{{Order: 24, Word: "editor.wizard.c.global", Fields: map[string]string{"global": "global"}, ChooseCmp: true, Compared: "count"}}},
}

var firstActions = []Op{
	{ID: 1, Signature: sig()}, // TRIG-ACTIONWRITE-135
	{ID: 2, Name: "Send message", Word: "editor.action.op2", Decoded: true, Offered: true, Signature: sig(number("message")), // TRIG-MSG-023
		Presets: []WizardPreset{
			{Order: 4, Word: "editor.wizard.a.dialog", DialogSource: "message", Fields: map[string]string{"message": "message"}},
			{Order: 34, Word: "editor.wizard.a.dialog", DialogSource: "file", Variant: true, ExtraSlots: number("source"), Fixed: []uint32{0, 1}},
		}},
	{ID: 3, Name: "Set variable", Word: "editor.action.op3", Decoded: true, Offered: true, Register: true, Signature: sig(number("variable"), number("value")), // TRIG-ACT-004
		Presets: []WizardPreset{{Order: 16, Word: "editor.wizard.a.setvar", Fields: map[string]string{"variable": "variable", "value": "value"}}}},
	{ID: 4, Name: "Win", Word: "editor.action.op4", Decoded: true, Offered: true, Signature: sig(), // TRIG-ACT-004
		Presets: []WizardPreset{{Order: 2, Word: "editor.wizard.a.win"}}},
	{ID: 5, Name: "Lose", Word: "editor.action.op5", Decoded: true, Offered: true, Signature: sig(), // TRIG-ACT-004
		Presets: []WizardPreset{{Order: 3, Word: "editor.wizard.a.lose"}}},
	{ID: 6, Name: "Group command", Word: "editor.action.op6", Decoded: true, Offered: true, Grouped: true, Variants: firstCommandVariants,
		Presets: []WizardPreset{
			{Order: 13, Word: "editor.wizard.a.order", Command: 4, GroupFirst: true},
			{Order: 30, Word: "editor.wizard.a.order.stand", Command: 3, GroupFirst: true, Variant: true},
			{Order: 31, Word: "editor.wizard.a.order.attack", Command: 10, GroupFirst: true, Variant: true},
			{Order: 32, Word: "editor.wizard.a.order.defend", Command: 11, GroupFirst: true, Variant: true, Fields: map[string]string{"range": "range"}},
			{Order: 33, Word: "editor.wizard.a.order.follow", Command: 15, GroupFirst: true, Variant: true, Fields: map[string]string{"range": "range"}},
		}},
	{ID: 7, Name: "Set formation", Offered: true, WizardSignature: sig(player("player"), number("value")),
		Presets: []WizardPreset{{Order: 29, Word: "editor.wizard.a.action7", Fields: map[string]string{"value": "value"}}}},
	{ID: 8, Name: "Increment variable", Word: "editor.action.op8", Decoded: true, Offered: true, Register: true, Signature: sig(number("variable")), // TRIG-ACT-004
		Presets: []WizardPreset{{Order: 15, Word: "editor.wizard.a.addvar", Fields: map[string]string{"variable": "variable"}}}},
	{ID: 9, Name: "No-op arm", Word: "editor.action.op9", Decoded: true, Signature: sig()}, // TRIG-ACT-004, a dead arm
	{ID: 10, Name: "Set relation", Word: "editor.action.op10", Decoded: true, Offered: true, Signature: sig(number("player"), number("player2"), number("value")), // TRIG-DIPLO-019
		Presets: []WizardPreset{{Order: 14, Hint: "editor.wizard.a.relation.hint", Fields: map[string]string{"player": "player", "player2": "player2", "value": "relation"}}}},
	{ID: 11, Word: "editor.action.op11", Decoded: true, Signature: sig(unit("unit"), unit("unit2"), item)}, // TRIG-XFERITEM-039
	{ID: 12, Name: "Create item", Word: "editor.action.op12", Decoded: true, Offered: true, Signature: sig(unit("unit"), item), // TRIG-ADDITEM-027
		Presets: []WizardPreset{{Order: 20, Word: "editor.wizard.a.giveitem"}}},
	{ID: 13, Name: "Take item", Word: "editor.action.op13", Decoded: true, Offered: true, Signature: sig(unit("unit"), item), // TRIG-TAKEITEM-038
		Presets: []WizardPreset{{Order: 19, Word: "editor.wizard.a.takeitem"}}},
	{ID: 16, Name: "Take unit off map", Word: "editor.action.op16", Decoded: true, Offered: true, Signature: sig(unit("unit")), // TRIG-OFFMAP-041
		Presets: []WizardPreset{{Order: 8, Word: "editor.wizard.a.unitoff"}}},
	{ID: 17, Name: "Return unit to map", Word: "editor.action.op17", Decoded: true, Offered: true, Signature: sig(unit("unit")), // TRIG-RETURN-042
		Presets: []WizardPreset{{Order: 9, Word: "editor.wizard.a.unitback"}}},
	{ID: 18, Name: "Replace unit on map", Word: "editor.action.op18", Decoded: true, Offered: true, Signature: sig(unit("unit"), unit("unit2")), // TRIG-MAPGROUP-043
		Presets: []WizardPreset{{Order: 22, Word: "editor.wizard.a.replace", Hint: "editor.wizard.a.replace.hint"}}},
	{ID: 19, Name: "Give unit to player", Word: "editor.action.op19", Decoded: true, Offered: true, Signature: sig(unit("unit"), player("player")), // PARTY-JOIN-025
		Presets: []WizardPreset{{Order: 5, Word: "editor.wizard.a.giveunit"}}},
	{ID: 20, Name: "Drop inventory", Word: "editor.action.op20", Decoded: true, Offered: true, Signature: sig(unit("unit")), // TRIG-DROPALL-024
		Presets: []WizardPreset{}},
	{ID: 21, Name: "Cast at cell", Word: "editor.action.op21", Decoded: true, Offered: true, Signature: sig(cell("cell"), cell("cell2"), number("spell"), number("power")), // TRIG-CAST-033
		Presets: []WizardPreset{{Order: 24, Word: "editor.wizard.a.castcell", Hint: "editor.wizard.a.cast.hint", Fields: map[string]string{"cell": "from", "spell": "spell", "power": "power"}}}},
	{ID: 22, Name: "Give group to player", Word: "editor.action.op22", Decoded: true, Offered: true, Signature: sig(group, player("player")), // PARTY-JOIN-025
		Presets: []WizardPreset{{Order: 6, Word: "editor.wizard.a.givegroup"}}},
	{ID: 23, Name: "Give money to player", Word: "editor.action.op23", Decoded: true, Offered: true, Signature: sig(player("player"), number("money")), // TRIG-MONEY-028
		Presets: []WizardPreset{{Order: 1, Fields: map[string]string{"money": "gold"}}}},
	{ID: 24, Name: "Cast at unit", Word: "editor.action.op24", Decoded: true, Offered: true, Signature: sig(cell("cell"), unit("unit"), number("spell"), number("power")), // TRIG-CAST-033
		Presets: []WizardPreset{{Order: 25, Word: "editor.wizard.a.castunit", Fields: map[string]string{"cell": "from", "spell": "spell", "power": "power"}}}},
	{ID: 25, Name: "Set cell tail", Word: "editor.action.op25", Decoded: true, Offered: true, Signature: sig(number("spell"), number("power"), cell("cell")), // TRIG-CELLTAIL-035
		Presets: []WizardPreset{{Order: 26, Word: "editor.wizard.a.trap", Fields: map[string]string{"spell": "spell", "power": "power"}}}},
	{ID: 26, Name: "Set structure field", Word: "editor.action.op26", Decoded: true, Offered: true, Signature: sig(structure, number("value")), // TRIG-CHECK-053
		Presets: []WizardPreset{}},
	{ID: 27, Name: "Move unit", Word: "editor.action.op27", Decoded: true, Signature: sig(unit("unit"), cell("cell")), // AI-SCRIPTGUARD-625
		Presets: []WizardPreset{{Order: 7, Word: "editor.wizard.a.move"}}},
	{ID: 28, Name: "Give entire inventory", Word: "editor.action.op28", Decoded: true, Offered: true, Signature: sig(unit("unit"), unit("unit2")), // TRIG-GIVEALL-025
		Presets: []WizardPreset{{Order: 21, Word: "editor.wizard.a.giveall", SubjectWord: "editor.wizard.s.items", Fields: map[string]string{"unit2": "receiver"}}}},
	{ID: 29, Name: "Set cell effect duration", Word: "editor.action.op29", Decoded: true, Offered: true, Signature: sig(cell("cell"), number("spell"), number("time")), // TRIG-EFFECTTIME-034
		Presets: []WizardPreset{{Order: 27, Word: "editor.wizard.a.celltime", Fields: map[string]string{"spell": "spell", "time": "time"}}}},
	{ID: 30, Name: "Set unit effect duration", Word: "editor.action.op30", Decoded: true, Offered: true, Signature: sig(unit("unit"), number("spell"), number("time")), // TRIG-EFFECTTIME-034
		Presets: []WizardPreset{{Order: 28, Word: "editor.wizard.a.unittime", Fields: map[string]string{"spell": "spell", "time": "time"}}}},
	{ID: 32, Name: "Take group off map", Word: "editor.action.op32", Decoded: true, Offered: true, Signature: sig(group), // TRIG-MAPGROUP-043
		Presets: []WizardPreset{{Order: 10, Word: "editor.wizard.a.groupoff"}}},
	{ID: 33, Name: "Return group to map", Word: "editor.action.op33", Decoded: true, Offered: true, Signature: sig(group), // TRIG-MAPGROUP-043
		Presets: []WizardPreset{{Order: 11, Word: "editor.wizard.a.groupback"}}},
	{ID: 34, Name: "Set unit field", Word: "editor.action.op34", Decoded: true, Offered: true, Signature: sig(unit("unit"), number("selector"), number("value")), // TRIG-PROPERTY-036
		Presets: []WizardPreset{{Order: 23, Word: "editor.wizard.a.property", Fields: map[string]string{"selector": "property", "value": "value"}}}},
	{ID: constantOp, Name: "Drop location (build time)", Word: "editor.action.op65538", Decoded: true, Offered: true, Signature: sig(cell("cell")), CopyRefused: "editor.trigger.copydropzone", // TRIG-DROP-013
		Presets: []WizardPreset{{Order: 12, Word: "editor.wizard.a.drop", SubjectWord: "editor.wizard.s.mission"}}},
	{ID: formOp, Name: "Object form (build time)", Offered: true},
	{ID: undying, Name: "Set undying (Againrom)", Word: "editor.action.op16640", Decoded: true, Offered: true, Again: true, Signature: sig(unit("unit"), number("undying"))}, // DIV-4009
	{ID: setGlobal, Name: "Set global (Againrom)", Word: "editor.action.op16643", Decoded: true, Again: true, Signature: sig(number("global"), number("value")), // DIV-5108
		Presets: []WizardPreset{{Order: 17, Word: "editor.wizard.a.setglobal", Fields: map[string]string{"global": "global", "value": "value"}}}},
	{ID: addGlobal, Name: "Add to global (Againrom)", Word: "editor.action.op16644", Decoded: true, Again: true, Signature: sig(number("global"), number("delta")), // DIV-5108
		Presets: []WizardPreset{{Order: 18, Word: "editor.wizard.a.addglobal", Fields: map[string]string{"global": "global", "delta": "delta"}}}},
}

// The second game's rows: the operations a claim reads the body of
// (R2-ENGINE-1313 to R2-ENGINE-1315, R2-ENGINE-1450), each named by what the
// body reads and by the final effects R2-ENGINE-1740 to R2-ENGINE-1749 read,
// with the type vector every installed node of the id stores (R2-ASSET-363).
// Action 6's vector follows its selector (R2-ENGINE-1726); check 65538
// stores two and carries none. An id absent here is shown by its raw number.
var secondChecks = []Op{
	{Ops: OpsSecond, Condition: true, ID: 1, Name: "Group member count", Word: "editor.second.check.1", Decoded: true, Offered: true, Signature: sig(group)},
	// No installed node holds check 2; its body tests U's cell against the
	// four corner scalars (R2-ENGINE-1730).
	{Ops: OpsSecond, Condition: true, ID: 2, Name: "Unit in rectangle", Word: "editor.second.check.2", Decoded: true, Offered: true, Signature: sig(unit("unit"), cell("corner"), cell("corner2"))},
	{Ops: OpsSecond, Condition: true, ID: 4, Name: "Unit field by selector", Word: "editor.second.check.4", Decoded: true, Offered: true, Signature: sig(unit("unit"), number(""))},
	{Ops: OpsSecond, Condition: true, ID: 5, Name: "Unit state is not 16", Word: "editor.second.check.5", Decoded: true, Offered: true, Signature: sig(unit("unit"))},
	{Ops: OpsSecond, Condition: true, ID: 6, Name: "Location metric between units", Word: "editor.second.check.6", Decoded: true, Offered: true, Signature: sig(unit("unit"), unit("unit2"))},
	{Ops: OpsSecond, Condition: true, ID: 7, Name: "Location metric from unit to cell", Word: "editor.second.check.7", Decoded: true, Offered: true, Signature: sig(unit("unit"), cell("cell"))},
	{Ops: OpsSecond, Condition: true, ID: 8, Name: "Member count of a player's groups", Word: "editor.second.check.8", Decoded: true, Offered: true, Signature: sig(player("player"))},
	{Ops: OpsSecond, Condition: true, ID: 9, Name: "Unit's kind-5 record word", Word: "editor.second.check.9", Decoded: true, Offered: true, Signature: sig(unit("unit"))},
	{Ops: OpsSecond, Condition: true, ID: 10, Name: "Relation between players", Word: "editor.second.check.10", Decoded: true, Offered: true, Signature: sig(player("player"), player("player2"))},
	{Ops: OpsSecond, Condition: true, ID: 14, Name: "Terrain test at cell", Word: "editor.second.check.14", Decoded: true, Offered: true, Signature: sig(cell("cell"))},
	{Ops: OpsSecond, Condition: true, ID: 15, Name: "Nearest metric from a player's units to cell", Word: "editor.second.check.15", Decoded: true, Offered: true, Signature: sig(player("player"), cell("cell"))},
	{Ops: OpsSecond, Condition: true, ID: 16, Name: "Metric from a unit's item to cell", Word: secondWord(true, 16), Decoded: true, Offered: true, Signature: sig(unit("unit"), item, cell("cell"))},
	{Ops: OpsSecond, Condition: true, ID: 17, Name: "Unit holds item", Word: "editor.second.check.17", Decoded: true, Offered: true, Signature: sig(unit("unit"), item)},
	{Ops: OpsSecond, Condition: true, ID: 18, Name: "Failure counter while unit is in state 16", Word: "editor.second.check.18", Decoded: true, Offered: true, Signature: sig(unit("unit"))},
	{Ops: OpsSecond, Condition: true, ID: 19, Name: "Map register", Word: "editor.second.check.19", Decoded: true, Offered: true, Register: true, Signature: sig(number("variable"))},
	{Ops: OpsSecond, Condition: true, ID: 21, Name: "Word 0x42 of the object with a type-4 id", Word: "editor.second.check.21", Decoded: true, Offered: true, Signature: []Slot{{slotStructure, ""}}}, // R2-ENGINE-1725
	{Ops: OpsSecond, Condition: true, ID: 23, Name: "Scenario variable", Word: "editor.second.check.23", Decoded: true, Offered: true, Signature: sig(number(""))},
	{Ops: OpsSecond, Condition: true, ID: 24, Name: "Scenario variable 752+n", Word: "editor.second.check.24", Decoded: true, Offered: true, Signature: sig(number(""))},
	{Ops: OpsSecond, Condition: true, ID: 25, Name: "Cell effect layer present", Word: "editor.second.check.25", Decoded: true, Offered: true, Signature: sig(cell("cell"), number(""))},
	{Ops: OpsSecond, Condition: true, ID: 26, Name: "Unit effect active", Word: "editor.second.check.26", Decoded: true, Offered: true, Signature: sig(unit("unit"), number(""))},
	{Ops: OpsSecond, Condition: true, ID: 27, Name: "Unit tile position", Word: "editor.second.check.27", Decoded: true, Offered: true},
	{Ops: OpsSecond, Condition: true, ID: constantOp, Name: "Compiler parameter", Word: "editor.second.check.65538", Decoded: true, Offered: true},
}

// secondWord is the word key of a second-game operation. Rows whose literal
// key would end in ".16" use it: the art guard reads such a literal as a
// picture name.
func secondWord(condition bool, id uint32) string {
	if condition {
		return "editor.second.check." + strconv.Itoa(int(id))
	}
	return "editor.second.action." + strconv.Itoa(int(id))
}

var secondActions = []Op{
	{Ops: OpsSecond, ID: 2, Name: "Send event", Word: "editor.second.action.2", Decoded: true, Offered: true, Signature: sig(number(""))},
	{Ops: OpsSecond, ID: 3, Name: "Set map register", Word: "editor.second.action.3", Decoded: true, Offered: true, Register: true, Signature: sig(number("variable"), number("value"))},
	{Ops: OpsSecond, ID: 4, Name: "Win: add to win counter", Word: "editor.second.action.4", Decoded: true, Offered: true, Signature: sig()},                    // R2-ENGINE-1729
	{Ops: OpsSecond, ID: 5, Name: "Lose: store failure reason", Word: "editor.second.action.5", Decoded: true, Offered: true, Signature: sig(number("reason"))}, // R2-ENGINE-1729
	{Ops: OpsSecond, ID: 6, Name: "Group arm by selector", Word: "editor.second.action.6", Decoded: true, Offered: true, Grouped: true,
		Variants: worded(secondCommandVariants, "editor.second.command.%d")}, // R2-ENGINE-1726
	{Ops: OpsSecond, ID: 8, Name: "Increment map register", Word: "editor.second.action.8", Decoded: true, Offered: true, Register: true, Signature: sig(number("variable"))},
	{Ops: OpsSecond, ID: 10, Name: "Set relation", Word: "editor.second.action.10", Decoded: true, Offered: true, Signature: sig(number(""), number(""), number("value"))},
	{Ops: OpsSecond, ID: 12, Name: "Build item into unit's bag", Word: "editor.second.action.12", Decoded: true, Offered: true, Signature: sig(unit("unit"), item)},                                              // R2-ENGINE-1747
	{Ops: OpsSecond, ID: 13, Name: "Take item out of unit's bag and destroy it", Word: "editor.second.action.13", Decoded: true, Offered: true, Signature: sig(unit("unit"), item)},                              // R2-ENGINE-1747
	{Ops: OpsSecond, ID: 16, Name: "Take unit off the map", Word: secondWord(false, 16), Decoded: true, Offered: true, Signature: sig(unit("unit"))},                                                             // R2-ENGINE-1743
	{Ops: OpsSecond, ID: 17, Name: "Put unit back on the map", Word: "editor.second.action.17", Decoded: true, Offered: true, Signature: sig(unit("unit"))},                                                      // R2-ENGINE-1744
	{Ops: OpsSecond, ID: 18, Name: "Replace unit by the second unit", Word: "editor.second.action.18", Decoded: true, Offered: true, Signature: sig(unit("unit"), unit("unit2"))},                                // R2-ENGINE-1744
	{Ops: OpsSecond, ID: 19, Name: "Give unit to player", Word: "editor.second.action.19", Decoded: true, Offered: true, Signature: sig(unit("unit"), player("player"))},                                         // R2-ENGINE-1745
	{Ops: OpsSecond, ID: 20, Name: "Drop unit's bag as a sack", Word: "editor.second.action.20", Decoded: true, Offered: true, Signature: sig(unit("unit"))},                                                     // R2-ENGINE-1748
	{Ops: OpsSecond, ID: 21, Name: "Cast spell from cell at cell", Word: "editor.second.action.21", Decoded: true, Offered: true, Signature: sig(cell("cell"), cell("cell2"), number("spell"), number("power"))}, // R2-ENGINE-1741
	{Ops: OpsSecond, ID: 22, Name: "Give each group member to player", Word: "editor.second.action.22", Decoded: true, Offered: true, Signature: sig(group, player("player"))},                                   // R2-ENGINE-1745
	{Ops: OpsSecond, ID: 23, Name: "Add gold to player", Word: "editor.second.action.23", Decoded: true, Offered: true, Signature: sig(player("player"), number("money"))},                                       // R2-ENGINE-1746
	{Ops: OpsSecond, ID: 24, Name: "Cast spell from cell at unit", Word: "editor.second.action.24", Decoded: true, Offered: true, Signature: sig(cell("cell"), unit("unit"), number("spell"), number("power"))},  // R2-ENGINE-1742
	// Action 25 writes the cell event (p0, p1, source, target) on cell
	// (p2, p3); the source and target are (p4, p5) and (p6, p7) when p4 is not
	// 0 (R2-ENGINE-1775). Event 23 moves a unit that reaches the cell's centre
	// to the target (R2-ENGINE-1778); its power and source are not read
	// (R2-ENGINE-1779, Medium). Every shipped node holds p1 255 and the source
	// equal to the cell (R2-ASSET-386, R2-ASSET-387). What another event does
	// is unread.
	{Ops: OpsSecond, ID: 25, Name: "Cell descriptor of eight bytes", Word: "editor.second.action.25", Decoded: true, Offered: true, EffectUnread: true, Signature: sig(number(""), number(""), cell("cell"), cell("cell2"), cell("")),
		Presets: []WizardPreset{{Order: 35, Word: "editor.second.action.25.teleport", Command: CellMoveEvent}},
		Variants: []Variant{{Value: CellMoveEvent, Name: "Teleport", Word: "editor.second.action.25.teleport",
			Signature: sig(number("event"), number("unreadpower"), cell("entry"), cell("source"), cell("exit")),
			Defaults:  map[int]uint32{1: 255}, Follows: map[int]int{4: 2}}}},
	{Ops: OpsSecond, ID: 28, Name: "Move unit's bag into the second unit's bag", Word: "editor.second.action.28", Decoded: true, Offered: true, Signature: sig(unit("unit"), unit("unit2"))},                        // R2-ENGINE-1748
	{Ops: OpsSecond, ID: 29, Name: "Set time left of a spell's area effects at cell", Word: "editor.second.action.29", Decoded: true, Offered: true, Signature: sig(cell("cell"), number("spell"), number("time"))}, // R2-ENGINE-1749
	{Ops: OpsSecond, ID: 30, Name: "Set duration of a spell's effects on unit", Word: "editor.second.action.30", Decoded: true, Offered: true, Signature: sig(unit("unit"), number("spell"), number("time"))},       // R2-ENGINE-1749
	{Ops: OpsSecond, ID: 32, Name: "Take each group member off the map", Word: "editor.second.action.32", Decoded: true, Offered: true, Signature: sig(group)},                                                      // R2-ENGINE-1743
	{Ops: OpsSecond, ID: 33, Name: "Put each group member back on the map", Word: "editor.second.action.33", Decoded: true, Offered: true, Signature: sig(group)},                                                   // R2-ENGINE-1744
	{Ops: OpsSecond, ID: 34, Name: "Set unit health, defence or absorption", Word: "editor.second.action.34", Decoded: true, Offered: true, Signature: sig(unit("unit"), number(""), number("value"))},              // R2-ENGINE-1749
	{Ops: OpsSecond, ID: 35, Name: "Set scenario variable", Word: "editor.second.action.35", Decoded: true, Offered: true, Signature: sig(number(""), number("value"))},
	{Ops: OpsSecond, ID: 36, Name: "Set scenario state 752+n", Word: "editor.second.action.36", Decoded: true, Offered: true, Signature: sig(number(""), number("value"))},
	{Ops: OpsSecond, ID: 38, Name: "Take item out of listed units' bags", Word: "editor.second.action.38", Decoded: true, Offered: true, Signature: sig(item)}, // R2-ENGINE-1747
	{Ops: OpsSecond, ID: 39, Name: "Clear group forced activity", Word: "editor.second.action.39", Decoded: true, Offered: true, Signature: sig(group)},
	// The compiler appends the node's value slots 0 and 1 as bytes to a list
	// whatever trigger names it (R2-ENGINE-042, R2-ENGINE-1727): a trigger
	// copy names the same node, so the list gains no entry. The list's reader
	// is unread.
	{Ops: OpsSecond, ID: constantOp, Name: "Compiler parameter", Word: "editor.second.action.65538", Decoded: true, Offered: true, Signature: sig(cell("")),
		CopyNamed: true, EffectUnread: true},
}

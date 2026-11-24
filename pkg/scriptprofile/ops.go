package scriptprofile

import (
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
// where none is stated; Commands is the group command list of an operation
// whose signature follows its first parameter.
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
	Commands  []uint32
	// CopyRefused is the word of the refusal of a trigger copy holding a
	// node of the operation, which takes the node's id; "" where it copies.
	CopyRefused string
}

// HasSignature reports that a claim states the operation's parameter slots.
func (o Op) HasSignature() bool { return o.Signature != nil }

// Vocabulary is what differs between the games' operation vocabularies
// beyond the rows: the fallback word of an operation with no word, whether
// the inspection names an undecoded operation by its number, whether its
// parameters carry the first game's message, spell, property and variable
// kinds, whether an undecoded operation's fields stay read-only, and the
// parameter type codes whose named record no claim reads, whose slots stay
// read-only.
type Vocabulary struct {
	CheckWord, ActionWord string
	Numbered              bool
	Kinds                 bool
	ReadOnlyUndecoded     bool
	UnreadTypes           []uint32
	// NewRecord is the word of the refusal of a new trigger or node, whose
	// record holds fields no claim reads; "" where a new one is written.
	NewRecord string
}

// TypeUnread reports that the record a slot of type code t names is unread.
func (v Vocabulary) TypeUnread(t uint32) bool { return slices.Contains(v.UnreadTypes, t) }

// Undecoded is the fallback word key of a condition or an action.
func (v Vocabulary) Undecoded(condition bool) string {
	if condition {
		return v.CheckWord
	}
	return v.ActionWord
}

var vocabularies = [...]Vocabulary{
	OpsFirst: {CheckWord: "editor.check.unnamed", ActionWord: "editor.action.unnamed", Kinds: true},
	// The record a type-9 lookup returns is unread (R2-ENGINE-1680). What a
	// new node's ten text fields and word +0x48 hold is unread (R2-ASSET-364,
	// R2-ENGINE-1681).
	OpsSecond: {CheckWord: "editor.op.undecoded", ActionWord: "editor.op.undecoded", Numbered: true, ReadOnlyUndecoded: true, UnreadTypes: []uint32{slotStructure},
		NewRecord: "editor.second.newrecord"},
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

// groupCommands are the commands action 6 can be authored with, in order.
var groupCommands = []uint32{1, 2, 3, 4, 5, 10, 11, 14, 15, 17, 18}

// The first game's rows. An operation with no row is read by its raw number.
var firstChecks = []Op{
	{Condition: true, ID: 1, Name: "Group count", Word: "editor.check.op1", Decoded: true, Offered: true, Signature: sig(group)},                                                                 // TRIG-COUNT-015
	{Condition: true, ID: 2, Name: "Unit in box", Word: "editor.check.op2", Decoded: true, Offered: true, Signature: sig(unit("unit"), cell("corner"), cell("corner2"))},                         // TRIG-CHECKGEOM-130
	{Condition: true, ID: 3, Name: "Unit in distance square", Word: "editor.check.op3", Decoded: true, Offered: true, Signature: sig(unit("unit"), cell("cell"), number("radius"))},              // TRIG-DIST-014
	{Condition: true, ID: 4, Name: "Unit field (selector 6)", Word: "editor.check.op4", Decoded: true, Offered: true, Signature: sig(unit("unit"), number("selector"))},                          // TRIG-CHECK-051
	{Condition: true, ID: 5, Name: "Unit alive", Word: "editor.check.op5", Decoded: true, Offered: true, Signature: sig(unit("unit"))},                                                           // TRIG-CHECKGEOM-130
	{Condition: true, ID: 6, Name: "Distance between units", Word: "editor.check.op6", Decoded: true, Offered: true, Signature: sig(unit("unit"), unit("unit2"))},                                // TRIG-DIST-014
	{Condition: true, ID: 7, Name: "Point to unit distance", Word: "editor.check.op7", Decoded: true, Offered: true, Signature: sig(unit("unit"), cell("cell"))},                                 // TRIG-DIST-014
	{Condition: true, ID: 8, Name: "Player population", Word: "editor.check.op8", Decoded: true, Offered: true, Signature: sig(player("player"))},                                                // TRIG-LISTREAD-131
	{Condition: true, ID: 9, Name: "Unit target ID", Word: "editor.check.op9", Decoded: true, Offered: true, Signature: sig(unit("unit"))},                                                       // TRIG-TARGETID-032
	{Condition: true, ID: 10, Name: "Player relation", Word: "editor.check.op10", Decoded: true, Offered: true, Signature: sig(player("player"), player("player2"))},                             // TRIG-DIPLO-020
	{Condition: true, ID: 11, Name: "No-write arm", Word: "editor.check.op11", Decoded: true, Offered: true, Signature: sig()},                                                                   // TRIG-COND-003, a dead arm
	{Condition: true, ID: 12, Name: "Item presence", Word: "editor.check.op12", Decoded: true, Offered: true, Signature: sig(unit("unit"), item)},                                                // TRIG-ITEMTEST-040
	{Condition: true, ID: 13, Name: "No-write arm", Word: "editor.check.op13", Decoded: true, Offered: true, Signature: sig()},                                                                   // TRIG-COND-003, a dead arm
	{Condition: true, ID: 14, Name: "Sack at cell", Word: "editor.check.op14", Decoded: true, Offered: true, Signature: sig(cell("cell"))},                                                       // TRIG-SACK-022
	{Condition: true, ID: 15, Name: "Nearest player unit distance", Word: "editor.check.op15", Decoded: true, Offered: true, Signature: sig(player("player"), cell("cell"))},                     // TRIG-NEAREST-029
	{Condition: true, ID: 16, Name: "Item to point distance", Word: "editor.check.op16", Decoded: true, Offered: true, Signature: sig(unit("unit"), item, cell("cell"))},                         // TRIG-CHECK-052
	{Condition: true, ID: 17, Name: "Item presence", Word: "editor.check.op17", Decoded: true, Offered: true, Signature: sig(unit("unit"), item)},                                                // TRIG-ITEMTEST-040
	{Condition: true, ID: 18, Name: "Lose if unit dies", Word: "editor.check.op18", Decoded: true, Offered: true, Signature: sig(unit("unit"))},                                                  // MISSION-VIP-004
	{Condition: true, ID: 19, Name: "Read variable", Word: "editor.check.op19", Decoded: true, Offered: true, Register: true, Signature: sig(number("variable"))},                                // TRIG-COND-003
	{Condition: true, ID: 20, Signature: sig(player("player"))},                                                                                                                                  // TRIG-CHECKCOUNT-132
	{Condition: true, ID: 21, Name: "Structure field", Word: "editor.check.op21", Decoded: true, Offered: true, Signature: sig(structure)},                                                       // TRIG-CHECK-053
	{Condition: true, ID: 22, Name: "Move object check", Word: "editor.check.op22", Decoded: true, Signature: sig(cell("cell"), cell("cell2"))},                                                  // TRIG-CELLRESULT-133
	{Condition: true, ID: constantOp, Name: "Constant / variable preset", Word: "editor.check.op65538", Decoded: true, Offered: true, Signature: sig(number("value"))},                           // TRIG-COND-003
	{Condition: true, ID: healthBelow, Name: "Health below (Againrom)", Word: "editor.check.op16641", Decoded: true, Offered: true, Again: true, Signature: sig(unit("unit"), number("health"))}, // DIV-4009
	{Condition: true, ID: readGlobal, Name: "Read global (Againrom)", Word: "editor.check.op16642", Decoded: true, Again: true, Signature: sig(number("global"))},                                // DIV-5108
}

var firstActions = []Op{
	{ID: 1, Signature: sig()}, // TRIG-ACTIONWRITE-135
	{ID: 2, Name: "Send message", Word: "editor.action.op2", Decoded: true, Offered: true, Signature: sig(number("message"))},                                   // TRIG-MSG-023
	{ID: 3, Name: "Set variable", Word: "editor.action.op3", Decoded: true, Offered: true, Register: true, Signature: sig(number("variable"), number("value"))}, // TRIG-ACT-004
	{ID: 4, Name: "Win", Word: "editor.action.op4", Decoded: true, Offered: true, Signature: sig()},                                                             // TRIG-ACT-004
	{ID: 5, Name: "Lose", Word: "editor.action.op5", Decoded: true, Offered: true, Signature: sig()},                                                            // TRIG-ACT-004
	{ID: 6, Name: "Group command", Word: "editor.action.op6", Decoded: true, Offered: true, Commands: groupCommands},
	{ID: 7, Name: "Set formation", Offered: true},
	{ID: 8, Name: "Increment variable", Word: "editor.action.op8", Decoded: true, Offered: true, Register: true, Signature: sig(number("variable"))},                        // TRIG-ACT-004
	{ID: 9, Name: "No-op arm", Word: "editor.action.op9", Decoded: true, Signature: sig()},                                                                                  // TRIG-ACT-004, a dead arm
	{ID: 10, Name: "Set relation", Word: "editor.action.op10", Decoded: true, Offered: true, Signature: sig(number("player"), number("player2"), number("value"))},          // TRIG-DIPLO-019
	{ID: 11, Word: "editor.action.op11", Decoded: true, Signature: sig(unit("unit"), unit("unit2"), item)},                                                                  // TRIG-XFERITEM-039
	{ID: 12, Name: "Create item", Word: "editor.action.op12", Decoded: true, Offered: true, Signature: sig(unit("unit"), item)},                                             // TRIG-ADDITEM-027
	{ID: 13, Name: "Take item", Word: "editor.action.op13", Decoded: true, Offered: true, Signature: sig(unit("unit"), item)},                                               // TRIG-TAKEITEM-038
	{ID: 16, Name: "Take unit off map", Word: "editor.action.op16", Decoded: true, Offered: true, Signature: sig(unit("unit"))},                                             // TRIG-OFFMAP-041
	{ID: 17, Name: "Return unit to map", Word: "editor.action.op17", Decoded: true, Offered: true, Signature: sig(unit("unit"))},                                            // TRIG-RETURN-042
	{ID: 18, Name: "Replace unit on map", Word: "editor.action.op18", Decoded: true, Offered: true, Signature: sig(unit("unit"), unit("unit2"))},                            // TRIG-MAPGROUP-043
	{ID: 19, Name: "Give unit to player", Word: "editor.action.op19", Decoded: true, Offered: true, Signature: sig(unit("unit"), player("player"))},                         // PARTY-JOIN-025
	{ID: 20, Name: "Drop inventory", Word: "editor.action.op20", Decoded: true, Offered: true, Signature: sig(unit("unit"))},                                                // TRIG-DROPALL-024
	{ID: 21, Name: "Cast at cell", Word: "editor.action.op21", Decoded: true, Offered: true, Signature: sig(cell("cell"), cell("cell2"), number("spell"), number("power"))}, // TRIG-CAST-033
	{ID: 22, Name: "Give group to player", Word: "editor.action.op22", Decoded: true, Offered: true, Signature: sig(group, player("player"))},                               // PARTY-JOIN-025
	{ID: 23, Name: "Give money to player", Word: "editor.action.op23", Decoded: true, Offered: true, Signature: sig(player("player"), number("money"))},                     // TRIG-MONEY-028
	{ID: 24, Name: "Cast at unit", Word: "editor.action.op24", Decoded: true, Offered: true, Signature: sig(cell("cell"), unit("unit"), number("spell"), number("power"))},  // TRIG-CAST-033
	{ID: 25, Name: "Set cell tail", Word: "editor.action.op25", Decoded: true, Offered: true, Signature: sig(number("spell"), number("power"), cell("cell"))},               // TRIG-CELLTAIL-035
	{ID: 26, Name: "Set structure field", Word: "editor.action.op26", Decoded: true, Offered: true, Signature: sig(structure, number("value"))},                             // TRIG-CHECK-053
	{ID: 27, Name: "Move unit", Word: "editor.action.op27", Decoded: true, Signature: sig(unit("unit"), cell("cell"))},                                                      // AI-SCRIPTGUARD-625
	{ID: 28, Name: "Give entire inventory", Word: "editor.action.op28", Decoded: true, Offered: true, Signature: sig(unit("unit"), unit("unit2"))},                          // TRIG-GIVEALL-025
	{ID: 29, Name: "Set cell effect duration", Word: "editor.action.op29", Decoded: true, Offered: true, Signature: sig(cell("cell"), number("spell"), number("time"))},     // TRIG-EFFECTTIME-034
	{ID: 30, Name: "Set unit effect duration", Word: "editor.action.op30", Decoded: true, Offered: true, Signature: sig(unit("unit"), number("spell"), number("time"))},     // TRIG-EFFECTTIME-034
	{ID: 32, Name: "Take group off map", Word: "editor.action.op32", Decoded: true, Offered: true, Signature: sig(group)},                                                   // TRIG-MAPGROUP-043
	{ID: 33, Name: "Return group to map", Word: "editor.action.op33", Decoded: true, Offered: true, Signature: sig(group)},                                                  // TRIG-MAPGROUP-043
	{ID: 34, Name: "Set unit field", Word: "editor.action.op34", Decoded: true, Offered: true, Signature: sig(unit("unit"), number("selector"), number("value"))},           // TRIG-PROPERTY-036
	{ID: constantOp, Name: "Drop location (build time)", Word: "editor.action.op65538", Decoded: true, Offered: true, Signature: sig(cell("cell")), // TRIG-DROP-013
		CopyRefused: "editor.trigger.copydropzone"},
	{ID: formOp, Name: "Object form (build time)", Offered: true},
	{ID: undying, Name: "Set undying (Againrom)", Word: "editor.action.op16640", Decoded: true, Offered: true, Again: true, Signature: sig(unit("unit"), number("undying"))}, // DIV-4009
	{ID: setGlobal, Name: "Set global (Againrom)", Word: "editor.action.op16643", Decoded: true, Again: true, Signature: sig(number("global"), number("value"))},             // DIV-5108
	{ID: addGlobal, Name: "Add to global (Againrom)", Word: "editor.action.op16644", Decoded: true, Again: true, Signature: sig(number("global"), number("delta"))},          // DIV-5108
}

// The second game's rows: the operations a claim reads the body of
// (R2-ENGINE-1313 to R2-ENGINE-1315, R2-ENGINE-1450), each named by what the
// body reads, with the type vector every installed node of the id stores
// (R2-ASSET-363). Action 6 and check 65538 store more than one vector and
// carry none. An id absent here is shown by its raw number, not decoded.
var secondChecks = []Op{
	{Ops: OpsSecond, Condition: true, ID: 1, Name: "Group member count", Word: "editor.second.check.1", Decoded: true, Offered: true, Signature: sig(group)},
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
	{Ops: OpsSecond, Condition: true, ID: 21, Name: "Word 0x42 of a type-9 object", Word: "editor.second.check.21", Decoded: true, Offered: true, Signature: []Slot{{slotStructure, ""}}},
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
	{Ops: OpsSecond, ID: 4, Name: "Add to win counter", Word: "editor.second.action.4", Decoded: true, Offered: true, Signature: sig()},
	{Ops: OpsSecond, ID: 5, Name: "Store failure reason", Word: "editor.second.action.5", Decoded: true, Offered: true, Signature: sig(number(""))},
	{Ops: OpsSecond, ID: 6, Name: "Group arm by selector", Word: "editor.second.action.6", Decoded: true, Offered: true},
	{Ops: OpsSecond, ID: 8, Name: "Increment map register", Word: "editor.second.action.8", Decoded: true, Offered: true, Register: true, Signature: sig(number("variable"))},
	{Ops: OpsSecond, ID: 10, Name: "Set relation", Word: "editor.second.action.10", Decoded: true, Offered: true, Signature: sig(number(""), number(""), number("value"))},
	{Ops: OpsSecond, ID: 12, Name: "Item into a unit's collection", Word: "editor.second.action.12", Decoded: true, Offered: true, Signature: sig(unit("unit"), item)},
	{Ops: OpsSecond, ID: 13, Name: "Item out of a unit's collection", Word: "editor.second.action.13", Decoded: true, Offered: true, Signature: sig(unit("unit"), item)},
	{Ops: OpsSecond, ID: 16, Name: "Unit call of group action 32", Word: secondWord(false, 16), Decoded: true, Offered: true, Signature: sig(unit("unit"))},
	{Ops: OpsSecond, ID: 17, Name: "Unit call of group action 33", Word: "editor.second.action.17", Decoded: true, Offered: true, Signature: sig(unit("unit"))},
	{Ops: OpsSecond, ID: 18, Name: "Call of action 16, then a location call on the second unit", Word: "editor.second.action.18", Decoded: true, Offered: true, Signature: sig(unit("unit"), unit("unit2"))},
	{Ops: OpsSecond, ID: 19, Name: "Owner call of unit and player", Word: "editor.second.action.19", Decoded: true, Offered: true, Signature: sig(unit("unit"), player("player"))},
	{Ops: OpsSecond, ID: 20, Name: "New item collection for unit", Word: "editor.second.action.20", Decoded: true, Offered: true, Signature: sig(unit("unit"))},
	{Ops: OpsSecond, ID: 21, Name: "Two-cell call with two numbers", Word: "editor.second.action.21", Decoded: true, Offered: true, Signature: sig(cell("cell"), cell("cell2"), number(""), number(""))},
	{Ops: OpsSecond, ID: 22, Name: "Owner call of each group member and player", Word: "editor.second.action.22", Decoded: true, Offered: true, Signature: sig(group, player("player"))},
	{Ops: OpsSecond, ID: 23, Name: "Player call with a number", Word: "editor.second.action.23", Decoded: true, Offered: true, Signature: sig(player("player"), number(""))},
	{Ops: OpsSecond, ID: 24, Name: "Cell and unit call with two numbers", Word: "editor.second.action.24", Decoded: true, Offered: true, Signature: sig(cell("cell"), unit("unit"), number(""), number(""))},
	{Ops: OpsSecond, ID: 25, Name: "Cell descriptor of eight bytes", Word: "editor.second.action.25", Decoded: true, Offered: true, Signature: sig(number(""), number(""), cell("cell"), cell("cell2"), cell(""))},
	{Ops: OpsSecond, ID: 28, Name: "Item collection moved to the second unit", Word: "editor.second.action.28", Decoded: true, Offered: true, Signature: sig(unit("unit"), unit("unit2"))},
	{Ops: OpsSecond, ID: 29, Name: "Word 0x50 of the matching cell object", Word: "editor.second.action.29", Decoded: true, Offered: true, Signature: sig(cell("cell"), number(""), number("value"))},
	{Ops: OpsSecond, ID: 30, Name: "Word 0x42 of the matching unit entry", Word: "editor.second.action.30", Decoded: true, Offered: true, Signature: sig(unit("unit"), number(""), number("value"))},
	{Ops: OpsSecond, ID: 32, Name: "Call of action 16 on each group member", Word: "editor.second.action.32", Decoded: true, Offered: true, Signature: sig(group)},
	{Ops: OpsSecond, ID: 33, Name: "Call of action 17 on each group member", Word: "editor.second.action.33", Decoded: true, Offered: true, Signature: sig(group)},
	{Ops: OpsSecond, ID: 34, Name: "Unit field by selector 6, 15 or 16", Word: "editor.second.action.34", Decoded: true, Offered: true, Signature: sig(unit("unit"), number(""), number("value"))},
	{Ops: OpsSecond, ID: 35, Name: "Set scenario variable", Word: "editor.second.action.35", Decoded: true, Offered: true, Signature: sig(number(""), number("value"))},
	{Ops: OpsSecond, ID: 36, Name: "Set scenario state 752+n", Word: "editor.second.action.36", Decoded: true, Offered: true, Signature: sig(number(""), number("value"))},
	{Ops: OpsSecond, ID: 38, Name: "Take item from units", Word: "editor.second.action.38", Decoded: true, Offered: true, Signature: sig(item)},
	{Ops: OpsSecond, ID: 39, Name: "Clear group forced activity", Word: "editor.second.action.39", Decoded: true, Offered: true, Signature: sig(group)},
	// The compiler appends the node's two low bytes to a list whatever
	// trigger names it; what reads the list is unread (R2-ENGINE-042).
	{Ops: OpsSecond, ID: constantOp, Name: "Compiler parameter", Word: "editor.second.action.65538", Decoded: true, Offered: true, Signature: sig(cell("")),
		CopyRefused: "editor.second.copycompilerparameter"},
}

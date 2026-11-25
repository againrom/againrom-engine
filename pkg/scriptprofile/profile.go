// Package scriptprofile is the one table of a map script's limits: how many
// condition pairs and actions a trigger holds, the comparison alphabet, the
// registers, the fire-once latches, the trigger cap and the node id cap, per
// game. The map format, the simulation, the editor and its window all read
// them here; it imports nothing, so the simulation can size its fixed arrays
// from it without importing a format decoder.
//
// It also names the fields of a trigger record the editor's field table and
// its window share, so a slot's key is spelled once.
package scriptprofile

import (
	"strconv"
	"strings"
)

// The original script machine's numbers.
const (
	// RecordPairs and RecordActions are the slots of one 184-byte trigger
	// record: three condition pairs and four action ids (ALM-TRIG-047).
	RecordPairs   = 3
	RecordActions = 4
	// MaxCompare is the highest comparison code (TRIG-CMP-006).
	MaxCompare = 5
	// FirstRegisters is the first game's check-result register file and
	// Latches its fire-once latch bytes, one per trigger position
	// (TRIG-STORE-002); a check owns a register (MISSION-SLOT-008).
	FirstRegisters = 100
	Latches        = 1000
	// Triggers is the trigger cap: a trigger past the last latch has none.
	Triggers = Latches
	// MaxID is the highest node id a trigger slot names (TRIG-BIND-010).
	MaxID = 5000
	// SecondRegisters is the second game's result slots (R2-SESSION-019).
	SecondRegisters = 1000
	// FirstTickRegister and SecondTickRegister are the register each script
	// pass increments (SAV-650, R2-ENGINE-1451).
	FirstTickRegister  = 93
	SecondTickRegister = 999
)

// The extended machine's numbers, an Againrom map script record's limits
// (owner decision): conditions and actions per trigger, the variable array
// and the trigger cap, one fire-once latch per trigger.
const (
	ExtendedConditions = 64
	ExtendedActions    = 64
	ExtendedVariables  = 4096
	ExtendedTriggers   = 10000
)

// The ids a mod's global variable takes (owner decision): a map names a
// global by its id, and one mod declares at most LastGlobal of them.
const (
	FirstGlobal = 1
	LastGlobal  = 1024
)

// Ops names which game's operation ids a profile's nodes use: the same
// number is a different operation in each game.
type Ops uint8

const (
	OpsFirst Ops = iota
	OpsSecond
)

// Profile is one game's script limits.
type Profile struct {
	Ops            Ops    // the operation vocabulary of the game's nodes
	Pairs, Actions int    // condition pairs and action ids per trigger
	MaxCompare     uint32 // the highest comparison code
	Registers      int    // check-result registers, 0 for one result per check
	TickRegister   int    // register incremented on each script pass
	Latches        int    // fire-once latch bytes
	Triggers       int    // triggers a map holds
	MaxID          uint32 // the highest node id a slot names
	// Variables is the variable array kept apart from the check results;
	// 0 where the variables are the check-result registers.
	Variables int
	// Groups is whether a trigger's conditions form OR-ed AND groups.
	Groups bool
	// Extended is whether the profile is the extended machine's record.
	Extended bool

	// The compiler's rules for a map script's nodes. Each rule selects one
	// body in the compiler's tables in pkg/mapload, so a rule is read only
	// through its index there.
	Empty          EmptyRule
	Drops          DropRule
	OmittedActions OmittedRule
	Consts         ConstRule
	Omit           OmitRule
	TriggerOmit    TriggerRule
	Units          UnitRule
	Late           LateRule
	// LateLow and LateHigh bound the values a LateReferences rule leaves for
	// the session table.
	LateLow, LateHigh uint32
	// InstantSlots: each instant carries the runtime slot it was numbered in.
	InstantSlots bool
	// AliveHeldBody: a health-below check's held-body argument is set from
	// the unit's zero-health state.
	AliveHeldBody bool
	// UntypedAsInt: a parameter whose type is unset reads as an integer.
	UntypedAsInt bool
}

// EmptyRule is what an empty map script compiles to.
type EmptyRule uint8

const (
	// EmptyCompiles compiles an empty script like any other.
	EmptyCompiles EmptyRule = iota
	// EmptyNil compiles an empty script to no script at all.
	EmptyNil
)

// DropRule is how a drop node is kept.
type DropRule uint8

const (
	// DropCells binds a drop node's parameters and names a cell.
	DropCells DropRule = iota
	// DropPacked names one packed coordinate and binds no references.
	DropPacked
)

// OmittedRule is where an omitted action is recorded.
type OmittedRule uint8

const (
	// OmittedUnbuilt records an omitted action as unbuilt, which a trigger's
	// roster check skips.
	OmittedUnbuilt OmittedRule = iota
	// OmittedBuildTime records an omitted action as a build-time node.
	OmittedBuildTime
)

// ConstRule is how a constant node binds.
type ConstRule uint8

const (
	// ConstBound binds a constant node like any other, and omits it when its
	// references fail.
	ConstBound ConstRule = iota
	// ConstValue carries the node's value in its first operand and is never
	// omitted for its references.
	ConstValue
)

// OmitRule is which failed reference omits a node.
type OmitRule uint8

const (
	// OmitByRole omits a node only when a role reference fails.
	OmitByRole OmitRule = iota
	// OmitByAnyFailure omits a node when any of its references fails.
	OmitByAnyFailure
)

// TriggerRule is whether a trigger is omitted when one of its checks is.
type TriggerRule uint8

const (
	// TriggersKept keeps every trigger.
	TriggersKept TriggerRule = iota
	// TriggersOmitOnCheck omits a trigger a pair of which names an omitted check.
	TriggersOmitOnCheck
)

// UnitRule is how a unit value resolves.
type UnitRule uint8

const (
	// UnitDirect resolves a unit value through its three id bands.
	UnitDirect UnitRule = iota
	// UnitByRole resolves a unit value through the role table first.
	UnitByRole
)

// LateRule is whether a value in the late range is left for the session.
type LateRule uint8

const (
	// LateKept resolves every value now.
	LateKept LateRule = iota
	// LateReferences leaves a value in [LateLow, LateHigh] under a group,
	// player or unit type for the session table.
	LateReferences
)

// First is the first game's profile. Second is the second game's: its
// session holds 1000 registers, and its other limits are the first game's.
// Extended is the extended machine's: a pair is one condition of a group,
// each check owns a result slot of its own, and the variables are an array
// of their own.
var (
	First = Profile{Pairs: RecordPairs, Actions: RecordActions, MaxCompare: MaxCompare,
		Registers: FirstRegisters, TickRegister: FirstTickRegister, Latches: Latches, Triggers: Triggers, MaxID: MaxID,
		Empty: EmptyNil}
	Second = Profile{Ops: OpsSecond, Pairs: RecordPairs, Actions: RecordActions, MaxCompare: MaxCompare,
		Registers: SecondRegisters, TickRegister: SecondTickRegister, Latches: Latches, Triggers: Triggers, MaxID: MaxID,
		OmittedActions: OmittedBuildTime, Drops: DropPacked, Consts: ConstValue, Omit: OmitByAnyFailure,
		TriggerOmit: TriggersOmitOnCheck, Units: UnitByRole, Late: LateReferences,
		InstantSlots: true, AliveHeldBody: true, UntypedAsInt: true, LateLow: 8000, LateHigh: 9999}
	Extended = First.onExtendedMachine()
)

// onExtendedMachine is p's game on the extended machine: the record's pair,
// action, latch and trigger counts, its own variables and groups; the
// game's operations and node rules stay.
func (p Profile) onExtendedMachine() Profile {
	p.Pairs, p.Actions, p.Registers = ExtendedConditions, ExtendedActions, 0
	p.Latches, p.Triggers, p.Variables = ExtendedTriggers, ExtendedTriggers, ExtendedVariables
	p.Groups, p.Extended = true, true
	return p
}

// For is the one script profile selector: the profile of operation
// vocabulary ops, on the extended machine when the map carries the
// Againrom script record.
func For(ops Ops, extended bool) Profile {
	p := First
	if ops == OpsSecond {
		p = Second
	}
	if extended {
		return p.onExtendedMachine()
	}
	return p
}

// LeftKey, RightKey and CmpKey name pair p's fields of a trigger record,
// GroupKey its group in the Againrom record, and ActionKey action slot a's,
// counting from 0.
func LeftKey(p int) string   { return "left" + strconv.Itoa(p) }
func RightKey(p int) string  { return "right" + strconv.Itoa(p) }
func CmpKey(p int) string    { return "cmp" + strconv.Itoa(p) }
func GroupKey(p int) string  { return "group" + strconv.Itoa(p) }
func ActionKey(a int) string { return "action" + strconv.Itoa(a) }

// TextKey names text field i of a node record, counting from 0. NodeTextKey
// reports a node's own text: its name or one of its text fields.
func TextKey(i int) string { return "text" + strconv.Itoa(i) }

func NodeTextKey(key string) bool {
	n, ok := strings.CutPrefix(key, "text")
	_, err := strconv.Atoi(n)
	return key == "name" || ok && err == nil
}

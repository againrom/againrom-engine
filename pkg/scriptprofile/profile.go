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

import "strconv"

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

// Profile is one game's script limits.
type Profile struct {
	Pairs, Actions int    // condition pairs and action ids per trigger
	MaxCompare     uint32 // the highest comparison code
	Registers      int    // check-result registers, 0 for one result per check
	Latches        int    // fire-once latch bytes
	Triggers       int    // triggers a map holds
	MaxID          uint32 // the highest node id a slot names
	// Variables is the variable array kept apart from the check results;
	// 0 where the variables are the check-result registers.
	Variables int
	// Groups is whether a trigger's conditions form OR-ed AND groups.
	Groups bool
}

// First is the first game's profile. Second is the second game's: its
// session holds 1000 registers, and its other limits are the first game's.
// Extended is the extended machine's: a pair is one condition of a group,
// each check owns a result slot of its own, and the variables are an array
// of their own.
var (
	First = Profile{Pairs: RecordPairs, Actions: RecordActions, MaxCompare: MaxCompare,
		Registers: FirstRegisters, Latches: Latches, Triggers: Triggers, MaxID: MaxID}
	Second = Profile{Pairs: RecordPairs, Actions: RecordActions, MaxCompare: MaxCompare,
		Registers: SecondRegisters, Latches: Latches, Triggers: Triggers, MaxID: MaxID}
	Extended = Profile{Pairs: ExtendedConditions, Actions: ExtendedActions, MaxCompare: MaxCompare,
		Latches: ExtendedTriggers, Triggers: ExtendedTriggers, MaxID: MaxID,
		Variables: ExtendedVariables, Groups: true}
)

// LeftKey, RightKey and CmpKey name pair p's fields of a trigger record, and
// ActionKey action slot a's, counting from 0.
func LeftKey(p int) string   { return "left" + strconv.Itoa(p) }
func RightKey(p int) string  { return "right" + strconv.Itoa(p) }
func CmpKey(p int) string    { return "cmp" + strconv.Itoa(p) }
func ActionKey(a int) string { return "action" + strconv.Itoa(a) }

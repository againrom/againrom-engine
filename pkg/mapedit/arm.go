package mapedit

import (
	"encoding/binary"
	"fmt"
)

// A win or lose arm is the script shape the shipped maps author: an action
// node that runs instant 4 (win) or 5 (lose), a condition node that measures
// the target, a constant node, and a trigger that compares the two once
// (MISSION-WIN-003, TRIG-ACT-004, TRIG-CMP-006, TRIG-FIRE-007). The four arms
// are rows of armTable; AddArm builds any of them from its row.

// ArmKind names a row of the arm table.
type ArmKind int

const (
	ArmWinUnit      ArmKind = iota // won when the target unit dies
	ArmWinGroup                    // won when the target group has no living member
	ArmWinStructure                // won when the target structure's health is at or below zero
	ArmLoseUnit                    // lost when the target unit dies
)

// The check opcodes an arm measures with (TRIG-COND-003) and the comparison
// codes of a trigger pair (TRIG-CMP-006).
const (
	checkGroupCount  = 1  // the group's member count (TRIG-COUNT-015, TRIG-EMPTY-018)
	checkAlive       = 5  // 0 when the unit is dead
	checkStructField = 21 // the structure's word at +0x42 (TRIG-CHECK-053)
	constOpcode      = 0x10002

	cmpEqual     = 0
	cmpLessEqual = 5
)

const (
	instantWin  = 4
	instantLose = 5

	// The binder numbers conditions into a 100-word register file and reads
	// one latch byte per trigger from a 1000-byte array (MISSION-SLOT-008);
	// an id above 5000 is a suspicious reference (TRIG-BIND-010).
	scriptRegisters = 100
	scriptTriggers  = 1000
	scriptMaxID     = 5000

	// Trigger record offsets (ALM-TRIG-047, ALM-TRIGREAD-233).
	triggerPairs   = 0x80
	triggerActions = 0x98
	triggerCompare = 0xa8
	triggerOnce    = 0xb4
)

// Arm is one row of the arm table: what the arm's records say. Target is the
// parameter type code of the check's one parameter (ALM-TRIG-046).
type Arm struct {
	Name        string // "win" or "lose"
	Instant     uint32
	ActionLabel string
	TriggerName string
	Check       uint32
	CheckLabel  string
	Target      uint32
	Compare     uint32 // trigger comparison code of (check, constant)
	ConstValue  uint32
}

var armTable = map[ArmKind]Arm{
	ArmWinUnit: {Name: "win", Instant: instantWin, ActionLabel: `Force "Mission Complete" state`, TriggerName: "Win",
		Check: checkAlive, CheckLabel: "Win unit alive?", Target: paramUnit, Compare: cmpEqual},
	ArmWinGroup: {Name: "win", Instant: instantWin, ActionLabel: `Force "Mission Complete" state`, TriggerName: "Win",
		Check: checkGroupCount, CheckLabel: "Win group count", Target: paramGroup, Compare: cmpEqual},
	ArmWinStructure: {Name: "win", Instant: instantWin, ActionLabel: `Force "Mission Complete" state`, TriggerName: "Win",
		Check: checkStructField, CheckLabel: "Win structure health", Target: paramStructure, Compare: cmpLessEqual},
	ArmLoseUnit: {Name: "lose", Instant: instantLose, ActionLabel: `Force "Mission Fail" state`, TriggerName: "Lose",
		Check: checkAlive, CheckLabel: "Lose unit alive?", Target: paramUnit, Compare: cmpEqual},
}

// ArmFor returns the row of kind.
func ArmFor(kind ArmKind) (Arm, bool) {
	a, ok := armTable[kind]
	return a, ok
}

// ArmConditions is how many condition nodes, and so binder registers, AddArm
// appends: the check and the constant it is compared with.
const ArmConditions = 2

// AddArm appends one arm for target as one edit: an instant node, the check
// node, a constant node holding the arm's limit and a trigger that compares
// them once. Every id is one above the highest of its array (ALM-TRIG-045);
// the unwritten bytes of each record are zero (DIV-3183). A map whose arrays
// have no room for the arm is refused with nothing written.
func (e *Editor) AddArm(kind ArmKind, target uint32) error {
	arm, ok := armTable[kind]
	if !ok {
		return fmt.Errorf("mapedit: no arm %d", kind)
	}
	switch arm.Target {
	case paramUnit, paramStructure:
		if target < 1 || target > 0xFFFF {
			return fmt.Errorf("mapedit: a %s target is 1..65535, not %d", arm.Name, target)
		}
	}
	if e.Count(KindCondition)+ArmConditions > scriptRegisters {
		return fmt.Errorf("mapedit: the script already uses %d of %d check registers", e.Count(KindCondition), scriptRegisters)
	}
	if e.Count(KindTrigger)+1 > scriptTriggers {
		return fmt.Errorf("mapedit: the script already holds %d of %d triggers", e.Count(KindTrigger), scriptTriggers)
	}
	actionID, err := e.NextScriptID(KindAction)
	if err != nil {
		return err
	}
	checkID, err := e.NextScriptID(KindCondition)
	if err != nil {
		return err
	}
	if actionID > scriptMaxID || checkID+1 > scriptMaxID {
		return fmt.Errorf("mapedit: the next node id passes %d", scriptMaxID)
	}
	action, err := NewScriptNode(arm.ActionLabel, arm.Instant, actionID)
	if err != nil {
		return err
	}
	check, err := NewScriptNode(arm.CheckLabel, arm.Check, checkID, ScriptParam{Type: arm.Target, Value: target})
	if err != nil {
		return err
	}
	limit, err := NewScriptNode("Constant", constOpcode, checkID+1, ScriptParam{Type: paramInt, Value: arm.ConstValue})
	if err != nil {
		return err
	}
	trigger := make([]byte, scriptTriggerSize)
	copy(trigger, arm.TriggerName)
	binary.LittleEndian.PutUint32(trigger[triggerPairs:], checkID)
	binary.LittleEndian.PutUint32(trigger[triggerPairs+4:], checkID+1)
	binary.LittleEndian.PutUint32(trigger[triggerActions:], actionID)
	binary.LittleEndian.PutUint32(trigger[triggerCompare:], arm.Compare)
	binary.LittleEndian.PutUint32(trigger[triggerOnce:], 1)
	_, err = e.PlaceAll([]Placement{{KindAction, action}, {KindCondition, check}, {KindCondition, limit}, {KindTrigger, trigger}})
	return err
}

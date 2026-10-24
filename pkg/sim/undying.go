package sim

import (
	"fmt"
	"math"
)

// The undying state and the health-below condition are Againrom's own script
// nodes (DIV-4009). ROM1 takes its death arm on health <= 0
// (HERO-DEATH-026); an undying unit never takes it.

// entityUndyingBit is Entity.Undying in the entity record's always-hits byte
// (+82), whose bit 0 is AlwaysHits. A world with no undying unit encodes
// exactly as before.
const entityUndyingBit byte = 0x02

// undyingHealthFloor is the least health an undying unit carries: the ROM1
// health field is a signed 16-bit word (HERO-HEALTH-032), and a SAV writes
// it at that width.
const undyingHealthFloor int32 = math.MinInt16

// healthBelow is the health-below condition's value: 1 iff e's current health
// is below x, else 0.
func healthBelow(e Entity, x int32) int32 {
	if e.HP < x {
		return 1
	}
	return 0
}

// setUndying sets or clears entity i's undying state.
//
// Setting it on a unit that is already not alive changes nothing: the unit
// has taken the death transition and the state only keeps a living unit out
// of it. Clearing it on a unit whose health is at or below zero runs the
// ordinary death transition at once, as a script health write does
// (setUnitProperty), so the dwell and the decay count from the next tick.
func (w *World) setUndying(i int, on bool) {
	e := &w.entities[i]
	if on {
		if e.Alive() {
			e.Undying = true
		}
		return
	}
	if !e.Undying {
		return
	}
	e.Undying = false
	w.clearFelled(i)
}

// AgainromNodeValue refuses value as the plain parameter of an Againrom node:
// a health-below X outside the signed 16-bit health field, or an undying
// state other than 0 or 1. check says op is a check opcode, else an instant
// opcode; every other node passes. The program compiler and the map editor
// both ask it.
func AgainromNodeValue(check bool, op int32, value int64) error {
	if check && op == ScriptCheckHealthBelow && (value < math.MinInt16 || value > math.MaxInt16) {
		return fmt.Errorf("health below %d is outside %d..%d", value, math.MinInt16, math.MaxInt16)
	}
	if !check && op == ScriptInstantUndying && value != 0 && value != 1 {
		return fmt.Errorf("undying is 1 (set) or 0 (clear), not %d", value)
	}
	return nil
}

// validateAgainromNodes refuses a program holding an Againrom node whose
// plain parameter AgainromNodeValue refuses.
func validateAgainromNodes(checks []ScriptCheck, instants []ScriptInstant) error {
	for i, c := range checks {
		if err := AgainromNodeValue(true, c.Op, int64(c.Args[0])); err != nil {
			return fmt.Errorf("sim: script check %d: %w", i, err)
		}
	}
	for i, in := range instants {
		if err := AgainromNodeValue(false, in.Op, int64(in.Args[0])); err != nil {
			return fmt.Errorf("sim: script instant %d: %w", i, err)
		}
	}
	return nil
}

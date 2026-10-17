package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// TestRepairSourceSkillsKeepsAStoredLowerLevel: a live level below a trained
// level its experience agrees with is a placement's skill store and stays; a
// live and trained level both below their experience band rise together.
func TestRepairSourceSkillsKeepsAStoredLowerLevel(t *testing.T) {
	build := func(live, base int16, xp uint32) sim.SourceActor {
		s := sim.SourceActor{Class: 2}
		binary.LittleEndian.PutUint16(s.Attack[2+2*1:], uint16(live))
		binary.LittleEndian.PutUint16(s.Base[2+2*1:], uint16(base))
		s.SkillXP[1] = xp
		return s
	}
	words := func(s sim.SourceActor) (int16, int16) {
		return int16(binary.LittleEndian.Uint16(s.Attack[4:])), int16(binary.LittleEndian.Uint16(s.Base[4:]))
	}
	xp := uint32(data.SkillXPFor(20))
	stored := build(5, 20, xp)
	repairSourceSkills(&stored)
	if live, base := words(stored); live != 5 || base != 20 {
		t.Fatalf("a stored level 5 over trained 20: live %d base %d, want 5 and 20", live, base)
	}
	lost := build(5, 5, xp)
	repairSourceSkills(&lost)
	want := int16(data.RepairSkillLevel(5, int32(xp)))
	if live, base := words(lost); want <= 5 || live != want || base != want {
		t.Fatalf("a lost rank: live %d base %d, want both %d", live, base, want)
	}
}

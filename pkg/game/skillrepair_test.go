package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestALoadedNativeSkillHoldsItsGamesLevel: a loaded native skill keeps the
// level the live award left it at. The second game raises at the next
// level's mark, so level 10 with 1619 experience (marks 1593 and 1853) stays
// 10 and one experience in an untrained skill stays 0; the first game raises
// past its own mark, so the same slots rise to 11 and 1, as before.
func TestALoadedNativeSkillHoldsItsGamesLevel(t *testing.T) {
	r := sim.Rules{}
	second := base.GameROM2.Edition()
	for _, c := range []struct {
		what   string
		table  *mapload.Table
		level  int32
		xp     int32
		want   int32
		raised bool
	}{
		{"second, between the marks", &mapload.Table{Edition: &second}, 10, r.SkillXP(10) + 26, 10, false},
		{"second, at the next mark", &mapload.Table{Edition: &second}, 10, r.SkillXP(11), 11, true},
		{"second, untrained", &mapload.Table{Edition: &second}, 0, 1, 0, false},
		{"second, two marks behind", &mapload.Table{Edition: &second}, 9, r.SkillXP(11) + 1, 11, true},
		{"first, between the marks", nil, 10, r.SkillXP(10) + 26, 11, true},
		{"first, at its own mark", nil, 10, r.SkillXP(10), 10, false},
		{"first, untrained", nil, 0, 1, 1, true},
	} {
		var skill, xp [data.SkillSlots]int32
		skill[1], xp[1] = c.level, c.xp
		raised := repairNativeSkills(&skill, xp, c.table)
		if skill[1] != c.want || raised != c.raised {
			t.Errorf("%s: level %d at %d experience loads as %d (raised %v), want %d (%v)", c.what, c.level, c.xp, skill[1], raised, c.want, c.raised)
		}
	}
}

// TestTheFirstGameHeldSkillLevelIsItsOldRepair: the first game's held level
// is the repair it had, the level above the last mark below the experience.
func TestTheFirstGameHeldSkillLevelIsItsOldRepair(t *testing.T) {
	r := sim.Rules{}
	for xp := int32(1); xp <= r.SkillXP(r.SkillCap())+3; xp += 1 + xp/64 {
		if got, want := sim.HeldSkillLevel(sim.ScriptROM1, r, xp), r.ClampSkill(r.SkillLevelFor(xp-1)+1); got != want {
			t.Fatalf("experience %d holds %d, want %d", xp, got, want)
		}
	}
}

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

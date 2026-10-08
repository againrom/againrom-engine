package game

import (
	"encoding/binary"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func repairNativeSkills(skill *[data.SkillSlots]int32, xp [data.SkillSlots]int32, rules sim.Rules) bool {
	repaired := false
	for i := 1; i < data.SkillSlots; i++ {
		if xp[i] <= 0 {
			continue
		}
		// A native school purchase stores the purchased rank's threshold plus one.
		if skill[i] > 0 && xp[i] == rules.SkillXP(skill[i])+1 {
			continue
		}
		level := rules.ClampSkill(rules.SkillLevelFor(xp[i]-1) + 1)
		if skill[i] < level {
			skill[i], repaired = level, true
		}
	}
	return repaired
}

// repairedSkillLevel is the level a loaded slot holds. A slot whose stored
// experience lies above its stored level's band takes the level that
// experience implies (data.RepairSkillLevel); every other slot is unchanged.
func repairedSkillLevel(level int32, xp uint32) int32 {
	if xp > 1<<31-1 {
		return level
	}
	return data.RepairSkillLevel(level, int32(xp))
}

// repairHeroSkills raises the six trained levels of a hero by the amount the
// saved level words are repaired. The test runs on the saved level word, the
// effective level, so a slot whose level is already above its experience band
// through an effect keeps its trained base.
func repairHeroSkills(skill *[data.SkillSlots]int32, saved [data.SkillSlots]uint16, xp [data.SkillSlots]uint32) {
	for i := range skill {
		level := int32(int16(saved[i]))
		skill[i] += repairedSkillLevel(level, xp[i]) - level
	}
}

// repairSourceSkills applies repairedSkillLevel to the attack-block and
// base-block level words of a source actor. The base word rises by the same
// amount as the attack word, so a later derive keeps their difference.
func repairSourceSkills(s *sim.SourceActor) {
	if s.Class != 2 {
		return
	}
	for i := 0; i < data.SkillSlots; i++ {
		at := binary.LittleEndian.Uint16(s.Attack[2+2*i:])
		fixed := repairedSkillLevel(int32(int16(at)), s.SkillXP[i])
		if fixed == int32(int16(at)) {
			continue
		}
		base := int32(int16(binary.LittleEndian.Uint16(s.Base[2+2*i:])))
		binary.LittleEndian.PutUint16(s.Attack[2+2*i:], uint16(fixed))
		binary.LittleEndian.PutUint16(s.Base[2+2*i:], uint16(base+fixed-int32(int16(at))))
	}
}

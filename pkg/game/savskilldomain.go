package game

import (
	"encoding/binary"

	"againrom/pkg/formats/sav"
	"againrom/pkg/rules"
)

// The attack block of a Human record is a field the original writes in its own
// domain: skill levels 1 to 5 clamped to the original cap, and a to-hit and
// damage base that carry the active skill's terms at that clamped level. The
// engine's live levels can sit above the cap (a worn bonus lifts base plus
// bonus above 100), and the live to-hit and damage base carry the extra terms.
// SAVE writes the original's values; a LOAD derives the lift again from the
// base and modifier blocks, which stay as they are.

const (
	savLevelBlock       = "UA6"
	savSkillToHitFactor = 3
	savSkillDamageDiv   = 5
)

// savHumanAttack returns the attack block of a Human record, or nil.
func savHumanAttack(r *sav.DocumentRecordData) []byte {
	if r.Class != "Human" {
		return nil
	}
	if b := modBlock(r, savLevelBlock); len(b) == 24 {
		return b
	}
	return nil
}

// projectOriginalAttack removes from the to-hit and damage base of every Human
// record the terms its active skill adds above the original cap. The level
// words are not touched here. A save under a mod set keeps the true levels in
// its mark and so keeps the to-hit that matches them; this runs without one.
func projectOriginalAttack(doc *sav.DocumentData) {
	for i := range doc.Objects {
		b := savHumanAttack(&doc.Objects[i])
		if b == nil {
			continue
		}
		active := int(b[16])
		if active < 1 || active > 5 {
			continue
		}
		level := int32(int16(binary.LittleEndian.Uint16(b[2+2*active:])))
		if level <= rules.ROM1SkillCap {
			continue
		}
		excess := level - rules.ROM1SkillCap
		toHit := binary.LittleEndian.Uint16(b) - uint16(savSkillToHitFactor*excess)
		binary.LittleEndian.PutUint16(b, toHit)
		b[14] -= uint8(level/savSkillDamageDiv - rules.ROM1SkillCap/savSkillDamageDiv)
	}
}

// clampOriginalLevels holds the five skill levels of every Human record to the
// original cap. A modded save already carries the projection; this is the
// ordinary path, and it never adds a mod mark.
func clampOriginalLevels(doc *sav.DocumentData) {
	for i := range doc.Objects {
		b := savHumanAttack(&doc.Objects[i])
		if b == nil {
			continue
		}
		for j := 1; j <= 5; j++ {
			if int32(int16(binary.LittleEndian.Uint16(b[2+2*j:]))) > rules.ROM1SkillCap {
				binary.LittleEndian.PutUint16(b[2+2*j:], uint16(rules.ROM1SkillCap))
			}
		}
	}
}

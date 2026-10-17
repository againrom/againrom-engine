package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// The placement spawner applies a record's override block in two parts
// (UNIT-PLACE-034). The four stat bytes come first and the virtual
// derivation runs after them: on a Human it is the whole recompute, which
// clamps the live health and mana to the new maxima (HERO-ORDER-014); on a
// creature it touches no stat a store reads (UNIT-DERIVE-003). The live
// health and mana words, defence, absorption, the five skills and the five
// protections follow the derivation, so each is the last word on its field.
// The creature arm's difficulty block runs before both parts
// (UNIT-GATE-013), which is the order blockFor already has.

// tailStats reports whether the block stores any of the four stats.
func tailStats(t alm.UnitTail) bool {
	return t.Body != 0 || t.Mind != 0 || t.Spirit != 0 || t.Reaction != 0
}

// humanWithTailStats is d with the record's present stat bytes stored.
func humanWithTailStats(d data.HumanDef, t alm.UnitTail) data.HumanDef {
	d.Body = storedStat(d.Body, t.Body)
	d.Mind = storedStat(d.Mind, t.Mind)
	d.Spirit = storedStat(d.Spirit, t.Spirit)
	d.Reaction = storedStat(d.Reaction, t.Reaction)
	return d
}

// withTailStats stores the present stat bytes into a block no derivation
// follows: a creature's or an unresolved placement's.
func (b spawnBlock) withTailStats(t alm.UnitTail) spawnBlock {
	if t.Body != 0 {
		b.nativeBasis = b.nativeBasis.WithBody(uint16(t.Body))
	}
	b.mind = storedStat(b.mind, t.Mind)
	b.spirit = storedStat(b.spirit, t.Spirit)
	b.reaction = storedStat(b.reaction, t.Reaction)
	return b
}

// storedStat is v, or the override byte when it is present.
func storedStat(v int32, override uint8) int32 {
	if override != 0 {
		return int32(override)
	}
	return v
}

// withTailWords applies the stores that follow the derivation to the actor
// definition of placement u, resolved on arm. An NPC-arm placement keeps its
// derived defence, absorption, skills and protections (DIV-3580).
func withTailWords(d sim.ActorDefinition, u alm.Unit, arm Arm) sim.ActorDefinition {
	if u.HasCurrentHP {
		d.HP = int32(u.CurrentHP)
	}
	t := u.Tail
	if t.HasMana {
		d.Mana = int32(t.Mana)
	}
	if arm == ArmNPC {
		return d
	}
	if t.Defence != 0 {
		d.Defence = int32(t.Defence)
	}
	if t.Absorption != 0 {
		d.Absorption = int32(t.Absorption)
	}
	for i, v := range t.Skill {
		if v != 0 && (i < len(t.Skill)-1 || storesShooting(t)) {
			d.Skill[i+1] = int32(v)
		}
	}
	for i, v := range t.Protection {
		if v != 0 {
			d.Protection[i] = int32(v)
		}
	}
	return d
}

// storesShooting reports that the record's Shooting byte (file +0x3a)
// reaches the actor: it is present and another of the four applied skill
// bytes is too. The original's state keeps the derived Shooting where the
// byte is the record's only skill override (DIV-3581); the cause is Unknown.
func storesShooting(t alm.UnitTail) bool {
	return t.Skill[4] != 0 && (t.Skill[0] != 0 || t.Skill[1] != 0 || t.Skill[2] != 0 || t.Skill[3] != 0)
}

// skipsGear reports that the Human constructor builds u's row without its
// equipment: flag bit 7 reaches the gate on the type-key and definition-id
// arms, and the NPC arm passes zero (ALM-FLAGPATH-109, UNIT-PLACEGEAR-098).
func skipsGear(u alm.Unit, arm Arm) bool {
	return u.Tail.SkipGear && (arm == ArmHumansByType || arm == ArmServerID)
}

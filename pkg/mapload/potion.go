package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// PotionHero folds only earned permanent gains, not a temporary modifier.
func PotionHero(h data.Hero, gains [4]int32) data.Hero {
	h.Body += gains[0]
	h.Reaction += gains[1]
	h.Mind += gains[2]
	h.Spirit += gains[3]
	return h
}

// PotionHeadroom is the remaining single-use gain under the ordinary attribute
// cap and the hard ceiling. Equipment bonuses remain modifiers, not earned stats.
func PotionHeadroom(h data.Hero, d data.Derived) (out [4]int32) {
	base := [4]int32{h.Body, h.Reaction, h.Mind, h.Spirit}
	live := [4]int32{d.Body, d.Reaction, d.Mind, d.Spirit}
	for i := range out {
		out[i] = min(50-base[i], 100-live[i])
		out[i] = max(0, min(100, out[i]))
	}
	return out
}

// PartyDisplayWithTable is the read-only town/preview projection. Spawn stays
// base-only; display folds a carried timer once without advancing or storing
// state.
func PartyDisplayWithTable(p PartyMember, t *Table) (d data.Derived, health, mana int32) {
	// The first two hire types use fixed creature templates, including in
	// the town card. StartMission resolves this same placement block.
	resolvedSiege := false
	if p.MercenaryType == 1 || p.MercenaryType == 2 {
		if b, err := blockFor(alm.Unit{ClassID: int16(p.Class), ClassSubID: uint16(p.FigureFace)}, t, DifficultyNormal); err == nil {
			d = data.Derived{Body: p.Hero.Body, Reaction: b.reaction, Mind: b.mind, Spirit: b.spirit,
				HealthMax: b.health, ManaMax: b.manaMax, Combat: b.combat,
				Protection: b.protection, Speed: b.speed, Sight: int32(b.sight),
				HealthRegeneration: b.healthRegeneration, ManaRegeneration: b.manaRegeneration,
				RotationSpeed: b.rotationSpeed, SecondaryDamage: b.secondaryDamage}
			for i, value := range b.resistance {
				d.Resistance[i] = int32(value)
			}
			health, mana = b.health, b.mana
			resolvedSiege = true
		}
	}
	if !resolvedSiege {
		d, health, mana = PartySpawnWithTable(p, t)
	}
	_, _, _, sourcePools := originalHumanSpawn(p, t)
	if p.Saved != nil && !sourcePools {
		health, d.HealthMax = p.Saved.HP, p.Saved.MaxHP
		mana, d.ManaMax = p.Saved.Mana, p.Saved.MaxMana
	}
	if p.PotionEffect == nil || p.Carry != nil && p.Carry.LiveLoad != nil && p.Carry.LiveLoad.Inventory.Source.Class == 2 {
		return
	}
	base := sim.Entity{HP: health, MaxHP: d.HealthMax, Mana: mana, MaxMana: d.ManaMax,
		Absorption: d.Combat.Absorption, HealthRegeneration: d.HealthRegeneration, ManaRegeneration: d.ManaRegeneration}
	if projected, ok := sim.ProjectPotionEffect(base, *p.PotionEffect); ok {
		d.Combat.Absorption = projected.Absorption
		d.HealthRegeneration = projected.HealthRegeneration
		d.ManaRegeneration = projected.ManaRegeneration
	}
	return
}

func initializePotions(w *sim.World, party []PartyMember, st Start, table *Table) {
	for i, p := range party {
		if i >= len(st.IDs) {
			break
		}
		d, _, _ := PartySpawnWithTable(p, table)
		w.SetPotionHeadroom(st.IDs[i], PotionHeadroom(p.Hero, d))
		if p.PotionEffect == nil {
			continue
		}
		// Only an earlier build's town use stores one; it is attached as
		// written and dropTownPotionRecords removes it.
		if p.Carry != nil && p.Carry.LiveLoad != nil && p.Carry.LiveLoad.Inventory.Source.Class == 2 {
			w.RestoreAppliedPotionEffect(st.IDs[i], *p.PotionEffect)
		} else if p.Carry != nil && p.Carry.NativeHistory != nil {
			w.RestoreCarriedPotionEffect(st.IDs[i], *p.PotionEffect)
		} else {
			w.RestorePotionEffect(st.IDs[i], *p.PotionEffect)
		}
	}
}

// dropTownPotionRecords expires every carried potion record before the first
// tick: no town potion reaches a mission (DIV-1691).
func dropTownPotionRecords(w *sim.World, party []PartyMember, st Start) {
	for i, p := range party {
		if i < len(st.IDs) && p.PotionEffect != nil {
			w.DropPotionEffect(st.IDs[i])
		}
	}
}

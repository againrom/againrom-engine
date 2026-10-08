package mapload

import (
	"fmt"

	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// siegeEntity mints the Entity a hired Catapult or Ballista (tavern type 1 or
// 2) enters a mission as. The two types are Units-table creatures
// (MERC-LEVEL-005): the Units row supplies their combat, domain, footprint and
// equipment, so the block resolves through the same placement path as a map
// creature. The caller sets the identity, the cell and the saved map unit id.
func siegeEntity(p PartyMember, t *Table) (sim.Entity, spawnBlock, error) {
	b, err := blockFor(alm.Unit{ClassID: int16(p.Class), ClassSubID: uint16(p.FigureFace)}, t, DifficultyNormal)
	if err != nil {
		return sim.Entity{}, spawnBlock{}, err
	}
	spellID, _ := SpellIDByToken(t, b.combat.SpellName)
	weaponSource := sim.WeaponSpellNone
	if itemSpell, itemPower, ok := b.worn[0].CastSpell(); ok {
		spellID, b.combat.SpellPower = itemSpell, itemPower
		weaponSource = sim.WeaponSpellItem
	} else if spellID != 0 || b.combat.SpellPower != 0 {
		weaponSource = sim.WeaponSpellInnate
	}
	return sim.Entity{
		Class: p.Class, Owner: sim.SelfSlot,
		HP: b.health, MaxHP: b.health, Domain: b.domain, Speed: b.speed,
		Capacity:   b.capacity,
		Protection: b.protection, Resistance: b.resistance, TokenSize: b.tokenSize,
		ScanRange: b.sight, SeeInvisible: b.seeInvisible, DyingTime: b.dying,
		Reach: reachOf(b.combat.Reach), AttackCharge: b.combat.AttackChargeTime,
		AttackRelax: b.combat.AttackRelaxTime, ToHit: b.combat.ToHit,
		Humanoid:    b.humanoid,
		NativeBasis: b.nativeBasis,
		Defence:     b.combat.Defence, Absorption: b.combat.Absorption,
		DamageBase: b.combat.DamageBase, DamageSpread: b.combat.DamageSpread,
		AlwaysHits: b.combat.AlwaysHits, WeaponSpell: spellID,
		WeaponSpellLevel: b.combat.SpellPower, WeaponSpellSource: weaponSource,
		Mana: b.mana, MaxMana: b.manaMax,
		HealthRegenPeriod: b.healthPeriod, ManaRegenPeriod: b.manaPeriod,
		HealthRegeneration: b.healthRegeneration, ManaRegeneration: b.manaRegeneration,
		RotationSpeed:   b.rotationSpeed,
		SecondaryDamage: simSecondaryDamage(b.secondaryDamage),
		KnownSpells:     b.knownSpells, Book: b.book, XPValue: b.xpValue, Reaction: b.reaction,
		Mind: b.mind, Spirit: b.spirit, XPSlot: uint8(b.combat.SkillSlot),
		GainsXP: b.gainsXP, TypeID: b.typeID, GoldChance: b.goldChance,
		TreasureMin: b.treasureMin, TreasureMax: b.treasureMax,
		Skill: b.skill, SkillXP: b.skillXP,
		SuppressCorpseLoot: b.suppressCorpseLoot,
	}, b, nil
}

// SiegeHire is a hired Catapult or Ballista as the constructor builds it: the
// actor basis (token row, face, type, tracked blocks) of the Units row and the
// worn set that row arms.
type SiegeHire struct {
	Basis sim.Entity
	Worn  [sim.EquipSlots]sim.ItemInstance
}

// SiegeHireActor constructs the hire's actor basis through the same
// constructor a map creature takes (ConstructActorBasis over the resolved
// Units row), then completes its load from the row's own worn set. A Unit
// carries own weight equal to its load.
func SiegeHireActor(p PartyMember, t *Table, key uint32) (SiegeHire, error) {
	if p.MercenaryType != 1 && p.MercenaryType != 2 {
		return SiegeHire{}, fmt.Errorf("mapload: tavern type %d is not a siege hire", p.MercenaryType)
	}
	e, b, err := siegeEntity(p, t)
	if err != nil {
		return SiegeHire{}, err
	}
	placement := alm.Unit{ClassID: int16(p.Class), ClassSubID: uint16(p.FigureFace)}
	// A generated binding needs a nonzero identity and runtime id; the SAVE
	// writer replaces both with the document's own.
	if key == 0 {
		key = 1
	}
	basis, _, err := ConstructActorBasis(e, p, &placement, t, key, 1, nil, 0)
	if err != nil {
		return SiegeHire{}, err
	}
	basis = ConstructActorLoad(basis, b.worn, nil)
	own := basis.ActorLoad.Source.Stats
	if own[5] == 0 {
		own[5] = own[6]
	}
	basis.ActorLoad.Source.Stats = own
	return SiegeHire{Basis: basis, Worn: cloneItemEquipment(b.worn)}, nil
}

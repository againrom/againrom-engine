package mapload

import "againrom/pkg/sim"

// actorDefinition is the resolved row as sim.NewActor takes it, for a map
// placement, a siege hire and a cheat spawn alike. The weapon spell is the
// one its combat block names through t; the worn weapon marks it as the
// item's only when the item casts that same spell at that same power.
func (b spawnBlock) actorDefinition(t *Table) sim.ActorDefinition {
	spellID, _ := SpellIDByToken(t, b.combat.SpellName)
	source := sim.WeaponSpellNone
	if spellID != 0 || b.combat.SpellPower != 0 {
		if itemSpell, itemPower, cast := b.worn[0].CastSpell(); cast && itemSpell == spellID && itemPower == b.combat.SpellPower {
			source = sim.WeaponSpellItem
		} else {
			source = sim.WeaponSpellInnate
		}
	}
	return sim.ActorDefinition{
		Class: b.class, TypeID: b.typeID, Humanoid: b.humanoid, Domain: b.domain,
		HP: b.health, MaxHP: b.health, Mana: b.mana, MaxMana: b.manaMax,
		HealthRegenPeriod: b.healthPeriod, ManaRegenPeriod: b.manaPeriod,
		HealthRegeneration: b.healthRegeneration, ManaRegeneration: b.manaRegeneration,
		Speed: b.speed, RotationSpeed: b.rotationSpeed, Capacity: b.capacity,
		ScanRange: b.sight, SeeInvisible: b.seeInvisible,
		Reach: reachOf(b.combat.Reach), TokenSize: b.tokenSize,
		DyingTime: b.dying, Withdraw: b.withdraw, Wimpy: b.wimpy,
		ToHit: b.combat.ToHit, Defence: b.combat.Defence, Absorption: b.combat.Absorption,
		DamageBase: b.combat.DamageBase, DamageSpread: b.combat.DamageSpread,
		SecondBase: b.combat.SecondBase, SecondSpread: b.combat.SecondSpread,
		SecondaryDamage: simSecondaryDamage(b.secondaryDamage), AlwaysHits: b.combat.AlwaysHits,
		AttackCharge: b.combat.AttackChargeTime, AttackRelax: b.combat.AttackRelaxTime,
		Protection: b.protection, Resistance: b.resistance,
		WeaponSpell: spellID, WeaponSpellLevel: b.combat.SpellPower, WeaponSpellSource: source,
		KnownSpells: b.knownSpells, Book: b.book, CreatureSpells: b.creatureSpells,
		XPValue: b.xpValue, Reaction: b.reaction, Mind: b.mind, Spirit: b.spirit,
		XPSlot: uint8(b.combat.SkillSlot), GainsXP: b.gainsXP,
		GoldChance: b.goldChance, TreasureMin: b.treasureMin, TreasureMax: b.treasureMax,
		Skill: b.skill, SkillXP: b.skillXP, SuppressCorpseLoot: b.suppressCorpseLoot,
		NativeBasis: b.nativeBasis, NativeClass: b.nativeClass,
	}
}

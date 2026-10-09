package mapload

import (
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// checkActor compares every field sim.ActorDefinition and sim.ActorPlacement
// name against got, by name. Each origin witness states want from the census
// source of each field, so a field the constructor drops, or an origin stops
// passing, fails where its source is nonzero.
func checkActor(t *testing.T, got sim.Entity, want sim.ActorDefinition, at sim.ActorPlacement, least int) {
	t.Helper()
	g := reflect.ValueOf(got)
	nonzero := 0
	for _, in := range []reflect.Value{reflect.ValueOf(want), reflect.ValueOf(at)} {
		for i := 0; i < in.NumField(); i++ {
			name := in.Type().Field(i).Name
			if !in.Field(i).IsZero() {
				nonzero++
			}
			if f := g.FieldByName(name); !reflect.DeepEqual(f.Interface(), in.Field(i).Interface()) {
				t.Errorf("%s = %+v, want %+v", name, f.Interface(), in.Field(i).Interface())
			}
		}
	}
	if nonzero < least {
		t.Fatalf("the witness states only %d nonzero fields; its fixture is too thin to catch a dropped field", nonzero)
	}
}

func originPlacement() alm.Unit {
	return alm.Unit{X: 9<<8 | 128, Y: 11<<8 | 128, ClassID: 64, ClassSubID: 2, Owner: 3, GroupID: 4, UnitID: 99}
}

// blockDefinition is the census mapping of a resolved block, written out
// field by field rather than through spawnBlock.actorDefinition.
func blockDefinition(b spawnBlock, spell uint16, source sim.WeaponSpellSource) sim.ActorDefinition {
	return sim.ActorDefinition{Class: b.class, TypeID: b.typeID, Humanoid: b.humanoid, Domain: b.domain,
		HP: b.health, MaxHP: b.health, Mana: b.mana, MaxMana: b.manaMax,
		HealthRegenPeriod: b.healthPeriod, ManaRegenPeriod: b.manaPeriod,
		HealthRegeneration: b.healthRegeneration, ManaRegeneration: b.manaRegeneration,
		Speed: b.speed, RotationSpeed: b.rotationSpeed, Capacity: b.capacity,
		ScanRange: b.sight, SeeInvisible: b.seeInvisible, Reach: uint8(b.combat.Reach), TokenSize: b.tokenSize,
		DyingTime: b.dying, Withdraw: b.withdraw, Wimpy: b.wimpy,
		ToHit: b.combat.ToHit, Defence: b.combat.Defence, Absorption: b.combat.Absorption,
		DamageBase: b.combat.DamageBase, DamageSpread: b.combat.DamageSpread,
		SecondaryDamage: simSecondaryDamage(b.secondaryDamage), AlwaysHits: b.combat.AlwaysHits,
		AttackCharge: b.combat.AttackChargeTime, AttackRelax: b.combat.AttackRelaxTime,
		Protection: b.protection, Resistance: b.resistance,
		WeaponSpell: spell, WeaponSpellLevel: b.combat.SpellPower, WeaponSpellSource: source,
		KnownSpells: b.knownSpells, Book: b.book, CreatureSpells: b.creatureSpells,
		XPValue: b.xpValue, Reaction: b.reaction, Mind: b.mind, Spirit: b.spirit,
		XPSlot: uint8(b.combat.SkillSlot), GainsXP: b.gainsXP,
		GoldChance: b.goldChance, TreasureMin: b.treasureMin, TreasureMax: b.treasureMax,
		Skill: b.skill, SkillXP: b.skillXP, SuppressCorpseLoot: b.suppressCorpseLoot,
		NativeBasis: b.nativeBasis, NativeClass: b.nativeClass}
}

// Census row "map placement": the resolved block at the mission difficulty,
// the record's cell, owner, group and map unit id.
func TestMapPlacementActorCarriesItsCensusFields(t *testing.T) {
	table := cheatActorTable()
	u := originPlacement()
	w, _, err := FromALMRoster(&alm.Map{Width: 40, Height: 40, Units: []alm.Unit{u}}, table, DifficultyHard)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := w.Entity(0)
	b, err := blockFor(u, table, DifficultyHard)
	if err != nil {
		t.Fatal(err)
	}
	checkActor(t, got, blockDefinition(b, 0, sim.WeaponSpellNone),
		sim.ActorPlacement{X: 9, Y: 11, Owner: 3, Group: 4, MapUnitID: 99}, 20)
}

// Census row "siege hire": the same block at normal difficulty, the player's
// slot, the start cell and the saved map unit id; no group.
func TestSiegeHireActorCarriesItsCensusFields(t *testing.T) {
	table := cheatActorTable()
	p := PartyMember{Name: "Siege", MercenaryType: 1, Class: 64, FigureFace: 2}
	w, st, err := StartMission(&alm.Map{Width: 40, Height: 40}, table, DifficultyHard, []PartyMember{p})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := w.Entity(st.IDs[0])
	b, err := blockFor(alm.Unit{ClassID: 64, ClassSubID: 2}, table, DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	checkActor(t, got, blockDefinition(b, 0, sim.WeaponSpellNone),
		sim.ActorPlacement{ID: st.IDs[0], X: st.Cells[0].X, Y: st.Cells[0].Y, Owner: sim.SelfSlot}, 20)
	hire, err := SiegeHireActor(p, table, 5)
	if err != nil {
		t.Fatal(err)
	}
	if hire.Basis.MaxHP != b.health || hire.Basis.TypeID != b.typeID || hire.Basis.Owner != sim.SelfSlot || hire.Basis.SourceBinding.Identity != 5 {
		t.Fatalf("SAV hire basis is not the same definition: %+v", hire.Basis)
	}
}

// Census row "party": a generated member's own fold, pools and spellbook, the
// player's slot and the start cell; no group and no row-borne dying time,
// sight-through-invisibility, retreat thresholds or treasure.
func TestPartyActorCarriesItsCensusFields(t *testing.T) {
	table := cheatActorTable()
	sword := data.Weapon{Name: "Blade", DamageBase: 3, DamageSpread: 2, AttackType: data.SkillBlade,
		ChargeTime: 6, RelaxTime: 4, Range: 2}
	p := PartyMember{Name: "Hero", Class: 100, Hero: data.NewCampaignHero(data.SkillBlade), Weapon: &sword,
		KnownSpells: 6, SuppressCorpseLoot: true, Profile: data.Profile{HealthColumn: true, ManaColumn: true}}
	w, st, err := StartMission(&alm.Map{Width: 40, Height: 40}, table, DifficultyHard, []PartyMember{p})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := w.Entity(st.IDs[0])
	p = OwnParty([]PartyMember{p})[0]
	d, hp, mana := PartySpawnWithTable(p, table)
	pool := restoredPools(p, hp, mana, data.UnitDefaults())
	worn := MemberItemEquipment(p, table)
	reward := p.Hero.Reward()
	typeID := sim.HeroTypeID(p.Mage, data.FigureDir(p.FigureDir).Female())
	want := sim.ActorDefinition{Class: 100, TypeID: typeID, Humanoid: true, Domain: sim.DomainGround,
		HP: pool.hp, MaxHP: pool.maxHP, Mana: pool.mana, MaxMana: pool.maxMana,
		HealthRegenPeriod: pool.healthPeriod, ManaRegenPeriod: pool.manaPeriod,
		HealthRegeneration: d.HealthRegeneration, ManaRegeneration: d.ManaRegeneration,
		Speed: d.Speed, RotationSpeed: d.RotationSpeed, Capacity: d.Capacity,
		ScanRange: uint8(d.Sight), Reach: uint8(d.Combat.Reach), TokenSize: 1,
		ToHit: d.Combat.ToHit, Defence: d.Combat.Defence, Absorption: d.Combat.Absorption,
		DamageBase: d.Combat.DamageBase, DamageSpread: d.Combat.DamageSpread,
		SecondBase: d.Combat.SecondBase, SecondSpread: d.Combat.SecondSpread,
		SecondaryDamage: simSecondaryDamage(d.SecondaryDamage), AlwaysHits: d.Combat.AlwaysHits,
		AttackCharge: d.Combat.AttackChargeTime, AttackRelax: d.Combat.AttackRelaxTime,
		Protection: d.Protection, Resistance: data.DamageKindResistance(d.Resistance),
		KnownSpells: 6, Book: p.Book,
		Reaction: d.Reaction, Mind: reward.Mind, Spirit: d.Spirit,
		XPSlot: uint8(d.Combat.SkillSlot), GainsXP: sim.InPersistBand(typeID),
		Skill: d.Skill, SkillXP: reward.SkillXP, SuppressCorpseLoot: true,
		NativeBasis:    nativeInitialModifier(nativeInitialBase(&p.Hero.Skill), &worn, table, p.Hero.Skill[0], true, false).WithBody(uint16(d.Body)),
		NativeClass:    sim.NativeClass{Present: true},
		NativeTraining: sim.NativeTraining{Present: true, Levels: p.Hero.Skill}}
	checkActor(t, got, want, sim.ActorPlacement{ID: st.IDs[0], X: st.Cells[0].X, Y: st.Cells[0].Y, Owner: sim.SelfSlot}, 25)
}

// Census row "SAV seed": the saved binding's type, and its Units row's
// policy fields the saved record does not carry.
func TestSavedUnitSeedCarriesItsCensusFields(t *testing.T) {
	table := cheatActorTable()
	params := cheatActorParams(41, map[int]int32{0: 23, 4: 80, 9: 6, 29: 64, 30: 2, 33: 14, 34: 7, 35: 3, 36: 2, 37: 50, 38: 20, 39: 5, 40: 9})
	table.Units = ghostResistanceCollection{{}, {}, {name: "Seeded", params: params}}
	got, err := SourceActorSeed(sim.SourceBinding{Class: 4, TokenRow: 2, TypeID: 64, Face: 2}, table)
	if err != nil {
		t.Fatal(err)
	}
	d, err := data.NewUnitDef("Seeded", params)
	if err != nil {
		t.Fatal(err)
	}
	want := sim.ActorDefinition{Class: 64, TypeID: 64, RotationSpeed: d.RotationSpeed, DyingTime: d.DyingTime,
		Withdraw: d.Withdraw, Wimpy: d.Wimpy, SeeInvisible: uint8(d.SeeInvisible), XPValue: d.XPValue,
		GoldChance: d.GoldChance, TreasureMin: d.TreasureMin, TreasureMax: d.TreasureMax, AlwaysHits: d.AlwaysHits}
	checkActor(t, got, want, sim.ActorPlacement{}, 10)
}

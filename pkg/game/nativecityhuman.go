package game

import (
	"encoding/binary"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

const nativeCityManaReserve = 95

// Legacy construction entry point. Current saves use applyCurrentCityHuman.
func nativeCityHumanState(member mapload.PartyMember, table *mapload.Table, unit sav.CityUnitData, approximate bool) (data.HumanState, error) {
	if member.OriginalHuman != nil && !approximate {
		return data.HumanState{}, originalCityUnsupportedf("native city retained Human basis requires source continuity; use lossless .ags")
	}
	if member.PotionEffect != nil {
		return data.HumanState{}, originalCityUnsupportedf("native city timed potion requires lossless .ags")
	}
	d, hp, mp := mapload.PartyDisplayWithTable(member, table)
	h, err := nativeCityHumanFromDerived(member, table, unit, d, hp, mp)
	if err != nil {
		return data.HumanState{}, err
	}
	if member.Carry != nil && member.Carry.LiveLoad != nil {
		load := member.Carry.LiveLoad
		if err := load.Validate(); err != nil {
			return data.HumanState{}, originalCityUnsupportedf("native city retained Human basis requires source continuity; use lossless .ags")
		}
		// HumanSourceActor has no data.HumanState input for these four:
		// ConstructActorBasis (actorconstructor.go) sets them straight from
		// the live entity, from the SAME d.Combat this function's own
		// PartyDisplayWithTable call above already produced (Reach/
		// AttackChargeTime/AttackRelaxTime), plus the constant true it
		// writes for every actor that has gone through construction at all.
		rebuilt := mapload.HumanSourceActor(h, 2)
		rebuilt.Reach, rebuilt.AttackCharge, rebuilt.AttackRelax, rebuilt.EquipmentRuntimePresent =
			uint8(d.Combat.Reach), uint8(d.Combat.AttackChargeTime), uint8(d.Combat.AttackRelaxTime), true
		// TypeID round-trips within its OWN document's numbering convention
		// (nativeCityUnitData's city-token construction here; whatever the
		// mission document used when this member's basis was first built);
		// the two need not agree and nothing downstream reads
		// SourceActor.TypeID across that boundary (h.TypeID, a different,
		// city-local copy, is what nativeCityApplyHumanState actually uses).
		retained := load.Inventory.Source
		retained.TypeID, rebuilt.TypeID = 0, 0
		if retained != rebuilt && !approximate {
			return data.HumanState{}, originalCityUnsupportedf("native city retained Human basis requires source continuity; use lossless .ags")
		}
	}
	return h, nil
}

// snapshotHumanField is a Human field a save takes from the loaded document
// because the engine does not model it.
type snapshotHumanField struct{ Field, Reason string }

// snapshotHumanFields is the whole list of Human fields a save still takes
// from the loaded document. Every other field the engine models is written
// from the current state; the retained document state is used for the whole
// record only while every modelled field still equals it
// (PartyMember.OriginalHumanState).
var snapshotHumanFields = []snapshotHumanField{
	{"Attack.Tail", "opaque words of the attack block with no decoded meaning"},
	{"Base.Tail", "opaque words of the base block with no decoded meaning"},
	{"Modifier.Attack.Tail", "opaque words of the modifier attack block with no decoded meaning"},
	{"TypeID", "definition identity, never changed in play"},
	{"ManaReservePercent", "per-player setting owned by the loaded owner record"},
}

// Opaque tails have identity ownership, independent of the retained arithmetic
// guard. A current actor load supersedes the member basis, including zero bytes.
func currentCityHumanTails(member mapload.PartyMember) ([3][2]byte, bool) {
	if h, _, ok := currentCityHuman(member); ok {
		return [3][2]byte{h.Attack.Tail, h.Base.Tail, h.Modifier.Attack.Tail}, true
	}
	if h := member.OriginalHuman; h != nil && h.Version == 1 && member.ID != "" && h.PartyID == member.ID {
		return [3][2]byte{h.State.Attack.Tail, h.State.Base.Tail, h.State.Modifier.Attack.Tail}, true
	}
	return [3][2]byte{}, false
}

func nativeCityHumanFromDerived(member mapload.PartyMember, table *mapload.Table, unit sav.CityUnitData, d data.Derived, hp, mp int32) (data.HumanState, error) {
	periods := data.UnitDefaults()
	// SAV-HUMEQUIP-447
	h := data.HumanState{Body: uint16(d.Body), Reaction: uint16(d.Reaction), Mind: uint16(d.Mind), Spirit: uint16(d.Spirit),
		Health: uint16(hp), HealthMax: uint16(d.HealthMax), Mana: uint16(mp), ManaMax: uint16(d.ManaMax),
		HealthPeriod: uint16(periods.HealthRegenPeriod), ManaPeriod: uint16(periods.ManaRegenPeriod),
		Speed: uint16(d.Speed), Capacity: uint16(d.Capacity), Fighter: member.Profile.Fighter, HasSpellbook: unit.SpellbookFlag != 0,
		TypeID: binary.LittleEndian.Uint16(unit.Token[17:]), HasOwner: true, ManaReservePercent: nativeCityManaReserve,
		InventoryWeight: int32(unit.ContainerTails[1]), Experience: uint32(d.Experience)}
	if member.Saved != nil {
		h.HealthPeriod, h.ManaPeriod = uint16(member.Saved.HealthRegenPeriod), uint16(member.Saved.ManaRegenPeriod)
	}
	// ManaReservePercent is a per-Player setting, not a per-Human one
	// (sav/actorgraph.go: "a.Character.Basis.Human.ManaReservePercent =
	// a.Owner.PlayerF58"); nativeCityManaReserve above is only the right
	// value for an actor with no established basis to inherit it from yet.
	// A member who already carries one — CarryParty's own Carry.LiveLoad,
	// set once a mission has actually run — keeps the SAME value gameplay
	// already established, the same way member.Saved above already
	// overrides the two regen periods.
	if member.Carry != nil && member.Carry.LiveLoad != nil {
		h.ManaReservePercent = member.Carry.LiveLoad.Inventory.Source.ManaReservePercent
	}
	for _, item := range mapload.MemberItemEquipment(member, table) {
		if item.Code != 0 {
			h.Weight += uint16(mapload.SourceConstructedItem(item, table).Weight)
		}
	}
	loadout := mapload.PartyLoadout(member, table)
	mapload.ApplyItemEffects(&loadout, mapload.MemberItemEquipment(member, table), h.Fighter)
	for i, v := range []int32{loadout.Mod.Body, loadout.Mod.Reaction, loadout.Mod.Mind, loadout.Mod.Spirit} {
		h.Modifier.StatCap[i] = int8(v)
	}
	for i := range h.Attack.Skill {
		h.Attack.Skill[i] = uint16(d.Skill[i])
		if i != 0 {
			h.Base.Skill[i] = uint16(member.Hero.Skill[i])
		}
		h.Modifier.Attack.Skill[i] = uint16(loadout.Mod.SkillBonus[i])
		h.SkillXP[i] = uint32(d.SkillXP[i])
	}
	c := d.Combat
	h.Attack.ToHit, h.Attack.DamageBase, h.Attack.DamageSpread, h.Attack.Active = uint16(c.ToHit), uint8(c.DamageBase), uint8(c.DamageSpread), uint8(c.SkillSlot)
	h.Attack.SecondBase, h.Attack.SecondSpread = c.SecondBase, c.SecondSpread
	h.Attack.ElementalBase, h.Attack.ElementalSpread = c.SecondaryDamage.Base, c.SecondaryDamage.Spread
	if h.Attack.ElementalBase != 0 || h.Attack.ElementalSpread != 0 {
		for i, selector := range data.ElementalSelectorOrder {
			if selector == c.SecondaryDamage.Selector {
				h.Attack.ElementalKind = uint8(i + 1)
			}
		}
	}
	h.Defence.Defence, h.Defence.Absorption = uint16(c.Defence), uint16(c.Absorption)
	for i := range d.Protection {
		h.Defence.Protection[i+1], h.Defence.Resistance[i+1] = uint16(d.Protection[i]), uint8(d.Resistance[i])
	}
	// Keep the actual equipment/effect contributions and maintained skills.
	// Derive below checks this basis; it does not invent a modifier by
	// subtracting two final projections or reconstruct source history.
	weapon := data.FoldWeapon(data.Combat{}, loadout.Weapon, member.Hero.Skill[0])
	m := &h.Modifier
	m.Speed = uint16(loadout.Mod.Speed)
	m.HealthMax, m.ManaMax = uint16(loadout.Mod.HealthMax), uint16(loadout.Mod.ManaMax)
	m.Sight = uint16(loadout.Mod.Sight * 256)
	m.HealthRegeneration, m.ManaRegeneration = uint16(d.HealthRegeneration), uint16(d.ManaRegeneration)
	m.Attack.ToHit = uint16(loadout.Mod.ToHit + weapon.ToHit)
	m.Attack.DamageBase, m.Attack.DamageSpread = uint8(loadout.Mod.DamageBase+weapon.DamageBase), uint8(loadout.Mod.DamageSpread+weapon.DamageSpread)
	m.Attack.SecondBase, m.Attack.SecondSpread = h.Attack.SecondBase, h.Attack.SecondSpread
	m.Attack.ElementalBase, m.Attack.ElementalSpread, m.Attack.ElementalKind = h.Attack.ElementalBase, h.Attack.ElementalSpread, h.Attack.ElementalKind
	m.Defence.Defence, m.Defence.Absorption = uint16(loadout.Mod.Defence+weapon.Defence), uint16(loadout.Mod.Absorption+weapon.Absorption)
	for i := range loadout.Mod.Protection {
		m.Defence.Protection[i+1] = uint16(loadout.Mod.Protection[i])
		m.Defence.Resistance[i+1] = uint8(loadout.Mod.Resistance[i])
	}
	if tails, present := currentCityHumanTails(member); present {
		h.Attack.Tail, h.Base.Tail, h.Modifier.Attack.Tail = tails[0], tails[1], tails[2]
	}
	next, err := h.Derive()
	if err != nil {
		return h, originalCityUnsupportedf("native city Human derive: %v", err)
	}
	// Stored combat and pool words remain current even outside the clamps a
	// later derive applies. SAVE must not normalize them or refuse the state.
	// These stored values include original load penalties and the fractional
	// sight word. The native preview exposes their unladen/integer forms.
	h.Speed, h.Load, h.Sight, h.ManaFloor, h.MoverSpeed = next.Speed, next.Load, next.Sight, next.ManaFloor, next.MoverSpeed
	return h, nil
}

func nativeCityApplyHuman(unit *sav.CityUnitData, member mapload.PartyMember, table *mapload.Table) error {
	return nativeCityApplyHumanDerived(unit, member, table, false)
}

// nativeCityApplyHumanApproximate is DIV-1320/DIV-1321's fallback shape of
// nativeCityApplyHuman: same signature, so nativeCityDataConstruct's existing
// injection point runs it unchanged, but a retained original or unreproducible
// mission Human basis writes the live-derived approximation instead of
// refusing (nativeCityHumanState's own doc comment).
func nativeCityApplyHumanApproximate(unit *sav.CityUnitData, member mapload.PartyMember, table *mapload.Table) error {
	return nativeCityApplyHumanDerived(unit, member, table, true)
}

func nativeCityApplyHumanDerived(unit *sav.CityUnitData, member mapload.PartyMember, table *mapload.Table, approximate bool) error {
	h, err := nativeCityHumanState(member, table, *unit, approximate)
	if err != nil {
		return err
	}
	if err := nativeCityApplyHumanState(unit, member, h); err != nil {
		return err
	}
	d, _, _ := mapload.PartyDisplayWithTable(member, table)
	unit.Scalar2[34], unit.Scalar2[39], unit.Scalar2[40] = byte(d.Combat.Reach), byte(d.Combat.AttackChargeTime), byte(d.Combat.AttackRelaxTime)
	return nil
}

func nativeCityApplyHumanState(unit *sav.CityUnitData, member mapload.PartyMember, h data.HumanState) error {
	_, face := memberFigure(member)
	if face < 1 || face > 127 {
		return originalCityUnsupportedf("native city %s has no valid portrait face", member.Name)
	}
	unit.Scalar1[2] = byte(face)
	binary.LittleEndian.PutUint16(unit.Token[17:], h.TypeID)
	if h.TypeID < 0x1a && data.FigureDir(member.FigureDir).Female() {
		unit.Scalar1[2] |= 0x80 // UNIT-PICT-035's separate table-Human arm.
	}
	if !h.Fighter {
		unit.Scalar1[3] |= 6 // HERO-HP-071; the drawable class uses TypeID.
	}
	if h.HasSpellbook {
		unit.Scalar1[3] |= 2 // UNIT-SPELL-007's independent book allocation bit.
	}
	u := cityHumanUpdate(sav.CityCharacter{}, h)
	for i, v := range u.Stats {
		binary.LittleEndian.PutUint16(unit.Scalar2[i*2:], v)
	}
	for i, v := range u.SkillXP {
		binary.LittleEndian.PutUint32(unit.XP[i*4:], v)
	}
	binary.LittleEndian.PutUint32(unit.Scalar2[35:], u.Experience)
	copy(unit.RawA6, u.Human.Attack[:])
	copy(unit.Raw114, u.Human.Base[:])
	copy(unit.RawBE, u.Human.Defence[:])
	copy(unit.RawD4, u.Human.Modifier[:])
	binary.LittleEndian.PutUint16(unit.Scalar2[30:], h.ManaFloor)
	binary.LittleEndian.PutUint16(unit.Scalar2[32:], h.Sight)
	unit.Raw154[10] = h.MoverSpeed
	return nil
}

func nativeCityDiary(table *mapload.Table) sav.CityDiaryData {
	n := 0
	if table != nil && table.Units != nil {
		n = table.Units.Len()
	}
	d := sav.CityDiaryData{DWords: make([]uint32, n), Words: make([]uint16, n), Reference: nativeCityPlayerIdentity}
	for i := range d.Words {
		d.Words[i] = 0x400 // SAV-667's fresh constructor defaults.
	}
	return d
}

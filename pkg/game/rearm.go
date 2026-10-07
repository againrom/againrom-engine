package game

import (
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// characterDerive pairs one canonical actor id with the loader inputs the
// simulation byte form intentionally does not duplicate. The slice is sorted
// by id when installed, so simultaneous raises are folded in a stable order.
type characterDerive struct {
	id     sim.EntityID
	member mapload.PartyMember
	table  *mapload.Table
}

type derivedSkillState struct {
	levels   [data.SkillSlots]int32
	training sim.NativeTraining
}

func (mw *mapWorld) installCharacterDerivations(ms *Mission, t *mapload.Table) {
	seen := make(map[sim.EntityID]bool)
	n := len(ms.Party)
	if len(ms.Start.IDs) < n {
		n = len(ms.Start.IDs)
	}
	for i := 0; i < n; i++ {
		id := ms.Start.IDs[i]
		mw.derives = append(mw.derives, characterDerive{id: id, member: ms.Party[i], table: t})
		seen[id] = true
	}
	for id, member := range ms.Start.Roster {
		if !seen[id] {
			mw.derives = append(mw.derives, characterDerive{id: id, member: member, table: t})
		}
	}
	slices.SortFunc(mw.derives, func(a, b characterDerive) int {
		switch {
		case a.id < b.id:
			return -1
		case a.id > b.id:
			return 1
		default:
			return 0
		}
	})
	mw.derivedSkills = make(map[sim.EntityID]derivedSkillState, len(mw.derives))
	mw.skillBonus = make(map[sim.EntityID][data.SkillSlots]int32, len(mw.derives))
	mw.derivedPotions = make(map[sim.EntityID][4]int32, len(mw.derives))
	for _, c := range mw.derives {
		if e, ok := mw.entity(c.id); ok {
			mw.derivedSkills[c.id] = derivedSkillState{e.Skill, e.NativeTraining}
			mw.derivedPotions[c.id] = e.PotionStats
			if items, ok := mw.world.EquippedItems(c.id); ok && e.ActorLoad.Source.Class == 0 {
				bonus := mapload.EquippedSkillBonus(items, c.member.Profile.Fighter)
				mw.skillBonus[c.id] = bonus
				// A save holds the levels the original writes, base plus bonus
				// held to the original cap. The level above it is derived
				// here, and the first post-step recompute carries it into the
				// combat block.
				if levels, lifted := liftedSkills(e.Skill, c.member.Hero.Skill, bonus, mw.world.Rules()); lifted && mw.world.SetSkillLevels(c.id, levels) {
					mw.derivedSkills[c.id] = derivedSkillState{}
				}
			}
			if e.ActorLoad.Source.Class != 0 {
				mw.refreshSourceCharacter(e)
			} else if e.PotionStats != ([4]int32{}) {
				// A native mission retains its entry PartyMember and stores
				// earned gains in the world. Seed the first visible sheet from
				// both without running SetDerived or reapplying the potion.
				p := c.member
				p.Hero = mapload.PotionHero(p.Hero, e.PotionStats)
				mw.chars[c.id] = partyPanelSubject(p, c.table).Char
			}
		}
	}
}

// recomputeRaisedSkills folds canonical awards into each live derived sheet.
func (mw *mapWorld) recomputeRaisedSkills() {
	for _, c := range mw.derives {
		e, ok := mw.entity(c.id)
		if ok && e.ActorLoad.Source.Class != 0 {
			mw.refreshSourceCharacter(e)
			mw.derivedSkills[c.id], mw.derivedPotions[c.id] = derivedSkillState{e.Skill, e.NativeTraining}, e.PotionStats
			continue
		}
		if !ok || (derivedSkillState{e.Skill, e.NativeTraining}) == mw.derivedSkills[c.id] && e.PotionStats == mw.derivedPotions[c.id] {
			continue
		}
		items, ok := mw.world.EquippedItems(c.id)
		if !ok {
			continue
		}
		if mw.world.NativeTrainingNeedsProducer(c.id) && e.PotionStats == mw.derivedPotions[c.id] &&
			e.NativeTraining == mw.derivedSkills[c.id].training && mapload.EquippedSkillBonus(items, c.member.Profile.Fighter) == mw.skillBonus[c.id] {
			mw.derivedSkills[c.id] = derivedSkillState{e.Skill, e.NativeTraining}
			continue
		}
		var eq data.Equipment
		for slot, item := range items {
			eq.SetCode(slot+1, data.ItemCode(item.Code))
		}
		// Read the current member's weapon latch; entry derivation inputs are
		// frozen and cannot record a later equip, unequip or sale.
		everEquipped := mw.resolveWeaponMaterialized(mw.missionPartyMember(c.id), eq)
		loadout, ok := mapload.ResolveItemLoadout(items, c.member.Weapon, everEquipped, c.table)
		if !ok {
			continue
		}
		loadout.RotationSpeed = mapload.RotationSpeedBase(c.member.Hired(), c.member.HiredRotationSpeed, c.member.Class, c.table)
		mapload.ApplyItemEffects(&loadout, items, c.member.Profile.Fighter)
		mapload.AddLayers(&loadout, mw.activeLayers(c.id), c.table)
		hero := mapload.PotionHero(c.member.Hero, e.PotionStats)
		hero.Skill = e.TrainedSkills(mw.skillBonus[c.id])
		d := hero.RecomputeWithSkillXP(c.member.Profile, loadout, e.SkillXP)
		// DIV-675: native modifier producers remain incomplete. Keep the
		// observed pair; do not manufacture a new one from item arithmetic.
		d.Combat.SecondBase, d.Combat.SecondSpread = e.SecondBase, e.SecondSpread
		spellID, _ := mapload.SpellIDByToken(c.table, d.Combat.SpellName)
		if e.WeaponSpellSource == sim.WeaponSpellInnate || e.WeaponSpellSource == sim.WeaponSpellLegacy {
			spellID = e.WeaponSpell
			d.Combat.SpellPower = e.WeaponSpellLevel
		}
		if !mw.world.SetDerived(c.id, simDerivedBlock(d, spellID, e.WeaponSpellSource)) {
			continue
		}
		reportOriginalProfileRetirement(mw.world, e)
		mw.settleSkillBonus(c.id, loadout.Mod.SkillBonus)
		mw.derivedSkills[c.id] = derivedSkillState{d.Skill, e.NativeTraining}
		mw.derivedPotions[c.id] = e.PotionStats
		mw.world.SetPotionHeadroom(c.id, mapload.PotionHeadroom(hero, d))
		mw.projectDerivedCharacter(c.id, d)
	}
}

// projectDerivedCharacter copies a native actor's derived sheet into the
// panel's character.
func (mw *mapWorld) projectDerivedCharacter(id sim.EntityID, d data.Derived) {
	ch, found := mw.chars[id]
	if !found {
		return
	}
	ch.Body, ch.Reaction, ch.Mind, ch.Spirit = int(d.Body), int(d.Reaction), int(d.Mind), int(d.Spirit)
	ch.Experience, ch.Sight = int(d.Experience), int(d.Sight)
	for i := range ch.Skills {
		ch.Skills[i] = int(d.Skill[i])
	}
	for i := range ch.Protection {
		ch.Protection[i], ch.Resistance[i] = int(d.Protection[i]), int(d.Resistance[i])
	}
	mw.chars[id] = ch
}

// refreshWornCharacter re-derives the inventory subject's panel character from
// the items he now wears, so a statistic an item raises or lowers reaches the
// sheet at the equip and not only at the next skill raise.
func (mw *mapWorld) refreshWornCharacter(id sim.EntityID) {
	e, ok := mw.entity(id)
	if !ok || e.ActorLoad.Source.Class != 0 {
		return
	}
	items, ok := mw.world.EquippedItems(id)
	if !ok {
		return
	}
	p := mw.invParty
	loadout, ok := mapload.ResolveItemLoadout(items, p.startWeapon, mw.invWeaponEverEquipped, p.table)
	if !ok {
		return
	}
	loadout.RotationSpeed = mapload.RotationSpeedBase(p.hired, p.hiredRotationSpeed, p.class, p.table)
	mapload.ApplyItemEffects(&loadout, items, p.profile.Fighter)
	mapload.AddLayers(&loadout, mw.activeLayers(id), p.table)
	hero := mapload.PotionHero(p.hero, e.PotionStats)
	hero.Skill = e.TrainedSkills(mw.skillBonus[id])
	mw.projectDerivedCharacter(id, hero.RecomputeWithSkillXP(p.profile, loadout, e.SkillXP))
}

// liftedSkills returns the levels of a native actor with each slot that sits at
// the original clamp raised to its base plus bonus, and whether any rose.
func liftedSkills(levels, base, bonus [data.SkillSlots]int32, r sim.Rules) ([data.SkillSlots]int32, bool) {
	lifted := false
	for j := int32(data.SkillGeneral) + 1; j < data.SkillSlots; j++ {
		if levels[j] != data.SkillCap && levels[j] != r.SkillCap() {
			continue
		}
		if want := r.EffectiveSkill(base[j], bonus[j]); want > levels[j] {
			levels[j], lifted = want, true
		}
	}
	return levels, lifted
}

// settleSkillBonus records the bonus the entity's levels now include.
func (mw *mapWorld) settleSkillBonus(id sim.EntityID, bonus [data.SkillSlots]int32) {
	if mw.skillBonus == nil {
		mw.skillBonus = make(map[sim.EntityID][data.SkillSlots]int32)
	}
	mw.skillBonus[id] = bonus
}

func simDerivedBlock(d data.Derived, spellID uint16, source sim.WeaponSpellSource) sim.DerivedBlock {
	if spellID == 0 && d.Combat.SpellPower == 0 {
		source = sim.WeaponSpellNone
	}
	return sim.DerivedBlock{
		MaxHP: d.HealthMax, MaxMana: d.ManaMax, Speed: d.Speed, ScanRange: derivedByte(d.Sight),
		Reaction: d.Reaction, Mind: d.Mind, Spirit: d.Spirit, Capacity: d.Capacity,
		Skill: d.Skill, SkillSet: true,
		HealthRegeneration: d.HealthRegeneration, ManaRegeneration: d.ManaRegeneration,
		RotationSpeed: d.RotationSpeed,
		Combat: sim.CombatBlock{
			DamageBase: d.Combat.DamageBase, DamageSpread: d.Combat.DamageSpread,
			SecondBase: d.Combat.SecondBase, SecondSpread: d.Combat.SecondSpread,
			ToHit: d.Combat.ToHit, Defence: d.Combat.Defence, Absorption: d.Combat.Absorption,
			AttackCharge: d.Combat.AttackChargeTime, AttackRelax: d.Combat.AttackRelaxTime,
			AlwaysHits: d.Combat.AlwaysHits, Reach: derivedByte(d.Combat.Reach),
			XPSlot: uint8(creditedSlot(d.Combat.SkillSlot)), WeaponSpell: spellID,
			WeaponSpellLevel: d.Combat.SpellPower, WeaponSpellSource: source, Protection: d.Protection,
			Resistance: data.DamageKindResistance(d.Resistance),
			SecondaryDamage: sim.SecondaryDamage{Base: d.SecondaryDamage.Base,
				Spread: d.SecondaryDamage.Spread, Selector: d.SecondaryDamage.Selector},
		},
	}
}

func derivedByte(v int32) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v)
}

func EquipTarget(c data.ItemCode, t *mapload.Table) (int, bool) {
	if t == nil || t.Shapes == nil || t.Materials == nil || t.Weapons == nil || t.Armors == nil {
		return 0, false
	}
	if _, err := data.WeaponFromCode(c, t.Shapes, t.Materials, t.Weapons); err == nil {
		return data.EquipSlotFor(c)
	}
	if t.Shields != nil {
		if _, err := data.ShieldFromCode(c, t.Shapes, t.Materials, t.Shields); err == nil {
			return data.EquipSlotFor(c)
		}
	}
	if piece, err := data.ArmorFromCode(c, t.Shapes, t.Materials, t.Armors); err == nil {
		return int(piece.Slot), true
	}
	return 0, false
}

// Rearm derives combat from the current owned items. An empty slot can use
// the starting weapon only before materialization. Source actors retain their
// own arithmetic; native actors receive the complete derived combat block.
func Rearm(w *sim.World, id sim.EntityID, h data.Hero, profile data.Profile, fallback *data.Weapon, everEquipped bool, t *mapload.Table, rotationBase int32) (*data.Weapon, bool) {
	return RearmWithLayers(w, id, h, profile, fallback, everEquipped, t, rotationBase, nil)
}

// RearmWithLayers is Rearm for a character who also wears clothing layers; their
// defence and absorption join the loadout's sum.
func RearmWithLayers(w *sim.World, id sim.EntityID, h data.Hero, profile data.Profile, fallback *data.Weapon, everEquipped bool, t *mapload.Table, rotationBase int32, layers []uint16) (*data.Weapon, bool) {
	items, _ := w.EquippedItems(id)
	for _, entity := range w.Entities() {
		if entity.ID == id && entity.ActorLoad.Source.Class != 0 {
			if !items[0].Empty() || everEquipped || fallback == nil {
				return mapload.CurrentItemWeapon(items[0], t), true
			}
			weapon := *fallback
			return &weapon, true
		}
	}
	loadout, ok := mapload.ResolveItemLoadout(items, fallback, everEquipped, t)
	if !ok {
		return nil, false
	}
	loadout.RotationSpeed = rotationBase
	mapload.AddLayers(&loadout, layers, t)
	return applyRearmLoadout(w, id, h, profile, loadout, items, t)
}

func applyRearmLoadout(w *sim.World, id sim.EntityID, h data.Hero, profile data.Profile,
	loadout data.Loadout, items [sim.EquipSlots]sim.ItemInstance, t *mapload.Table) (*data.Weapon, bool) {
	var before sim.Entity
	for _, e := range w.Entities() {
		if e.ID == id {
			before = e
			h = mapload.PotionHero(h, e.PotionStats)
			break
		}
	}
	weapon := loadout.Weapon
	if before.ActorLoad.Source.Class != 0 {
		return weapon, true
	}
	mapload.ApplyItemEffects(&loadout, items, profile.Fighter)
	derived := h.Recompute(profile, loadout)
	// THE SPELL, off derived.Combat.SpellName through mapload's own token-to-id
	// lookup (0139 FR-1b) and derived.Combat.SpellPower directly: the id a
	// re-armed entity releases is a fact about the INSTALLED table, exactly as
	// the weapon and the armour above already are, so it is resolved off the
	// same t this call already reads rather than carried as a string past
	// pkg/sim's determinism wall. A nil t, or one with no spell collection,
	// answers (0, false) on SpellIDByToken's own leniency — the same fold
	// this function already extends to a missing armour table above — so a
	// caller that cannot say what a spell is does not also refuse the weapon
	// and the armour it CAN resolve.
	spellID, _ := mapload.SpellIDByToken(t, derived.Combat.SpellName)
	source := sim.WeaponSpellNone
	var authoritativeSpell uint16
	var authoritativePower int32
	for _, entity := range w.Entities() {
		if entity.ID == id {
			// Same explicit retention policy as the post-award recompute.
			derived.Combat.SecondBase, derived.Combat.SecondSpread = entity.SecondBase, entity.SecondSpread
			source = entity.WeaponSpellSource
			authoritativeSpell = entity.WeaponSpell
			authoritativePower = entity.WeaponSpellLevel
			break
		}
	}
	if source == sim.WeaponSpellInnate || source == sim.WeaponSpellLegacy {
		spellID = authoritativeSpell
		derived.Combat.SpellPower = authoritativePower
	}
	if spellID != 0 || derived.Combat.SpellPower != 0 {
		if source == sim.WeaponSpellNone || source == sim.WeaponSpellItem {
			source = sim.WeaponSpellLegacy
		}
	}
	wrote := w.SetDerived(id, simDerivedBlock(derived, spellID, source))
	if wrote {
		reportOriginalProfileRetirement(w, before)
		w.SetPotionHeadroom(id, mapload.PotionHeadroom(h, derived))
	}
	return weapon, wrote
}

// creditedSlot narrows a recompute's own skill slot to one of the six an
// entity's experience array holds, answering data.SkillGeneral for anything
// else. MOVED HERE FROM world.go UNCHANGED: it belongs beside Rearm, its one
// caller, not beside the front-end code that used to hold both.
//
// THE NARROWING IS NOT INVENTED HERE — it is Recompute's own, read one field
// further. `activeSkill` answers a melee weapon's attack type, and "melee" is
// `AttackType < 0xa` (data/weapon.go) while there are six slots, so a row of
// attack type 6..9, or a negative one, is a melee weapon whose skill this
// build holds no slot for. data/recompute.go's step 6a already decides what
// such a slot is worth: its to-hit and damage terms are added only `if slot >
// SkillGeneral && slot < SkillSlots`, so a slot outside that window
// contributes no skill to the blow at all — which is exactly the state
// `activeSkill` names SkillGeneral for when it is handed no melee weapon.
// This function is that same window, and the same answer outside it.
//
// IT IS ALSO WHAT MAKES THE uint8 CONVERSION AT THE CALL SITE SAFE. Without
// it a slot of -1 narrows to 255 and 256 narrows silently to 0 — the first
// refused by SetCombat's own guard, the second accepted as a legal but wrong
// slot. Narrowing first means the byte handed over is always one of the six,
// so the guard on the far side is a second fence rather than the only one.
// THE ROW IS REAL: the shipped Weapons table carries one of attack type -1,
// on both lawful roots, and it resolves through data.WeaponFromCode as melee.
func creditedSlot(slot int32) int32 {
	if slot <= data.SkillGeneral || slot >= data.SkillSlots {
		return data.SkillGeneral
	}
	return slot
}

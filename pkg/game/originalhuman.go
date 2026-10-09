package game

import (
	"encoding/binary"
	"fmt"
	"reflect"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

const maxCityTraining = 4096

func equalTrainingMember(a, b mapload.PartyMember) bool {
	av := reflect.ValueOf(&a).Elem().FieldByName("Weapon" + "Materialized")
	bv := reflect.ValueOf(&b).Elem().FieldByName("Weapon" + "Materialized")
	av.SetBool(false)
	bv.SetBool(false)
	return reflect.DeepEqual(a, b)
}

func equalCurrentCityTrainingMember(a, b mapload.PartyMember) bool {
	if a.Weapon != nil && b.Weapon != nil && a.Weapon.Code != 0 && a.Weapon.Code == b.Weapon.Code &&
		mapload.MemberItemEquipment(a, nil)[0].Code == uint16(a.Weapon.Code) {
		actual, expected := *a.Weapon, *b.Weapon
		actual.Name, expected.Name = "", ""
		a.Weapon, b.Weapon = &actual, &expected
	}
	return equalTrainingMember(a, b)
}

func readHumanAttack(b []byte) data.HumanAttack {
	a := data.HumanAttack{ToHit: binary.LittleEndian.Uint16(b), DamageBase: b[14], DamageSpread: b[15], Active: b[16],
		SecondBase: b[17], SecondSpread: b[18], ElementalBase: b[19], ElementalSpread: b[20], ElementalKind: b[21], Tail: [2]byte{b[22], b[23]}}
	for i := range a.Skill {
		a.Skill[i] = binary.LittleEndian.Uint16(b[2+2*i:])
	}
	return a
}

func writeHumanAttack(b []byte, a data.HumanAttack) {
	binary.LittleEndian.PutUint16(b, a.ToHit)
	for i, v := range a.Skill {
		binary.LittleEndian.PutUint16(b[2+2*i:], v)
	}
	copy(b[14:], []byte{a.DamageBase, a.DamageSpread, a.Active, a.SecondBase, a.SecondSpread, a.ElementalBase, a.ElementalSpread, a.ElementalKind, a.Tail[0], a.Tail[1]})
}

func readHumanDefence(b []byte) data.HumanDefence {
	d := data.HumanDefence{Defence: binary.LittleEndian.Uint16(b), Absorption: binary.LittleEndian.Uint16(b[2:])}
	for i := range d.Protection {
		d.Protection[i] = binary.LittleEndian.Uint16(b[4+2*i:])
		d.Resistance[i] = b[16+i]
	}
	return d
}

func writeHumanDefence(b []byte, d data.HumanDefence) {
	binary.LittleEndian.PutUint16(b, d.Defence)
	binary.LittleEndian.PutUint16(b[2:], d.Absorption)
	for i, v := range d.Protection {
		binary.LittleEndian.PutUint16(b[4+2*i:], v)
		b[16+i] = d.Resistance[i]
	}
}

func cityHumanState(c sav.CityCharacter, source sav.CityHuman) data.HumanState {
	s, f := c.Stats, source.Fields
	h := data.HumanState{Body: s[sav.StatBody], Reaction: s[sav.StatReaction], Mind: s[sav.StatMind], Spirit: s[sav.StatSpirit],
		Speed: s[sav.StatSpeed], Weight: s[sav.StatOwnWeight], Load: s[sav.StatLoad], Capacity: s[sav.StatCapacity],
		Health: s[sav.StatHealth], HealthMax: s[sav.StatHealthMax], HealthPeriod: s[sav.StatHealthRegen],
		Mana: s[sav.StatMana], ManaMax: s[sav.StatManaMax], ManaPeriod: s[sav.StatManaRegen], ManaFloor: f.ManaFloor, Sight: f.Sight,
		Attack: readHumanAttack(f.Attack[:]), Base: readHumanAttack(f.Base[:]), Defence: readHumanDefence(f.Defence[:]),
		SkillXP: c.SkillXP, Experience: c.Experience, MoverSpeed: f.MoverSpeed,
		Fighter: source.Fighter, HasSpellbook: source.HasSpellbook, TypeID: source.TypeID,
		InventoryWeight: source.InventoryWeight, HasOwner: source.HasOwner, ManaReservePercent: source.ManaReservePercent}
	b := f.Modifier[:]
	for i := range h.Modifier.StatCap {
		h.Modifier.StatCap[i] = int8(b[i])
	}
	h.Modifier.Speed, h.Modifier.Capacity = binary.LittleEndian.Uint16(b[4:]), binary.LittleEndian.Uint16(b[6:])
	h.Modifier.HealthMax, h.Modifier.HealthRegeneration = binary.LittleEndian.Uint16(b[8:]), binary.LittleEndian.Uint16(b[10:])
	h.Modifier.ManaMax, h.Modifier.ManaRegeneration, h.Modifier.Sight = binary.LittleEndian.Uint16(b[12:]), binary.LittleEndian.Uint16(b[14:]), binary.LittleEndian.Uint16(b[16:])
	h.Modifier.Attack, h.Modifier.Defence = readHumanAttack(b[18:42]), readHumanDefence(b[42:64])
	return h
}

func cityHumanUpdate(c sav.CityCharacter, h data.HumanState) sav.CityCharacterUpdate {
	u := originalCityBaselineUpdate(c)
	u.Stats = [sav.UnitStatWords]uint16{h.Body, h.Reaction, h.Mind, h.Spirit, h.Speed, h.Weight, h.Load, h.Capacity,
		h.Health, h.HealthMax, h.HealthPeriod, h.Mana, h.ManaMax, h.ManaPeriod}
	u.SkillLevels, u.SkillXP, u.Experience = h.Attack.Skill, h.SkillXP, h.Experience
	f := &sav.CityHumanFields{ManaFloor: h.ManaFloor, Sight: h.Sight, MoverSpeed: h.MoverSpeed}
	writeHumanAttack(f.Attack[:], h.Attack)
	writeHumanAttack(f.Base[:], h.Base)
	writeHumanDefence(f.Defence[:], h.Defence)
	b := f.Modifier[:]
	for i, v := range h.Modifier.StatCap {
		b[i] = byte(v)
	}
	for i, v := range []uint16{h.Modifier.Speed, h.Modifier.Capacity, h.Modifier.HealthMax, h.Modifier.HealthRegeneration, h.Modifier.ManaMax, h.Modifier.ManaRegeneration, h.Modifier.Sight} {
		binary.LittleEndian.PutUint16(b[4+2*i:], v)
	}
	writeHumanAttack(b[18:42], h.Modifier.Attack)
	writeHumanDefence(b[42:64], h.Modifier.Defence)
	u.Human = f
	return u
}

func bindCityHuman(document originalCityDocument, c sav.CityCharacter, member *mapload.PartyMember) {
	doc, ok := document.(*sav.CityProvenance)
	if !ok || member.Hired() || member.Mage && member.Book.State == sim.BookLegacy {
		return
	}
	source, err := doc.Human(c.Identity)
	if err != nil || source.Fighter == member.Mage || source.HasSpellbook != member.SpellbookPresent {
		return
	}
	h := cityHumanState(c, source)
	if h.Hero() != member.Hero || member.Carry == nil || member.Saved == nil {
		return
	}
	mapload.ApplyOriginalHuman(member, h)
}

// expectedMember replays only admitted school and completed sale operations against the
// independently reconstructed source baseline. A forged current member or
// modifier cannot authorize its own export by forging a matching baseline.
func (b originalCityBinding) expectedMember() (mapload.PartyMember, error) {
	return replayCityMember(b)
}

func (t *townScreen) trainOriginalCityFighter(slot int, member *mapload.PartyMember) (bool, string) {
	if mapload.HasSourceActor(*member) {
		h, _, ok := currentCityHuman(*member)
		if !ok {
			return true, "training requires consistent current Human fields"
		}
		if err := mapload.ValidatePartyLoad(*member); err != nil {
			return true, err.Error()
		}
		if err := h.ProjectionError(); err != nil {
			return true, err.Error()
		}
		candidate, price, err := mapload.TrainSourceParty(*member, slot)
		if err != nil {
			return true, err.Error()
		}
		next, _, ok := currentCityHuman(candidate)
		if !ok {
			return true, "training produced inconsistent current Human fields"
		}
		if err := next.ProjectionError(); err != nil {
			return true, err.Error()
		}
		if price <= 0 {
			return true, "training price is outside the supported purse domain"
		}
		if candidate.OriginalHuman != nil {
			candidate.OriginalHuman = mapload.BindOriginalHuman(candidate, next)
		}
		refreshDerivedPartyBook(&candidate, t.in.Table)
		graph, err := prepareCityTrainingBook(t.sess.originalCity, t.sess.Town.cityObjects, *member, candidate)
		if err != nil {
			return true, err.Error()
		}
		if !t.sess.Town.spend(int(price)) {
			return true, fmt.Sprintf("training costs %d", price)
		}
		*member, t.sess.Town.cityObjects = candidate, graph
		if t.room == roomSchool {
			t.schoolPage().Event("train")
		}
		t.composeShopFaces()
		return true, fmt.Sprintf("trained %s to %d for %d", schoolSkillName(member.Mage, slot), member.Hero.Skill[slot], price)
	}
	if t.sess.originalCity == nil {
		return false, ""
	}
	for i := range t.sess.originalCity.bindings {
		b := &t.sess.originalCity.bindings[i]
		if b.partyID != member.ID {
			continue
		}
		if b.returned != nil && mapload.HasSourceActor(*member) {
			return false, ""
		}
		if b.baseline.OriginalHuman == nil {
			return false, ""
		}
		comparison := *b
		if b.salesVersion == 0 {
			comparison.sales = nil
		}
		expected, err := comparison.expectedMember()
		if err != nil {
			return true, err.Error()
		}
		basis := mapload.CloneParty([]mapload.PartyMember{expected})[0]
		if member.OriginalHuman != nil && member.OriginalHuman.Retired {
			retained := *member.OriginalHuman
			retained.Retired = false
			if !reflect.DeepEqual(&retained, expected.OriginalHuman) {
				return true, "SAV training has invalid retained Human state"
			}
			if err := retained.State.FighterProjectionError(); err != nil {
				return true, err.Error()
			}
			// Without current actor operands, use native school arithmetic.
			return false, ""
		}
		var cityHuman *data.HumanState
		if t.sess.Town.cityObjects != nil {
			if expected.OriginalHuman != nil {
				state := expected.OriginalHuman.State
				cityHuman = &state
			}
			if err := member.Book.Validate(member.KnownSpells); err != nil {
				return true, err.Error()
			}
			expected.Book, expected.KnownSpells = member.Book, member.KnownSpells
			// A current-city SAV owns the materialized Human, item instances and
			// runtime mirrors. The retained import baseline still supplies the
			// identity and authored fields, but these views are reconstructed from
			// the live city graph and therefore must not be compared by pointer or
			// runtime identity here.
			expected.OriginalHuman = member.OriginalHuman
			expected.WornItems, expected.CarriedItems = member.WornItems, member.CarriedItems
			expected.Carry, expected.Saved = member.Carry, member.Saved
			expected.SpellbookRestored, expected.SpellbookPresent = member.SpellbookRestored, member.SpellbookPresent
		}
		var unchanged bool
		if t.sess.Town.cityObjects != nil {
			unchanged = equalCurrentCityTrainingMember(*member, expected)
		} else {
			unchanged = equalTrainingMember(*member, expected)
		}
		if !unchanged && member.OriginalHuman == nil {
			// Native current state, as for a retired Human.
			return false, ""
		}
		if !unchanged {
			return true, "SAV training requires unchanged items and character state"
		}
		h, ok := member.OriginalHumanState()
		if !ok && cityHuman != nil {
			h, ok = *cityHuman, true
		}
		if !ok {
			return true, "SAV training has no valid Human state"
		}
		if err := h.ProjectionError(); err != nil {
			return true, err.Error()
		}
		price, err := h.TrainingPrice(slot)
		if err != nil {
			return true, err.Error()
		}
		next, err := h.Train(slot)
		if err != nil {
			return true, err.Error()
		}
		if err := next.ProjectionError(); err != nil {
			return true, err.Error()
		}
		if price <= 0 {
			return true, "SAV training price is outside the supported purse domain"
		}
		if len(b.training) >= maxCityTraining {
			return true, "SAV training history exceeds the save bound"
		}
		candidate := mapload.CloneParty([]mapload.PartyMember{*member})[0]
		if candidate.Carry == nil {
			candidate.Carry = basis.Carry
		}
		if candidate.Saved == nil {
			candidate.Saved = basis.Saved
		}
		mapload.ApplyOriginalHuman(&candidate, next)
		refreshPartyBook(b.gameRules, &candidate, b.spellRules)
		graph, err := prepareCityTrainingBook(t.sess.originalCity, t.sess.Town.cityObjects, *member, candidate)
		if err != nil {
			return true, err.Error()
		}
		if !t.sess.Town.spend(int(price)) {
			return true, fmt.Sprintf("training costs %d", price)
		}
		*member = candidate
		t.sess.Town.cityObjects = graph
		b.training = append(b.training, uint8(slot))
		if t.room == roomSchool {
			t.schoolPage().Event("train")
		}
		t.composeShopFaces()
		return true, fmt.Sprintf("trained %s to %d for %d", schoolSkillName(member.Mage, slot), member.Hero.Skill[slot], price)
	}
	return false, ""
}

func memberSchoolPrice(m mapload.PartyMember, slot int) int {
	if h, ok := m.OriginalHumanState(); ok {
		price, err := h.TrainingPrice(slot)
		if err == nil && price > 0 {
			return int(price)
		}
		return 0
	}
	return heroSkillPrice(m.Hero.Skill[slot])
}

func originalCityUpgradeParty(party []mapload.PartyMember, state *originalCitySaveState) []mapload.PartyMember {
	if state == nil {
		return party
	}
	party = mapload.CloneParty(party)
	for i := range party {
		if party[i].OriginalHuman != nil {
			continue
		}
		for _, b := range state.bindings {
			if b.returned != nil {
				continue
			}
			if b.partyID != party[i].ID || len(b.training) != 0 {
				continue
			}
			legacy := b.baseline
			legacy.OriginalHuman = nil
			if reflect.DeepEqual(party[i], legacy) {
				party[i] = mapload.CloneParty([]mapload.PartyMember{b.baseline})[0]
			} else if b.baseline.OriginalHuman != nil {
				// Before retained Human state existed, ordinary native item and
				// school changes carried no marker. The independently validated
				// source basis must not reclaim those changed native characters.
				party[i].OriginalHuman = mapload.CloneParty([]mapload.PartyMember{b.baseline})[0].OriginalHuman
				party[i].RetireOriginalHuman()
			}
		}
	}
	return party
}

func humanSpeedStat(e sim.Entity) int {
	if raw, ok := e.RetainedHumanSpeed(); ok {
		return int(raw)
	}
	return int(e.Speed)
}

// The original aggregate is independent stored state, not an assertion that
// it equals the six XP slots. Keep it while those slots still match the
// retained entry basis. Later native award/rearm producers remain DIV-675.
func (mw *mapWorld) retainedHumanExperience(e sim.Entity, native int) int {
	if mw.mission == nil {
		return native
	}
	for i, id := range mw.mission.ids {
		if id != e.ID || i >= len(mw.mission.party) {
			continue
		}
		h, ok := mw.mission.party[i].OriginalHumanState()
		if !ok {
			return native
		}
		for slot, xp := range e.SkillXP {
			if uint32(xp) != h.SkillXP[slot] {
				return native
			}
		}
		return int(int32(h.Experience))
	}
	return native
}

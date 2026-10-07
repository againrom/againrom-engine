package game

import (
	"encoding/binary"
	"math"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// This file closes the item and spellbook remainder of DIV-889.
// city_inventory.go's own reader is this file's byte-shape reference
// (Piece.W52/W6A/W50, A52/A50, S50): SourceEquipment.Attack IS W52, .Defence
// IS W6A/A52/S50 and .OwnKind IS W50/A50 by that type's own doc comment, so
// the shapes agree by construction, not by parallel guesswork.

// nativeCityItemToken extends nativeCityToken with the one generic Token
// field only an Item/Weapon/Armor/Shield head uses: price at this+0x1c
// (bytes 25:29 — SAV-OBJ-014's own byte map: "u32 this+0x1c" follows the
// u16 marker at this+0x0e and precedes the identity/reference pair every
// non-Player class shares). Every other offset is the same shared 37-byte
// head nativeCityToken already builds and city_test.go already proves valid
// for a Human; SAV-OBJ-014 (promoted) establishes that head is
// byte-identical in role across nine of the ten non-Player classes, Item
// among them.
//
// HERO-SKILLGATE-074
func nativeCityItemToken(identity, owner uint32, row byte, price int32) []byte {
	b := nativeCityToken(identity, owner, row, 0x21)
	binary.LittleEndian.PutUint32(b[25:29], uint32(price))
	return b
}

// nativeCityItemFields writes the established Item fields. Shape/material
// and the uncolumned F48 constructor result are supplied separately below.
// The final F47 byte remains outside the admitted constructor's known fields.
func nativeCityItemFields(code, stack uint16, kind uint8, weight int16) []byte {
	f := make([]byte, 12)
	binary.LittleEndian.PutUint16(f[0:2], code)
	binary.LittleEndian.PutUint16(f[2:4], stack)
	f[4] = kind
	binary.LittleEndian.PutUint16(f[9:11], uint16(weight))
	return f
}

type nativeCityItemConstruction struct {
	Shape, Material uint8
	F48             uint16
}

// ITEM-PICT-051 and corrected ITEM-SCALE-017 assign C/A to Item+45/+46.
// ITEM-ARMFILL-032 establishes the same uncolumned F48 product for all three
// equipment classes. Match data's scale-table indices and ftol convention;
// a partial table cannot supply a source constructor's missing operand.
func nativeCityItemConstructionFor(code uint16, table *mapload.Table) (nativeCityItemConstruction, error) {
	c := data.ItemCode(code)
	if table == nil || table.Shapes == nil || table.Materials == nil ||
		c.C() >= table.Shapes.Len() || c.A() >= table.Materials.Len() ||
		len(table.Shapes.EntryDoubles(c.C())) < 9 || len(table.Materials.EntryDoubles(c.A())) < 9 {
		return nativeCityItemConstruction{}, originalCityUnsupportedf(
			"native city item code %d has no complete shape/material F48 constructor operands", code)
	}
	v := table.Materials.EntryDoubles(c.A())[8] * table.Shapes.EntryDoubles(c.C())[8]
	if math.IsNaN(v) || math.IsInf(v, 0) || math.Trunc(v) < math.MinInt32 || math.Trunc(v) > math.MaxInt32 {
		return nativeCityItemConstruction{}, originalCityUnsupportedf(
			"native city item code %d has an unsupported F48 constructor product", code)
	}
	return nativeCityItemConstruction{Shape: uint8(c.C()), Material: uint8(c.A()), F48: uint16(int32(math.Trunc(v)))}, nil
}

// nativeCityItemObject reuses the concrete combat and weight constructor.
// A retained registry identity requires its original field and edge carriers,
// which an isolated ItemInstance cannot supply. Fresh base Items use the
// ITEM-CLASS-001 constructor weight1 only when no explicit weight is present.
func nativeCityItemObject(item sim.ItemInstance, identity, owner uint32, table *mapload.Table) (sav.CityObjectData, error) {
	if item.ObjectID != 0 {
		return sav.CityObjectData{}, originalCityUnsupportedf(
			"native city item code %d retains object identity %d without its saved field/edge registry; requires lossless .ags", item.Code, item.ObjectID)
	}
	if err := item.ValidateWeight(); err != nil {
		return sav.CityObjectData{}, originalCityUnsupportedf("native city item code %d has unsupported current operands: %v", item.Code, err)
	}
	class := "Item"
	var derived []byte
	row := item.SourceEquipment.DefinitionRow
	switch item.SourceEquipment.Class {
	case sim.SourceWeapon:
		class = "Weapon"
		derived = make([]byte, 47)
		copy(derived[:24], item.SourceEquipment.Attack[:])
		copy(derived[24:46], item.SourceEquipment.Defence[:])
		derived[46] = item.SourceEquipment.OwnKind
	case sim.SourceArmor:
		class = "Armor"
		derived = make([]byte, 23)
		copy(derived[:22], item.SourceEquipment.Defence[:])
		derived[22] = item.SourceEquipment.OwnKind
	case sim.SourceShield:
		class = "Shield"
		derived = make([]byte, 22)
		copy(derived, item.SourceEquipment.Defence[:])
	default:
		if b := data.ItemCode(item.Code).B(); b >= 1 && b <= 12 {
			return sav.CityObjectData{}, originalCityUnsupportedf(
				"native city save has no installed Weapons/Armor/Shields row to bind item code %d", item.Code)
		}
		row = 0
		if !item.WeightPresent {
			item.Weight, item.WeightPresent = 1, true
		}
	}
	if !item.WeightPresent {
		return sav.CityObjectData{}, originalCityUnsupportedf("native city item code %d has no explicit equipment weight", item.Code)
	}
	if item.SourceEquipment.EffectsUnsupported {
		return sav.CityObjectData{}, originalCityUnsupportedf("native city item code %d has unsupported retained Effect state", item.Code)
	}
	if class != "Item" {
		row = equipmentDefinitionRow(item.Code, row)
	}
	token := nativeCityItemToken(identity, owner, row, item.Price)
	fields := nativeCityItemFields(item.Code, 1, item.Kind, item.Weight)
	if class != "Item" {
		construction, err := nativeCityItemConstructionFor(item.Code, table)
		if err != nil {
			return sav.CityObjectData{}, err
		}
		fields[5], fields[6] = construction.Shape, construction.Material
		binary.LittleEndian.PutUint16(fields[7:9], construction.F48)
	}
	return sav.CityObjectData{Class: class, Item: &sav.CityItemData{Token: token, Fields: fields, Derived: derived}}, nil
}

// A saved owned Spell is current state. Only a freshly equipped weapon needs
// ITEM-SPELLMOVE-132's first-kind41 constructor; a carried weapon is unequipped.
// No matching Effect leaves an existing Spell intact, as sourcePrepareWeapon
// does. This edge is distinct from every Effect and spellbook object.
func nativeCityWeaponSpell(item sim.ItemInstance, equipped bool, table *mapload.Table) (sim.SourceItemSpell, error) {
	if item.SourceEquipment.Class != sim.SourceWeapon {
		return sim.SourceItemSpell{}, nil
	}
	if item.SourceEquipment.Spell.Present {
		return item.SourceEquipment.Spell, nil
	}
	if !equipped {
		return sim.SourceItemSpell{}, nil
	}
	for _, effect := range item.Effects {
		if effect.Kind != 41 {
			continue
		}
		id := uint8(effect.Operand)
		// A first kind-41 effect with ID zero is an explicit empty
		// attachment. It is preserved as an Effect but creates no owned Spell,
		// matching SourceEquippedItem and the ordinary item reader.
		if id == 0 {
			return sim.SourceItemSpell{}, nil
		}
		rules := mapload.SpellRules(table)
		if int(id) > len(rules) || rules[id-1].ID != uint16(id) {
			return sim.SourceItemSpell{}, originalCityUnsupportedf(
				"native city weapon code %d has no installed Spells row for first kind41 id %d", item.Code, id)
		}
		fields := nativeCitySpellFields(rules[id-1], 0)
		return sim.SourceItemSpell{Present: true, ID: fields[0], Range: fields[1], Defensive: fields[2],
			ManaCost: binary.LittleEndian.Uint16(fields[3:5])}, nil
	}
	return sim.SourceItemSpell{}, nil
}

// nativeCitySpellFields builds the 9-byte Spell fields block, the same shape
// city_inventory.go's WeaponSpell reads (SavedSpell.ID/.Range/.Defensive/
// .ManaCost = fields[0]/[1]/[2]/[3:5]). Bytes 5:9 are this class's own
// identity: unlike every other identity-bearing class, Spell has no Token at
// all (CitySpellData carries only Fields), and city_semantic.go's own
// cityObjectIdentity reads a Spell's identity from exactly this slice
// ("obj.spell.fields[5:9]") rather than from a shared 37-byte head.
// remintCityIdentities refuses any identity-bearing object whose read-back
// identity is zero, so this needs a real, per-object-distinct value the same
// way nativeCityItemToken supplies one for Item/Weapon/Armor/Shield.
func nativeCitySpellFields(rule sim.SpellRule, identity uint32) []byte {
	f := make([]byte, 9)
	f[0] = uint8(rule.ID)
	f[1] = rule.MaxRange
	if rule.Defensive {
		f[2] = 1
	}
	binary.LittleEndian.PutUint32(f[5:9], identity)
	mana := rule.ManaCost
	if mana < 0 {
		mana = 0
	}
	if mana > 0xffff {
		mana = 0xffff
	}
	binary.LittleEndian.PutUint16(f[3:5], uint16(mana))
	return f
}

// nativeCityMemberSpells resolves member's KnownSpells bitmask against the
// installed Spells table through mapload.SpellRules, the same conversion
// chargen's own preview and the mission spell table both already use
// (pkg/mapload/spell.go). Bit N of KnownSpells names spell id N
// (spellbooks.go's savedSpellMembership: "mask |= 1 << spell.ID"), and id N
// is rules[N-1] (SpellRules' own one-based id, spellRules' own doc). ok is
// false only when a known bit names an id the installed table does not
// reach — every other outcome, including no known spells at all, is fine.
func nativeCityMemberSpells(member mapload.PartyMember, table *mapload.Table) ([]sim.SpellRule, bool) {
	if member.KnownSpells == 0 {
		return nil, true
	}
	rules := mapload.SpellRules(table)
	var out []sim.SpellRule
	for bit := uint(1); bit <= 28; bit++ {
		if member.KnownSpells&(uint32(1)<<bit) == 0 {
			continue
		}
		idx := int(bit) - 1
		if idx < 0 || idx >= len(rules) {
			return nil, false
		}
		out = append(out, rules[idx])
	}
	return out, true
}

// nativeCityAttachItems fills unit's equipment, container and spellbook
// fields from member's live worn/carried items and known spells, minting a
// fresh CityObjectData per item and per spell and appending each to objects.
// seq is a per-document counter so every minted identity stays distinct;
// (*CityProvenance).Marshal reassigns every identity before the bytes are
// final (remintCityIdentities), so these only need internal consistency, the
// same contract nativeCityIdentity's own doc states.
func nativeCityAttachItems(objects []sav.CityObjectData, unit *sav.CityUnitData, member mapload.PartyMember, table *mapload.Table, owner uint32, seq *int) ([]sav.CityObjectData, error) {
	if member.OriginalHuman != nil {
		return nil, originalCityUnsupportedf(
			"native city member %q retains source item/container state without its saved field/edge registry; requires lossless .ags", member.Name)
	}
	if member.Carry != nil && member.Carry.OrderedStacks != nil && member.Carry.LiveLoad == nil {
		// An ordered pack with no accompanying current-load snapshot has no
		// InsertIndex/Accumulator to verify a re-derived container total
		// against (nativeCityAttachItemsConstruct's own weight-consistency
		// check below needs current for that); accepting it here would
		// flatten a stack this writer cannot actually attest to. Unchanged
		// from before this story.
		return nil, originalCityUnsupportedf(
			"native city member %q retains source item/container state without its saved field/edge registry; requires lossless .ags", member.Name)
	}
	if member.Carry != nil && (member.Carry.OrderedStacks != nil || member.Carry.LiveLoad != nil) {
		// A member who actually played a mission carries CarryParty's own
		// mission-earned Carry (pkg/mapload/carry.go) forward: pack contents and
		// worn weight the SIMULATION wrote, not a retained import binding
		// (member.OriginalHuman, refused above, is the source state this writer
		// still cannot represent). generatedworld1171.go's ConstructActorBasis now
		// gives every from-nothing generated Human Source.Class==2 too
		// (Stats/Attack/ Defence/Modifier built fresh from that same actor's own
		// data.HumanState) — a from-nothing party member's Carry.LiveLoad no
		// longer implies retained import continuity, so this call site takes
		// nativeCityAttachItemsConstruct directly, and the sibling
		// nativeCityHumanState (called right after this by
		// nativeCityDataConstruct's own per-member loop, human(&unit, member,
		// table)) verifies that basis fidelity itself, by reconstructing the exact
		// sim.SourceActor a fresh derive of this same member would produce and
		// comparing it field-for-field, rather than trusting Source.Class as a
		// proxy. An ordinary TOWN save reaches this member after the mission ended
		// and f.live is nil (frontend.go: "the town — where live is nil"), so
		// there is no live sim.World left to query; member.Carry.OrderedStacks is
		// that same live query's own answer, already taken once at the identical
		// return boundary (CarryParty: "c.OrderedStacks, _ =
		// w.CarriedStacks(ids[i])") — a later read of the same value, not a
		// different source for it. nativeCityAttachItemsConstruct's own existing
		// weight-consistency check (current.Inventory.Accumulator against the
		// re-derived container total) still refuses a genuinely inconsistent
		// container exactly as before.
		if member.Carry.LiveLoad != nil {
			if err := member.Carry.LiveLoad.Validate(); err != nil {
				return nil, originalCityUnsupportedf(
					"native city member %q has an invalid current load snapshot: %v", member.Name, err)
			}
		}
		return nativeCityAttachItemsConstruct(objects, unit, member, table, owner, seq, member.Carry.LiveLoad, member.Carry.OrderedStacks, true)
	}
	return nativeCityAttachItemsConstruct(objects, unit, member, table, owner, seq, nil, nil, false)
}

func nativeCityAttachItemsConstruct(objects []sav.CityObjectData, unit *sav.CityUnitData, member mapload.PartyMember, table *mapload.Table, owner uint32, seq *int, current *sim.ActorLoadSnapshot, ordered []sim.ItemStack, currentBook bool) ([]sav.CityObjectData, error) {
	mint := func() uint32 {
		*seq++
		return nativeCityIdentity(4096 + *seq)
	}
	appendItem := func(item sim.ItemInstance, equipped bool, count uint32) (uint16, error) {
		if count == 0 || count > 65535 {
			return 0, originalCityUnsupportedf("native city item count is outside uint16")
		}
		// This path writes current native state, not a retained source document.
		// ObjectID belongs to the in-memory graph that supplied the item and has
		// no saved field/edge registry here. Preserve current values and effects,
		// but let this producer mint the SAV identity instead of treating the
		// runtime handle as an imported object.
		item = item.Clone()
		item.ObjectID = 0
		spell, err := nativeCityWeaponSpell(item, equipped, table)
		if err != nil {
			return 0, err
		}
		extra := 1
		if spell.Present {
			extra++
		}
		if len(objects) > 65535-extra || len(item.Effects) > 65535-extra-len(objects) {
			return 0, originalCityUnsupportedf("native city item effects exceed the object reference range")
		}
		obj, err := nativeCityItemObject(item, mint(), owner, table)
		if err != nil {
			return 0, err
		}
		binary.LittleEndian.PutUint16(obj.Item.Fields[2:4], uint16(count))
		objects = append(objects, obj)
		ref := uint16(len(objects))
		// ITEM-SAVE-014 / ITEM-EFFSAVE-077 place a counted, ordered list of
		// Effect references before the Item fields, outside its Derived tail.
		// Native item effects have state zero (SAV-776).
		for _, effect := range item.Effects {
			fields := make([]byte, 7)
			fields[0], fields[1] = effect.Kind, effect.Mode
			binary.LittleEndian.PutUint32(fields[2:6], effect.Operand)
			objects = append(objects, sav.CityObjectData{Class: "Effect", Effect: &sav.CityEffectData{
				Token: nativeCityToken(mint(), owner, 0, 0), Fields: fields,
			}})
			obj.Item.Effects = append(obj.Item.Effects, uint16(len(objects)))
		}
		if spell.Present {
			fields := nativeCitySpellFields(sim.SpellRule{ID: uint16(spell.ID), MaxRange: spell.Range,
				ManaCost: int32(spell.ManaCost)}, mint())
			fields[2] = spell.Defensive
			objects = append(objects, sav.CityObjectData{Class: "Spell", Spell: &sav.CitySpellData{Fields: fields}})
			obj.Item.WeaponExtra = uint16(len(objects))
		}
		return ref, nil
	}
	unit.Equipment = make([]uint16, 13)
	for i, item := range mapload.MemberItemEquipment(member, table) {
		if item.Code == 0 {
			continue
		}
		item = mapload.SourceConstructedItem(item, table)
		ref, err := appendItem(item, true, 1)
		if err != nil {
			return nil, err
		}
		switch i + 1 {
		case 1:
			unit.Reference74 = ref
		case 2:
			unit.Reference78 = ref
		default:
			unit.Equipment[i] = ref
		}
	}

	carried := ordered
	if current == nil && ordered == nil {
		carried = sim.FoldItems(mapload.MemberCarriedItems(member, table))
	}
	// DIV-895
	unit.ContainerFlag = 1
	var totalWeight int32
	for _, stack := range carried {
		item := stack.Instance()
		if item.Code == 0 {
			continue
		}
		item = mapload.SourceConstructedItem(item, table)
		ref, err := appendItem(item, false, stack.Count)
		if err != nil {
			return nil, err
		}
		unit.Container = append(unit.Container, ref)
		totalWeight += int32(int16(binary.LittleEndian.Uint16(objects[ref-1].Item.Fields[9:11]))) * int32(stack.Count)
	}
	unit.ContainerTails = [2]uint32{0, uint32(totalWeight)}
	if current != nil {
		if totalWeight != current.Inventory.Accumulator || !current.Inventory.ContainerPresent && len(carried) != 0 {
			return nil, originalCityUnsupportedf("native city current container weight differs from ordered contents")
		}
		unit.ContainerFlag = 0
		if current.Inventory.ContainerPresent {
			unit.ContainerFlag = 1
		}
		unit.ContainerTails = [2]uint32{current.Inventory.InsertIndex, uint32(current.Inventory.Accumulator)}
	}

	if currentBook && member.Book.State != sim.BookLegacy {
		if err := member.Book.Validate(member.KnownSpells); err != nil {
			return nil, err
		}
		if !member.Book.HasInstances() {
			return objects, nil
		}
		unit.SpellbookFlag = 1
		for i, slot := range member.Book.Slots {
			if member.KnownSpells&(uint32(1)<<uint(i+1)) == 0 {
				continue
			}
			for len(unit.Spells) <= i {
				unit.Spells = append(unit.Spells, 0)
			}
			fields := make([]byte, 9)
			fields[0], fields[1], fields[2] = byte(i+1), slot.Range, slot.Defensive
			binary.LittleEndian.PutUint16(fields[3:5], slot.ManaCost)
			binary.LittleEndian.PutUint32(fields[5:9], mint())
			objects = append(objects, sav.CityObjectData{Class: "Spell", Spell: &sav.CitySpellData{Fields: fields}})
			unit.Spells[i] = uint16(len(objects))
		}
		unit.SpellbookCount = uint32(len(unit.Spells)) + 1
		return objects, nil
	}
	rules, ok := nativeCityMemberSpells(member, table)
	if !ok {
		return nil, originalCityUnsupportedf(
			"native city save has no installed Spells row to bind %s's known spell", member.Name)
	}
	if len(rules) > 0 || member.SpellbookRestored && member.SpellbookPresent {
		if len(objects) > 65535-len(rules) {
			return nil, originalCityUnsupportedf("native city spellbook exceeds the object reference range")
		}
		unit.SpellbookFlag = 1
		// city.go's own reader (cityCharacterDescriptor) requires slot N to
		// hold spell ID N exactly ("int(id) != slot+1" refuses otherwise):
		// this is a POSITIONAL array indexed by id-1, not a compacted list in
		// bit order, so a spellbook with IDs {1,6,12,...} needs five empty
		// (zero-reference, ref(0)==nil, skipped) slots between id 1 and id 6.
		widest := 0
		for _, rule := range rules {
			if int(rule.ID) > widest {
				widest = int(rule.ID)
			}
		}
		unit.Spells = make([]uint16, widest)
		for _, rule := range rules {
			objects = append(objects, sav.CityObjectData{Class: "Spell", Spell: &sav.CitySpellData{Fields: nativeCitySpellFields(rule, mint())}})
			unit.Spells[rule.ID-1] = uint16(len(objects))
		}
		unit.SpellbookCount = uint32(len(unit.Spells)) + 1
	}
	return objects, nil
}

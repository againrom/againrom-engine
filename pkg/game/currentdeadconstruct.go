package game

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func (b *generatedDocumentBuilder) currentDeadRecord(record *sav.DocumentRecordData, body sim.OriginalDeadRecord, entity bool) error {
	s, c := body.Source, body.Current
	for _, v := range []sav.DocumentValueData{{Name: "Identity", Value: s.Identity}, {Name: "Reference", Value: s.OwnerKey}, {Name: "RuntimeID", Value: c.RuntimeID}, {Name: "T08", Value: uint32(s.MapUnitID)},
		{Name: "Stage", Value: uint32(c.Stage)}, {Name: "Health", Value: uint32(uint16(c.HP))}, {Name: "U6C", Value: uint32(uint8(c.Timer))},
		{Name: "U5C", Value: s.References[0]}, {Name: "U64", Value: s.References[1]}, {Name: "U44", Value: s.References[2]}, {Name: "U40", Value: s.References[4]}} {
		if err := savedActorSetValue(record, v.Name, v.Value); err != nil {
			return err
		}
	}
	if s.MaxHPPresent {
		if err := savedActorSetValue(record, "HealthMax", uint32(uint16(s.MaxHP))); err != nil {
			return err
		}
	}
	p := constructedPositionBlock(int32(c.Cell&255), int32(c.Cell>>8), s.TerrainKey)
	p[4], p[5] = c.FineX, c.FineY
	mustSetRaw(record, "Block12", p)
	if s.ContainerPresent {
		mustSetValue(record, "HasInventory", 1)
		mustSetValue(record, "Inventory1C", s.ContainerTail[0])
		mustSetValue(record, "Inventory20", s.ContainerTail[1])
	}
	if s.References[3] != 0 {
		var object uint16
		for i, r := range b.doc.Objects {
			key, err := savedStructureValue(&r, "Identity")
			if err == nil && key == s.References[3] {
				if object != 0 {
					return fmt.Errorf("retained actor reference has repeated identity")
				}
				object = uint16(i + 1)
			}
		}
		if object == 0 {
			return fmt.Errorf("retained actor reference has no current object")
		}
		mustSetRefs(record, "U68", []uint16{object})
	}
	if weapon := s.HeldWeapon; weapon.Present {
		r := mustNewRecord("Weapon")
		mustSetToken(&r, nativeCityToken(weapon.Identity, weapon.Reference, equipmentDefinitionRow(weapon.F40, weapon.T0C), weapon.T0E))
		for _, v := range []sav.DocumentValueData{{Name: "RuntimeID", Value: weapon.RuntimeID}, {Name: "T08", Value: weapon.T08}, {Name: "T18", Value: uint32(weapon.T18)}, {Name: "T1C", Value: weapon.T1C},
			{Name: "F40", Value: uint32(weapon.F40)}, {Name: "F42", Value: uint32(weapon.F42)}, {Name: "F44", Value: uint32(weapon.F44)}, {Name: "F45", Value: uint32(weapon.F45)}, {Name: "F46", Value: uint32(weapon.F46)},
			{Name: "F47", Value: uint32(weapon.F47)}, {Name: "F48", Value: uint32(weapon.F48)}, {Name: "F4A", Value: uint32(weapon.F4A)}, {Name: "W50", Value: uint32(weapon.W50)}} {
			mustSetValue(&r, v.Name, v.Value)
		}
		position := constructedPositionBlock(int32(weapon.Cell&255), int32(weapon.Cell>>8), weapon.TerrainKey)
		binary.LittleEndian.PutUint16(position[2:], weapon.PackedCell)
		position[4], position[5] = weapon.FineX, weapon.FineY
		binary.LittleEndian.PutUint16(position[6:], weapon.PositionU06)
		mustSetRaw(&r, "Block12", position)
		mustSetRaw(&r, "W52", weapon.W52[:])
		mustSetRaw(&r, "W6A", weapon.W6A[:])
		object, err := b.append(r)
		if err != nil {
			return err
		}
		mustSetRefs(record, savHeldWeaponField, []uint16{object})
	}
	if err := b.retainDeadBook(record, s, entity); err != nil {
		return err
	}
	return b.retainDeadEffects(record, body.ID, s)
}

// loadedDeadObject is the loaded document's DeadActors root of the dead
// source's Identity and class, or 0.
func (b *generatedDocumentBuilder) loadedDeadObject(s sim.OriginalDeadSource) (uint16, error) {
	if b.loaded == nil {
		return 0, nil
	}
	var object uint16
	for _, root := range b.loaded.DeadActors {
		if root == 0 || int(root) > len(b.loaded.Objects) {
			continue
		}
		r := &b.loaded.Objects[root-1]
		if key, err := savedStructureValue(r, "Identity"); err == nil && key == s.Identity && r.Class == savedActorClass(s.Class) && root != object {
			if object != 0 {
				return 0, fmt.Errorf("retained dead actor %#x has two loaded records", s.Identity)
			}
			object = root
		}
	}
	return object, nil
}

// loadedDeadBook reads the spellbook of a loaded dead record: its presence
// flag, header word, slot count and the loaded Spell object of each slot. ok
// is false when the record holds no well-formed book fields.
func loadedDeadBook(doc *sav.DocumentData, object uint16) (has, header, count uint32, slots []uint16, ok bool) {
	r := &doc.Objects[object-1]
	has, err := savedStructureValue(r, "HasSpellbook")
	if err != nil || has > 1 {
		return 0, 0, 0, nil, false
	}
	if has == 0 {
		return 0, 0, 0, nil, true
	}
	if header, err = savedStructureValue(r, "SpellsHeader"); err != nil {
		return 0, 0, 0, nil, false
	}
	refs, found := savedObjectRefs(r, "Spells")
	present := false
	for _, c := range r.Counts {
		if c.Name == "Spells" {
			count, present = c.Count, true
		}
	}
	if !found || !present {
		return 0, 0, 0, nil, false
	}
	for _, ref := range refs {
		if ref != 0 && (int(ref) > len(doc.Objects) || doc.Objects[ref-1].Class != "Spell") {
			return 0, 0, 0, nil, false
		}
	}
	return has, header, count, refs, true
}

// retainDeadBook writes the spellbook of a dead record whose body the World
// holds as no entity from the loaded document's record of the same Identity
// and class. The World holds no book for such a body, so the loaded record is
// its source: presence flag, header word, slot count, and each slot's Spell
// copied with its loaded This when reserveRetainedDeadEffectKeys claimed it.
// A loaded Spell two slots share stays one object. A body the World holds
// as an entity writes the entity's book (DIV-4121); without a loaded record
// the book stays as built.
func (b *generatedDocumentBuilder) retainDeadBook(record *sav.DocumentRecordData, s sim.OriginalDeadSource, entity bool) error {
	if entity {
		return nil
	}
	object, err := b.loadedDeadObject(s)
	if err != nil || object == 0 {
		return err
	}
	has, header, count, refs, ok := loadedDeadBook(b.loaded, object)
	if !ok {
		return nil
	}
	record.Values = slices.DeleteFunc(slices.Clone(record.Values), func(v sav.DocumentValueData) bool { return v.Name == "SpellsHeader" })
	record.RefSlots = slices.DeleteFunc(slices.Clone(record.RefSlots), func(v sav.DocumentRefsData) bool { return v.Name == "Spells" })
	record.Counts = slices.DeleteFunc(slices.Clone(record.Counts), func(v sav.DocumentCountData) bool { return v.Name == "Spells" })
	savedObjectSetValue(record, "HasSpellbook", has)
	if has == 0 {
		return nil
	}
	savedObjectSetValue(record, "SpellsHeader", header)
	slots := make([]uint16, len(refs))
	for i, ref := range refs {
		if ref == 0 {
			continue
		}
		if b.retainedSpells == nil {
			b.retainedSpells = map[uint16]uint16{}
		}
		if index := b.retainedSpells[ref]; index != 0 {
			slots[i] = index
			continue
		}
		spell := b.loaded.Objects[ref-1]
		spell.Values = slices.Clone(spell.Values)
		if key, err := savedStructureValue(&spell, "This"); err != nil || !b.retainedKeys[key] {
			savedObjectSetValue(&spell, "This", b.identity())
		}
		index, err := b.append(spell)
		if err != nil {
			return err
		}
		b.retainedSpells[ref], slots[i] = index, index
	}
	savedObjectSetRefs(record, "Spells", slots, true)
	mustSetCount(record, "Spells", count)
	return nil
}

// retainDeadEffects writes the Effects a dead record keeps unexecuted, read
// from the loaded document's record of the same identity and class, with that
// record's spell mask U144, unless that body holds live attachments in the
// loaded document or the World. The World holds no part of these Effects, so
// each Token is the loaded one. Its RuntimeID is kept even where another
// object holds the same handle, as original files repeat one on these Effects;
// an Identity reserveRetainedDeadEffectKeys could not claim is left to the
// minting pass.
func (b *generatedDocumentBuilder) retainDeadEffects(record *sav.DocumentRecordData, body sim.EntityID, s sim.OriginalDeadSource) error {
	object, err := b.loadedDeadObject(s)
	if err != nil {
		return err
	}
	effects, ok := retainedDeadEffects(b.loaded, object)
	if !ok || b.liveEffects[body] {
		return nil
	}
	if (&liveEffectReader{doc: b.loaded}).holds(object) {
		return nil
	}
	refs := make([]uint16, 0, len(effects))
	for _, effect := range effects {
		token := effect.Token
		if !b.retainedKeys[token.Identity] {
			token.Identity = 0
		}
		index, err := b.append(savedCurrentEffectRecord(sim.SavedEffectObject{E0C: effect.E0C, Value: effect.Value, Token: token}))
		if err != nil {
			return err
		}
		refs = append(refs, index)
	}
	mask, err := savedStructureValue(&b.loaded.Objects[object-1], "U144")
	if err != nil {
		return err
	}
	savedObjectSetRefs(record, "Effects", refs, true)
	savedObjectSetValue(record, "U144", mask)
	return nil
}

// reserveRetainedDeadEffectKeys claims, before any key is minted, the Identity
// of each Effect a loaded dead root keeps unexecuted (retainedDeadEffects) and
// the This of each Spell in the loaded book of a dead body the World holds as
// no entity (retainDeadBook).
func (b *generatedDocumentBuilder) reserveRetainedDeadEffectKeys(w *sim.World) error {
	if b.loaded == nil {
		return nil
	}
	claim := func(key uint32) {
		if key != 0 && !b.claimHeldKey(key) {
			if b.retainedKeys == nil {
				b.retainedKeys = map[uint32]bool{}
			}
			b.retainedKeys[key] = true
		}
	}
	live := &liveEffectReader{doc: b.loaded}
	for _, root := range b.loaded.DeadActors {
		effects, ok := retainedDeadEffects(b.loaded, root)
		if !ok || live.holds(root) {
			continue
		}
		for _, effect := range effects {
			claim(effect.Token.Identity)
		}
	}
	for _, body := range w.OriginalDeadActors() {
		if _, entity := w.Entity(body.ID); entity {
			continue
		}
		object, err := b.loadedDeadObject(body.Source)
		if err != nil {
			return err
		}
		if object == 0 {
			continue
		}
		if _, _, _, refs, ok := loadedDeadBook(b.loaded, object); ok {
			for _, ref := range refs {
				if ref != 0 {
					key, _ := savedStructureValue(&b.loaded.Objects[ref-1], "This")
					claim(key)
				}
			}
		}
	}
	return nil
}

// constructedSearchRadius gives an actor record this build constructs in a
// first-game document the constructor's mover byte +0x08, 5 (AI-692). The
// World's own Mover replaces it for an actor the World still holds
// (projectMotion), so it stands for a body that is gone; the byte a loaded
// record of that body held is not carried (DIV-3942).
func (b *generatedDocumentBuilder) constructedSearchRadius(r *sav.DocumentRecordData, w *sim.World) error {
	if w.SecondGame() {
		return nil
	}
	mover, err := savedActorRaw(r, "U154", 180)
	if err != nil {
		return err
	}
	mover[8] = sim.ConstructorSearchRadius
	return nil
}

// appendAbsentDeadRecords writes each retained dead actor no written record
// holds. A dead actor with a map placement takes its actor values from the
// placement and the tables, as a fresh mission constructs them; its dead
// tuple, references, held weapon and the items it kept at its terminal stage
// come from the World. A dead actor whose Diary the World holds gets a Diary
// record of diaryRows entries.
func (b *generatedDocumentBuilder) appendAbsentDeadRecords(w *sim.World, ms *Mission, hero mapload.PartyMember, diff mapload.Difficulty, diaries map[sim.EntityID]bool, diaryRows int) (bool, error) {
	changed := false
	var placed map[uint16]sim.Entity
	passive := map[sim.EntityID]sim.PassiveBody{}
	for _, p := range w.PassiveBodies() {
		passive[p.ID] = p
	}
	for _, body := range w.OriginalDeadActors() {
		found := false
		for _, r := range b.doc.Objects {
			key, err := savedStructureValue(&r, "Identity")
			if err == nil && key == body.Source.Identity {
				if r.Class != savedActorClass(body.Source.Class) {
					return false, fmt.Errorf("retained actor identity has a different current class")
				}
				found = true
			}
		}
		if found {
			continue
		}
		flags := []string{}
		if body.Source.ContainerPresent {
			flags = append(flags, "HasInventory")
		}
		r := mustNewRecord(savedActorClass(body.Source.Class), flags...)
		if err := b.constructedSearchRadius(&r, w); err != nil {
			return false, err
		}
		if id := body.Source.MapUnitID; id != 0 && ms.Map != nil {
			if placed == nil {
				var err error
				if placed, err = placementActors(ms.Map, b.table, diff); err != nil {
					return false, err
				}
			}
			if e, ok := placed[id]; ok {
				var placement *alm.Unit
				for i := range ms.Map.Units {
					if ms.Map.Units[i].UnitID == id {
						placement = &ms.Map.Units[i]
					}
				}
				basis, _, err := currentRecordActor(e, mapload.PartyMember{}, hero, placement, b.table, body.Source.Identity, body.Current.RuntimeID)
				if err != nil {
					return false, err
				}
				if r, err = savedActorValueRecord(r, basis, true); err != nil {
					return false, err
				}
			}
		}
		if err := b.currentDeadRecord(&r, body, false); err != nil {
			return false, err
		}
		if typeID, present := ms.DeadArt[body.ID]; present {
			mustSetValue(&r, "T0E", uint32(typeID))
		}
		if r.Class != "Unit" && diaries[body.ID] {
			index, err := b.append(mustNewDiaryRecord(diaryRows, 0))
			if err != nil {
				return false, err
			}
			mustSetRefs(&r, "Diary", []uint16{index})
		}
		index, err := b.append(r)
		if err != nil {
			return false, err
		}
		if kept, held := passive[body.ID]; held {
			if err := b.passiveDeadContents(index, body, kept, w); err != nil {
				return false, err
			}
		}
		changed = true
	}
	return changed, nil
}

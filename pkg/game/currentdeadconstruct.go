package game

import (
	"encoding/binary"
	"fmt"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func (b *generatedDocumentBuilder) currentDeadRecord(record *sav.DocumentRecordData, body sim.OriginalDeadRecord) error {
	s, c := body.Source, body.Current
	for _, v := range []sav.DocumentValueData{{Name: "Identity", Value: s.Identity}, {Name: "Reference", Value: s.OwnerKey}, {Name: "RuntimeID", Value: c.RuntimeID}, {Name: "T08", Value: uint32(s.MapUnitID)},
		{Name: "Stage", Value: uint32(c.Stage)}, {Name: "Health", Value: uint32(uint16(c.HP))}, {Name: "U6C", Value: uint32(uint8(c.Timer))},
		{Name: "U5C", Value: s.References[0]}, {Name: "U64", Value: s.References[1]}, {Name: "U44", Value: s.References[2]}, {Name: "U40", Value: s.References[4]}} {
		if err := savedActorSetValue(record, v.Name, v.Value); err != nil {
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
		mustSetRefs(record, "HeldWeapon", []uint16{object})
	}
	return nil
}

// appendAbsentDeadRecords writes each retained dead actor no written record
// holds. A dead actor with a map placement takes its actor values from the
// placement and the tables, as a fresh mission constructs them; its dead
// tuple, references and held weapon come from the World. A dead actor whose
// Diary the World holds gets a Diary record of diaryRows entries.
func (b *generatedDocumentBuilder) appendAbsentDeadRecords(w *sim.World, ms *Mission, hero mapload.PartyMember, diff mapload.Difficulty, diaries map[sim.EntityID]bool, diaryRows int) (bool, error) {
	changed := false
	var placed map[uint16]sim.Entity
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
				runtime := body.Current.RuntimeID
				if runtime == 0 {
					runtime = b.runtime()
				}
				basis, _, err := currentRecordActor(e, mapload.PartyMember{}, hero, placement, b.table, body.Source.Identity, runtime)
				if err != nil {
					return false, err
				}
				if r, err = savedActorValueRecord(r, basis, true); err != nil {
					return false, err
				}
			}
		}
		if err := b.currentDeadRecord(&r, body); err != nil {
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
		if _, err := b.append(r); err != nil {
			return false, err
		}
		changed = true
	}
	return changed, nil
}

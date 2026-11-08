package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// virtualCasterRecords builds the records of the world's caster list for the
// second game's list after the Sacks (R2-ENGINE-1011): the Token head holds the
// building's location and the owner's Player key, the reach byte and the
// six-byte block hold the rest. The Token's other words and the block's last two
// bytes have no source in the world and are written zero (DIV-4718).
func virtualCasterRecords(w *sim.World, doc *sav.DocumentData) ([]sav.DocumentRecordData, error) {
	list := w.BuildingCasters()
	if len(list) == 0 || doc.World == nil {
		return nil, nil
	}
	used := make(map[uint32]bool, len(doc.Objects))
	for _, object := range doc.Objects {
		for _, v := range object.Values {
			if v.Name == "Identity" || v.Name == "This" {
				used[v.Value] = true
			}
		}
	}
	key := uint32(virtualCasterKeyBase)
	records := make([]sav.DocumentRecordData, 0, len(list))
	for _, c := range list {
		for used[key] {
			key++
		}
		used[key] = true
		r := mustNewRecord("VirtualCaster")
		mustSetValue(&r, "Identity", key)
		mustSetValue(&r, "VC3C", uint32(c.Reach))
		mustSetRaw(&r, "Block12", constructedPositionBlock(int32(c.Cell&0xff), int32(c.Cell>>8), doc.World.TerrainIdentity))
		mustSetRaw(&r, "VC40", []byte{c.Spell, c.Power, c.X, c.Y, 0, 0})
		for _, object := range doc.Players {
			player := &doc.Objects[object-1]
			slot, err := savedStructureValue(player, "Slot")
			if err != nil {
				return nil, err
			}
			if slot != c.Owner {
				continue
			}
			reference, err := savedStructureValue(player, "This")
			if err != nil {
				return nil, err
			}
			mustSetValue(&r, "Reference", reference)
			break
		}
		records = append(records, r)
	}
	return records, nil
}

// virtualCasterKeyBase opens the identity range the caster records take; no
// other generated record writes this high byte.
const virtualCasterKeyBase = 0x5f000000

// applyOriginalBuildingCasters replaces the world's caster list with the
// file's. A list with no records rebuilds from the map for the buildings still
// standing: an original file where the list is empty, and every save this
// engine wrote before it carried the list (DIV-4718). A file's caster whose
// reference names no Player takes the owner of the map's caster on its cell
// word.
func applyOriginalBuildingCasters(ms *Mission, casters []sav.VirtualCaster) {
	if ms == nil || ms.Map == nil || ms.World == nil || !ms.World.BuildsCasters() {
		return
	}
	fromMap := mapload.BuildingCasters(ms.Map, ms.World)
	if len(casters) == 0 {
		ms.World.DeclareBuildingCasters(fromMap)
		return
	}
	owners := make(map[uint16]uint32, len(fromMap))
	for _, c := range fromMap {
		owners[c.Cell] = c.Owner
	}
	list := make([]sim.BuildingCaster, len(casters))
	for i, c := range casters {
		owner := c.OwnerSlot
		if !c.OwnerKnown {
			owner = owners[c.Cell]
		}
		list[i] = sim.BuildingCaster{X: c.X, Y: c.Y, Spell: c.Spell, Power: c.Power, Reach: c.Reach, Owner: owner, Cell: c.Cell}
	}
	ms.World.DeclareBuildingCasters(list)
}

// restoreCurrentBuildingCasters puts back the list the engine's own document
// carried. A document that does not say it carried one keeps the list the map
// rebuild gave: every save written before the list was carried.
func restoreCurrentBuildingCasters(w *sim.World, a *currentActionData) {
	if a != nil && a.BuildingCastersWritten && w.BuildsCasters() {
		w.DeclareBuildingCasters(a.BuildingCasters)
	}
}

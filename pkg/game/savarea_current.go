package game

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Native roots are constructed only on the detached final SAV document. The
// live world and its import bindings keep their independent identities.
func projectNativeAreas(doc *sav.DocumentData, world *sim.World) error {
	areas, err := world.NativeAreaSaveStates()
	if err != nil || len(areas) == 0 {
		return err
	}
	if len(doc.Objects)+2*len(areas) > 65534 {
		return fmt.Errorf("current areas exceed archive object limit")
	}
	keys, err := sav.ReserveDocumentKeys(*doc, 2*len(areas))
	if err != nil {
		return err
	}
	actions, err := readCurrentActions(doc)
	if err != nil {
		return err
	}
	if actions == nil {
		return fmt.Errorf("native area lacks current action bindings")
	}
	for n, a := range areas {
		token := sim.SavedObjectToken{T0C: uint8(a.Spell), T0E: uint16(2*a.Spell + 9), Identity: keys[2*n]}
		token.Position[0], token.Position[1] = uint8(a.Key), uint8(a.Key>>8)
		token.Position[2], token.Position[3] = token.Position[0], token.Position[1]
		token.Position[4], token.Position[5] = 128, 128
		binary.LittleEndian.PutUint32(token.Position[8:], doc.World.TerrainIdentity)
		if a.Mode == sim.AreaModeCloud {
			token.T08 = 1
		} else if a.Mode == sim.AreaModeRing {
			token.T08 = 2
		}
		root := savedObjectRecord("AreaEffect", &token)
		stopped := uint32(0)
		if a.Stopped {
			stopped = 1
		}
		savedObjectSetValue(&root, "SE40", stopped)
		savedObjectSetValue(&root, "SE41", 0)
		savedObjectSetValue(&root, "AE4C", uint32(a.Remaining))
		root.Raw = append(root.Raw, sav.DocumentRawData{Name: "AE48", Bytes: []byte{1, a.Radius, a.Direction << 5, a.Stage}})
		slices.SortFunc(root.Raw, func(a, b sav.DocumentRawData) int { return strings.Compare(a.Name, b.Name) })
		payload := a.Payload
		child := savedCurrentEffectRecord(sim.SavedEffectObject{Token: sim.SavedObjectToken{Identity: keys[2*n+1]}, E0C: payload.E0C,
			Value: sim.ItemEffect{Kind: payload.E3C, Mode: payload.E3D, Operand: payload.E40}})
		child.Class = payload.Class
		if child.Class == "Effect_DirectDamage" {
			child.Raw = append(child.Raw, sav.DocumentRawData{Name: "EDD48", Bytes: slices.Clone(payload.DirectDamage[:])})
		}
		index := uint16(len(doc.Objects) + 1)
		actions.NativeAreas = append(actions.NativeAreas, currentNativeArea{Object: index, Policy: a.Policy})
		savedObjectSetRefs(&root, "AE44", []uint16{index + 1}, false)
		doc.Objects = append(doc.Objects, root, child)
		doc.World.Effects = append(doc.World.Effects, index)
		if a.Mode != sim.AreaModeCloud {
			continue
		}
		layer, ok := areaLayerIndex(a.Spell)
		if !ok {
			return fmt.Errorf("current cloud %d has no cell layer", a.Spell)
		}
		for _, key := range a.Cells {
			at := slices.IndexFunc(doc.World.Cells, func(c sav.DocumentCellData) bool { return c.Cell == key })
			if at < 0 {
				// Insert before the first later key, the place a LOAD's ordered
				// cell list gives this record on the next SAVE.
				at = slices.IndexFunc(doc.World.Cells, func(c sav.DocumentCellData) bool { return c.Cell > key })
				if at < 0 {
					at = len(doc.World.Cells)
				}
				doc.World.Cells = slices.Insert(doc.World.Cells, at, currentAreaCellRecord(doc, world, key))
			}
			cell := &doc.World.Cells[at]
			cell.Layers[layer] = token.Identity
			cell.LayerCount = 0
			for _, pointer := range cell.Layers {
				if pointer != 0 {
					cell.LayerCount++
				}
			}
		}
	}
	raw, err := json.Marshal(actions)
	if err != nil {
		return err
	}
	if err := sav.SetNativeActions(&doc.State, raw); err != nil {
		return err
	}
	indexed, _, err := sav.ReindexDocumentData(*doc)
	if err != nil {
		return err
	}
	*doc = indexed
	return nil
}

// A cloud over a cell with no record creates one, as the layer attach does
// (MAGIC-AREAEND-041, SAV-CELLENTRY-582): 52 zero bytes, then the cell's
// current cost and static bytes as its baselines. Static drops the record bit
// 0x20, which no cell without a record carries (TERR-PASS-051). Without
// original planes the bytes are the ones this producer writes for the cell's
// block row (projectCurrentTerrain) and its terrain cost.
func currentAreaCellRecord(doc *sav.DocumentData, world *sim.World, key uint16) sav.DocumentCellData {
	if planes, ok := world.SavedCellPlanes(); ok && planes.CostKnown[key] != 0 {
		return sav.DocumentCellData{Cell: key, Cost: planes.Cost[key], Static: planes.Static[key] &^ 0x20}
	}
	p := world.CurrentPolicy()
	at := int(key>>8)*int(world.Bounds().Width) + int(key&255)
	var static, cost uint8
	for _, block := range doc.World.Blocks {
		if block.Cell == key {
			static = block.Static
		}
	}
	if at < len(p.Terrain.Block) {
		grid := p.Terrain.Block[at]
		static = static&^7 | grid&3 | (grid&8)>>1
	}
	if at < len(p.Terrain.Cost) {
		cost = p.Terrain.Cost[at]
	}
	return sav.DocumentCellData{Cell: key, Cost: cost, Static: static &^ 0x20}
}

func restoreCurrentAreas(ms *Mission, rows []currentNativeArea) error {
	if len(rows) == 0 {
		return nil
	}
	state := ms.savedDocument
	meta := state.WorldEffects
	if meta == nil {
		return fmt.Errorf("current area has no world-effect binding")
	}
	if err := ensureCurrentSpellGraph(ms); err != nil {
		return err
	}
	var native []sim.CurrentAreaRestore
	converted := map[uint32]bool{}
	for _, row := range rows {
		at := slices.IndexFunc(meta.SpellNodes, func(b SnapshotSAVSpellNode) bool { return b.ObjectIndex == row.Object })
		if row.Object == 0 || at < 0 {
			return fmt.Errorf("current area lost its ordinary node")
		}
		native = append(native, sim.CurrentAreaRestore{Node: uint32(at + 1), Policy: row.Policy})
		converted[uint32(at+1)] = true
	}
	if err := ms.World.RestoreCurrentAreas(native); err != nil {
		return err
	}
	remapCurrentSpellMetadata(meta, converted)
	return nil
}

func ensureCurrentSpellGraph(ms *Mission) error {
	state := ms.savedDocument
	meta := state.WorldEffects
	if meta == nil {
		return fmt.Errorf("current effect has no ordinary binding")
	}
	if ms.World.SavedSpellGraph() == nil {
		drivers := ms.World.SavedWorldEffectDrivers()
		if drivers == nil {
			drivers = &sim.SavedWorldEffects{}
		}
		drivers.Areas, meta.Areas, meta.SpellNodes = nil, nil, nil
		g, err := bindSavedSpellGraph(state, ms.World, meta, drivers)
		if err != nil {
			return err
		}
		if err := ms.World.ImportOriginalWorldEffectDrivers(drivers); err != nil {
			return err
		}
		if err := ms.World.ImportSavedSpellGraph(g); err != nil {
			return err
		}
	}
	return nil
}

func remapCurrentSpellMetadata(meta *SnapshotSAVWorldEffects, converted map[uint32]bool) {
	indices := make([]uint32, len(meta.SpellNodes)+1)
	var nodes []SnapshotSAVSpellNode
	for i, node := range meta.SpellNodes {
		if !converted[uint32(i+1)] {
			indices[i+1] = uint32(len(nodes) + 1)
			nodes = append(nodes, node)
		}
	}
	meta.SpellNodes = nodes
	meta.Areas = slices.DeleteFunc(meta.Areas, func(a SnapshotSAVArea) bool { return converted[a.ID] })
	for i := range meta.Areas {
		meta.Areas[i].ID = indices[meta.Areas[i].ID]
	}
}

// Keep the ordinary indices stable until every decoded current binding has
// been consumed. Only then can ownership transfer retire its transport nodes.
func retireCurrentAreaDocument(ms *Mission, rows []currentNativeArea, deliveries []currentNativeDelivery) error {
	if len(rows) == 0 && len(deliveries) == 0 {
		return nil
	}
	state := ms.savedDocument
	var removed []uint16
	for _, row := range rows {
		removed = append(removed, row.Object)
	}
	for _, row := range deliveries {
		removed = append(removed, row.Object)
	}
	state.Document.World.Effects = slices.DeleteFunc(state.Document.World.Effects, func(id uint16) bool { return slices.Contains(removed, id) })
	retired, err := retiredActorClosure(state.Document, removed, nil)
	if err != nil {
		return err
	}
	doc, permutation, err := sav.RetireDocumentData(*state.Document, retired)
	if err != nil {
		return err
	}
	state.Document = &doc
	return remapSavedSackDocument(state, permutation)
}

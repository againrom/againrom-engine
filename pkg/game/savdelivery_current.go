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

func currentSpellToken(doc *sav.DocumentData, identity uint32, spell uint16, key uint16) sim.SavedObjectToken {
	t := sim.SavedObjectToken{Identity: identity, T0C: uint8(spell), T0E: 2*spell + 9}
	t.Position[0], t.Position[1] = byte(key), byte(key>>8)
	t.Position[2], t.Position[3], t.Position[4], t.Position[5] = byte(key), byte(key>>8), 128, 128
	binary.LittleEndian.PutUint32(t.Position[8:], doc.World.TerrainIdentity)
	return t
}

func currentSpellPayload(p sim.SavedEffect, token sim.SavedObjectToken) sav.DocumentRecordData {
	r := savedCurrentEffectRecord(sim.SavedEffectObject{Token: token, E0C: p.E0C, Value: sim.ItemEffect{Kind: p.E3C, Mode: p.E3D, Operand: p.E40}})
	r.Class = p.Class
	if p.Class == "Effect_DirectDamage" {
		r.Raw = append(r.Raw, sav.DocumentRawData{Name: "EDD48", Bytes: slices.Clone(p.DirectDamage[:])})
	}
	return r
}

func projectNativeSpellDeliveries(doc *sav.DocumentData, state *SnapshotSAVDocument, world *sim.World) error {
	deliveries := world.NativeSpellDeliverySaveStates()
	if len(deliveries) == 0 {
		return nil
	}
	if len(doc.Objects)+3*len(deliveries) > 65534 {
		return fmt.Errorf("current deliveries exceed archive object limit")
	}
	keys, err := sav.ReserveDocumentKeys(*doc, 3*len(deliveries))
	if err != nil {
		return err
	}
	targets := map[sim.EntityID]uint32{}
	for _, e := range world.Entities() {
		if e.SourceBinding.Class != 0 {
			targets[e.ID] = e.SourceBinding.Identity
		}
	}
	for _, a := range state.Actors {
		if a.Retired {
			continue
		}
		key, _ := savedStructureValue(&state.Document.Objects[a.ObjectIndex-1], "Identity")
		targets[a.EntityID] = key
	}
	for _, d := range world.OriginalDeadActors() {
		targets[d.ID] = d.Source.Identity
	}
	actions, err := readCurrentActions(doc)
	if err != nil {
		return err
	}
	if actions == nil {
		return fmt.Errorf("current delivery has no action bindings")
	}
	for i, d := range deliveries {
		a := d.Area
		t := currentSpellToken(doc, keys[3*i], 0, uint16(d.FromX)|uint16(d.FromY)<<8)
		t.T0E = 0
		root := savedObjectRecord("SpellTransport", &t)
		savedObjectSetValue(&root, "SE40", 0)
		if d.TransportStopped {
			savedObjectSetValue(&root, "SE40", 1)
		}
		savedObjectSetValue(&root, "SE41", 1)
		savedObjectSetValue(&root, "ST4C", uint32(d.Remaining))
		t = currentSpellToken(doc, keys[3*i+1], a.Spell, a.Key)
		class := "PointEffect"
		if d.AtCell {
			class = "AreaEffect"
			if a.Mode == sim.AreaModeCloud {
				t.T08 = 1
			} else if a.Mode == sim.AreaModeRing {
				t.T08 = 2
			}
		}
		child := savedObjectRecord(class, &t)
		savedObjectSetValue(&child, "SE40", 0)
		if d.ChildStopped {
			savedObjectSetValue(&child, "SE40", 1)
		}
		flag := uint32(0)
		if d.Attribution {
			flag = 1
		}
		savedObjectSetValue(&child, "SE41", flag)
		index := uint16(len(doc.Objects) + 1)
		if d.AtCell {
			savedObjectSetValue(&child, "AE4C", uint32(a.Remaining))
			child.Raw = append(child.Raw, sav.DocumentRawData{Name: "AE48", Bytes: []byte{0, a.Radius, a.Direction << 5, a.Stage}})
			savedObjectSetRefs(&child, "AE44", []uint16{index + 2}, false)
		} else {
			savedObjectSetValue(&child, "PE44", targets[d.Target])
			savedObjectSetRefs(&child, "PE48", []uint16{index + 2}, false)
		}
		slices.SortFunc(child.Raw, func(a, b sav.DocumentRawData) int { return strings.Compare(a.Name, b.Name) })
		payload := currentSpellPayload(a.Payload, sim.SavedObjectToken{Identity: keys[3*i+2]})
		savedObjectSetRefs(&root, "ST44", []uint16{index + 1}, false)
		savedObjectSetRefs(&root, "ST48", []uint16{0}, false)
		if d.Released {
			field := "PE48"
			if d.AtCell {
				field = "AE44"
			}
			savedObjectSetRefs(&child, field, []uint16{index + 1}, false)
			doc.Objects = append(doc.Objects, child, payload)
		} else {
			doc.Objects = append(doc.Objects, root, child, payload)
		}
		doc.World.Effects = append(doc.World.Effects, index)
		policy := d.Policy
		if d.HasTarget && targets[d.Target] == 0 {
			id := d.Target
			policy.MissingTarget = &id
		}
		actions.NativeDeliveries = append(actions.NativeDeliveries, currentNativeDelivery{Object: index, Policy: policy})
	}
	if actions != nil {
		b, err := json.Marshal(actions)
		if err != nil {
			return err
		}
		if err := sav.SetNativeActions(&doc.State, b); err != nil {
			return err
		}
	}
	indexed, _, err := sav.ReindexDocumentData(*doc)
	if err != nil {
		return err
	}
	*doc = indexed
	return nil
}

func restoreCurrentSpellDeliveries(ms *Mission, rows []currentNativeDelivery) error {
	if len(rows) == 0 {
		return nil
	}
	if err := ensureCurrentSpellGraph(ms); err != nil {
		return err
	}
	state := ms.savedDocument
	meta, graph := state.WorldEffects, ms.World.SavedSpellGraph()
	var current []sim.CurrentDeliveryRestore
	removed := map[uint32]bool{}
	point := func(index uint16) (sim.CellPoint, error) {
		if index == 0 || int(index) > len(state.Document.Objects) {
			return sim.CellPoint{}, fmt.Errorf("current delivery position is unbound")
		}
		p, err := savedMotionRaw(&state.Document.Objects[index-1], "Block12", 12)
		if err != nil {
			return sim.CellPoint{}, err
		}
		return sim.CellPoint{X: int32(p[0]), Y: int32(p[1])}, nil
	}
	for _, row := range rows {
		at := slices.IndexFunc(meta.SpellNodes, func(b SnapshotSAVSpellNode) bool { return b.ObjectIndex == row.Object })
		if at < 0 {
			return fmt.Errorf("current delivery lost its ordinary node")
		}
		id := uint32(at + 1)
		child := id
		if graph.Nodes[at].Value.Class == "SpellTransport" {
			child = graph.Nodes[at].Primary
			if child == 0 {
				child = graph.Nodes[at].Fallback
			}
		}
		if child == 0 || int(child) > len(meta.SpellNodes) {
			return fmt.Errorf("current delivery lost its ordinary child")
		}
		from, err := point(row.Object)
		if err != nil {
			return err
		}
		to, err := point(meta.SpellNodes[child-1].ObjectIndex)
		if err != nil {
			return err
		}
		current = append(current, sim.CurrentDeliveryRestore{Node: id, From: from, Destination: to, Policy: row.Policy})
		removed[id], removed[child] = true, true
	}
	if err := ms.World.RestoreCurrentSpellDeliveries(current); err != nil {
		return err
	}
	remapCurrentSpellMetadata(meta, removed)
	return nil
}

func projectWorldEffectOrder(doc *sav.DocumentData, world *sim.World) error {
	areas, err := world.NativeAreaSaveStates()
	if err != nil {
		return err
	}
	deliveries := world.NativeSpellDeliverySaveStates()
	savedCount := len(doc.World.Effects) - len(areas) - len(deliveries)
	if savedCount < 0 {
		return fmt.Errorf("current effect population differs")
	}
	indices := map[sim.WorldEffectRef]uint16{}
	if g := world.SavedSpellGraph(); g != nil {
		for i, id := range g.Roots {
			indices[sim.WorldEffectRef{Kind: sim.EffectSavedGraph, Index: id}] = doc.World.Effects[i]
		}
	} else if drivers := world.SavedWorldEffectDrivers(); drivers != nil {
		for _, d := range drivers.Areas {
			if d.Root >= 0 {
				indices[sim.WorldEffectRef{Kind: sim.EffectSavedArea, Index: d.ID}] = doc.World.Effects[d.Root]
			}
		}
	}
	for i := range areas {
		indices[sim.WorldEffectRef{Kind: sim.EffectNativeArea, Index: uint32(i)}] = doc.World.Effects[savedCount+i]
	}
	for i := range deliveries {
		indices[sim.WorldEffectRef{Kind: sim.EffectNativeDelivery, Index: uint32(i)}] = doc.World.Effects[savedCount+len(areas)+i]
	}
	order := world.CurrentWorldEffectOrder()
	if len(order) != len(doc.World.Effects) {
		return fmt.Errorf("current effect order population differs")
	}
	for i, r := range order {
		id := indices[r]
		if id == 0 {
			return fmt.Errorf("current effect order binding lost")
		}
		doc.World.Effects[i] = id
	}
	indexed, _, err := sav.ReindexDocumentData(*doc)
	if err != nil {
		return err
	}
	*doc = indexed
	return nil
}

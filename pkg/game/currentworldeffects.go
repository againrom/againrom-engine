package game

import (
	"encoding/binary"
	"fmt"
	"strings"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// worldEffectFamily lists the loaded records the World's original spell
// effects reach from the World effect list, in visit order. It is empty when
// the World holds no original spell graph and no original area.
func worldEffectFamily(loaded *SnapshotSAVDocument, w *sim.World) ([]uint16, error) {
	drivers := w.SavedWorldEffectDrivers()
	if loaded == nil || loaded.Document == nil || loaded.Document.World == nil || loaded.WorldEffects == nil ||
		w.SavedSpellGraph() == nil && (drivers == nil || len(drivers.Areas) == 0) {
		return nil, nil
	}
	doc := loaded.Document
	seen := map[uint16]bool{}
	var order []uint16
	var visit func(uint16) error
	visit = func(index uint16) error {
		if index == 0 || seen[index] {
			return nil
		}
		if int(index) > len(doc.Objects) {
			return fmt.Errorf("world effect is outside the loaded graph")
		}
		r := &doc.Objects[index-1]
		if !strings.HasSuffix(r.Class, "Effect") && r.Class != "SpellTransport" && !strings.HasPrefix(r.Class, "Effect_") {
			return fmt.Errorf("world effect reaches a %s record", r.Class)
		}
		seen[index] = true
		order = append(order, index)
		for _, slot := range r.RefSlots {
			for _, child := range slot.Objects {
				if err := visit(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, root := range doc.World.Effects {
		if err := visit(root); err != nil {
			return nil, err
		}
	}
	return order, nil
}

// reserveWorldEffectKeys claims the identities of the loaded world-effect
// records before any key is minted; an area's identity is its cell-layer key.
func (b *generatedDocumentBuilder) reserveWorldEffectKeys(loaded *SnapshotSAVDocument, w *sim.World) error {
	family, err := worldEffectFamily(loaded, w)
	if err != nil {
		return err
	}
	for _, index := range family {
		key, err := savedStructureValue(&loaded.Document.Objects[index-1], "Identity")
		if err == nil && key != 0 && b.claimHeldKey(key) {
			return fmt.Errorf("world effect identity is claimed twice")
		}
	}
	return nil
}

// currentWorldEffects writes the World's original spell effects. The World
// holds each node's spell, payload, timers and references and each area's
// identity, mode and cell; the world-effect projection writes those over the
// record. The rest of each Token comes from the loaded record (DIV-2502).
func (b *generatedDocumentBuilder) currentWorldEffects(loaded *SnapshotSAVDocument, w *sim.World) error {
	family, err := worldEffectFamily(loaded, w)
	if err != nil || len(family) == 0 {
		return err
	}
	source, err := cloneSavedDocument(loaded)
	if err != nil {
		return err
	}
	moved := map[uint16]uint16{}
	for _, index := range family {
		r := source.Document.Objects[index-1]
		if _, err := savedStructureValue(&r, "RuntimeID"); err == nil {
			savedObjectSetValue(&r, "RuntimeID", b.runtime())
		}
		if moved[index], err = b.append(r); err != nil {
			return err
		}
	}
	for _, index := range family {
		r := &b.doc.Objects[moved[index]-1]
		for i := range r.RefSlots {
			for j, child := range r.RefSlots[i].Objects {
				r.RefSlots[i].Objects[j] = moved[child]
			}
		}
	}
	for _, root := range source.Document.World.Effects {
		b.doc.World.Effects = append(b.doc.World.Effects, moved[root])
	}
	meta := &SnapshotSAVWorldEffects{Version: 1, Unavailable: source.WorldEffects.Unavailable}
	for _, row := range source.WorldEffects.SpellNodes {
		meta.SpellNodes = append(meta.SpellNodes, SnapshotSAVSpellNode{ObjectIndex: moved[row.ObjectIndex], PayloadIndex: moved[row.PayloadIndex]})
	}
	for _, row := range source.WorldEffects.Areas {
		meta.Areas = append(meta.Areas, SnapshotSAVArea{ID: row.ID, ObjectIndex: moved[row.ObjectIndex], ChildIndex: moved[row.ChildIndex]})
	}
	b.state.WorldEffects = meta
	return b.projectWorldEffectTokens(meta, w)
}

// projectWorldEffectTokens writes the Token and payload fields the World
// holds: every node's spell and payload, and every area's identity, mode,
// spell, cell key, common state, countdown and stage.
func (b *generatedDocumentBuilder) projectWorldEffectTokens(meta *SnapshotSAVWorldEffects, w *sim.World) error {
	payload := func(index uint16, p *sim.SavedEffect) error {
		if index == 0 || p == nil {
			return nil
		}
		r := &b.doc.Objects[index-1]
		if r.Class != p.Class {
			return fmt.Errorf("world effect payload class differs")
		}
		for _, v := range []sav.DocumentValueData{{Name: "E0C", Value: uint32(p.E0C)}, {Name: "E3C", Value: uint32(p.E3C)}, {Name: "E3D", Value: uint32(p.E3D)}, {Name: "E40", Value: p.E40}} {
			savedObjectSetValue(r, v.Name, v.Value)
		}
		if p.Class == "Effect_DirectDamage" {
			raw, err := savedMotionRaw(r, "EDD48", 24)
			if err != nil {
				return err
			}
			copy(raw, p.DirectDamage[:])
		}
		return nil
	}
	if g := w.SavedSpellGraph(); g != nil {
		if len(g.Nodes) != len(meta.SpellNodes) {
			return fmt.Errorf("spell graph binding count differs")
		}
		for i, n := range g.Nodes {
			row := meta.SpellNodes[i]
			if row.ObjectIndex == 0 {
				continue
			}
			savedObjectSetValue(&b.doc.Objects[row.ObjectIndex-1], "T0C", uint32(n.Spell))
			p := n.Value.PE48
			if p == nil {
				p = n.Value.AE44
			}
			if err := payload(row.PayloadIndex, p); err != nil {
				return err
			}
		}
	}
	drivers := w.SavedWorldEffectDrivers()
	if drivers == nil {
		return nil
	}
	effects := w.SavedSpellEffects()
	for _, d := range drivers.Areas {
		var row SnapshotSAVArea
		for _, r := range meta.Areas {
			if r.ID == d.ID {
				row = r
			}
		}
		if row.ObjectIndex == 0 {
			continue
		}
		r := &b.doc.Objects[row.ObjectIndex-1]
		mode, err := savedStructureValue(r, "T08")
		if err != nil {
			return err
		}
		// The mode bits change only when the World's mode differs: a ring
		// may carry both bits.
		held := uint8(sim.AreaModeBlast)
		if mode&2 != 0 {
			held = sim.AreaModeRing
		} else if mode&1 != 0 {
			held = sim.AreaModeCloud
		}
		if held != d.Mode {
			mode &^= 3
			switch d.Mode {
			case sim.AreaModeCloud:
				mode |= 1
			case sim.AreaModeRing:
				mode |= 2
			}
			savedObjectSetValue(r, "T08", mode)
		}
		savedObjectSetValue(r, "T0C", uint32(d.Spell))
		savedObjectSetValue(r, "Identity", d.Identity)
		block, err := savedMotionRaw(r, "Block12", 12)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint16(block, d.Key)
		if w.SavedSpellGraph() == nil && d.Root >= 0 && int(d.Root) < len(effects) {
			e := effects[d.Root]
			savedObjectSetValue(r, "SE40", uint32(e.SE40))
			savedObjectSetValue(r, "SE41", uint32(e.SE41))
			savedObjectSetValue(r, "AE4C", uint32(e.AE4C))
			stage, err := savedMotionRaw(r, "AE48", 4)
			if err != nil {
				return err
			}
			copy(stage, e.AE48[:])
			if err := payload(row.ChildIndex, e.AE44); err != nil {
				return err
			}
		}
	}
	return nil
}

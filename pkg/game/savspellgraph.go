package game

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type SnapshotSAVSpellNode struct {
	ObjectIndex, PayloadIndex uint16
}

// Legacy AGS keeps its historical inert graph until an explicit SAV projection.
// The current carrier supplies counters and payloads; the Document supplies IDs.
func armCarriedSpellGraph(state *SnapshotSAVDocument, world *sim.World) error {
	if world.SavedSpellGraph() != nil || len(world.SavedSpellEffects()) == 0 || state.Document == nil || state.Document.World == nil || !needsSpellGraph(state.Document) {
		return nil
	}
	meta := state.WorldEffects
	if meta == nil {
		meta = &SnapshotSAVWorldEffects{Version: 1}
		state.WorldEffects = meta
	}
	drivers := world.SavedWorldEffectDrivers()
	if drivers == nil {
		drivers = &sim.SavedWorldEffects{}
	}
	drivers.Areas = nil
	meta.Areas = nil
	meta.SpellNodes = nil
	g, err := bindSavedSpellGraph(state, world, meta, drivers)
	if err != nil {
		return err
	}
	current := world.SavedSpellEffects()
	if len(current) != len(g.Roots) {
		return fmt.Errorf("legacy spell root count differs")
	}
	seen := map[uint32]bool{}
	var copyNode func(uint32, sim.SavedSpellEffect) error
	copyNode = func(id uint32, v sim.SavedSpellEffect) error {
		n := &g.Nodes[id-1]
		primary, fallback := v.ST44, v.ST48
		v.ST44, v.ST48 = nil, nil
		if v.Class != n.Value.Class || (primary == nil) != (n.Primary == 0) || (fallback == nil) != (n.Fallback == 0) {
			return fmt.Errorf("legacy spell graph shape differs")
		}
		if seen[id] {
			if !reflect.DeepEqual(n.Value, v) {
				return fmt.Errorf("legacy shared spell fields disagree")
			}
			return nil
		}
		seen[id] = true
		n.Value = v
		if primary != nil {
			if err := copyNode(n.Primary, *primary); err != nil {
				return err
			}
		}
		if fallback != nil {
			if err := copyNode(n.Fallback, *fallback); err != nil {
				return err
			}
		}
		return nil
	}
	for i, id := range g.Roots {
		if err := copyNode(id, current[i]); err != nil {
			return err
		}
	}
	if err := world.ImportOriginalWorldEffectDrivers(drivers); err != nil {
		return err
	}
	if err := world.ImportSavedSpellGraph(g); err != nil {
		return err
	}
	for _, notice := range []string{"SpellTransport/PointEffect scheduling, PE44 and shared world-effect lifetimes remain unbound", "legacy native world effects have no current continuation bindings"} {
		meta.Unavailable = strings.ReplaceAll(meta.Unavailable, notice, "")
	}
	return nil
}

func needsSpellGraph(doc *sav.DocumentData) bool {
	incoming := savedDocumentIncoming(doc)
	for _, id := range doc.World.Effects {
		r := &doc.Objects[id-1]
		if r.Class != "AreaEffect" || incoming[id] != 1 {
			return true
		}
		refs, _ := savedObjectRefs(r, "AE44")
		if len(refs) == 1 && refs[0] != 0 && incoming[refs[0]] != 1 {
			return true
		}
	}
	return false
}

func bindSavedSpellGraph(state *SnapshotSAVDocument, world *sim.World, meta *SnapshotSAVWorldEffects, drivers *sim.SavedWorldEffects) (*sim.SavedSpellGraph, error) {
	doc := state.Document
	g := &sim.SavedSpellGraph{}
	ids := map[uint16]uint32{}
	actor := map[uint32]sim.EntityID{}
	for _, a := range state.Actors {
		if a.Retired {
			continue
		}
		key, _ := savedStructureValue(&doc.Objects[a.ObjectIndex-1], "Identity")
		if key != 0 {
			actor[key] = a.EntityID
		}
	}
	for _, d := range world.OriginalDeadActors() {
		if d.Source.Identity != 0 {
			actor[d.Source.Identity] = d.ID
		}
	}
	var visit func(uint16) (uint32, error)
	visit = func(index uint16) (uint32, error) {
		if index == 0 {
			return 0, nil
		}
		if id := ids[index]; id != 0 {
			return id, nil
		}
		if int(index) > len(doc.Objects) {
			return 0, fmt.Errorf("spell graph object outside archive")
		}
		id := uint32(len(g.Nodes) + 1)
		ids[index] = id
		g.Nodes = append(g.Nodes, sim.SavedSpellNode{})
		meta.SpellNodes = append(meta.SpellNodes, SnapshotSAVSpellNode{ObjectIndex: index})
		r := &doc.Objects[index-1]
		value := func(name string) uint32 { v, _ := savedStructureValue(r, name); return v }
		v := sim.SavedSpellEffect{Class: r.Class, SE40: uint8(value("SE40")), SE41: uint8(value("SE41"))}
		n := sim.SavedSpellNode{Spell: uint16(value("T0C"))}
		ref := func(name string) uint16 {
			v, _ := savedObjectRefs(r, name)
			if len(v) == 1 {
				return v[0]
			}
			return 0
		}
		var payload uint16
		switch r.Class {
		case "SpellTransport":
			v.ST4C = uint16(value("ST4C"))
			var err error
			n.Primary, err = visit(ref("ST44"))
			if err != nil {
				return 0, err
			}
			n.Fallback, err = visit(ref("ST48"))
			if err != nil {
				return 0, err
			}
		case "PointEffect":
			v.PE44 = value("PE44")
			n.Target, n.HasTarget = actor[v.PE44]
			payload = ref("PE48")
		case "AreaEffect":
			payload = ref("AE44")
			v.AE4C = uint16(value("AE4C"))
			block, err := savedMotionRaw(r, "AE48", 4)
			if err != nil {
				return 0, err
			}
			copy(v.AE48[:], block)
			pos, err := savedMotionRaw(r, "Block12", 12)
			if err != nil {
				return 0, err
			}
			d := sim.SavedAreaDriver{ID: id, Root: int32(slices.Index(doc.World.Effects, index)), Identity: value("Identity"), Key: binary.LittleEndian.Uint16(pos), Spell: n.Spell, Mode: sim.AreaModeBlast, Layer: 255}
			if value("T08")&2 != 0 {
				d.Mode = sim.AreaModeRing
			} else if value("T08")&1 != 0 {
				d.Mode = sim.AreaModeCloud
			}
			if d.Mode == sim.AreaModeCloud {
				if layer, ok := areaLayerIndex(d.Spell); ok {
					d.Layer = layer
				}
				for _, c := range world.SavedCellRecords() {
					if d.Layer < 6 && c.SpellEffects[d.Layer] == d.Identity {
						d.Cells = append(d.Cells, c.Cell)
					}
				}
			}
			drivers.Areas = append(drivers.Areas, d)
			meta.Areas = append(meta.Areas, SnapshotSAVArea{ID: id, ObjectIndex: index, ChildIndex: payload})
		case "SpellEffect":
		default:
			return 0, fmt.Errorf("invalid spell node class %s", r.Class)
		}
		if payload != 0 {
			p := &doc.Objects[payload-1]
			plain := *p
			plain.Class = "Effect"
			row, err := savedEffectRecord(&plain)
			if err != nil {
				return 0, err
			}
			e := &sim.SavedEffect{Class: p.Class, E0C: row.E0C, E3C: row.Value.Kind, E3D: row.Value.Mode, E40: row.Value.Operand}
			if p.Class == "Effect_DirectDamage" {
				block, err := savedMotionRaw(p, "EDD48", 24)
				if err != nil {
					return 0, err
				}
				copy(e.DirectDamage[:], block)
			}
			if r.Class == "AreaEffect" {
				v.AE44 = e
			} else {
				v.PE48 = e
			}
			meta.SpellNodes[id-1].PayloadIndex = payload
		}
		n.Value = v
		g.Nodes[id-1] = n
		return id, nil
	}
	for _, index := range doc.World.Effects {
		id, err := visit(index)
		if err != nil {
			return nil, err
		}
		g.Roots = append(g.Roots, id)
	}
	slices.SortFunc(drivers.Areas, func(a, b sim.SavedAreaDriver) int { return int(a.ID) - int(b.ID) })
	slices.SortFunc(meta.Areas, func(a, b SnapshotSAVArea) int { return int(a.ID) - int(b.ID) })
	return g, nil
}

func projectSpellGraphFields(state *SnapshotSAVDocument, graph *sim.SavedSpellGraph) error {
	rows := state.WorldEffects.SpellNodes
	if len(rows) != len(graph.Nodes) {
		return fmt.Errorf("spell graph binding count differs")
	}
	index := func(id uint32) uint16 {
		if id == 0 {
			return 0
		}
		return rows[id-1].ObjectIndex
	}
	state.Document.World.Effects = nil
	if len(graph.Roots) > 0 {
		state.Document.World.Effects = make([]uint16, len(graph.Roots))
	}
	for i, id := range graph.Roots {
		state.Document.World.Effects[i] = index(id)
	}
	for i, n := range graph.Nodes {
		if rows[i].ObjectIndex == 0 {
			if !n.Retired {
				return fmt.Errorf("live spell graph binding lost")
			}
			continue
		}
		r := &state.Document.Objects[rows[i].ObjectIndex-1]
		v := n.Value
		if r.Class != v.Class {
			return fmt.Errorf("spell graph class changed")
		}
		savedObjectSetValue(r, "SE40", uint32(v.SE40))
		savedObjectSetValue(r, "SE41", uint32(v.SE41))
		switch v.Class {
		case "SpellTransport":
			savedObjectSetValue(r, "ST4C", uint32(v.ST4C))
			savedObjectSetRefs(r, "ST44", []uint16{index(n.Primary)}, false)
			savedObjectSetRefs(r, "ST48", []uint16{index(n.Fallback)}, false)
		case "PointEffect":
			savedObjectSetValue(r, "PE44", v.PE44)
		case "AreaEffect":
			savedObjectSetValue(r, "AE4C", uint32(v.AE4C))
			block, err := savedMotionRaw(r, "AE48", 4)
			if err != nil {
				return err
			}
			copy(block, v.AE48[:])
		}
	}
	return nil
}

func validateSavedSpellGraph(state *SnapshotSAVDocument, world *sim.World) error {
	g := world.SavedSpellGraph()
	if g == nil {
		if state != nil && state.WorldEffects != nil && len(state.WorldEffects.SpellNodes) > 0 &&
			(len(world.SavedSpellEffects()) != 0 || spellGraphAreasRemain(world)) {
			return fmt.Errorf("saved spell graph lost")
		}
		return nil
	}
	if state == nil || state.Document == nil || state.WorldEffects == nil {
		return fmt.Errorf("spell graph document lost")
	}
	copy, err := cloneSavedDocument(state)
	if err != nil {
		return err
	}
	if err := projectSpellGraphFields(copy, g); err != nil {
		return err
	}
	for i, n := range g.Nodes {
		row := state.WorldEffects.SpellNodes[i]
		if row.ObjectIndex == 0 {
			continue
		}
		r := &state.Document.Objects[row.ObjectIndex-1]
		spell, _ := savedStructureValue(r, "T0C")
		if uint16(spell) != n.Spell {
			return fmt.Errorf("spell node %d has a different spell", i+1)
		}
		payload := n.Value.PE48
		if payload == nil {
			payload = n.Value.AE44
		}
		if (payload == nil) != (row.PayloadIndex == 0) {
			return fmt.Errorf("spell node payload binding differs")
		}
		if payload != nil {
			child := &state.Document.Objects[row.PayloadIndex-1]
			if child.Class != payload.Class {
				return fmt.Errorf("spell payload class differs")
			}
			for _, field := range []sav.DocumentValueData{{Name: "E3C", Value: uint32(payload.E3C)}, {Name: "E3D", Value: uint32(payload.E3D)}, {Name: "E0C", Value: uint32(payload.E0C)}, {Name: "E40", Value: payload.E40}} {
				value, err := savedStructureValue(child, field.Name)
				if err != nil || value != field.Value {
					return fmt.Errorf("spell payload %s differs", field.Name)
				}
			}
			if child.Class == "Effect_DirectDamage" {
				raw, err := savedMotionRaw(child, "EDD48", 24)
				if err != nil || !slices.Equal(raw, payload.DirectDamage[:]) {
					return fmt.Errorf("spell damage payload differs")
				}
			}
			field := "PE48"
			if n.Value.Class == "AreaEffect" {
				field = "AE44"
			}
			refs, ok := savedObjectRefs(r, field)
			if !ok || !slices.Equal(refs, []uint16{row.PayloadIndex}) {
				return fmt.Errorf("spell payload reference differs")
			}
		}
		if n.Value.Class == "PointEffect" {
			var target sim.EntityID
			found := false
			for _, actor := range state.Actors {
				if actor.ObjectIndex == 0 {
					continue
				}
				key, _ := savedStructureValue(&state.Document.Objects[actor.ObjectIndex-1], "Identity")
				if key == n.Value.PE44 && key != 0 {
					target, found = actor.EntityID, true
					break
				}
			}
			for _, dead := range world.OriginalDeadActors() {
				if dead.Source.Identity == n.Value.PE44 && n.Value.PE44 != 0 {
					target, found = dead.ID, true
				}
			}
			present := false
			for _, entity := range world.Entities() {
				present = present || entity.ID == n.Target
			}
			if found && (!n.HasTarget || n.Target != target) || !found && n.HasTarget && present {
				return fmt.Errorf("spell target binding differs")
			}
		}
	}
	if drivers := world.SavedWorldEffectDrivers(); drivers != nil {
		for _, d := range drivers.Areas {
			if d.ID == 0 || int(d.ID) > len(g.Nodes) || g.Nodes[d.ID-1].Value.Class != "AreaEffect" || d.Root != int32(slices.Index(g.Roots, d.ID)) {
				return fmt.Errorf("spell area root binding differs")
			}
			row := state.WorldEffects.SpellNodes[d.ID-1]
			if row.ObjectIndex == 0 {
				continue
			}
			r := &state.Document.Objects[row.ObjectIndex-1]
			identity, _ := savedStructureValue(r, "Identity")
			spell, _ := savedStructureValue(r, "T0C")
			mode, _ := savedStructureValue(r, "T08")
			want := uint8(sim.AreaModeBlast)
			if mode&2 != 0 {
				want = sim.AreaModeRing
			} else if mode&1 != 0 {
				want = sim.AreaModeCloud
			}
			pos, err := savedMotionRaw(r, "Block12", 12)
			if err != nil || identity != d.Identity || uint16(spell) != d.Spell || d.Mode != want || binary.LittleEndian.Uint16(pos) != d.Key {
				return fmt.Errorf("spell area Token binding differs")
			}
		}
	}
	if !reflect.DeepEqual(copy.Document, state.Document) {
		for i, r := range copy.Document.Objects {
			if !reflect.DeepEqual(r, state.Document.Objects[i]) {
				return fmt.Errorf("current spell graph object %d differs", i+1)
			}
		}
		return fmt.Errorf("current spell graph roots differ: want %v got %v", copy.Document.World.Effects, state.Document.World.Effects)
	}
	return nil
}

func projectSavedSpellGraph(state *SnapshotSAVDocument, world *sim.World) error {
	g := world.SavedSpellGraph()
	if g == nil {
		if len(state.WorldEffects.SpellNodes) == 0 {
			return nil
		}
		if len(world.SavedSpellEffects()) != 0 || spellGraphAreasRemain(world) {
			return fmt.Errorf("spell graph carrier lost while effects remain")
		}
		retiredRoots := make(map[uint16]bool, len(state.WorldEffects.SpellNodes))
		for _, row := range state.WorldEffects.SpellNodes {
			retiredRoots[row.ObjectIndex] = true
		}
		state.Document.World.Effects = slices.DeleteFunc(state.Document.World.Effects, func(index uint16) bool { return retiredRoots[index] })
	} else {
		if err := projectSpellGraphFields(state, g); err != nil {
			return err
		}
	}
	// Only this family's unreachable records may retire. Shared payloads remain
	// while any root or another object's reference still reaches them.
	incoming := savedDocumentIncoming(state.Document)
	removed := map[uint16]bool{}
	var retired []uint16
	for changed := true; changed; {
		changed = false
		for _, row := range state.WorldEffects.SpellNodes {
			for _, id := range []uint16{row.ObjectIndex, row.PayloadIndex} {
				if id == 0 || removed[id] || incoming[id] != 0 {
					continue
				}
				removed[id] = true
				retired = append(retired, id)
				changed = true
				for _, slot := range state.Document.Objects[id-1].RefSlots {
					for _, child := range slot.Objects {
						if child != 0 {
							incoming[child]--
						}
					}
				}
			}
		}
	}
	if len(retired) == 0 {
		if g == nil {
			state.WorldEffects.SpellNodes = nil
		}
		return nil
	}
	for i := range state.WorldEffects.SpellNodes {
		r := &state.WorldEffects.SpellNodes[i]
		if removed[r.ObjectIndex] {
			r.ObjectIndex = 0
		}
		if removed[r.PayloadIndex] {
			r.PayloadIndex = 0
		}
	}
	for i := range state.WorldEffects.Areas {
		r := &state.WorldEffects.Areas[i]
		if removed[r.ObjectIndex] {
			r.ObjectIndex, r.ChildIndex = 0, 0
		}
	}
	doc, permutation, err := sav.RetireDocumentData(*state.Document, retired)
	if err != nil {
		return err
	}
	state.Document = &doc
	if err := remapSavedSackDocument(state, permutation); err != nil {
		return err
	}
	if g == nil {
		state.WorldEffects.SpellNodes = nil
	}
	return nil
}

// spellGraphAreasRemain reports whether an area driver still needs the spell
// graph. A projectile driver has its own records (bindProjectileDrivers), so
// an arrow fired after the loaded graph retired does not refuse the SAVE.
func spellGraphAreasRemain(world *sim.World) bool {
	d := world.SavedWorldEffectDrivers()
	return d != nil && len(d.Areas) > 0
}

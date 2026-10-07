package sim

import (
	"encoding/binary"
	"fmt"
	"slices"
)

type CurrentAreaPolicy struct {
	DerivedPayload bool `json:",omitempty"`
	Power          uint16
	CloudPhase     uint8
	RingLife       uint16
	Caster         EntityID
	HasCaster      bool
}

type CurrentAreaPayload struct {
	Radius  uint8
	Stopped bool
	Payload SavedEffect
}

type CurrentAreaRestore struct {
	Node   uint32
	Policy CurrentAreaPolicy
}

func (w *World) RestoreCurrentAreas(rows []CurrentAreaRestore) error {
	if len(rows) == 0 {
		return nil
	}
	n := *w
	n.savedSpellGraph = cloneSavedSpellGraph(w.savedSpellGraph)
	n.savedWorldEffects = cloneSavedWorldEffects(w.savedWorldEffects)
	n.savedMotion = cloneActorMotions(w.savedMotion)
	n.savedCellRecords = slices.Clone(w.savedCellRecords)
	n.effects = slices.Clone(w.effects)
	if n.savedSpellGraph == nil || n.savedWorldEffects == nil {
		return fmt.Errorf("sim: native areas lack decoded nodes")
	}
	order := n.CurrentWorldEffectOrder()
	converted := map[uint32]uint32{}
	for _, row := range rows {
		id, p := row.Node, row.Policy
		if id == 0 || int(id) > len(n.savedSpellGraph.Nodes) || !p.HasCaster && p.Caster != 0 {
			return fmt.Errorf("sim: invalid native area binding")
		}
		if _, ok := converted[id]; ok {
			return fmt.Errorf("sim: repeated native area binding")
		}
		node := &n.savedSpellGraph.Nodes[id-1]
		if node.Retired || node.Value.Class != "AreaEffect" || node.Value.AE44 == nil {
			return fmt.Errorf("sim: native area node changed class")
		}
		count := 0
		for _, root := range n.savedSpellGraph.Roots {
			if root == id {
				count++
			}
		}
		for _, other := range n.savedSpellGraph.Nodes {
			if other.Primary == id || other.Fallback == id {
				count++
			}
		}
		if count != 1 {
			return fmt.Errorf("sim: native area binding acquired aliases")
		}
		at := slices.IndexFunc(n.savedWorldEffects.Areas, func(d SavedAreaDriver) bool { return d.ID == id })
		if at < 0 {
			return fmt.Errorf("sim: native area driver is absent")
		}
		d := &n.savedWorldEffects.Areas[at]
		v := node.Value
		e := cellEffect{Key: d.Key, Spell: d.Spell, Mode: d.Mode, Direction: v.AE48[2] >> 5, Remaining: v.AE4C,
			Caster: p.Caster, HasCaster: p.HasCaster, Power: p.Power, Phase: p.CloudPhase, Cells: slices.Clone(d.Cells),
			Current: &CurrentAreaPayload{Radius: v.AE48[1], Stopped: v.SE40 != 0, Payload: *v.AE44}}
		if d.Mode == areaModeRing {
			if v.AE48[3] < 1 || v.AE4C > 2 {
				return fmt.Errorf("sim: invalid native ring phase")
			}
			e.Phase = (v.AE48[3]-1)*3 + uint8(2-v.AE4C)
			e.Remaining = p.RingLife
			e.Cells = canonicalCells(n.ringStageCells(e, int(e.Phase/3)))
		}
		// Absence belongs to the native constructor only while all ordinary
		// payload operands still match it. An edited ordinary value stays live.
		if p.DerivedPayload {
			if rule, ok := n.findSpell(uint32(e.Spell)); ok {
				// A direct-damage area may have obtained its pair from the
				// caster at impact (Fire Sacrifice). Publish that pair on the
				// native cellEffect before asking whether the payload is derived;
				// otherwise a cold SAVE would retain an artificial Current node
				// and report a different constructor shape.
				if e.Spell == 4 && e.Current.Payload.Class == "Effect_DirectDamage" && SavedAreaPayloadSupported(&e.Current.Payload) {
					e.DamageMin = int32(e.Current.Payload.DirectDamage[19])
					e.DamageMax = e.DamageMin + int32(e.Current.Payload.DirectDamage[20])
				}
				if !e.Current.Stopped && e.Current.Radius == rule.Radius && e.Current.Payload == areaSavePayload(&n, e, rule) {
					e.Current = nil
				}
			}
		}

		converted[id] = uint32(len(n.effects))
		n.effects = append(n.effects, e)
		for i := range n.savedCellRecords {
			r := &n.savedCellRecords[i]
			for layer, key := range r.SpellEffects {
				if key != d.Identity || key == 0 {
					continue
				}
				r.SpellEffects[layer] = 0
				if c := n.motionCell(r.Cell); c != nil {
					binary.LittleEndian.PutUint32(c.Payload[20+layer*4:], 0)
				}
			}
			r.LayerCount = 0
			for _, key := range r.SpellEffects {
				if key != 0 {
					r.LayerCount++
				}
			}
			if c := n.motionCell(r.Cell); c != nil {
				c.Payload[2] = r.LayerCount
			}
		}
		d.Cells, d.Root = nil, -1
		n.retireSavedSpellNode(id)
	}
	indices := make([]uint32, len(n.savedSpellGraph.Nodes)+1)
	var nodes []SavedSpellNode
	for i, node := range n.savedSpellGraph.Nodes {
		id := uint32(i + 1)
		if _, native := converted[id]; !native {
			indices[id] = uint32(len(nodes) + 1)
			nodes = append(nodes, node)
		}
	}
	for i := range nodes {
		nodes[i].Primary, nodes[i].Fallback = indices[nodes[i].Primary], indices[nodes[i].Fallback]
	}
	for i, id := range n.savedSpellGraph.Roots {
		n.savedSpellGraph.Roots[i] = indices[id]
	}
	n.savedSpellGraph.Nodes = nodes
	n.savedWorldEffects.Areas = slices.DeleteFunc(n.savedWorldEffects.Areas, func(d SavedAreaDriver) bool {
		_, native := converted[d.ID]
		return native
	})
	for i := range n.savedWorldEffects.Areas {
		n.savedWorldEffects.Areas[i].ID = indices[n.savedWorldEffects.Areas[i].ID]
	}
	n.refreshSavedSpellGraph()
	n.effectOrder = order
	for i, ref := range n.effectOrder {
		if ref.Kind == EffectSavedGraph {
			if at, ok := converted[ref.Index]; ok {
				n.effectOrder[i] = WorldEffectRef{EffectNativeArea, at}
			} else {
				n.effectOrder[i].Index = indices[ref.Index]
			}
		}
	}
	n.compactEffectOrder()
	if err := n.worldEffectOrderFault(); err != nil {
		return err
	}
	if err := n.savedSpellGraphFault(true); err != nil {
		return err
	}
	if err := n.savedWorldEffectsFault(); err != nil {
		return err
	}
	*w = n
	return nil
}

package sim

import (
	"fmt"
	"slices"
)

// These operands have no ordinary transport/child field. The current timer,
// spell, target, attribution and prepared payload remain in their own records.
type CurrentDeliveryPolicy struct {
	BirthTick uint64
	FanHead   bool
	Caster    EntityID
	HasCaster bool
	// DerivedArea records the native absence of a materialised area payload.
	// A current restore must let the ordinary landing construct that payload;
	// retaining it as an explicit Current value changes the SAV projection.
	DerivedArea   bool `json:",omitempty"`
	Power         int32
	School        *uint8 `json:",omitempty"`
	Distribution  uint8
	Restorative   bool
	Origin        *CellPoint               `json:",omitempty"`
	ReleasedTimer *uint16                  `json:",omitempty"`
	MissingTarget *EntityID                `json:",omitempty"`
	Special       *CurrentDeliveryOperands `json:",omitempty"`
}

type CurrentDeliveryOperands struct {
	Damaging             bool
	DamageMin, DamageMax int32
	Kind                 EffectKind
	Mode                 EffectMode
	Magnitude            int32
	Duration             uint16
	SpellDuration        int32
}

type CurrentDeliveryRestore struct {
	Node              uint32
	From, Destination CellPoint
	Policy            CurrentDeliveryPolicy
}

func (w *World) RestoreCurrentSpellDeliveries(rows []CurrentDeliveryRestore) error {
	if len(rows) == 0 {
		return nil
	}
	n := *w
	n.savedSpellGraph = cloneSavedSpellGraph(w.savedSpellGraph)
	n.savedWorldEffects = cloneSavedWorldEffects(w.savedWorldEffects)
	n.deliveries = slices.Clone(w.deliveries)
	if n.savedSpellGraph == nil {
		return fmt.Errorf("sim: current delivery has no ordinary graph")
	}
	order := n.CurrentWorldEffectOrder()
	removed, converted := map[uint32]bool{}, map[uint32]WorldEffectRef{}
	incoming := make([]int, len(n.savedSpellGraph.Nodes)+1)
	for _, id := range n.savedSpellGraph.Roots {
		incoming[id]++
	}
	for _, node := range n.savedSpellGraph.Nodes {
		if node.Primary != 0 {
			incoming[node.Primary]++
		}
		if node.Fallback != 0 {
			incoming[node.Fallback]++
		}
	}
	for _, row := range rows {
		id, p := row.Node, row.Policy
		if id == 0 || int(id) > len(n.savedSpellGraph.Nodes) || removed[id] || incoming[id] != 1 {
			return fmt.Errorf("sim: invalid current delivery root")
		}
		root := n.savedSpellGraph.Nodes[id-1]
		if root.Retired || !slices.Contains(n.savedSpellGraph.Roots, id) {
			return fmt.Errorf("sim: current delivery root is not active")
		}
		d := spellDelivery{BirthTick: p.BirthTick, FanHead: p.FanHead, Caster: p.Caster, HasCaster: p.HasCaster, Power: p.Power,
			FromX: row.From.X, FromY: row.From.Y, X: row.Destination.X, Y: row.Destination.Y}
		child := root
		if root.Value.Class == "SpellTransport" {
			childID := root.Primary
			if childID == 0 {
				childID = root.Fallback
			}
			if childID == 0 || removed[childID] || incoming[childID] != 1 {
				return fmt.Errorf("sim: current delivery child acquired aliases")
			}
			child = n.savedSpellGraph.Nodes[childID-1]
			removed[childID] = true
			d.Remaining = root.Value.ST4C
			d.TransportStopped = root.Value.SE40 != 0
		} else {
			d.Released = true
			if p.Origin == nil {
				return fmt.Errorf("sim: released delivery lost its origin")
			}
			d.FromX, d.FromY = p.Origin.X, p.Origin.Y
			if p.ReleasedTimer != nil {
				d.Remaining = *p.ReleasedTimer
			}
		}
		if child.Retired || child.Value.Class != "PointEffect" && child.Value.Class != "AreaEffect" {
			return fmt.Errorf("sim: invalid current delivery child")
		}
		rule, _ := n.findSpell(uint32(child.Spell))
		baseRule := rule
		rule.ID, rule.Delivery, rule.Area, rule.Defensive = child.Spell, 1, child.Value.Class == "AreaEffect", child.Value.SE41 == 0
		if p.School != nil {
			rule.School = *p.School
		}
		rule.Restorative, rule.Distribution = p.Restorative, p.Distribution
		d.Rule, d.AtCell = rule, rule.Area
		d.ChildStopped = child.Value.SE40 != 0
		payload := child.Value.PE48
		if d.AtCell {
			payload = child.Value.AE44
			at := -1
			if n.savedWorldEffects != nil {
				at = slices.IndexFunc(n.savedWorldEffects.Areas, func(v SavedAreaDriver) bool { return v.ID == childNodeID(n.savedSpellGraph, row.Node) })
			}
			if at < 0 {
				return fmt.Errorf("sim: current area delivery lost its driver")
			}
			area := n.savedWorldEffects.Areas[at]
			d.Current = &AreaSaveState{Key: cellKey(d.X, d.Y), Spell: child.Spell, Remaining: child.Value.AE4C, Mode: area.Mode,
				Radius: child.Value.AE48[1], Direction: child.Value.AE48[2] >> 5, Stage: child.Value.AE48[3]}
			d.Rule.Radius = d.Current.Radius
		} else if child.HasTarget {
			d.Target = child.Target
		} else if p.MissingTarget != nil {
			d.Target = *p.MissingTarget
		} else {
			return fmt.Errorf("sim: current point delivery lost its target")
		}
		if payload == nil {
			return fmt.Errorf("sim: current delivery lost its ordinary payload")
		}
		v := *payload
		d.Payload = &v
		if d.Current != nil {
			d.Current.Payload = v
		}
		if p.Special != nil && !SavedAreaPayloadSupported(payload) {
			d.ConstructSacrifice = d.AtCell && child.Spell == 4
			x := p.Special
			d.Rule.Damaging = x.Damaging
			d.Rule.DamageMin, d.Rule.DamageMax, d.Rule.EffectKind, d.Rule.EffectMode = x.DamageMin, x.DamageMax, x.Kind, x.Mode
			d.Rule.EffectMagnitude, d.Rule.EffectDuration, d.Rule.SpellDuration = x.Magnitude, x.Duration, x.SpellDuration
		} else if v.Class == "Effect_DirectDamage" {
			d.Rule.Damaging, d.Rule.Restorative, d.Rule.School = true, false, v.DirectDamage[21]
		} else if kind, mode, ok := savedAreaPayload(&v); ok {
			d.Rule.Damaging, d.Rule.EffectKind, d.Rule.EffectMode = false, kind, mode
			d.Rule.EffectMagnitude = int32(v.E40)
			if mode != 0 {
				d.Rule.EffectMagnitude = int32(int16(v.E40))
			}
		}
		if d.AtCell && p.DerivedArea && d.Current != nil && child.Spell != 4 {
			expected := SavedEffect{Class: "Effect", E0C: uint8(child.Spell)}
			probe := cellEffect{Spell: child.Spell, Power: uint16(d.Power)}
			expected = areaSavePayload(&n, probe, baseRule)
			if d.Current.Radius == baseRule.Radius && d.Current.Payload == expected {
				d.Current = nil
			}
		}
		removed[id] = true
		converted[id] = WorldEffectRef{EffectNativeDelivery, uint32(len(n.deliveries))}
		n.deliveries = append(n.deliveries, d)
	}
	n.detachCurrentSpellNodes(removed, converted, order)
	if err := n.spellDeliveryFault(); err != nil {
		return err
	}
	if err := n.savedSpellGraphFault(true); err != nil {
		return err
	}
	if err := n.savedWorldEffectsFault(); err != nil {
		return err
	}
	if err := n.worldEffectOrderFault(); err != nil {
		return err
	}
	*w = n
	return nil
}

func childNodeID(g *SavedSpellGraph, root uint32) uint32 {
	n := g.Nodes[root-1]
	if n.Value.Class != "SpellTransport" {
		return root
	}
	if n.Primary != 0 {
		return n.Primary
	}
	return n.Fallback
}

func (w *World) detachCurrentSpellNodes(removed map[uint32]bool, replacements map[uint32]WorldEffectRef, order []WorldEffectRef) {
	indices := make([]uint32, len(w.savedSpellGraph.Nodes)+1)
	var nodes []SavedSpellNode
	for i, node := range w.savedSpellGraph.Nodes {
		if !removed[uint32(i+1)] {
			indices[i+1] = uint32(len(nodes) + 1)
			nodes = append(nodes, node)
		}
	}
	for i := range nodes {
		nodes[i].Primary, nodes[i].Fallback = indices[nodes[i].Primary], indices[nodes[i].Fallback]
	}
	var roots []uint32
	for _, id := range w.savedSpellGraph.Roots {
		if indices[id] != 0 {
			roots = append(roots, indices[id])
		}
	}
	w.savedSpellGraph.Nodes, w.savedSpellGraph.Roots = nodes, roots
	if w.savedWorldEffects != nil {
		w.savedWorldEffects.Areas = slices.DeleteFunc(w.savedWorldEffects.Areas, func(d SavedAreaDriver) bool { return removed[d.ID] })
		for i := range w.savedWorldEffects.Areas {
			w.savedWorldEffects.Areas[i].ID = indices[w.savedWorldEffects.Areas[i].ID]
		}
	}
	w.refreshSavedSpellGraph()
	w.effectOrder = nil
	for _, r := range order {
		if r.Kind == EffectSavedGraph {
			if to, ok := replacements[r.Index]; ok {
				r = to
			} else {
				r.Index = indices[r.Index]
				if r.Index == 0 {
					continue
				}
			}
		}
		w.effectOrder = append(w.effectOrder, r)
	}
	w.compactEffectOrder()
}

package sim

import (
	"fmt"
	"slices"
)

const (
	EffectSavedGraph uint8 = iota + 1
	EffectSavedArea
	EffectNativeArea
	EffectNativeDelivery
)

type WorldEffectRef struct {
	Kind  uint8
	Index uint32
}

func (w *World) currentEffectPopulation() []WorldEffectRef {
	var out []WorldEffectRef
	if w.savedSpellGraph != nil {
		for _, id := range w.savedSpellGraph.Roots {
			out = append(out, WorldEffectRef{EffectSavedGraph, id})
		}
	} else if w.savedWorldEffects != nil {
		for _, d := range w.savedWorldEffects.Areas {
			if d.Root >= 0 {
				out = append(out, WorldEffectRef{EffectSavedArea, d.ID})
			}
		}
	}
	for i := range w.effects {
		out = append(out, WorldEffectRef{EffectNativeArea, uint32(i)})
	}
	for i := range w.deliveries {
		out = append(out, WorldEffectRef{EffectNativeDelivery, uint32(i)})
	}
	return out
}

// CurrentWorldEffectOrder is also the original SAV World.Effects order.
func (w *World) CurrentWorldEffectOrder() []WorldEffectRef {
	if w.effectOrder != nil {
		return slices.Clone(w.effectOrder)
	}
	return w.currentEffectPopulation()
}

func (w *World) prepareEffectAppend() {
	if !w.effectWalking && w.effectOrder == nil {
		w.effectOrder = w.currentEffectPopulation()
	}
}

func (w *World) noteEffectAppend(r WorldEffectRef) {
	if !w.effectWalking {
		w.effectOrder = append(w.effectOrder, r)
		w.compactEffectOrder()
	}
}

func (w *World) compactEffectOrder() {
	if slices.Equal(w.effectOrder, w.currentEffectPopulation()) {
		w.effectOrder = nil
	}
}

func (w *World) worldEffectOrderFault() error {
	if w.effectOrder == nil {
		return nil
	}
	counts := map[WorldEffectRef]int{}
	for _, r := range w.currentEffectPopulation() {
		counts[r]++
	}
	for _, r := range w.effectOrder {
		counts[r]--
	}
	for _, n := range counts {
		if n != 0 {
			return fmt.Errorf("sim: world effect order differs from current roots")
		}
	}
	return nil
}

// MAGIC-187/197: the cursor advances before Tick. A tail append waits, but
// a preceding append is reachable in this pass, after the previous tail.
func (w *World) stepWorldSpellEffects(obs *castObs) {
	walk := w.CurrentWorldEffectOrder()
	if len(walk) == 0 {
		return
	}
	w.effectWalking = true
	defer func() { w.effectWalking = false }()
	consumed := make([]bool, len(walk))
	deadAreas := map[uint32]bool{}
	deadDeliveries := map[uint32]bool{}
	seen := map[WorldEffectRef]bool{}
	appendRef := func(r WorldEffectRef) {
		walk = append(walk, r)
		consumed = append(consumed, false)
	}
	for at := 0; at < len(walk); at++ {
		r := walk[at]
		hasNext := at+1 < len(walk)
		if seen[r] {
			continue
		}
		seen[r] = true
		switch r.Kind {
		case EffectSavedGraph:
			g := w.savedSpellGraph
			n := &g.Nodes[r.Index-1]
			if n.Retired {
				consumed[at] = true
				break
			}
			switch n.Value.Class {
			case "SpellTransport":
				n.Value.ST4C--
				if int16(n.Value.ST4C) > 0 {
					if n.Value.SE40 != 0 {
						w.retireSavedSpellNode(r.Index)
					}
					break
				}
				child := n.Primary
				if child == 0 {
					child = n.Fallback
				}
				w.retireSavedSpellNode(r.Index)
				if child != 0 && !g.Nodes[child-1].Retired && !slices.Contains(g.Roots, child) {
					g.Roots = append(g.Roots, child)
					appendRef(WorldEffectRef{EffectSavedGraph, child})
				}
			case "PointEffect":
				w.applySavedPoint(*n)
				w.retireSavedSpellNode(r.Index)
			case "AreaEffect":
				w.refreshSavedSpellGraph()
				w.tickSavedAreaID(r.Index, obs)
				w.captureSavedGraphAreas()
			case "SpellEffect":
				if n.Value.SE40 != 0 {
					w.retireSavedSpellNode(r.Index)
				}
			}
			w.refreshSavedSpellGraph()
		case EffectSavedArea:
			w.tickSavedAreaID(r.Index, obs)
		case EffectNativeArea:
			deadAreas[r.Index] = !w.tickCellEffect(int(r.Index), obs)
		case EffectNativeDelivery:
			d := &w.deliveries[r.Index]
			if !d.Released {
				d.Remaining--
				if int16(d.Remaining) > 0 {
					if d.TransportStopped {
						deadDeliveries[r.Index] = true
					}
					break
				}
				d.Released, d.FanHead = true, false
				consumed[at] = true
				delete(seen, r)
				appendRef(r)
			} else {
				before := len(w.effects)
				if !d.ChildStopped {
					w.applySpellDelivery(*d, obs)
				}
				deadDeliveries[r.Index] = true
				for i := before; i < len(w.effects); i++ {
					added := WorldEffectRef{EffectNativeArea, uint32(i)}
					// Landing ran the first paint or ring stage zero.
					seen[added] = true
					appendRef(added)
				}
			}
		}
		if !hasNext {
			break
		}
	}
	if w.savedSpellGraph != nil {
		w.retireUnreachableSpellNodes()
		w.refreshSavedSpellGraph()
	}
	areaMap, deliveryMap := map[uint32]uint32{}, map[uint32]uint32{}
	areas := w.effects[:0]
	for i, e := range w.effects {
		if !deadAreas[uint32(i)] {
			areaMap[uint32(i)] = uint32(len(areas))
			areas = append(areas, e)
		}
	}
	w.effects = areas
	deliveries := w.deliveries[:0]
	for i, d := range w.deliveries {
		if !deadDeliveries[uint32(i)] {
			deliveryMap[uint32(i)] = uint32(len(deliveries))
			deliveries = append(deliveries, d)
		}
	}
	w.deliveries = deliveries
	active := map[WorldEffectRef]bool{}
	for _, r := range w.currentEffectPopulation() {
		active[r] = true
	}
	w.effectOrder = nil
	for i, r := range walk {
		if consumed[i] {
			continue
		}
		if r.Kind == EffectNativeArea {
			if deadAreas[r.Index] {
				continue
			}
			r.Index = areaMap[r.Index]
		}
		if r.Kind == EffectNativeDelivery {
			if deadDeliveries[r.Index] {
				continue
			}
			r.Index = deliveryMap[r.Index]
		}
		if active[r] {
			w.effectOrder = append(w.effectOrder, r)
		}
	}
	w.compactEffectOrder()
}

func (w *World) tickSavedAreaID(id uint32, obs *castObs) {
	if w.savedWorldEffects != nil {
		for i, d := range w.savedWorldEffects.Areas {
			if d.ID == id {
				w.tickSavedArea(i, obs)
				return
			}
		}
	}
}

func (w *World) retireUnreachableSpellNodes() {
	g := w.savedSpellGraph
	live := map[uint32]bool{}
	var visit func(uint32)
	visit = func(id uint32) {
		if id == 0 || live[id] {
			return
		}
		live[id] = true
		n := g.Nodes[id-1]
		visit(n.Primary)
		visit(n.Fallback)
	}
	for _, id := range g.Roots {
		visit(id)
	}
	for i := range g.Nodes {
		if !live[uint32(i+1)] {
			if !g.Nodes[i].Retired && g.Nodes[i].Value.Class == "AreaEffect" && w.savedWorldEffects != nil {
				for j, d := range w.savedWorldEffects.Areas {
					if d.ID == uint32(i+1) {
						w.retireSavedArea(j)
					}
				}
			}
			w.retireSavedSpellNode(uint32(i + 1))
		}
	}
}

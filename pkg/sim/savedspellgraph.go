package sim

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"slices"
	"sort"
)

type SavedSpellNode struct {
	Value             SavedSpellEffect
	Spell             uint16
	Primary, Fallback uint32
	Target            EntityID
	HasTarget         bool
	Retired           bool
	// Native caster attribution has no established original wire field.
	// Omitted fields retain the historical imported graph's unknown caster.
	Caster    EntityID `json:"Caster,omitempty"`
	HasCaster bool     `json:"HasCaster,omitempty"`
}

// Node IDs are one-based import ordinals. Roots preserve list order and aliases.
type SavedSpellGraph struct {
	Nodes []SavedSpellNode
	Roots []uint32
}

func cloneSavedSpellGraph(g *SavedSpellGraph) *SavedSpellGraph {
	if g == nil {
		return nil
	}
	out := &SavedSpellGraph{Nodes: slices.Clone(g.Nodes), Roots: slices.Clone(g.Roots)}
	for i := range out.Nodes {
		n := &out.Nodes[i]
		if n.Value.PE48 != nil {
			v := *n.Value.PE48
			n.Value.PE48 = &v
		}
		if n.Value.AE44 != nil {
			v := *n.Value.AE44
			n.Value.AE44 = &v
		}
	}
	return out
}

func (w *World) SavedSpellGraph() *SavedSpellGraph { return cloneSavedSpellGraph(w.savedSpellGraph) }

func (w *World) ImportSavedSpellGraph(g *SavedSpellGraph) error {
	next := *w
	next.savedSpellGraph = cloneSavedSpellGraph(g)
	next.savedWorldEffects = cloneSavedWorldEffects(w.savedWorldEffects)
	if err := next.savedSpellGraphFault(false); err != nil {
		return err
	}
	next.refreshSavedSpellGraph()
	if err := next.savedWorldEffectsFault(); err != nil {
		return err
	}
	w.savedSpellGraph, w.savedSpellEffects, w.savedWorldEffects = next.savedSpellGraph, next.savedSpellEffects, next.savedWorldEffects
	return nil
}

func (w *World) savedSpellGraphFault(compare bool) error {
	g := w.savedSpellGraph
	if g == nil {
		return nil
	}
	fail := func() error { return fmt.Errorf("sim: invalid saved spell graph") }
	if len(g.Nodes) > 65534 || len(g.Roots) > 65534 {
		return fail()
	}
	for _, id := range g.Roots {
		if id == 0 || int(id) > len(g.Nodes) || g.Nodes[id-1].Retired {
			return fail()
		}
	}
	for _, n := range g.Nodes {
		if n.Value.ST44 != nil || n.Value.ST48 != nil || savedSpellEffectTreeFault(n.Value, 1) != nil || !n.HasTarget && n.Target != 0 || !n.HasCaster && n.Caster != 0 {
			return fail()
		}
		if n.Value.Class != "SpellTransport" && (n.Primary != 0 || n.Fallback != 0) {
			return fail()
		}
		for _, id := range []uint32{n.Primary, n.Fallback} {
			if int(id) > len(g.Nodes) || id != 0 && g.Nodes[id-1].Retired {
				return fail()
			}
		}
		if n.Fallback != 0 && g.Nodes[n.Fallback-1].Value.Class != "AreaEffect" {
			return fail()
		}
	}
	heights := make([]int, len(g.Nodes))
	var height func(uint32) int
	height = func(id uint32) int {
		if id == 0 {
			return 0
		}
		if heights[id-1] != 0 {
			return heights[id-1]
		}
		heights[id-1] = savedSpellEffectMaxDepth + 1
		n := g.Nodes[id-1]
		h := 1 + max(height(n.Primary), height(n.Fallback))
		heights[id-1] = h
		return h
	}
	for i := range g.Nodes {
		if height(uint32(i+1)) > savedSpellEffectMaxDepth {
			return fail()
		}
	}
	if compare && !reflect.DeepEqual(w.savedSpellEffects, g.values()) {
		return fail()
	}
	return nil
}

func (g *SavedSpellGraph) values() []SavedSpellEffect {
	if len(g.Roots) == 0 {
		return nil
	}
	cache := make(map[uint32]*SavedSpellEffect)
	var node func(uint32) *SavedSpellEffect
	node = func(id uint32) *SavedSpellEffect {
		if id == 0 {
			return nil
		}
		if v := cache[id]; v != nil {
			return v
		}
		n := g.Nodes[id-1]
		v := n.Value
		cache[id] = &v
		v.ST44, v.ST48 = node(n.Primary), node(n.Fallback)
		return &v
	}
	out := make([]SavedSpellEffect, len(g.Roots))
	for i, id := range g.Roots {
		out[i] = *node(id)
	}
	return out
}

func (w *World) refreshSavedSpellGraph() {
	g := w.savedSpellGraph
	w.savedSpellEffects = g.values()
	if w.savedWorldEffects != nil {
		for i := range w.savedWorldEffects.Areas {
			d := &w.savedWorldEffects.Areas[i]
			d.Root = int32(slices.Index(g.Roots, d.ID))
		}
	}
}

func (w *World) retireSavedSpellNode(id uint32) {
	g := w.savedSpellGraph
	g.Nodes[id-1].Retired = true
	g.Nodes[id-1].Primary, g.Nodes[id-1].Fallback = 0, 0
	g.Roots = slices.DeleteFunc(g.Roots, func(root uint32) bool { return root == id })
	for i := range g.Nodes {
		n := &g.Nodes[i]
		if n.Primary == id {
			n.Primary = 0
		}
		if n.Fallback == id {
			n.Fallback = 0
		}
	}
}

func (w *World) paintSavedArea(i int) {
	defer w.holdLayerCosts()()
	d := &w.savedWorldEffects.Areas[i]
	v, bound := w.savedAreaValue(*d)
	if !bound || d.Mode != AreaModeCloud || v.AE48[0] != 0 {
		return
	}
	rule, ok := w.findSpell(uint32(d.Spell))
	if !ok {
		return
	}
	x, y := keyCell(d.Key)
	e := cellEffect{Key: d.Key, Spell: d.Spell, Mode: AreaModeCloud, Direction: v.AE48[2] >> 5}
	if rule.Distribution == distributionWall {
		e.Cells = w.wallCells(x, y, e.Direction)
		if w.spellArm(d.Spell) == 19 {
			e.Cells = w.skipOccupiedGround(e.Cells)
		}
	} else {
		e.Cells = w.diamondCells(x, y, int32(v.AE48[1]))
	}
	w.resolveLayerConflicts(&e)
	d.Cells = canonicalCells(e.Cells)
	for _, key := range d.Cells {
		if w.createSavedCell(key) != "" {
			continue
		}
		at := sort.Search(len(w.savedCellRecords), func(i int) bool { return w.savedCellRecords[i].Cell >= key })
		if at == len(w.savedCellRecords) || w.savedCellRecords[at].Cell != key {
			w.savedCellRecords = slices.Insert(w.savedCellRecords, at, SavedCellRecord{Cell: key})
		}
		c := w.motionCell(key)
		r := &w.savedCellRecords[at]
		r.SpellEffects[d.Layer] = d.Identity
		r.LayerCount = 0
		for _, identity := range r.SpellEffects {
			if identity != 0 {
				r.LayerCount++
			}
		}
		c.Payload[2] = r.LayerCount
		binary.LittleEndian.PutUint32(c.Payload[20+4*d.Layer:], d.Identity)
		w.recomputeSavedCell(key)
	}
	w.scorchCells(w.spellArm(d.Spell), e.Cells)
	if w.spellArm(d.Spell) == 19 {
		w.setWallCells(d.Cells, true)
	}
	w.refreshSavedPlaneBlocks()
	if d.Root >= 0 {
		w.savedSpellEffects[d.Root].AE48[0] = 1
	}
	if g := w.savedSpellGraph; g != nil {
		g.Nodes[d.ID-1].Value.AE48[0] = 1
		w.refreshSavedSpellGraph()
	}
}

func (w *World) applySavedPoint(n SavedSpellNode) {
	if !n.HasTarget {
		return
	}
	ti := indexOfEntity(w.entities, n.Target)
	rule, ok := w.findSpell(uint32(n.Spell))
	if !ok {
		return
	}
	rule.Defensive = n.Value.SE41 == 0
	ci := -1
	if n.HasCaster {
		ci = indexOfEntity(w.entities, n.Caster)
	}
	w.applyFrozenPoint(ci, ti, rule, n.Value.PE48)
}

func (w *World) applyFrozenPoint(ci, ti int, rule SpellRule, payload *SavedEffect) {
	if ti < 0 || payload == nil || !spellTargetable(w.entities[ti], rule) {
		return
	}
	if payload.Class == "Effect_DirectDamage" && SavedAreaPayloadSupported(payload) {
		rule.Delivery, rule.Damaging, rule.Restorative = 1, true, false
		rule.DamageMin = int32(payload.DirectDamage[19])
		rule.DamageMax = rule.DamageMin + int32(payload.DirectDamage[20])
		rule.School = payload.DirectDamage[21]
		if w.ordinaryEffect(ci, ti, rule, 0) {
			w.markSpellEffect(ti, rule.ID)
		}
		return
	}
	kind, mode, known := savedAreaPayload(payload)
	if !known {
		return
	}
	applied := false
	if mode == 0 {
		_, ok := w.applyEffectDelta(ti, kind, int32(payload.E40))
		applied = ok || w.entities[ti].ActorLoad.Source.Class == 0
		if applied {
			w.clearFelled(ti)
		}
	} else {
		caster := ^EntityID(0)
		if ci >= 0 {
			caster = w.entities[ci].ID
		}
		applied = w.attachEffect(w.entities[ti].ID, caster, rule, kind, int32(int16(payload.E40)), uint16(payload.E40>>16), mode)
	}
	if applied {
		w.pointAttribution(ci, ti, rule)
		w.markSpellEffect(ti, rule.ID)
	}
}

func (w *World) captureSavedGraphAreas() {
	if w.savedSpellGraph == nil || w.savedWorldEffects == nil {
		return
	}
	for _, d := range w.savedWorldEffects.Areas {
		if d.Root >= 0 {
			w.savedSpellGraph.Nodes[d.ID-1].Value = w.savedSpellEffects[d.Root]
		}
	}
}

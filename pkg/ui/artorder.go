package ui

import (
	"image"
	"sort"

	"againrom/pkg/render/terrain"
)

// artOrder describes admitted draws, not the original pointer grids' lifetime.
// ANIM-CELL-085/AIRPASS-086/WALKORDER-088 separate phases, then cells (row
// ascending, column descending), then category dispatches within a cell.
type artOrder struct {
	phase, rank int
	cell        image.Point
	index       int
	part        int // a caster's shadow precedes its body and attached marks
}

const (
	artStructureShadow = 1
	artAlternate       = 3
	artMainCell        = 4
	artAirShadow       = 5
	artProjectile      = 6
	artCloud           = 7
	artAirBody         = 8
	artOutline         = 9 // the Z sack outlines, over all art
)

func (a artOrder) before(b artOrder) bool {
	if a.phase != b.phase {
		return a.phase < b.phase
	}
	if a.cell.Y != b.cell.Y {
		return a.cell.Y < b.cell.Y
	}
	if a.cell.X != b.cell.X {
		return a.cell.X > b.cell.X
	}
	if a.rank != b.rank {
		return a.rank < b.rank
	}
	if a.index != b.index {
		return a.index < b.index
	}
	return a.part < b.part
}

func entityArtOrder(p terrain.StaticPlacement, e MapEntity, index int) artOrder {
	k := artOrder{phase: artMainCell, cell: p.Cell, rank: 3, index: index, part: 1}
	switch e.DrawCategory {
	case terrain.UnitAlternate:
		k.phase = artAlternate
	case terrain.UnitAir:
		k.phase = artAirBody
	default:
		// Native sacks have no proven CBackPack lifecycle join. Preserve the
		// owner's dead-body < sack < living-unit tie in the main cell phase.
		// A fallen CUnit is still ordinary; do not label stage1 as selector4.
		if p.DepthTie == terrain.TieCorpse {
			k.rank = 1
		}
	}
	return k
}

type artDraw struct {
	order  artOrder
	sprite *staticScreenRect
	shadow *shadowDraw
	spell  *effectScreenRect
}

// drawArt follows the accepted category-specific composition. Structure
// shadows precede flat decoration. The main cell sweep ends each cell with
// static art and Fire/Earth. Only CAirUnit gets separated late shadow/body
// sweeps around the projectile collection and Freezing/Poison cell sweep.
// Native multi-entry cells retain stable input order (DIV-1142).
func (v *Viewer) drawArt(target imageTarget) {
	shadows := v.shadowDraws()
	var draws []artDraw
	for i := range shadows {
		d := &shadows[i]
		if d.order.phase == artStructureShadow {
			continue
		}
		draws = append(draws, artDraw{order: d.order, shadow: d})
	}
	// The original structure-shadow phase precedes both Flat body phases.
	var structureShadows []shadowDraw
	for _, d := range shadows {
		if d.order.phase == artStructureShadow {
			structureShadows = append(structureShadows, d)
		}
	}
	sort.SliceStable(structureShadows, func(i, j int) bool { return structureShadows[i].order.before(structureShadows[j].order) })
	for _, d := range structureShadows {
		v.drawShadow(target, d)
	}
	v.drawStructuresFlat(target)
	sprites := v.planeSprites()
	for i := range sprites {
		draws = append(draws, artDraw{order: sprites[i].order, sprite: &sprites[i]})
	}
	spells := v.spellArtPlacements()
	for i := range spells {
		s := &spells[i]
		k := artOrder{phase: artProjectile, index: i}
		switch s.Pass {
		case SpellProjectiles:
		case SpellOverlayA:
			k.phase, k.cell, k.rank = artMainCell, s.Cell, 5
		case SpellOverlayB:
			k.phase, k.cell = artCloud, s.Cell
		default:
			continue
		}
		draws = append(draws, artDraw{order: k, spell: s})
	}
	sort.SliceStable(draws, func(i, j int) bool { return draws[i].order.before(draws[j].order) })
	for _, d := range draws {
		switch {
		case d.sprite != nil:
			v.drawPlaneSprite(target, *d.sprite)
		case d.shadow != nil:
			v.drawShadow(target, *d.shadow)
		case d.spell != nil:
			v.drawSpellArt(target, []effectScreenRect{*d.spell}, d.spell.Pass)
		}
	}
}

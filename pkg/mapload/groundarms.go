package mapload

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
)

// GroundArm is one reason a cell is closed to a ground mover.
type GroundArm uint8

const (
	GroundBorder    GroundArm = 1 << iota // the eight-cell ring round the map
	GroundMountain                        // the classifier's Mountain
	GroundWater                           // the raw water index range
	GroundFlag                            // the tile word's impassable bit 13
	GroundObject                          // a static object on the overlay plane
	GroundStructure                       // a placed structure's footprint
)

type markCell struct {
	arms
	tile      uint16
	overlay   uint8
	structure bool
}

// MarkRow names a matching cell and supplies its stored value.
type MarkRow struct {
	Arm   GroundArm
	Word  string
	Value uint8
	on    func(markCell) bool
	final bool // this value overrides all later marks
}

// MarkTable lists marks in hover order. A format without stored values leaves PlaneValueRows empty.
type MarkTable struct {
	Rows           []MarkRow
	PlaneValueRows []PlaneValueRow
	groundPlane    bool // apply the world's terrain groups and footprint overrides
}

func borderMark(c markCell) bool   { return c.border }
func mountainMark(c markCell) bool { return c.mountain }
func flagMark(c markCell) bool     { return c.impassable }
func objectMark(c markCell) bool   { return c.scenery }

var FirstMarks = MarkTable{
	groundPlane: true,
	Rows: []MarkRow{
		{Arm: GroundBorder, Word: "editor.pass.border", on: borderMark},
		{Arm: GroundMountain, Word: "editor.pass.mountain", on: mountainMark},
		{Arm: GroundWater, Word: "editor.pass.water", on: func(c markCell) bool { return c.water }},
		{Arm: GroundFlag, Word: "editor.pass.flag", on: flagMark},
		{Arm: GroundObject, Word: "editor.pass.object", on: objectMark},
		{Arm: GroundStructure, Word: "editor.pass.structure", on: func(c markCell) bool { return c.structure }},
	},
}

// GroundArms records every matching mark in a cell. Attached cells use the
// footprint's final ground bit in place of their terrain.
func (marks MarkTable) GroundArms(m *alm.Map, t *Table) []GroundArm {
	w, h, ok := extent(m)
	if !ok {
		return nil
	}
	var plane []byte
	var taken []bool
	var groups *mod.TerrainData
	if marks.groundPlane {
		plane, _, taken = deriveTaken(m, t)
		groups = t.ModTerrain()
	}
	out := make([]GroundArm, w*h)
	for i := range out {
		var c markCell
		if taken != nil && taken[i] {
			if plane[i]&blockGround == 0 {
				continue
			}
			c.structure = true
		} else {
			c.tile = planeAt(m.Tiles, i)
			c.overlay = planeAt(m.Overlay, i)
			c.arms = cellArms(c.tile, c.overlay, i%w, i/w, w, h, groups)
		}
		for _, r := range marks.Rows {
			if r.on(c) {
				out[i] |= r.Arm
			}
		}
	}
	return out
}

// GroundType3 is a nonzero type-3 byte, which the second game's terrain build
// marks with the value 5.
const GroundType3 GroundArm = GroundStructure << 1

// GroundClass8 is a tile word the second game's classifier gives class 8,
// which its terrain build marks with the value 1.
const GroundClass8 GroundArm = GroundType3 << 1

var SecondMarks = MarkTable{
	Rows: []MarkRow{
		{Arm: GroundBorder, Word: "editor.pass.border", Value: RingValue, on: borderMark, final: true},
		{Arm: GroundFlag, Word: "editor.mark.flag", Value: 1, on: flagMark},
		{Arm: GroundClass8, Word: "editor.mark.class8", Value: 1, on: mountainMark},
		{Arm: GroundWater, Word: "editor.mark.water", Value: 1, on: func(c markCell) bool { return alm.TileIndex(c.tile)&0x300 == 0x200 }},
		{Arm: GroundType3, Word: "editor.mark.type3", Value: 5, on: objectMark},
	},
	PlaneValueRows: planeValueRows([]PlaneValueRow{
		{Value: 0, Word: "editor.pass.value.0"},
		{Value: 1, Word: "editor.pass.value.1"},
		{Value: 5, Word: "editor.pass.value.5"},
		{Value: RingValue, Word: "editor.pass.value.31"},
	}, sim.SecondGameMoveKindMasks()),
}

// RingValue is what the ring routine stores over the 8-cell border ring
// (R2-ENGINE-1341).
const RingValue = 0x1f

// PlaneValueRow is one value the second game's terrain build stores for a
// cell, the move kinds it closes (bit k-1 for kind k) and the word the editor
// names it by.
type PlaneValueRow struct {
	Value, Kinds uint8
	Word         string
}

func planeValueRows(rows []PlaneValueRow, masks [3]uint8) []PlaneValueRow {
	for i := range rows {
		for k, mask := range masks {
			if rows[i].Value&mask != 0 {
				rows[i].Kinds |= 1 << k
			}
		}
	}
	return rows
}

// MarkValue returns the last matching store unless a final mark takes precedence.
func (marks MarkTable) MarkValue(arms GroundArm) uint8 {
	var v uint8
	for _, r := range marks.Rows {
		if arms&r.Arm != 0 {
			v = r.Value
			if r.final {
				break
			}
		}
	}
	return v
}

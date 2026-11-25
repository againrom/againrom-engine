package mapload

import (
	"againrom/pkg/formats/alm"
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

// GroundArmRow is one arm with the word ID the editor names it by.
type GroundArmRow struct {
	Arm  GroundArm
	Word string
}

// GroundArmRows is every arm, in the order the editor names a cell's arms.
// It is the one table of the arms: the host's refusals and the editor's
// passability view both read it.
var GroundArmRows = [...]GroundArmRow{
	{Arm: GroundBorder, Word: "editor.pass.border"},
	{Arm: GroundMountain, Word: "editor.pass.mountain"},
	{Arm: GroundWater, Word: "editor.pass.water"},
	{Arm: GroundFlag, Word: "editor.pass.flag"},
	{Arm: GroundObject, Word: "editor.pass.object"},
	{Arm: GroundStructure, Word: "editor.pass.structure"},
}

// GroundArms is, per in-bounds cell and row-major, why PassabilityWith(m, t)
// closes the cell to a ground mover: zero exactly where that plane's ground
// bit is clear. It reads the plane and the occupancy of the one derivation
// the world is built from. A cell a structure attached carries
// GroundStructure alone, since the footprint overrides every terrain arm
// there; any other closed cell carries each terrain arm that holds for it.
func GroundArms(m *alm.Map, t *Table) []GroundArm {
	plane, _, taken := deriveTaken(m, t)
	w, h, ok := extent(m)
	if !ok {
		return nil
	}
	out := make([]GroundArm, w*h)
	for i := range out {
		if plane[i]&blockGround == 0 {
			continue
		}
		if taken != nil && taken[i] {
			out[i] = GroundStructure
			continue
		}
		a := cellArms(planeAt(m.Tiles, i), planeAt(m.Overlay, i), i%w, i/w, w, h, t.ModTerrain())
		for _, arm := range [...]struct {
			on  bool
			arm GroundArm
		}{{a.border, GroundBorder}, {a.mountain, GroundMountain}, {a.water, GroundWater}, {a.impassable, GroundFlag}, {a.scenery, GroundObject}} {
			if arm.on {
				out[i] |= arm.arm
			}
		}
	}
	return out
}

// GroundType3 is a nonzero type-3 byte, which the second game's terrain build
// marks with the value 5.
const GroundType3 GroundArm = GroundStructure << 1

// MarkRow is one cell word the second game's terrain build marks in its
// plane, with the value it stores and the word the editor names it by.
type MarkRow struct {
	Arm   GroundArm
	Word  string
	Value uint8
	On    func(tile uint16, overlay uint8) bool
}

// GroundClass8 is a tile word the second game's classifier gives class 8,
// which its terrain build marks with the value 1.
const GroundClass8 GroundArm = GroundType3 << 1

// MarkRows are the cells the second game's terrain build marks, in the order
// it stores them (R2-ENGINE-1340).
var MarkRows = [...]MarkRow{
	{Arm: GroundFlag, Word: "editor.mark.flag", Value: 1, On: func(t uint16, _ uint8) bool { return alm.Impassable(t) }},
	{Arm: GroundClass8, Word: "editor.mark.class8", Value: 1, On: func(t uint16, _ uint8) bool { return TileClass(t) == classMountain }},
	{Arm: GroundWater, Word: "editor.mark.water", Value: 1, On: func(t uint16, _ uint8) bool { return alm.TileIndex(t)&0x300 == 0x200 }},
	{Arm: GroundType3, Word: "editor.mark.type3", Value: 5, On: func(_ uint16, o uint8) bool { return o != 0 }},
}

// RingValue is what the ring routine stores over the 8-cell border ring
// (R2-ENGINE-1341).
const RingValue = 0x1f

// MoveKindMasks are the block masks of move kinds 1, 2 and 3: a cell whose
// value AND the mask is nonzero is closed to that kind (R2-ENGINE-809,
// R2-ENGINE-1804).
// They are read from pkg/sim, where the creature-aim profile holds them.
var MoveKindMasks = sim.SecondGameMoveKindMasks()

// PlaneValueRow is one value the second game's terrain build stores for a
// cell, the move kinds it closes (bit k-1 for kind k) and the word the editor
// names it by.
type PlaneValueRow struct {
	Value, Kinds uint8
	Word         string
}

// planeValues are the values the terrain build stores from a map's bytes:
// none, the marks' 1 and 5 and the ring's (R2-ENGINE-1805, Medium).
var planeValues = [...]PlaneValueRow{
	{Value: 0, Word: "editor.pass.value.0"},
	{Value: 1, Word: "editor.pass.value.1"},
	{Value: 5, Word: "editor.pass.value.5"},
	{Value: RingValue, Word: "editor.pass.value.31"},
}

// PlaneValueRows are planeValues with the kinds each closes by MoveKindMasks.
func PlaneValueRows() []PlaneValueRow {
	out := append([]PlaneValueRow(nil), planeValues[:]...)
	for i := range out {
		for k, mask := range MoveKindMasks {
			if out[i].Value&mask != 0 {
				out[i].Kinds |= 1 << k
			}
		}
	}
	return out
}

// MarkValue is the value the terrain build leaves for a cell of arms: the
// ring's when the ring holds it, else the last mark stored, else 0.
func MarkValue(arms GroundArm) uint8 {
	if arms&GroundBorder != 0 {
		return RingValue
	}
	var v uint8
	for _, r := range MarkRows {
		if arms&r.Arm != 0 {
			v = r.Value
		}
	}
	return v
}

// MarkArmRows is the arm table of a map whose cells carry marks: the ring and
// each mark.
func MarkArmRows() []GroundArmRow {
	out := []GroundArmRow{GroundArmRows[0]}
	for _, r := range MarkRows {
		out = append(out, GroundArmRow{Arm: r.Arm, Word: r.Word})
	}
	return out
}

// MarkedArms is, per cell and row-major, the border ring and each mark the
// second game's terrain build stores for the cell; MarkValue gives the value
// they leave.
func MarkedArms(m *alm.Map) []GroundArm {
	w, h, ok := extent(m)
	if !ok {
		return nil
	}
	out := make([]GroundArm, w*h)
	for i := range out {
		if InBorderRing(m, i%w, i/w) {
			out[i] = GroundBorder
		}
		for _, r := range MarkRows {
			if r.On(planeAt(m.Tiles, i), planeAt(m.Overlay, i)) {
				out[i] |= r.Arm
			}
		}
	}
	return out
}

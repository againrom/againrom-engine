package mapload

import (
	"image"

	"againrom/pkg/formats/alm"
)

// A trigger's places on the map: the cells, boxes and distance squares its
// conditions and actions name. X and Y slots are tile coordinates
// (ALM-TRIG-046); check 2 is a box between two corners, inclusive
// (TRIG-CHECKGEOM-130); every distance a check measures is Chebyshev, so a
// radius bounds a square of side 2r+1 (TRIG-DIST-014).

// ScriptAreaShape is how an area is drawn.
type ScriptAreaShape uint8

const (
	AreaCell   ScriptAreaShape = iota // one cell
	AreaBox                           // a box between two corners
	AreaSquare                        // the cells within a Chebyshev radius of a centre
)

// ScriptArea is one place a trigger names. Rect is the cells it covers, Max
// exclusive and clipped to the map; Centre and Radius are a square's.
type ScriptArea struct {
	Trigger   int // index in the script's triggers
	Condition bool
	Node      int // index in the conditions or the actions
	Opcode    uint32
	Shape     ScriptAreaShape
	Rect      image.Rectangle
	Centre    image.Point
	Radius    int
	// Slots are the node's parameter slots its cells come from: a box's
	// X, Y, X2, Y2, a cell's or a square's centre X, Y in the first two;
	// -1 where none, as for a square around a unit.
	Slots [4]int
	// RadiusNode is the condition whose slot RadiusSlot holds the radius
	// plus RadiusBias; -1 when no parameter holds it.
	RadiusNode, RadiusSlot, RadiusBias int
}

// noSlots marks an area no parameter places.
var noSlots = [4]int{-1, -1, -1, -1}

// ScriptPlaceNote is one node a trigger names, with whether it is drawn and,
// when not, why: the census's row.
type ScriptPlaceNote struct {
	Trigger   int
	Condition bool
	Node      int
	Opcode    uint32
	Drawn     bool
	Reason    string
}

// The reasons a named node draws no area.
const (
	NoPlace       = "names no place"
	UnitNotPlaced = "its unit stands on no map cell (hero ordinal, static name or no unique placement)"
	OutsideMap    = "its cell lies outside the map"
	ReversedBox   = "its box has a corner above the opposite corner: it never holds"
	PointOnly     = "a distance with no constant to compare: its point is drawn"
	Unresolved    = "its id names no single node"
	WideRadius    = "its radius is missing or outside 0..255"
)

// ScriptAreas are the areas of every trigger of sc on m, and one note per
// node a trigger names.
func ScriptAreas(m *alm.Map, sc alm.Script) ([]ScriptArea, []ScriptPlaceNote) {
	p := placer{m: m, sc: sc, units: map[uint32][]image.Point{}}
	for _, u := range m.Units {
		p.units[uint32(u.UnitID)] = append(p.units[uint32(u.UnitID)], image.Pt(int(u.X>>8), int(u.Y>>8)))
	}
	for t, tr := range sc.Triggers {
		seen := map[[2]uint32]bool{}
		for k := range tr.Left {
			for side, id := range []uint32{tr.Left[k], tr.Right[k]} {
				other := tr.Right[k]
				if side == 1 {
					other = tr.Left[k]
				}
				if id == 0 || seen[[2]uint32{1, id}] {
					continue
				}
				seen[[2]uint32{1, id}] = true
				p.node(t, true, id, comparison{other: other, code: tr.Cmp[k], left: side == 0})
			}
		}
		for _, id := range tr.Acts {
			if id == 0 || seen[[2]uint32{0, id}] {
				continue
			}
			seen[[2]uint32{0, id}] = true
			p.node(t, false, id, comparison{})
		}
	}
	return p.areas, p.notes
}

type placer struct {
	m     *alm.Map
	sc    alm.Script
	units map[uint32][]image.Point
	areas []ScriptArea
	notes []ScriptPlaceNote
}

// comparison is the other side of the pair a condition stands in: its
// condition id, the comparison code and whether the condition is the left.
type comparison struct {
	other, code uint32
	left        bool
}

func scriptNodeIndex(nodes []alm.ScriptNode, id uint32) int {
	at := -1
	for i, n := range nodes {
		if n.ID == id {
			if at >= 0 {
				return -1
			}
			at = i
		}
	}
	return at
}

func (p *placer) node(t int, condition bool, id uint32, c comparison) {
	nodes := p.sc.Actions
	if condition {
		nodes = p.sc.Conditions
	}
	i := scriptNodeIndex(nodes, id)
	note := ScriptPlaceNote{Trigger: t, Condition: condition, Node: i}
	if i < 0 {
		note.Reason = Unresolved
		p.notes = append(p.notes, note)
		return
	}
	n := nodes[i]
	note.Opcode = n.Opcode
	add := func(shape ScriptAreaShape, r image.Rectangle, centre image.Point, radius int, slots [4]int, rad radiusSource) {
		p.areas = append(p.areas, ScriptArea{Trigger: t, Condition: condition, Node: i, Opcode: n.Opcode, Shape: shape,
			Rect: r.Intersect(image.Rect(0, 0, p.m.Width, p.m.Height)), Centre: centre, Radius: radius,
			Slots: slots, RadiusNode: rad.node, RadiusSlot: rad.slot, RadiusBias: rad.bias})
		note.Drawn = true
	}
	cells, at := scriptCells(n), scriptCellSlots(n)
	cellSlots := func(k int) [4]int { return [4]int{at[k], at[k] + 1, -1, -1} }
	reason := NoPlace
	switch {
	case condition && n.Opcode == 2 && len(cells) == 2:
		a, b := cells[0], cells[1]
		if a.X > b.X || a.Y > b.Y {
			reason = ReversedBox
			break
		}
		if !p.inside(a) && !p.inside(b) {
			reason = OutsideMap
			break
		}
		add(AreaBox, image.Rectangle{Min: a, Max: b.Add(image.Pt(1, 1))}, a, 0, [4]int{at[0], at[0] + 1, at[1], at[1] + 1}, noRadius)
	case condition && n.Opcode == 3 && len(cells) == 1:
		r, ok := scriptPlain(n, 0)
		slot := scriptPlainSlot(n, 0)
		if !ok || r > 255 {
			reason = WideRadius
			break
		}
		if !p.inside(cells[0]) {
			reason = OutsideMap
			break
		}
		add(AreaSquare, square(cells[0], int(r)), cells[0], int(r), cellSlots(0), radiusSource{node: i, slot: slot})
	case condition && n.Opcode == 6:
		reason = p.unitSquare(n, c, add)
	case condition && (n.Opcode == 7 || n.Opcode == 15 || n.Opcode == 16) && len(cells) == 1:
		if !p.inside(cells[0]) {
			reason = OutsideMap
			break
		}
		if r, rad, ok := p.radius(c); ok {
			add(AreaSquare, square(cells[0], r), cells[0], r, cellSlots(0), rad)
		} else {
			add(AreaCell, square(cells[0], 0), cells[0], 0, cellSlots(0), noRadius)
			reason = PointOnly
		}
	default:
		reason = OutsideMap
		for k, cl := range cells {
			if p.inside(cl) {
				add(AreaCell, square(cl, 0), cl, 0, cellSlots(k), noRadius)
			}
		}
		if len(cells) == 0 {
			reason = NoPlace
		}
	}
	if !note.Drawn || reason == PointOnly {
		note.Reason = reason
	}
	p.notes = append(p.notes, note)
}

// unitSquare draws check 6's distance as a square around its first unit's
// placement, the radius its constant comparison gives.
func (p *placer) unitSquare(n alm.ScriptNode, c comparison, add func(ScriptAreaShape, image.Rectangle, image.Point, int, [4]int, radiusSource)) string {
	var id uint32
	found := false
	for s, tag := range n.Type {
		if tag == ScriptUnit {
			id, found = n.Value[s], true
			break
		}
	}
	at := p.units[id]
	if !found || id >= 10001 || len(at) != 1 {
		return UnitNotPlaced
	}
	if !p.inside(at[0]) {
		return OutsideMap
	}
	r, _, ok := p.radius(c)
	if !ok {
		add(AreaCell, square(at[0], 0), at[0], 0, noSlots, noRadius)
		return PointOnly
	}
	add(AreaSquare, square(at[0], r), at[0], r, noSlots, noRadius)
	return ""
}

// radiusSource is the condition slot that holds a square's radius plus bias.
type radiusSource struct{ node, slot, bias int }

var noRadius = radiusSource{node: -1, slot: -1}

// radius is the distance a pair compares its condition with when the other
// side is a constant (TRIG-COND-003): the outline of the cells within that
// distance, false when the comparison leaves none; and the constant's slot.
func (p *placer) radius(c comparison) (int, radiusSource, bool) {
	i := scriptNodeIndex(p.sc.Conditions, c.other)
	if c.other == 0 || i < 0 || p.sc.Conditions[i].Opcode != ScriptConstant {
		return 0, noRadius, false
	}
	bias := 0
	v := int64(int32(p.sc.Conditions[i].Value[0]))
	code := c.code
	if !c.left { // constant OP distance reads distance OP' constant
		code = [6]uint32{0, 1, 3, 2, 5, 4}[min(code, 5)]
		if c.code > 5 {
			code = c.code
		}
	}
	switch code {
	case 3, 4: // < v, >= v: the cells at distance up to v-1
		v--
		bias = 1
	case 0, 1, 2, 5: // ==, !=, > v, <= v: the cells at distance up to v
	default:
		return 0, noRadius, false
	}
	if v < 0 || v > 255 {
		return 0, noRadius, false
	}
	return int(v), radiusSource{node: i, slot: 0, bias: bias}, true
}

func (p *placer) inside(c image.Point) bool {
	return c.In(image.Rect(0, 0, p.m.Width, p.m.Height))
}

func square(c image.Point, r int) image.Rectangle {
	return image.Rect(c.X-r, c.Y-r, c.X+r+1, c.Y+r+1)
}

// scriptCells are the node's cells: each X slot followed by a Y slot.
func scriptCells(n alm.ScriptNode) []image.Point {
	var out []image.Point
	for s := 0; s+1 < len(n.Type); s++ {
		if n.Type[s] == ScriptX && n.Type[s+1] == ScriptY {
			out = append(out, image.Pt(int(n.Value[s]), int(n.Value[s+1])))
			s++
		}
	}
	return out
}

// scriptCellSlots are the X slots of the node's cells, in scriptCells' order.
func scriptCellSlots(n alm.ScriptNode) []int {
	var out []int
	for s := 0; s+1 < len(n.Type); s++ {
		if n.Type[s] == ScriptX && n.Type[s+1] == ScriptY {
			out = append(out, s)
			s++
		}
	}
	return out
}

// scriptPlainSlot is the slot of the node's k-th int slot, -1 for none.
func scriptPlainSlot(n alm.ScriptNode, k int) int {
	for s, tag := range n.Type {
		if tag == ScriptInt || tag == ScriptConst {
			if k == 0 {
				return s
			}
			k--
		}
	}
	return -1
}

// scriptPlain is the node's plain parameter k past its cells: the k-th int
// slot.
func scriptPlain(n alm.ScriptNode, k int) (uint32, bool) {
	for s, tag := range n.Type {
		if tag == ScriptInt || tag == ScriptConst {
			if k == 0 {
				return n.Value[s], true
			}
			k--
		}
	}
	return 0, false
}

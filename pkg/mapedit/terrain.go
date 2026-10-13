package mapedit

import "strings"

// The terrain classes, numbered as the movement classifier numbers them
// (TERR-PASS-050). The editor never classifies a word itself: a caller hands
// every stroke the classifier it trusts, so the class a stroke reads and the
// class the mission build reads are one function.
const (
	ClassLand uint8 = iota + 1
	ClassGrass
	ClassFlowers
	ClassSand
	ClassCracked
	ClassStones
	ClassSavanna
	ClassMountain
	ClassWater
	ClassRoad
)

// ClassCount is one past the last class number.
const ClassCount = int(ClassRoad) + 1

// PaletteEntry is one terrain a brush can paint: its class, its base tile word
// (TERR-242: the word the original's terrain tool stores) and the name the
// window shows.
type PaletteEntry struct {
	Class uint8
	Word  uint16
	Name  string
}

// Palette lists the paintable terrains in the order the window shows them.
var Palette = [...]PaletteEntry{
	{ClassGrass, 0x11, "Grass"},
	{ClassLand, 0x14, "Land"},
	{ClassSand, 0x91, "Sand"},
	{ClassCracked, 0x51, "Cracked"},
	{ClassFlowers, 0x191, "Flowers"},
	{ClassStones, 0x111, "Stones"},
	{ClassSavanna, 0xd1, "Savanna"},
	{ClassMountain, 0x1d1, "Mountain"},
	{ClassWater, 0x211, "Water"},
	{ClassRoad, 0x311, "Road"},
}

// strip is one strip group: the two classes its tile images blend
// (TERR-PASS-050: group, primary, secondary).
type strip struct {
	group              uint16
	primary, secondary uint8
}

var strips = [...]strip{
	{0, ClassGrass, ClassLand},
	{1, ClassCracked, ClassLand},
	{2, ClassSand, ClassLand},
	{3, ClassSavanna, ClassLand},
	{4, ClassStones, ClassLand},
	{5, ClassCracked, ClassStones},
	{6, ClassFlowers, ClassSavanna},
	{7, ClassMountain, ClassStones},
	{12, ClassRoad, ClassLand},
}

// waterGroup is the strip group the editor writes for water. The four water
// groups 8 to 11 draw one image set in turn, and every one of them reads as
// water (TERR-PASS-050).
const waterGroup = 8

// hub is the class a cell blends toward when a neighbour's class shares no
// strip with its own: the neighbour is drawn as the hub. Land has none.
var hub = [ClassCount]uint8{
	ClassGrass:    ClassLand,
	ClassCracked:  ClassLand,
	ClassSand:     ClassLand,
	ClassSavanna:  ClassLand,
	ClassStones:   ClassLand,
	ClassRoad:     ClassLand,
	ClassFlowers:  ClassSavanna,
	ClassMountain: ClassStones,
}

// word is the in-strip part of a tile word: blend column and sub-cell.
type word struct{ b, s uint8 }

// Direction bits of a neighbourhood, in reading order.
const (
	dirNW = 1 << iota
	dirN
	dirNE
	dirW
	dirE
	dirSW
	dirS
	dirSE
)

var neighbourOffset = [8][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}

// layout lists the words of one strip family, by what the cell shows. A land
// strip carries the transition on its primary side (an edge or an inner corner
// of the other class) and a tip on a secondary cell where three primary cells
// meet; a water strip also has an outer corner. These words and their shapes
// are read from the installed maps' tile planes, not from a claim: DIV-3199.
type layout struct {
	pureP, pureS []word
	edge         [4][]word // cardinal order N, E, S, W: the other class lies there
	inner        [4][]word // diagonal order NW, NE, SW, SE: the other class lies there
	outer        [4][]word // two adjacent cardinals and their diagonal (water only)
	tip          [4][]word // secondary cell; the primary class lies at that corner
}

var landLayout = layout{
	pureP: []word{{1, 1}, {3, 0}, {3, 1}, {3, 2}, {3, 3}, {3, 4}, {3, 5}},
	pureS: []word{{1, 4}},
	edge:  [4][]word{{{1, 0}, {1, 5}}, {{0, 4}, {2, 1}}, {{1, 2}, {1, 3}}, {{0, 1}, {2, 4}}},
	inner: [4][]word{{{1, 12}, {2, 5}}, {{0, 5}, {2, 12}}, {{1, 7}, {2, 3}}, {{0, 3}, {2, 7}}},
	tip:   [4][]word{{{2, 2}, {2, 8}}, {{0, 2}, {1, 8}}, {{2, 0}, {2, 11}}, {{0, 0}, {1, 11}}},
}

var waterLayout = layout{
	pureP: []word{{1, 1}, {3, 0}, {3, 1}, {3, 2}, {3, 3}, {3, 4}},
	edge:  [4][]word{{{1, 0}, {1, 5}}, {{0, 4}, {2, 1}}, {{1, 2}, {1, 3}}, {{0, 1}, {2, 4}}},
	inner: [4][]word{{{2, 5}, {3, 7}}, {{0, 5}, {2, 7}}, {{0, 6}, {2, 3}}, {{0, 3}, {1, 6}}},
	outer: [4][]word{{{0, 0}, {3, 6}}, {{2, 0}, {2, 6}}, {{0, 2}, {0, 7}}, {{1, 7}, {2, 2}}},
}

// ShapeKind is what a cell shows of its neighbours.
type ShapeKind uint8

const (
	ShapePure ShapeKind = iota
	ShapeEdge
	ShapeInner
	ShapeOuter
	ShapeTip
)

// Shape names what a tile word shows, independent of its strip group and of
// which look-alike variant it is: Primary is true when the cell is the strip's
// primary class, Dir is the index into the kind's four-entry table.
type Shape struct {
	Primary bool
	Kind    ShapeKind
	Dir     uint8
}

// shapeKey locates a word in a layout: the strip family and its (b, s).
type shapeKey struct {
	water bool
	w     word
}

var shapeOfWord = map[shapeKey]Shape{}

func init() {
	for _, l := range []struct {
		water bool
		l     *layout
	}{{false, &landLayout}, {true, &waterLayout}} {
		add := func(ws []word, s Shape) {
			for _, w := range ws {
				k := shapeKey{l.water, w}
				if _, dup := shapeOfWord[k]; dup {
					panic("mapedit: a tile word sits in two shapes")
				}
				shapeOfWord[k] = s
			}
		}
		add(l.l.pureP, Shape{Primary: true, Kind: ShapePure})
		add(l.l.pureS, Shape{Kind: ShapePure})
		for d := 0; d < 4; d++ {
			add(l.l.edge[d], Shape{Primary: true, Kind: ShapeEdge, Dir: uint8(d)})
			add(l.l.inner[d], Shape{Primary: true, Kind: ShapeInner, Dir: uint8(d)})
			add(l.l.outer[d], Shape{Primary: true, Kind: ShapeOuter, Dir: uint8(d)})
			add(l.l.tip[d], Shape{Kind: ShapeTip, Dir: uint8(d)})
		}
	}
}

// WordShape reports the shape of an installed-style tile word and the strip
// group it sits in; ok is false for a word outside the listed shapes.
func WordShape(w uint16) (s Shape, group uint16, ok bool) {
	i := w & 0x3ff
	group = i >> 6 & 15
	if i&0x300 == 0x200 {
		group = waterGroup
	}
	k := shapeKey{water: i&0x300 == 0x200, w: word{uint8(i >> 4 & 3), uint8(i & 15)}}
	s, ok = shapeOfWord[k]
	return s, group, ok
}

func stripOf(c, x uint8) (strip, bool) {
	for _, s := range strips {
		if s.primary == c && s.secondary == x || s.primary == x && s.secondary == c {
			return s, true
		}
	}
	return strip{}, false
}

// ownStrip is the strip that gives class c its pure word: the first strip it
// is primary of, and for Land the first strip.
func ownStrip(c uint8) strip {
	for _, s := range strips {
		if s.primary == c {
			return s
		}
	}
	return strips[0]
}

func compose(group uint16, w word) uint16 { return group<<6 | uint16(w.b)<<4 | uint16(w.s) }

func pickVariant(ws []word, x, y int) word {
	h := uint32(x)*0x9e3779b1 ^ uint32(y)*0x85ebca77
	h ^= h >> 15
	return ws[h%uint32(len(ws))]
}

// Blend is what a cell's word depends on: the strip group it blends in, whether
// the cell is that strip's primary class, the mask of its neighbours of the
// other class (bits in reading order NW, N, NE, W, E, SW, S, SE) and whether the
// strip is water. Active is false for a cell with no neighbour of another class
// that it blends toward.
type Blend struct {
	Group   uint16
	Primary bool
	Mask    int
	Water   bool
	Active  bool
}

// BlendOf reduces a cell of class c and the classes of its eight neighbours
// (reading order; 0 or a class beyond ClassRoad is ignored, as is water for a
// land cell) to a Blend.
//
// The cell blends toward at most one other class: the one most neighbours
// name, the lower class number winning a tie. A neighbour whose class shares no
// strip with c is read as c's hub. A water cell blends toward every neighbour
// that is not water.
func BlendOf(c uint8, around [8]uint8) Blend {
	if c == 0 || int(c) >= ClassCount {
		return Blend{}
	}
	if c == ClassWater {
		var mask int
		for d, n := range around {
			if n != 0 && int(n) < ClassCount && n != ClassWater {
				mask |= 1 << d
			}
		}
		return Blend{Group: waterGroup, Primary: true, Mask: mask, Water: true, Active: mask != 0}
	}
	var count [ClassCount]int
	var masks [ClassCount]int
	for d, n := range around {
		if n == 0 || int(n) >= ClassCount || n == ClassWater || n == c {
			continue
		}
		if _, ok := stripOf(c, n); !ok {
			if n = hub[c]; n == 0 || n == c {
				continue
			}
		}
		count[n]++
		masks[n] |= 1 << d
	}
	best := uint8(0)
	for n := 1; n < ClassCount; n++ {
		if count[n] > count[best] {
			best = uint8(n)
		}
	}
	if best == 0 {
		return Blend{}
	}
	s, _ := stripOf(c, best)
	return Blend{Group: s.group, Primary: s.primary == c, Mask: masks[best], Active: true}
}

// Derive returns the tile word of the cell at (x, y) whose class is c and whose
// eight neighbours' classes are around. The result keeps the class of c: a
// primary cell takes an edge or corner word, a secondary cell takes a tip word
// or its pure word. The shape comes from the transition table (DIV-3199) and the
// cell's position picks among the shape's look-alike words.
func Derive(c uint8, around [8]uint8, x, y int) uint16 {
	if c == 0 || int(c) >= ClassCount {
		return 0
	}
	b := BlendOf(c, around)
	switch {
	case !b.Active:
		return pureWord(c, x, y)
	case b.Water:
		return deriveIn(&waterLayout, b.Group, shapeTable.water, b.Mask, x, y, c)
	case b.Primary:
		return deriveIn(&landLayout, b.Group, shapeTable.primary, b.Mask, x, y, c)
	}
	return deriveIn(&landLayout, b.Group, shapeTable.secondary, b.Mask, x, y, c)
}

func pureWord(c uint8, x, y int) uint16 {
	if c == ClassWater {
		return compose(waterGroup, pickVariant(waterLayout.pureP, x, y))
	}
	if c == ClassLand {
		return compose(strips[0].group, landLayout.pureS[0])
	}
	return compose(ownStrip(c).group, pickVariant(landLayout.pureP, x, y))
}

// Code numbers a shape for the transition table: 0 pure, 1 to 4 an edge, 5 to 8
// an inner corner, 9 to 12 an outer corner, 13 to 16 a tip, each in direction
// table order.
func (s Shape) Code() int {
	switch s.Kind {
	case ShapeEdge:
		return 1 + int(s.Dir)
	case ShapeInner:
		return 5 + int(s.Dir)
	case ShapeOuter:
		return 9 + int(s.Dir)
	case ShapeTip:
		return 13 + int(s.Dir)
	}
	return 0
}

// codeWords is the words a table code shows in one layout, nil for a pure cell
// or a shape the layout has no word for.
func (l *layout) codeWords(code int) []word {
	switch {
	case code >= 13:
		return l.tip[code-13]
	case code >= 9:
		return l.outer[code-9]
	case code >= 5:
		return l.inner[code-5]
	case code >= 1:
		return l.edge[code-1]
	}
	return nil
}

// deriveIn picks the word of one strip family for a cell on its primary or
// secondary side, from the mask of neighbours of the other class: the table
// names the shape and the cell's position picks among the shape's look-alike
// words.
func deriveIn(l *layout, group uint16, table string, mask, x, y int, c uint8) uint16 {
	code := strings.IndexByte(codeDigits, table[mask])
	if ws := l.codeWords(code); ws != nil {
		return compose(group, pickVariant(ws, x, y))
	}
	return pureWord(c, x, y)
}

const codeDigits = "0123456789abcdefg"

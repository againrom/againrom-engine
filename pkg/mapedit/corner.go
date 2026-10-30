package mapedit

import "slices"

// The terrain stroke's rule. A tile word shows the terrain at the four corners
// of its cell: a strip word splits them between the strip's two classes, a
// pure word gives all four one class (TERR-252: a family base plus a state of
// four corner occupancies). Two cells that share a corner show one class
// there, or the plane shows a hard edge. A stroke gives every corner of a
// painted cell its class and rewrites each neighbour from its corners, as the
// fixer rewrites the 3x3 (TERR-243); where no strip joins two classes the
// neighbour takes the inserted class and the rewrite spreads out from it until
// every cell it touches shows its corners, as the fixer recurses with no depth
// limit.

// Corner indices: north-west, north-east, south-west, south-east. Bit i of a
// state is set when corner i holds the strip's primary class.
const (
	cornerNW = iota
	cornerNE
	cornerSW
	cornerSE
)

// quad is the class at each corner of one cell.
type quad [4]uint8

// family is a set of words that blend two classes: a strip, or water whose
// other class is Land.
type family struct {
	group              uint16
	primary, secondary uint8
	water              bool
}

// stateWords lists the in-strip words of each state, for land strips and for
// water; a state no word shows (the two diagonal splits) is empty.
var stateWords [2][16][]word

// stateOfWord is the state each listed word shows.
var stateOfWord = map[shapeKey]uint8{}

func init() {
	edge := [4]uint8{1<<cornerSW | 1<<cornerSE, 1<<cornerNW | 1<<cornerSW, 1<<cornerNW | 1<<cornerNE, 1<<cornerNE | 1<<cornerSE}
	for k, l := range []*layout{&landLayout, &waterLayout} {
		add := func(state uint8, ws []word) {
			stateWords[k][state] = append(stateWords[k][state], ws...)
			for _, w := range ws {
				stateOfWord[shapeKey{k == 1, w}] = state
			}
		}
		add(15, l.pureP)
		add(0, l.pureS)
		for d := 0; d < 4; d++ {
			add(edge[d], l.edge[d])
			add(15&^(1<<d), l.inner[d])
			add(1<<(3-d), l.outer[d])
			add(1<<d, l.tip[d])
		}
	}
}

// familyOf is the family that blends classes a and b.
func familyOf(a, b uint8) (family, bool) {
	if a == ClassWater && b == ClassLand || a == ClassLand && b == ClassWater {
		return family{group: waterGroup, primary: ClassWater, secondary: ClassLand, water: true}, true
	}
	s, ok := stripOf(a, b)
	return family{group: s.group, primary: s.primary, secondary: s.secondary}, ok && a != b
}

// groupFamily is the family of the words of strip group g.
func groupFamily(g uint16) (family, bool) {
	if g == waterGroup {
		return familyOf(ClassWater, ClassLand)
	}
	for _, s := range strips {
		if s.group == g {
			return family{group: g, primary: s.primary, secondary: s.secondary}, true
		}
	}
	return family{}, false
}

// wordQuad reads the corners of word w: a listed strip or water word by its
// state, any other word as its class at all four corners. ok is false for a
// word of no terrain class.
func wordQuad(w uint16, class func(uint16) uint8) (quad, bool) {
	i := w & 0x3ff
	water := i&0x300 == 0x200
	g := i >> 6 & 15
	if water {
		g = waterGroup
	}
	if st, ok := stateOfWord[shapeKey{water, word{uint8(i >> 4 & 3), uint8(i & 15)}}]; ok {
		if f, ok := groupFamily(g); ok {
			var q quad
			for k := range q {
				q[k] = f.secondary
				if st>>k&1 == 1 {
					q[k] = f.primary
				}
			}
			return q, true
		}
	}
	c := class(w)
	if c == 0 || int(c) >= ClassCount {
		return quad{}, false
	}
	return quad{c, c, c, c}, true
}

// quadWords is the family and state of the words that show q; ok is false
// when no word does: three classes, two that share no family, or a diagonal
// split. Flowers and Grass meet without a seam, so a corner of one may be
// shown as the other, as the installed maps show it; prefer names the class
// tried first.
func quadWords(q quad, prefer uint8) (family, []word, bool) {
	if f, ws, ok := stateOf(q); ok {
		return f, ws, true
	}
	from, to := uint8(ClassFlowers), uint8(ClassGrass)
	if prefer == ClassFlowers {
		from, to = to, from
	}
	for range 2 {
		s := q
		for k := range s {
			if s[k] == from {
				s[k] = to
			}
		}
		if f, ws, ok := stateOf(s); ok {
			return f, ws, true
		}
		from, to = to, from
	}
	return family{}, nil, false
}

// stateOf is the family and state of the words that show exactly q.
func stateOf(q quad) (f family, ws []word, ok bool) {
	a, b := q[0], q[0]
	for _, c := range q {
		if c != a {
			b = c
		}
	}
	for _, c := range q {
		if c != a && c != b {
			return family{}, nil, false
		}
	}
	if a == b {
		return family{primary: a}, nil, true
	}
	if f, ok = familyOf(a, b); !ok {
		return family{}, nil, false
	}
	var st uint8
	for k, c := range q {
		if c == f.primary {
			st |= 1 << k
		}
	}
	k := 0
	if f.water {
		k = 1
	}
	ws = stateWords[k][st]
	return f, ws, len(ws) > 0
}

// seen is q with Flowers read as Grass: two quads a word shows alike.
func seen(q quad) quad {
	for k := range q {
		if q[k] == ClassFlowers {
			q[k] = ClassGrass
		}
	}
	return q
}

// joined reports whether corners of classes a and b may meet in one cell.
func joined(a, b uint8) bool {
	_, ok := familyOf(a, b)
	return ok || a == ClassFlowers && b == ClassGrass || a == ClassGrass && b == ClassFlowers
}

// quadWord is the word of q at (x, y), the cell's position picking among
// look-alike words.
func quadWord(q quad, prefer uint8, x, y int) uint16 {
	f, ws, _ := quadWords(q, prefer)
	if ws == nil {
		return pureWord(f.primary, x, y)
	}
	return compose(f.group, pickVariant(ws, x, y))
}

// baseClass is the class the fixer inserts between class c and a neighbour
// whose class no family joins to c (TERR-243 base): Stones for Land and
// Mountain, Grass for Flowers, Land for every other class.
func baseClass(c uint8) uint8 {
	switch c {
	case ClassLand, ClassMountain:
		return ClassStones
	case ClassFlowers:
		return ClassGrass
	}
	return ClassLand
}

// hops is the number of family steps between two classes.
var hops [ClassCount][ClassCount]int

func init() {
	for a := 1; a < ClassCount; a++ {
		for b := range hops[a] {
			hops[a][b] = ClassCount
		}
		hops[a][a] = 0
		for frontier := []int{a}; len(frontier) > 0; {
			var next []int
			for _, c := range frontier {
				for n := 1; n < ClassCount; n++ {
					if joined(uint8(c), uint8(n)) && hops[a][n] == ClassCount {
						hops[a][n] = hops[a][c] + 1
						next = append(next, n)
					}
				}
			}
			frontier = next
		}
	}
}

// rank orders the classes a corner may take in a stroke of class c: by family
// steps from c, then by class number. A corner moves only to a class of lower
// rank than the cell that moves it shows there, so a stroke ends.
func rank(c, x uint8) int { return hops[c][x]*ClassCount + int(x) }

// towards is the class one family step from x on a shortest way to c: the
// fixer's base class of c when it is one, else the lowest-numbered class.
func towards(c, x uint8) uint8 {
	if b := baseClass(c); hops[c][b] == 1 && hops[c][x] == 2 {
		if joined(b, x) {
			return b
		}
	}
	for n := 1; n < ClassCount; n++ {
		if joined(x, uint8(n)) && hops[c][n] == hops[c][x]-1 {
			return uint8(n)
		}
	}
	return c
}

// TileFix applies the terrain stroke's rule to a tile plane it reads and does
// not write. One TileFix serves any number of strokes on the same plane.
type TileFix struct {
	w, h  int
	tiles []uint16
	class func(uint16) uint8

	quads   map[int]quad // corners of each cell the stroke has read
	known   map[int]bool // the cell's word has a terrain class
	painted map[int]bool
	// Fill chooses the word of a painted cell that shows only the painted
	// class; the zero value keeps the position pattern.
	Fill  Fill
	fixed map[int]bool // the corners of the painted cells
	moved map[int]bool // the corners the stroke has written
	// Seams counts cells left with corners no word shows; each takes the
	// word nearest its corners. A cell no word shows always has a corner
	// that may move, so the rule leaves none.
	Seams int
}

// NewTileFix reads a w x h tile plane through class, the classifier a stroke
// trusts.
func NewTileFix(tiles []uint16, w, h int, class func(uint16) uint8) *TileFix {
	return &TileFix{w: w, h: h, tiles: tiles, class: class}
}

// TileChange is one cell a stroke rewrites and its new word.
type TileChange struct {
	Cell int
	Word uint16
}

// Paint is the words a stroke of class c over the cells (indices into the
// plane) gives, in cell order. Every corner of a painted cell takes c in each
// cell that shares it. Then, while a touched cell's corners are ones no word
// shows, one of its corners moves to a class of lower rank, in every cell
// that shares it (move), until every touched cell shows its corners; the
// classes between c and a class no strip joins to it are so inserted ring by
// ring. A cell left with corners no word shows takes the word nearest them
// (nearestQuad). A cell whose corners still show its word keeps the word; a
// painted cell loses bit 13, any other keeps it.
func (f *TileFix) Paint(cells []int, c uint8) []TileChange {
	f.quads, f.known, f.painted, f.Seams = map[int]quad{}, map[int]bool{}, map[int]bool{}, 0
	f.fixed, f.moved = map[int]bool{}, map[int]bool{}
	cells = slices.Sorted(slices.Values(cells))
	var queue []int
	for _, i := range cells {
		f.painted[i] = true
		f.read(i)
		f.known[i] = true
		for k := range 4 {
			f.fixed[f.corner(i, k)] = true
			queue = f.set(i, k, c, queue)
		}
	}
	for head := 0; head < len(queue); head++ {
		i := queue[head]
		q := f.quads[i]
		if _, _, ok := quadWords(q, 0); ok {
			continue
		}
		k, x := f.move(i, q, c)
		if k < 0 {
			continue
		}
		queue = append(f.set(i, k, x, queue), i)
	}
	order := make([]int, 0, len(f.quads))
	for i := range f.quads {
		order = append(order, i)
	}
	slices.Sort(order)
	var out []TileChange
	for _, i := range order {
		q, old := f.quads[i], f.tiles[i]
		prefer := f.class(old)
		if f.painted[i] {
			prefer = q[0]
		}
		if !f.known[i] {
			continue
		}
		if _, _, ok := quadWords(q, prefer); !ok {
			f.Seams++
			q = nearestQuad(q, old, f.class)
		}
		next := old
		if w, ok := f.centreWord(i, q, prefer); ok {
			next = w
		} else if cur, ok := wordQuad(old, f.class); !ok || cur != q && (f.painted[i] || seen(cur) != seen(q)) {
			next = quadWord(q, prefer, i%f.w, i/f.w)
		}
		next &^= blockedBit
		if !f.painted[i] {
			next |= old & blockedBit
		}
		if next != old {
			out = append(out, TileChange{Cell: i, Word: next})
		}
	}
	return out
}

// move is the corner of cell i, whose corners q no word shows, that moves in a
// stroke of class c, and its new class; -1 when no corner of i may move. A
// corner of a painted cell never moves. Where two classes no strip joins meet,
// the corner whose class lies most family steps from c moves one step towards
// c (deepest, towards). A diagonal split of two classes takes the class of
// lower rank at one more corner. Three classes that strips join pairwise
// (Cracked, Stones, Land) are a junction no word shows, and moving a corner
// towards c would carry it along the seam of the other two; so a class gives
// way to one of lower rank, preferring one the stroke has not written in this
// cell beside a third it has, and the junction moves along the stroke's edge.
func (f *TileFix) move(i int, q quad, c uint8) (int, uint8) {
	var movable [4]bool
	for k := range movable {
		movable[k] = !f.fixed[f.corner(i, k)]
	}
	var classes []uint8
	for _, x := range q {
		if !slices.Contains(classes, x) {
			classes = append(classes, x)
		}
	}
	pairwise := true
	for _, a := range classes {
		for _, b := range classes {
			pairwise = pairwise && (a == b || joined(a, b))
		}
	}
	if pairwise && len(classes) <= 3 {
		k, x, best := -1, uint8(0), [5]int{}
		for _, a := range classes {
			for _, b := range classes {
				if rank(c, b) >= rank(c, a) {
					continue
				}
				aWritten, thirdOld, at := 0, 1, -1
				for n := range q {
					w := f.moved[f.corner(i, n)]
					if q[n] == a && w {
						aWritten = 1
					}
					if q[n] != a && q[n] != b && w {
						thirdOld = 0
					}
					if q[n] == a && movable[n] && at < 0 {
						at = n
					}
				}
				if len(classes) == 2 {
					thirdOld = 0
				}
				score := [5]int{thirdOld, aWritten, b2i(b == c), -rank(c, a), -rank(c, b)}
				if at >= 0 && (k < 0 || slices.Compare(score[:], best[:]) < 0) {
					k, x, best = at, b, score
				}
			}
		}
		if k >= 0 {
			return k, x
		}
	}
	k := deepest(q, c, movable)
	if k < 0 {
		return -1, 0
	}
	return k, towards(c, q[k])
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// deepest is the corner of q to move towards c: of the movable corners whose
// class lies most steps from c, one whose class the fewest corners hold, the
// lowest-numbered breaking a tie; -1 when every movable corner holds c.
func deepest(q quad, c uint8, movable [4]bool) int {
	best := -1
	count := func(x uint8) (n int) {
		for _, y := range q {
			if y == x {
				n++
			}
		}
		return n
	}
	for k, x := range q {
		if x == c || int(x) >= ClassCount || !movable[k] {
			continue
		}
		if best < 0 || hops[c][x] > hops[c][q[best]] || hops[c][x] == hops[c][q[best]] && count(x) < count(q[best]) {
			best = k
		}
	}
	return best
}

// read is the corners of cell i as the stroke holds them.
func (f *TileFix) read(i int) quad {
	q, ok := f.quads[i]
	if !ok {
		var known bool
		q, known = wordQuad(f.tiles[i], f.class)
		f.quads[i], f.known[i] = q, known
	}
	return q
}

// nearestQuad is the quad of a word that differs from q at the fewest
// corners, each corner taking a class q holds; the corners of old, the cell's
// word, win a tie, then the lowest classes in corner order.
func nearestQuad(q quad, old uint16, class func(uint16) uint8) quad {
	best, diff := q, 5
	try := func(s quad) {
		if _, _, ok := quadWords(s, 0); !ok {
			return
		}
		d := 0
		for k := range s {
			if s[k] != q[k] {
				d++
			}
		}
		if d < diff {
			best, diff = s, d
		}
	}
	if cur, ok := wordQuad(old, class); ok {
		try(cur)
	}
	for n := range 256 {
		var s quad
		for k := range s {
			s[k] = q[n>>(2*k)&3]
		}
		try(s)
	}
	return best
}

// corner is the index of corner k of cell i on the (w+1) x (h+1) corner grid.
func (f *TileFix) corner(i, k int) int {
	return (i/f.w+k>>1)*(f.w+1) + i%f.w + k&1
}

// set gives corner k of cell i the class x in every cell of a terrain class
// that shares it, and queues each cell it changed. A corner of a painted cell
// keeps the painted class.
func (f *TileFix) set(i, k int, x uint8, queue []int) []int {
	p := f.corner(i, k)
	if f.fixed[p] && f.moved[p] {
		return queue
	}
	f.moved[p] = true
	px, py := i%f.w+k&1, i/f.w+k>>1
	for _, s := range [4][3]int{{-1, -1, cornerSE}, {0, -1, cornerSW}, {-1, 0, cornerNE}, {0, 0, cornerNW}} {
		cx, cy := px+s[0], py+s[1]
		if cx < 0 || cy < 0 || cx >= f.w || cy >= f.h {
			continue
		}
		j := cy*f.w + cx
		if q := f.read(j); f.known[j] && q[s[2]] != x {
			q[s[2]] = x
			f.quads[j] = q
			queue = append(queue, j)
		}
	}
	return queue
}

// centreWord is the fill's word for painted cell i showing only one class, and
// false for any other cell or for the pattern.
func (f *TileFix) centreWord(i int, q quad, prefer uint8) (uint16, bool) {
	if !f.painted[i] || f.Fill.Mode == FillPattern {
		return 0, false
	}
	fam, ws, ok := quadWords(q, prefer)
	if !ok || ws != nil {
		return 0, false
	}
	return f.fillWord(fam.primary)
}

package sim

import "testing"

// claimWalkStamp is the second game's fog stamp written from the claim text
// alone, apart from the march: the seed (word >> 1) + 64 at the window's
// centre (R2-ENGINE-644); rings 1..19, ring cells outside 7..W-8 and 7..H-8
// skipped, a cell's budget its predecessor's minus trunc(128*sqrt(a*a+b*b)/
// max(a,b)) and its height over the centre's, an empty ring ending the walk
// (R2-ENGINE-645); the 2x2 block per positive budget cell around the body's
// cell plus trunc(size/2) (R2-ENGINE-646).
func claimWalkStamp(heights []byte, b Bounds, x0, y0, word, size int) []byte {
	width, height := int(b.Width), int(b.Height)
	var budget [41][41]int
	root := func(n int) int {
		r := 0
		for (r+1)*(r+1) <= n {
			r++
		}
		return r
	}
	abs := func(v int) int { return max(v, -v) }
	sign := func(v int) int {
		if v < 0 {
			return -1
		}
		return 1
	}
	rise := func(x, y int) int {
		if heights == nil {
			return 0
		}
		return int(int8(heights[y*width+x]))
	}
	h0 := rise(x0, y0)
	budget[20][20] = word>>1 + 64
	for r := 1; r <= 19; r++ {
		lit := false
		for k := -r; k <= r; k++ {
			for _, d := range [4][2]int{{k, -r}, {k, r}, {-r, k}, {r, -k}} {
				x, y := x0+d[0], y0+d[1]
				if x < 7 || x >= width-7 || y < 7 || y >= height-7 {
					continue
				}
				a, c := abs(d[0]), abs(d[1])
				sx, sy := -1, -1
				switch {
				case c < a>>1:
					sx, sy = -1, 0
				case c > 2*a:
					sx, sy = 0, -1
				}
				if c == 0 && a == 1 {
					sx, sy = -1, 0
				}
				px, py := d[0]+sx*sign(d[0]), d[1]+sy*sign(d[1])
				cost := root(128*128*(a*a+c*c)) / max(a, c)
				v := budget[px+20][py+20] - (cost + rise(x, y) - h0)
				budget[d[0]+20][d[1]+20] = v
				lit = lit || v > 0
			}
		}
		if !lit {
			break
		}
	}
	out := make([]byte, width*height)
	cx, cy := x0+size/2, y0+size/2
	for i := 0; i < 41; i++ {
		for j := 0; j < 41; j++ {
			if budget[i][j] <= 0 {
				continue
			}
			x, y := cx-20+i, cy-20+j
			for _, p := range [4][2]int{{x, y}, {x + 1, y}, {x, y + 1}, {x + 1, y + 1}} {
				if p[0] >= 0 && p[0] < width && p[1] >= 0 && p[1] < height {
					out[p[1]*width+p[0]] = 1
				}
			}
		}
	}
	return out
}

// fogWalkHero is the second game's unmodified first hero: Mind 25, Agility
// 36, sight byte 6, the sight word 1648 (R2-ENGINE-290).
func fogWalkHero(x, y int32, size uint8) Entity {
	return Entity{ID: 1, X: x, Y: y, Owner: 1, ScanRange: 6, Humanoid: true, Mind: 25, Reaction: 36, HP: 10, MaxHP: 10, TokenSize: size}
}

func fogWalkSight(t *testing.T, heights []byte, owner uint32, es ...Entity) []byte {
	t.Helper()
	w, err := NewTerrainWorld(1, sightBounds, ModeCanonical, Terrain{Height: heights}, es, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.rom2 = &rom2ScriptState{}
	return w.Sight(owner)
}

func sameStamp(a, b []byte) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// The fog stamp is the claims' walk on flat ground, and a height rise, the
// map edge, a body of size 2 and an owner without the reveal bit each change
// it as the claims state.
func TestTheFogStampIsTheSecondGameWalk(t *testing.T) {
	t.Parallel()
	const word = 1648
	if w := fogWalkHero(0, 0, 1).SightWord(true); w != word {
		t.Fatalf("sight word %d, want %d", w, word)
	}
	flat := fogWalkSight(t, nil, 1, fogWalkHero(22, 22, 1))
	stampDiff(t, "flat ground", flat, claimWalkStamp(nil, sightBounds, 22, 22, word, 1))

	// A ridge east of the hero costs its rise; a hill under him pays every
	// cell its fall.
	ridge := sightPlane(sightBounds, 0)
	for y := 16; y <= 28; y++ {
		ridge[y*int(sightBounds.Width)+25] = 127
	}
	got := fogWalkSight(t, ridge, 1, fogWalkHero(22, 22, 1))
	stampDiff(t, "ridge", got, claimWalkStamp(ridge, sightBounds, 22, 22, word, 1))
	if sameStamp(got, flat) || got[22*int(sightBounds.Width)+29] != 0 || flat[22*int(sightBounds.Width)+29] == 0 {
		t.Errorf("the ridge at column 25 leaves the stamp past it as on flat ground")
	}
	hill := sightPlane(sightBounds, 0)
	hill[22*int(sightBounds.Width)+22] = 60
	got = fogWalkSight(t, hill, 1, fogWalkHero(22, 22, 1))
	stampDiff(t, "hill", got, claimWalkStamp(hill, sightBounds, 22, 22, word, 1))
	if litCount(got) <= litCount(flat) {
		t.Errorf("the hero on a hill lights %d cells, on flat ground %d", litCount(got), litCount(flat))
	}

	// Near the west edge no ring cell left of column 7 is walked: the block
	// of the cell at column 7 is the stamp's west end.
	edge := fogWalkSight(t, nil, 1, fogWalkHero(9, 22, 1))
	stampDiff(t, "west edge", edge, claimWalkStamp(nil, sightBounds, 9, 22, word, 1))
	west := int(sightBounds.Width)
	for i, v := range edge {
		if v != 0 && i%int(sightBounds.Width) < west {
			west = i % int(sightBounds.Width)
		}
	}
	if west != 7 {
		t.Errorf("the stamp's west end is column %d, want 7", west)
	}

	// A body of size 2 walks from its cell and writes one cell to +x and +y.
	uneven := sightPlane(sightBounds, 0)
	for i := range uneven {
		uneven[i] = byte(i * 37 % 101)
	}
	one := fogWalkSight(t, uneven, 1, fogWalkHero(21, 23, 1))
	two := fogWalkSight(t, uneven, 1, fogWalkHero(21, 23, 2))
	stampDiff(t, "size 2", two, claimWalkStamp(uneven, sightBounds, 21, 23, word, 2))
	cols := int(sightBounds.Width)
	for i, v := range one {
		x, y := i%cols, i/cols
		if v != 0 && x+1 < cols && y+1 < int(sightBounds.Height) && two[(y+1)*cols+x+1] == 0 {
			t.Fatalf("size 2 does not hold the size-1 cell %d/%d moved by one", x, y)
		}
	}
	if sameStamp(one, two) {
		t.Error("size 2 stamps where size 1 does")
	}

	// A body whose owner lacks the viewer's reveal bit stamps nothing for
	// that viewer; under the viewer's own owner it adds its cells.
	other := fogWalkHero(32, 32, 1)
	other.ID, other.Owner = 2, 2
	hero := fogWalkHero(14, 14, 1)
	alone := claimWalkStamp(nil, sightBounds, 14, 14, word, 1)
	stampDiff(t, "other owner", fogWalkSight(t, nil, 1, hero, other), alone)
	stampDiff(t, "other owner's own view", fogWalkSight(t, nil, 2, hero, other), claimWalkStamp(nil, sightBounds, 32, 32, word, 1))
	other.Owner = 1
	if both := fogWalkSight(t, nil, 1, hero, other); litCount(both) <= litCount(alone) {
		t.Errorf("a second own body adds no cells: %d against %d", litCount(both), litCount(alone))
	}
}

package game

import (
	"fmt"
	"sort"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
)

// installedMapKeys lists the catalogue keys of every first-game map.
func installedMapKeys(x *MapInspector) []string {
	var keys []string
	for _, m := range x.Maps {
		if m.FromArchive {
			keys = append(keys, "scenario/"+m.Source)
		} else {
			keys = append(keys, "loose/"+m.Source)
		}
	}
	return keys
}

// stripClasses is the primary and secondary class of each strip group, water
// (group 8) blending into Land.
var stripClasses = map[uint16][2]uint8{
	0: {mapedit.ClassGrass, mapedit.ClassLand}, 1: {mapedit.ClassCracked, mapedit.ClassLand},
	2: {mapedit.ClassSand, mapedit.ClassLand}, 3: {mapedit.ClassSavanna, mapedit.ClassLand},
	4: {mapedit.ClassStones, mapedit.ClassLand}, 5: {mapedit.ClassCracked, mapedit.ClassStones},
	6: {mapedit.ClassFlowers, mapedit.ClassSavanna}, 7: {mapedit.ClassMountain, mapedit.ClassStones},
	8: {mapedit.ClassWater, mapedit.ClassLand}, 12: {mapedit.ClassRoad, mapedit.ClassLand},
}

// installedCorners is the class word w shows at its corners north-west,
// north-east, south-west and south-east, read from its shape: a strip word
// splits them between its strip's classes, any other word shows its class at
// all four. The family names the word's strip, or "pure" and its class.
func installedCorners(w uint16) (q [4]uint8, family string) {
	c := mapload.TileClass(w)
	s, g, ok := mapedit.WordShape(w)
	if !ok || s.Kind == mapedit.ShapePure {
		return [4]uint8{c, c, c, c}, fmt.Sprintf("pure %d", c)
	}
	p := stripClasses[g]
	var primary [4]bool
	switch s.Kind {
	case mapedit.ShapeEdge:
		far := [4][2]int{{2, 3}, {0, 2}, {0, 1}, {1, 3}}[s.Dir]
		primary[far[0]], primary[far[1]] = true, true
	case mapedit.ShapeInner:
		primary = [4]bool{true, true, true, true}
		primary[s.Dir] = false
	case mapedit.ShapeOuter:
		primary[3-s.Dir] = true
	case mapedit.ShapeTip:
		primary[s.Dir] = true
	}
	for k := range q {
		q[k] = p[1]
		if primary[k] {
			q[k] = p[0]
		}
	}
	return q, fmt.Sprintf("strip %d/%d", p[0], p[1])
}

// terrainPrediction is the corners a rule gives cell (x, y) of an installed
// plane from its eight neighbours; ok is false when it gives none.
type terrainPrediction func(tiles []uint16, w, x, y int) (q [4]uint8, ok bool)

// cornerPrediction is the corner rule's reading: each corner takes the class
// at least two of the three other cells sharing it show there.
func cornerPrediction(tiles []uint16, w, x, y int) (q [4]uint8, ok bool) {
	for k := range q {
		px, py := x+k&1, y+k>>1
		var seen []uint8
		for _, s := range [4][3]int{{-1, -1, 3}, {0, -1, 2}, {-1, 0, 1}, {0, 0, 0}} {
			cx, cy := px+s[0], py+s[1]
			if cx == x && cy == y {
				continue
			}
			c, _ := installedCorners(tiles[cy*w+cx])
			seen = append(seen, c[s[2]])
		}
		switch {
		case seen[0] == seen[1] || seen[0] == seen[2]:
			q[k] = seen[0]
		case seen[1] == seen[2]:
			q[k] = seen[1]
		default:
			return q, false
		}
	}
	return q, true
}

// seenAlike reports whether two corner sets look alike: Flowers and Grass
// meet with no strip between them.
func seenAlike(a, b [4]uint8) bool {
	grass := func(c uint8) uint8 {
		if c == mapedit.ClassFlowers {
			return mapedit.ClassGrass
		}
		return c
	}
	for k := range a {
		if grass(a[k]) != grass(b[k]) {
			return false
		}
	}
	return true
}

// terrainShare counts, over the playable cells of every installed map with a
// neighbour of another class, those whose installed corners the rule gives,
// exactly and with Flowers read as Grass, in total and per family of the
// installed word.
func terrainShare(t *testing.T, x *MapInspector, predict terrainPrediction) (n [3]int, byFamily map[string][3]int) {
	t.Helper()
	byFamily = map[string][3]int{}
	for _, key := range installedMapKeys(x) {
		d, err := x.Open(key)
		if err != nil {
			t.Fatal(err)
		}
		m, err := alm.Open(d.SourceBytes)
		if err != nil {
			t.Fatal(err)
		}
		w := m.Width
		for y := 8; y < m.Height-8; y++ {
			for xx := 8; xx < w-8; xx++ {
				i := y*w + xx
				c := mapload.TileClass(m.Tiles[i])
				if c == 0 || int(c) >= mapedit.ClassCount {
					continue
				}
				differs := false
				for _, o := range [8][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}} {
					differs = differs || mapload.TileClass(m.Tiles[(y+o[1])*w+xx+o[0]]) != c
				}
				if !differs {
					continue
				}
				have, family := installedCorners(m.Tiles[i])
				f := byFamily[family]
				got, ok := predict(m.Tiles, w, xx, y)
				for k, hit := range [3]bool{true, ok && got == have, ok && seenAlike(got, have)} {
					if hit {
						n[k]++
						f[k]++
					}
				}
				byFamily[family] = f
			}
		}
	}
	return n, byFamily
}

func share(n [3]int) string {
	pc := func(k int) float64 { return 100 * float64(n[k]) / float64(max(1, n[0])) }
	return fmt.Sprintf("%d of %d (%.1f%%), Flowers as Grass %d (%.1f%%)", n[1], n[0], pc(1), n[2], pc(2))
}

func logTerrainShare(t *testing.T, rule string, n [3]int, byFamily map[string][3]int) {
	t.Helper()
	t.Logf("%s: cells with a neighbour of another class: %s", rule, share(n))
	var families []string
	for f := range byFamily {
		families = append(families, f)
	}
	sort.Strings(families)
	for _, f := range families {
		t.Logf("%s: %s: %s", rule, f, share(byFamily[f]))
	}
}

// The corner rule re-derives each installed cell from the corners its eight
// neighbours show. The share is the rule's agreement with the installed
// planes of the root, a measurement reported per family.
func TestReleaseTerrainRuleOverInstalledMaps(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	n, byFamily := terrainShare(t, x, cornerPrediction)
	logTerrainShare(t, "corner rule", n, byFamily)
	if n[2]*100 < n[0]*90 {
		t.Errorf("the corner rule gives the installed corners of %s cells", share(n))
	}
}

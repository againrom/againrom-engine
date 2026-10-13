package game

import (
	"testing"

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

// Every shape word the transition table can write classifies, through the
// mission build's classifier, as the class the table gives the cell.
func TestTerrainTableWordsClassifyAsTheTableSays(t *testing.T) {
	for c := uint8(1); c <= mapedit.ClassRoad; c++ {
		for x := uint8(1); x <= mapedit.ClassRoad; x++ {
			for mask := 0; mask < 256; mask++ {
				var around [8]uint8
				for d := range around {
					around[d] = c
					if mask>>d&1 == 1 {
						around[d] = x
					}
				}
				for _, at := range [][2]int{{0, 0}, {3, 5}, {17, 2}, {9, 31}} {
					if w := mapedit.Derive(c, around, at[0], at[1]); mapload.TileClass(w) != c {
						t.Fatalf("class %d beside %d at %08b: word %#x classifies as %d", c, x, mask, w, mapload.TileClass(w))
					}
				}
			}
		}
	}
	for _, p := range mapedit.Palette {
		if got := mapload.TileClass(p.Word); got != p.Class {
			t.Errorf("palette %s: word %#x classifies as %d, not %d", p.Name, p.Word, got, p.Class)
		}
	}
}

// The transition rule re-derives each installed cell's word from the classes
// of its 3x3 neighbourhood. The counts are the rule's agreement with the
// installed planes, reported per root; they are a measurement, not a target.
func TestReleaseTerrainRuleOverInstalledMaps(t *testing.T) {
	x := installedInspector(t)
	var cells, uniform, unknown, uniformPure, mixed, mixedShape, mixedStrip int
	for _, key := range installedMapKeys(x) {
		d, err := x.Open(key)
		if err != nil {
			t.Fatal(err)
		}
		m, err := alm.Open(d.SourceBytes)
		if err != nil {
			t.Fatal(err)
		}
		cls := make([]uint8, len(m.Tiles))
		for i, w := range m.Tiles {
			cls[i] = mapload.TileClass(w)
		}
		for y := 1; y < m.Height-1; y++ {
			for xx := 1; xx < m.Width-1; xx++ {
				i := y*m.Width + xx
				c := cls[i]
				if c == 0 || int(c) >= mapedit.ClassCount {
					continue
				}
				var around [8]uint8
				differs := false
				for k, o := range [8][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}} {
					around[k] = cls[(y+o[1])*m.Width+xx+o[0]]
					differs = differs || around[k] != c
				}
				cells++
				word := mapedit.Derive(c, around, xx, y)
				if mapload.TileClass(word) != c {
					t.Fatalf("%s (%d,%d): class %d got word %#x of class %d", key, xx, y, c, word, mapload.TileClass(word))
				}
				got, gg, gok := mapedit.WordShape(word)
				have, hg, hok := mapedit.WordShape(m.Tiles[i])
				if !hok || !gok {
					unknown++
					continue
				}
				if !differs {
					uniform++
					if have.Kind == mapedit.ShapePure {
						uniformPure++
					}
					continue
				}
				mixed++
				if have == got {
					mixedShape++
					if have.Kind == mapedit.ShapePure || hg == gg {
						mixedStrip++
					}
				}
			}
		}
	}
	t.Logf("cells %d; word outside the table %d; uniform neighbourhood %d (pure %d); differing neighbour %d: same shape %d, same shape and strip %d",
		cells, unknown, uniform, uniformPure, mixed, mixedShape, mixedStrip)
	if mixedShape*100 < mixed*70 || uniformPure*100 < uniform*90 {
		t.Errorf("the rule agrees with too few installed cells: %d of %d with a differing neighbour, %d of %d uniform", mixedShape, mixed, uniformPure, uniform)
	}
}

package game

import (
	"image"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
)

// strokeBound is the distance, in cells on the square distance, from a terrain
// stroke's footprint within which the inserted rings of the two classes most
// strip steps apart (Mountain and Flowers: Stones, Land, Grass) and the cells
// their corners touch lie. A stroke rewrites past it only where it cuts a band
// one corner wide between two classes no strip joins, which takes the
// inserted class whole.
func strokeBound() int { return 4 }

// strokeReach measures one stroke from before to after: the words rewritten
// below the top eight rows (the save normalisation rewrites those whole), the
// unpainted cells whose class changed, the farthest rewritten word from the
// footprint, and the cells past the bound that changed a word or a class or
// opened in the game's passability plane.
type strokeReach struct {
	words, classes, reach, far, opened, farOpened int
}

func measureStroke(before, after *alm.Map, fp []mapedit.Cell) strokeReach {
	w := before.Width
	in := map[int]bool{}
	for _, c := range fp {
		in[c.Y*w+c.X] = true
	}
	dist := func(i int) int {
		d := 1 << 30
		for _, c := range fp {
			d = min(d, max(absInt(c.X-i%w), absInt(c.Y-i/w)))
		}
		return d
	}
	var r strokeReach
	for i := range before.Tiles {
		if before.Tiles[i] == after.Tiles[i] || i/w < 8 {
			continue
		}
		r.words++
		d := dist(i)
		r.reach = max(r.reach, d)
		if !in[i] && mapload.TileClass(before.Tiles[i]) != mapload.TileClass(after.Tiles[i]) {
			r.classes++
		}
		if d > strokeBound() {
			r.far++
		}
	}
	pb, pa := mapload.Passability(before), mapload.Passability(after)
	for i := range pb {
		if pb[i]&1 == 1 && pa[i]&1 == 0 {
			r.opened++
			if dist(i) > strokeBound() {
				r.farOpened++
			}
		}
	}
	return r
}

// catalogueMapBytes is the installed bytes of a catalogue key.
func catalogueMapBytes(t *testing.T, x *MapInspector, key string) []byte {
	t.Helper()
	var b []byte
	var err error
	if name, ok := strings.CutPrefix(key, "loose/"); ok {
		b, err = x.archives.Loose.ReadFile(name)
	} else {
		b, err = x.archives.Containers.ReadFile(key)
	}
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// A terrain stroke on an installed map leaves every cell a word for its
// corners (TileFix.Seams), adds no hard corner and opens no closed cell more
// than strokeBound cells from its footprint. The one-cell Cracked click at
// (40,40) on scenario/31.alm, through the window, sits on a Cracked, Stones
// and Land junction beside mountain ridges that a walk along the seam would
// open; it rewrites no word past the bound. The sweep paints every listed
// class at sizes 0, 1, 3 and 8, both shapes, at the centre, the west edge, the
// far corner, the top rows, a three-centre drag and up to three cells beside
// water, on four maps; every written word is drawn and a painted cell shows
// its class. The sweep reports the strokes that rewrite words past the bound.
func TestReleaseMapEditorStrokeStaysNearTheBrush(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := openBrushEditor(t, x, "scenario/31.alm")
	_ = e.Key("terrain")
	setBrush(t, e, mapedit.ClassCracked, 0, 0)
	at := image.Pt(40, 40)
	e.ShowCell(at)
	before := shownMap(t, e)
	pointerStroke(e, []image.Point{at})
	r := measureStroke(before, shownMap(t, e), []mapedit.Cell{{X: at.X, Y: at.Y}})
	t.Logf("scenario/31.alm Cracked size 0 at %v: %d words rewritten, %d unpainted class changes, reach %d, %d cells opened",
		at, r.words, r.classes, r.reach, r.opened)
	if r.far != 0 || r.farOpened != 0 {
		t.Errorf("scenario/31.alm click: %d words and %d opened cells past %d cells", r.far, r.farOpened, strokeBound())
	}

	keys := installedMapKeys(x)
	strokes, seamCells, seamStrokes, newHard, reach, past, pastOpened, classes := 0, 0, 0, 0, 0, 0, 0, 0
	for _, key := range []string{keys[0], keys[len(keys)/3], keys[2*len(keys)/3], keys[len(keys)-1]} {
		src := catalogueMapBytes(t, x, key)
		m0, err := alm.Open(src)
		if err != nil {
			t.Fatal(err)
		}
		w, h := m0.Width, m0.Height
		positions := [][]mapedit.Cell{{{X: w / 2, Y: h / 2}}, {{X: 0, Y: h / 2}}, {{X: w - 1, Y: h - 1}}, {{X: 3, Y: 0}},
			{{X: w / 3, Y: h / 3}, {X: w/3 + 1, Y: h/3 + 1}, {X: w/3 + 2, Y: h / 3}}}
		found := 0
		for y := 9; y < h-9 && found < 3; y += 3 {
			for cx := 9; cx < w-9 && found < 3; cx += 5 {
				if mapload.TileClass(m0.Tiles[y*w+cx]) == mapedit.ClassWater {
					continue
				}
				if mapload.TileClass(m0.Tiles[y*w+cx+1]) == mapedit.ClassWater || mapload.TileClass(m0.Tiles[(y+1)*w+cx]) == mapedit.ClassWater {
					positions = append(positions, []mapedit.Cell{{X: cx, Y: y}})
					found++
				}
			}
		}
		for _, p := range mapedit.Palette {
			for _, size := range []int{0, 1, 3, 8} {
				for _, shape := range []mapedit.BrushShape{mapedit.BrushCircle, mapedit.BrushSquare} {
					for _, at := range positions {
						ed, err := mapedit.New(src)
						if err != nil {
							t.Fatal(err)
						}
						b := mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(p.Class), Radius: size, Shape: shape}
						if _, err := ed.Paint(b, at, mapload.TileClass); err != nil {
							t.Fatalf("%s %s size %d: %v", key, p.Name, size, err)
						}
						m1, err := alm.Open(ed.Bytes())
						if err != nil {
							t.Fatal(err)
						}
						strokes++
						fp := b.Footprint(at, w, h)
						r := measureStroke(m0, m1, fp)
						reach = max(reach, r.reach)
						_, hc := hardEdges(m0, m1)
						classes += r.classes
						newHard += hc
						what := key + " " + p.Name
						if r.far != 0 {
							past++
						}
						pastOpened += r.farOpened
						var cells []int
						for _, c := range fp {
							cells = append(cells, c.Y*w+c.X)
						}
						fix := mapedit.NewTileFix(m0.Tiles, w, h, mapload.TileClass)
						fix.Paint(cells, p.Class)
						seamCells += fix.Seams
						if fix.Seams > 0 {
							seamStrokes++
						}
						if r.farOpened != 0 || fix.Seams != 0 || hc != 0 {
							t.Errorf("%s size %d shape %d at %v: %d closed cells opened past %d cells (reach %d), %d cells with no word, %d new hard corners",
								what, size, shape, at, r.farOpened, strokeBound(), r.reach, fix.Seams, hc)
						}
						for i := range m1.Tiles {
							if m1.Tiles[i] == m0.Tiles[i] {
								continue
							}
							ref := terrain.Resolve(m1.Tiles[i] &^ 0x2000)
							if x.tiles.Slot(ref.Slot).SubCell(ref.Sub) == nil {
								t.Fatalf("%s: cell %d takes word %#x, not in the tile set", what, i, m1.Tiles[i])
							}
						}
						for _, c := range fp {
							if c.Y >= 8 && mapload.TileClass(m1.Tiles[c.Y*w+c.X]) != p.Class {
								t.Fatalf("%s: painted cell %v shows word %#x", what, c, m1.Tiles[c.Y*w+c.X])
							}
						}
					}
				}
			}
		}
	}
	t.Logf("sweep: %d strokes, farthest rewritten word %d cells from the footprint (bound %d), %d strokes rewrite past the bound, %d closed cells opened past it; %d cells in %d strokes take the nearest word, %d new hard corners, %d unpainted class changes",
		strokes, reach, strokeBound(), past, pastOpened, seamCells, seamStrokes, newHard, classes)
}

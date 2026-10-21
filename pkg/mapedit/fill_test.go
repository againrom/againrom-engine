package mapedit_test

import (
	"bytes"
	"math/rand/v2"
	"slices"
	"testing"

	"againrom/pkg/mapedit"
)

// seeded is a random fill drawing from one seed.
func seeded(seed uint64) mapedit.Fill {
	r := rand.New(rand.NewPCG(seed, 7))
	return mapedit.Fill{Mode: mapedit.FillRandom, Draw: r.IntN}
}

// fillStroke paints a wide square of class c over a Land map with fill and
// returns the saved bytes and the tile plane.
func fillStroke(t *testing.T, c uint8, fill mapedit.Fill) ([]byte, []uint16) {
	t.Helper()
	ed, err := mapedit.New(brushMap(0x14, 40))
	if err != nil {
		t.Fatal(err)
	}
	b := mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(c), Radius: 8, Shape: mapedit.BrushSquare, Fill: fill}
	if _, err := ed.Paint(b, []mapedit.Cell{{X: 12, Y: 12}}, testClass); err != nil {
		t.Fatal(err)
	}
	tiles, _, _ := planes(t, ed.Bytes())
	return ed.Bytes(), tiles
}

// A fixed variant writes only that variant inside the area and the very
// words of the position pattern on every edge and corner cell; the pattern
// is the loss control and shows several variants.
func TestFixedFillWritesOneVariantAndKeepsEdges(t *testing.T) {
	for _, c := range []uint8{mapedit.ClassGrass, mapedit.ClassWater, mapedit.ClassSand} {
		vars := mapedit.Variants(c)
		_, base := fillStroke(t, c, mapedit.Fill{})
		seen := map[uint16]bool{}
		for _, w := range base {
			if slices.Contains(vars, w) {
				seen[w] = true
			}
		}
		if len(seen) < 2 {
			t.Fatalf("class %d: the pattern shows %d variants; the loss control needs several", c, len(seen))
		}
		for k := range vars {
			_, got := fillStroke(t, c, mapedit.Fill{Mode: mapedit.FillVariant, Variant: k})
			inside := 0
			for i := range base {
				if slices.Contains(vars, base[i]) {
					inside++
					if got[i] != vars[k] {
						t.Fatalf("class %d variant %d: interior cell %d is %#x, not %#x", c, k, i, got[i], vars[k])
					}
				} else if got[i] != base[i] {
					t.Fatalf("class %d variant %d: edge or outside cell %d is %#x, the pattern wrote %#x", c, k, i, got[i], base[i])
				}
			}
			if inside < 100 {
				t.Fatalf("class %d: only %d interior cells", c, inside)
			}
		}
	}
}

// A random fill draws from its seed: one seed gives the same interior, two
// seeds give different ones, and the edges equal the pattern's.
func TestRandomFillDiffersPerSeedAndKeepsEdges(t *testing.T) {
	vars := mapedit.Variants(mapedit.ClassGrass)
	baseBytes, base := fillStroke(t, mapedit.ClassGrass, mapedit.Fill{})
	aBytes, a := fillStroke(t, mapedit.ClassGrass, seeded(1))
	again, _ := fillStroke(t, mapedit.ClassGrass, seeded(1))
	_, b := fillStroke(t, mapedit.ClassGrass, seeded(2))
	if !bytes.Equal(aBytes, again) {
		t.Fatal("one seed gave two interiors")
	}
	if bytes.Equal(aBytes, baseBytes) {
		t.Fatal("the random fill wrote the pattern")
	}
	if slices.Equal(a, b) {
		t.Fatal("two seeds gave one interior")
	}
	for name, got := range map[string][]uint16{"seed 1": a, "seed 2": b} {
		used := map[uint16]bool{}
		for i := range base {
			if slices.Contains(vars, base[i]) {
				if !slices.Contains(vars, got[i]) {
					t.Fatalf("%s: interior cell %d is %#x, not a grass variant", name, i, got[i])
				}
				used[got[i]] = true
			} else if got[i] != base[i] {
				t.Fatalf("%s: edge cell %d is %#x, the pattern wrote %#x", name, i, got[i], base[i])
			}
		}
		if len(used) < len(vars)-1 {
			t.Fatalf("%s: only %d of %d variants used", name, len(used), len(vars))
		}
	}
	if _, err := mapedit.New(aBytes); err != nil {
		t.Fatalf("the map does not load back: %v", err)
	}
}

// Every variant of every class reads back as its own class.
func TestVariantsAreWordsOfTheirClass(t *testing.T) {
	for _, p := range mapedit.Palette {
		vs := mapedit.Variants(p.Class)
		if len(vs) == 0 || !slices.Contains(vs, p.Word) {
			t.Fatalf("%s: variants %v lack the palette word %#x", p.Name, vs, p.Word)
		}
		for _, w := range vs {
			if got := testClass(w); got != p.Class {
				t.Fatalf("%s: variant %#x reads as class %d", p.Name, w, got)
			}
		}
	}
}

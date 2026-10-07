package game_test

// The margin plane's trip from a decoded map to the viewer that draws it
// (AC-8, AC-9). Fixtures are synthetic map streams built in test code; no
// game install is read and no window is opened.

import (
	"encoding/binary"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
)

// waterWord is a tile word whose index falls inside the derivation's water
// range, so the cell it names blocks a ground mover and does NOT block an air
// one. 512 is the range's first index.
const waterWord = 0x0200

// marginMap is a synthetic w*h map, plain ground everywhere except a single
// WATER cell at its centre, and nothing placed on it.
//
// That one cell is the whole reason this fixture is not an all-zero grid. Water
// is blocked to a ground mover and not to an air one, so the map carries a cell
// that is blocked WITHOUT being margin. Without it, every blocked cell here
// would be a margin cell, a count that read the plane for "any block bit" would
// agree with one that read bit 1 on every shape below, and the distinction the
// count is built on would be invisible to all of them.
func marginMap(w, h int) []byte {
	tiles := make([]byte, 2*w*h)
	binary.LittleEndian.PutUint16(tiles[2*((h/2)*w+w/2):], waterWord)
	return synth.ALM(synth.ALMOptions{Width: w, Height: h, Type1Payload: tiles})
}

// loadViewer runs the one load path both front-ends use.
func loadViewer(t *testing.T, data []byte) *game.MapView {
	t.Helper()
	mv, err := game.LoadMapViewer(&terrain.Tileset{}, data, "fixture", game.Markers{}, game.StaticLayer{}, game.StructureLayer{})
	if err != nil {
		t.Fatalf("LoadMapViewer: %v", err)
	}
	return mv
}

func TestLoadedViewerCarriesTheMargin(t *testing.T) {
	// The depth is read out of the derivation itself rather than written down
	// here: this test asserts the SHAPE of the margin, and nothing in it should
	// have to be edited if the one place that owns the width is corrected.
	d := derivedDepth(t)
	if d <= 0 {
		t.Fatalf("derived margin depth = %d, want a positive ring", d)
	}

	for _, s := range []struct{ w, h int }{{20, 20}, {24, 19}, {19, 24}} {
		mv := loadViewer(t, marginMap(s.w, s.h))
		interior := (s.w - 2*d) * (s.h - 2*d)
		want := s.w*s.h - interior
		if got := mv.Viewer.MarginCells(); got != want {
			t.Errorf("%dx%d map: MarginCells = %d, want %d (a ring %d deep on four sides)",
				s.w, s.h, got, want, d)
		}
	}
}

func TestLoadedNarrowMapIsMarginThroughout(t *testing.T) {
	d := derivedDepth(t)

	for _, s := range []struct{ w, h int }{{2 * d, 40}, {40, 2 * d}, {3, 3}, {1, 1}} {
		mv := loadViewer(t, marginMap(s.w, s.h))
		want := s.w * s.h
		if got := mv.Viewer.MarginCells(); got != want {
			t.Errorf("%dx%d map: MarginCells = %d, want %d — every cell is margin", s.w, s.h, got, want)
		}
	}
}

func TestViewerMarginIsTheSimulationMargin(t *testing.T) {
	const w, h = 24, 19
	data := marginMap(w, h)

	m, err := alm.Open(data)
	if err != nil {
		t.Fatalf("alm.Open: %v", err)
	}
	// Census is the simulation tier's own count of its own plane, arm by arm.
	c := mapload.Census(m)
	want := c.Border

	mv := loadViewer(t, data)
	if got := mv.Viewer.MarginCells(); got != want {
		t.Fatalf("MarginCells = %d, but the simulation plane's border arm counts %d", got, want)
	}

	// The fixture has to be able to tell the two readings apart, and saying so
	// here is what keeps that a checked property rather than a hope: the map
	// must carry a cell that is blocked to a ground mover WITHOUT being margin,
	// or a count of "any block bit" would come out at exactly this number too.
	if want == 0 || want == w*h {
		t.Fatalf("fixture drift: the %dx%d census counts %d margin cells of %d, so the "+
			"comparison is between two trivial answers", w, h, want, w*h)
	}
	if c.Ground <= c.Border {
		t.Fatalf("fixture drift: %d cells block a ground mover and %d are margin, so no cell "+
			"is blocked without being margin and a plane read for any block bit would agree here",
			c.Ground, c.Border)
	}
}

// derivedDepth recovers the margin depth from the derivation itself, by
// censusing a tall single-column map: every cell of a map one cell wide is
// margin whatever the depth, so the discriminating shape is a WIDE, SHORT one —
// a map 1 cell tall is margin throughout, and a map 2d+1 tall has exactly one
// playable row. It walks upward until a playable cell appears.
//
// This exists so no test above restates a number that one production constant
// already owns.
func derivedDepth(t *testing.T) int {
	t.Helper()
	const wide = 400 // wider than any plausible depth, so only the rows decide
	for h := 1; h <= 64; h++ {
		m, err := alm.Open(marginMap(wide, h))
		if err != nil {
			t.Fatalf("alm.Open: %v", err)
		}
		c := mapload.Census(m)
		if c.Border < wide*h {
			// The first height with a playable row is 2d+1.
			if h%2 == 0 {
				t.Fatalf("the first height with a playable row is %d, which is even: a ring "+
					"of equal depth on both sides can only make an odd one", h)
			}
			return (h - 1) / 2
		}
	}
	t.Fatal("no height up to 64 had a playable row; the margin depth is not recoverable")
	return 0
}

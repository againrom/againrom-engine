package sim

import "testing"

// blockStamp is the client stamp built independently of fogStamp: the AI
// reader's single-cell march from walk, each lit cell shifted by shift and
// widened to the 2x2 block (x,y)..(x+1,y+1) (TERR-FOG-266). On a world with no
// border the two readers' rectangles agree, so the march is the same walk.
func blockStamp(w *World, walk cell, shift, sight256 int32) []byte {
	var acc [sightArea]int32
	single := make([]byte, len(w.grid))
	w.marchInto(aiSight, single, &acc, walk, sight256)
	out := make([]byte, len(w.grid))
	cols := int(w.bounds.Width)
	for i, v := range single {
		if v == 0 {
			continue
		}
		x, y := int32(i%cols)+shift, int32(i/cols)+shift
		for _, c := range [4]cell{{x, y}, {x + 1, y}, {x, y + 1}, {x + 1, y + 1}} {
			if j, ok := w.cellIndex(c.x, c.y); ok {
				out[j] = 1
			}
		}
	}
	return out
}

func stampDiff(t *testing.T, label string, got, want []byte) {
	t.Helper()
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: cell %d is %d, want %d", label, i, got[i], want[i])
		}
	}
}

// The first game's fog stamp is the 2x2 block per positive accumulator cell,
// and on flat ground its counts are the dilated disc's.
func TestTheFogStampMarksTheTwoByTwoBlock(t *testing.T) {
	t.Parallel()
	centre := cell{x: 22, y: 22}
	for _, tc := range []struct {
		r    uint8
		want int
	}{{1, 16}, {2, 32}, {3, 60}, {4, 88}, {5, 128}, {6, 172}} {
		w, err := NewTerrainWorld(1, sightBounds, ModeCanonical, Terrain{}, []Entity{
			{ID: 1, X: centre.x, Y: centre.y, Owner: 1, ScanRange: tc.r},
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		got := w.Sight(1)
		stampDiff(t, "flat ground", got, blockStamp(w, centre, 0, int32(tc.r)<<8))
		if n := litCount(got); n != tc.want {
			t.Errorf("range %d: the fog stamp lights %d cells, want %d", tc.r, n, tc.want)
		}
	}
}

// The walk centres on the body's cell; the write centre adds half the
// footprint side, so a two- or three-cell body's block moves by one cell while
// its walk, read on uneven ground, stays where the body stands.
func TestTheFogStampWritesAroundTheWriteCentre(t *testing.T) {
	t.Parallel()
	heights := sightPlane(sightBounds, 0)
	for i := range heights {
		heights[i] = byte(i * 37 % 101)
	}
	for _, tc := range []struct {
		size  uint8
		shift int32
	}{{0, 0}, {1, 0}, {2, 1}, {3, 1}, {4, 2}} {
		from := cell{x: 21, y: 23}
		w, err := NewTerrainWorld(1, sightBounds, ModeCanonical, Terrain{Height: heights}, []Entity{
			{ID: 1, X: from.x, Y: from.y, Owner: 1, ScanRange: 5, TokenSize: tc.size},
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		stampDiff(t, "uneven ground", w.Sight(1), blockStamp(w, from, tc.shift, 5<<8))
		if tc.shift != 0 {
			moved := blockStamp(w, cell{x: from.x + tc.shift, y: from.y + tc.shift}, 0, 5<<8)
			same := true
			for i, v := range moved {
				same = same && v == w.Sight(1)[i]
			}
			if same {
				t.Fatalf("size %d: a walk from the write centre gives the same stamp; the fixture does not separate them", tc.size)
			}
		}
	}
}

// A Human seeds the stamp with its sight word, whose fraction comes from Mind
// and Reaction (HERO-SIGHT-007); a loaded low byte takes precedence.
func TestTheFogStampSeedsWithTheSightWord(t *testing.T) {
	t.Parallel()
	from := cell{x: 22, y: 22}
	human := Entity{ID: 1, X: from.x, Y: from.y, Owner: 1, ScanRange: 6, Humanoid: true, Mind: 42, Reaction: 24}
	if word := human.SightWord(true); word != 1699 {
		t.Fatalf("sight word %d, want 1699", word)
	}
	w, err := NewTerrainWorld(1, sightBounds, ModeCanonical, Terrain{}, []Entity{human}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := w.Sight(1)
	stampDiff(t, "human", got, blockStamp(w, from, 0, 1699))
	if whole := blockStamp(w, from, 0, 6<<8); litCount(got) <= litCount(whole) {
		t.Fatalf("the sight word lights %d cells, the whole range %d", litCount(got), litCount(whole))
	}
	loaded := human
	loaded.NativeBasis.ScalarsPresent = true
	loaded.NativeBasis.ScalarKnown |= 1 << ScalarUA4
	loaded.NativeBasis.Scalars[ScalarUA4] = 6<<8 | 0x10
	if word := loaded.SightWord(true); word != 6<<8|0x10 {
		t.Fatalf("loaded sight word %d", word)
	}
}

// The AI reader and the second game's fog stay one cell per positive
// accumulator cell.
func TestTheFogStampLeavesTheAIReaderAndTheSecondGame(t *testing.T) {
	t.Parallel()
	e := Entity{ID: 1, X: 22, Y: 22, Owner: 1, ScanRange: 4, HP: 10, MaxHP: 10}
	w, err := NewTerrainWorld(1, sightBounds, ModeCanonical, Terrain{}, []Entity{e}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n := litCount(w.actorSight(0)); n != 69 {
		t.Errorf("the AI reader lights %d cells, want the range-4 disc's 69", n)
	}
	w.rom2 = &rom2ScriptState{}
	if n := litCount(w.Sight(1)); n != 69 {
		t.Errorf("the second game's fog lights %d cells, want the range-4 disc's 69", n)
	}
	if n := litCount(w.SightOf(1, []EntityID{1})); n != 69 {
		t.Errorf("the second game's moved-body fog lights %d cells, want 69", n)
	}
}

package terrain_test

import (
	"testing"

	"againrom/pkg/render/terrain"
)

// tileWord assembles a type1 tile word from its render fields: strip group g
// (bits 6..12), blend column b (bits 4..5) and sub-cell (bits 0..3).
func tileWord(g, b, sub int) uint16 {
	return uint16(g<<6) | uint16(b<<4) | uint16(sub)
}

// corpusCells is the sub-cell count of the strip a corpus-valid group resolves
// to: water (g 8..11 -> tile3) is 8 cells tall, every other group is 14.
func corpusCells(g int) int {
	if g >= 8 && g <= 11 {
		return 8
	}
	return 14
}

// TestResolveMapping - AC-5: over every corpus group (0..12), every blend column
// and every sub-cell, the resolved slot is g*4+b and equals the file identity
// tileG-VV.bmp with G = (g>>2)+1, V = (g&3)*4+b; the sub-cell is w & 0xf; and
// water (g 8..11) is forced to group 8.
func TestResolveMapping(t *testing.T) {
	for g := 0; g <= 12; g++ {
		for b := 0; b < 4; b++ {
			for sub := 0; sub < 16; sub++ {
				word := tileWord(g, b, sub)
				ref := terrain.Resolve(word)

				wantWater := g >= 8 && g <= 11
				effective := g
				if wantWater {
					effective = 8 // phase 0
				}
				wantSlot := effective*4 + b
				if ref.Slot != wantSlot {
					t.Fatalf("g=%d b=%d sub=%d: Slot = %d, want %d", g, b, sub, ref.Slot, wantSlot)
				}
				if ref.Sub != sub {
					t.Fatalf("g=%d b=%d sub=%d: Sub = %d, want %d", g, b, sub, ref.Sub, sub)
				}
				if ref.Water != wantWater {
					t.Fatalf("g=%d: Water = %v, want %v", g, ref.Water, wantWater)
				}

				// The slot must be the same thing as the filename the research
				// names: tileG-VV.bmp at tiles[(G-1)*16 + V].
				fileG := (effective >> 2) + 1
				fileV := (effective&3)*4 + b
				if got := terrain.SlotIndex(fileG, fileV); got != ref.Slot {
					t.Fatalf("g=%d b=%d: slot %d != SlotIndex(G=%d,V=%d) = %d",
						g, b, ref.Slot, fileG, fileV, got)
				}
				if fileG < 1 || fileG > 4 {
					t.Fatalf("g=%d: file group G = %d, want 1..4", g, fileG)
				}
			}
		}
	}

	// Water's four stored groups all collapse onto tile3-0b at phase 0.
	for g := 8; g <= 11; g++ {
		for b := 0; b < 4; b++ {
			ref := terrain.Resolve(tileWord(g, b, 0))
			if want := terrain.SlotIndex(3, b); ref.Slot != want {
				t.Fatalf("water g=%d b=%d: Slot = %d, want tile3-0%d = %d", g, b, ref.Slot, b, want)
			}
		}
	}

	// Road is the single group 12 -> tile4-00..03.
	for b := 0; b < 4; b++ {
		if got, want := terrain.Resolve(tileWord(12, b, 0)).Slot, terrain.SlotIndex(4, b); got != want {
			t.Fatalf("road b=%d: Slot = %d, want tile4-0%d = %d", b, got, b, want)
		}
	}
}

func TestResolveIgnoresNonMappingBits(t *testing.T) {
	for g := 0; g <= 12; g++ {
		for b := 0; b < 4; b++ {
			for sub := 0; sub < 16; sub++ {
				base := tileWord(g, b, sub)
				plain := terrain.Resolve(base)
				for _, extra := range []uint16{terrain.ImpassableBit, 0x4000, 0x8000, 0xE000} {
					if got := terrain.Resolve(base | extra); got != plain {
						t.Fatalf("word %#04x | %#04x resolved to %+v, want %+v", base, extra, got, plain)
					}
				}
				if terrain.IsImpassable(base) {
					t.Fatalf("word %#04x should not be impassable", base)
				}
				if !terrain.IsImpassable(base | terrain.ImpassableBit) {
					t.Fatalf("word %#04x should be impassable", base|terrain.ImpassableBit)
				}
			}
		}
	}
}

func TestResolveIsTotalAndPure(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Resolve panicked: %v", r)
		}
	}()
	for w := 0; w <= 0xffff; w++ {
		word := uint16(w)
		ref := terrain.Resolve(word)
		if ref != terrain.Resolve(word) {
			t.Fatalf("word %#04x: Resolve is not deterministic", word)
		}
		if ref.Slot < 0 {
			t.Fatalf("word %#04x: negative slot %d", word, ref.Slot)
		}
		if ref.Sub < 0 || ref.Sub > 15 {
			t.Fatalf("word %#04x: sub-cell %d outside 0..15", word, ref.Sub)
		}
		if ref.Sub != int(word&0xf) {
			t.Fatalf("word %#04x: Sub = %d, want %d", word, ref.Sub, word&0xf)
		}
	}
}

// TERR-VER-005
func TestResolveCorpusDomainHitsShippedFiles(t *testing.T) {
	ts := terrain.LoadTileset(shippedSource())

	for g := 0; g <= 12; g++ {
		for b := 0; b < 4; b++ {
			for sub := 0; sub < corpusCells(g); sub++ {
				word := tileWord(g, b, sub)
				ref := terrain.Resolve(word)

				strip := ts.Slot(ref.Slot)
				if strip == nil {
					t.Fatalf("g=%d b=%d sub=%d: slot %d is absent on a shipped install",
						g, b, sub, ref.Slot)
				}
				if strip.SubCell(ref.Sub) == nil {
					t.Fatalf("g=%d b=%d sub=%d: sub-cell %d out of range for a %d-cell strip",
						g, b, sub, ref.Sub, len(strip.SubCells))
				}
			}
		}
	}

	// The discriminating half: water strips are only 8 cells tall, so a
	// land-range sub-cell must NOT resolve inside a water strip.
	for g := 8; g <= 11; g++ {
		ref := terrain.Resolve(tileWord(g, 0, 13))
		if ts.Slot(ref.Slot).SubCell(ref.Sub) != nil {
			t.Fatalf("water g=%d sub=13 resolved inside the strip; tile3 has only 8 cells", g)
		}
	}
}

func TestDirtSubCell(t *testing.T) {
	for _, tc := range []struct{ col, row, want int }{
		{0, 0, 0}, {1, 0, 1}, {2, 0, 2}, {3, 0, 3}, {4, 0, 0},
		{0, 1, 1}, {1, 1, 2}, {0, 2, 2}, {0, 3, 3}, {7, 9, 0},
	} {
		if got := terrain.DirtSubCell(tc.col, tc.row); got != tc.want {
			t.Errorf("DirtSubCell(%d,%d) = %d, want %d", tc.col, tc.row, got, tc.want)
		}
	}
	for col := 0; col < 64; col++ {
		for row := 0; row < 64; row++ {
			if got := terrain.DirtSubCell(col, row); got < 0 || got > 3 {
				t.Fatalf("DirtSubCell(%d,%d) = %d, outside dirt's four cells", col, row, got)
			}
		}
	}
}

package terrain

import (
	"slices"
	"testing"
)

func TestRenderTileWordsPreservesOtherFlagsAndSource(t *testing.T) {
	stored := []uint16{0x0000, 0x2051, 0x4000, 0x8000, 0xffff}
	before := slices.Clone(stored)
	got := RenderTileWords(stored)
	if !slices.Equal(got, []uint16{0x0000, 0x0051, 0x4000, 0x8000, 0xdfff}) || !slices.Equal(stored, before) {
		t.Fatalf("render=%04x stored=%04x", got, stored)
	}
	got[0] = 0x2000
	if stored[0] != 0 {
		t.Fatal("later fire mutated the main tile plane")
	}
}

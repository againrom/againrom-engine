package mapload_test

import (
	"encoding/binary"
	"fmt"
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// These are implementation witnesses from promoted TERR-PASS-049/050 and
// ALM-TERR-043, not new observations about ROM1. No expected value calls the
// production classifier, Passability, Cost, Height or a native World reader.
func TestOriginalTerrainCostTable1115DefaultsAndNarrowing(t *testing.T) {
	defaults := [11]byte{255, 8, 8, 9, 14, 6, 12, 11, 16, 8, 6}
	for _, tc := range []struct {
		name string
		raw  []byte
		want [11]byte
	}{
		{"missing Terrain section", synth.Reg(0, nil), defaults},
		{"empty Terrain section", originalTerrainREG1115(nil), defaults},
		{"missing keys retain executable defaults", originalTerrainREG1115([]synth.RegNode{
			{Name: "CostLand", Kind: 2, Int: 37},
			{Name: "CostWater", Kind: 2, Int: 0},
			{Name: "PassFlowers", Kind: 2, Int: 99},
		}), [11]byte{255, 37, 8, 9, 14, 6, 12, 11, 16, 0, 6}},
		{"integer values narrow to their low byte", originalTerrainREG1115([]synth.RegNode{
			{Name: "CostLand", Kind: 2, Int: -1},
			{Name: "CostGrass", Kind: 2, Int: 256},
			{Name: "CostFlowers", Kind: 2, Int: 257},
			{Name: "CostSand", Kind: 2, Int: -242},
			{Name: "CostCracked", Kind: 2, Int: 262},
			{Name: "CostStones", Kind: 2, Int: -244},
			{Name: "CostSavanna", Kind: 2, Int: 65547},
			{Name: "CostMountain", Kind: 2, Int: 272},
			{Name: "CostWater", Kind: 2, Int: 511},
			{Name: "CostRoad", Kind: 2, Int: -250},
		}), [11]byte{255, 255, 0, 1, 14, 6, 12, 11, 16, 255, 6}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := slices.Clone(tc.raw)
			got, err := data.OriginalTerrainCostTable(tc.raw)
			if err != nil || got != tc.want {
				t.Fatalf("cost table = %v, %v; want %v", got, err, tc.want)
			}
			if !slices.Equal(before, tc.raw) {
				t.Fatal("cost reader changed its registry input")
			}
		})
	}
}

func originalTerrainREG1115(keys []synth.RegNode) []byte {
	return synth.Reg(0, []synth.RegNode{{Name: "Terrain", Kind: 1, Children: keys}})
}

func TestOriginalTerrainCostTable1115MalformedRegistry(t *testing.T) {
	valid := originalTerrainREG1115([]synth.RegNode{{Name: "CostLand", Kind: 2, Int: 7}})
	badMagic := slices.Clone(valid)
	badMagic[0] ^= 1
	badCount := slices.Clone(valid)
	binary.LittleEndian.PutUint32(badCount[0x10:], ^uint32(0))
	for _, tc := range []struct {
		name string
		raw  []byte
	}{
		{"missing file bytes", nil},
		{"short header", valid[:23]},
		{"truncated heap footer", valid[:len(valid)-1]},
		{"bad magic", badMagic},
		{"unbounded node count", badCount},
		{"root outside node table", synth.RegRaw(2, 1, 0, 1, []synth.RegRawNode{{Name: []byte("Terrain"), Kind: 1}}, nil)},
		{"unsupported value type", synth.RegRaw(0, 1, 0, 1, []synth.RegRawNode{{Name: []byte("Terrain"), Kind: 8}}, nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := data.OriginalTerrainCostTable(tc.raw); err == nil {
				t.Fatal("malformed registry was accepted as constructor defaults")
			}
		})
	}
}

// The unusual values catch premature narrowing and upward rounding:
// e.g. (3*211+246)>>2 must be 219, with a wide accumulator and truncation.
func originalTerrainCosts1115() [11]byte {
	return [11]byte{255, 3, 251, 19, 23, 29, 211, 31, 246, 241, 37}
}

func originalTerrainMap1115(width, height int) *alm.Map {
	m := &alm.Map{Width: width, Height: height,
		Tiles: make([]uint16, width*height), Altitudes: make([]byte, width*height), Overlay: make([]byte, width*height)}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			// Literal group0/column1/subcell4 selects level1 and Land.
			m.Tiles[i] = 0x0014
			m.Altitudes[i] = byte(x*13 + y*29 + 137)
		}
	}
	return m
}

type originalTerrainCell1115 struct {
	name                  string
	x, y                  int
	word                  uint16
	overlay, cost, static byte
}

// This oracle describes the SET of the four indexed stores, rather than
// repeating the producer's stamp loops. TERR-PASS-049's square branch touches
// only the N-square; its nonsquare branch first stamps the entire array ring,
// then only the map's right and bottom strips. Outside-map differences matter.
func originalTerrainBorder1115(width, height, x, y int) bool {
	if width == height {
		return x < width && y < height && (x < 8 || y < 8 || x >= width-8 || y >= height-8)
	}
	arrayRing := x < 8 || y < 8 || x >= 248 || y >= 248
	mapRight := y < height && x >= width-8 && x < width
	mapBottom := x < width && y >= height-8 && y < height
	return arrayRing || mapRight || mapBottom
}

func assertOriginalTerrainPlanes1115(t *testing.T, got *sim.SavedCellPlanes, width, height int, edits []originalTerrainCell1115) {
	t.Helper()
	if got == nil {
		t.Fatal("missing raw original planes")
	}
	if got.Costs != originalTerrainCosts1115() {
		t.Fatalf("retained cost parameters = %v", got.Costs)
	}
	byCell := make(map[int]originalTerrainCell1115, len(edits))
	for _, edit := range edits {
		byCell[edit.y*256+edit.x] = edit
	}
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			cell := y*256 + x
			cost, heightByte, static := byte(1), byte(0), byte(0)
			if x < width && y < height {
				cost = 3 // fixture's literal Land cell, not a derived classifier answer
				heightByte = byte(x*13 + y*29 + 137)
			}
			if edit, exists := byCell[cell]; exists {
				cost, static = edit.cost, edit.static
			}
			if originalTerrainBorder1115(width, height, x, y) {
				static = 0x1f
			}
			if got.Cost[cell] != cost || got.Height[cell] != heightByte || got.Static[cell] != static || got.Dynamic[cell] != static || got.CostKnown[cell] != 1 {
				t.Fatalf("%dx%d cell (%d,%d)/%04x: cost/height/static/dynamic/known = %d/%d/%02x/%02x/%d, want %d/%d/%02x/%02x/1",
					width, height, x, y, cell, got.Cost[cell], got.Height[cell], got.Static[cell], got.Dynamic[cell], got.CostKnown[cell], cost, heightByte, static, static)
			}
		}
	}
}

func TestOriginalTerrainPlanes1115FullExtentAndExactBorders(t *testing.T) {
	for _, size := range [][2]int{{8, 8}, {16, 16}, {24, 24}, {256, 256}, {24, 40}, {40, 24}, {8, 256}, {256, 8}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			m := originalTerrainMap1115(size[0], size[1])
			got, err := mapload.OriginalTerrainPlanes(m, originalTerrainCosts1115())
			if err != nil {
				t.Fatal(err)
			}
			assertOriginalTerrainPlanes1115(t, got, size[0], size[1], nil)
			// The returned arrays cannot alias the decoded map's source slices.
			before := m.Altitudes[0]
			got.Height[0] ^= 255
			if m.Altitudes[0] != before {
				t.Fatal("raw height plane aliases the ALM source")
			}
		})
	}
}

func TestOriginalTerrainPlanes1115LiteralIngestAndCustomBlends(t *testing.T) {
	// Expected costs/classes are independent literals from the promoted pair
	// and selector tables. The five Mountain/Stones vectors cover every blend
	// arm, including the change of returned class between levels2 and3.
	cases := []originalTerrainCell1115{
		{name: "level1 secondary Stones", word: 0x01d4, cost: 211},
		{name: "level2 three quarters Stones", word: 0x01c0, cost: 219},
		{name: "level3 midpoint Mountain", word: 0x01c1, cost: 228, static: 1},
		{name: "level4 three quarters Mountain", word: 0x01c3, cost: 237, static: 1},
		{name: "level5 primary Mountain", word: 0x01d1, cost: 246, static: 1},
		{name: "group0 primary Grass", word: 0x0011, cost: 251},
		{name: "group0 secondary Land", word: 0x0014, cost: 3},
		{name: "group1 primary Cracked", word: 0x0051, cost: 29},
		{name: "group1 secondary Land", word: 0x0054, cost: 3},
		{name: "group2 primary Sand", word: 0x0091, cost: 23},
		{name: "group2 secondary Land", word: 0x0094, cost: 3},
		{name: "group3 primary Savanna", word: 0x00d1, cost: 31},
		{name: "group3 secondary Land", word: 0x00d4, cost: 3},
		{name: "group4 primary Stones", word: 0x0111, cost: 211},
		{name: "group4 secondary Land", word: 0x0114, cost: 3},
		{name: "group5 primary Cracked", word: 0x0151, cost: 29},
		{name: "group5 secondary Stones", word: 0x0154, cost: 211},
		{name: "group6 primary Flowers", word: 0x0191, cost: 19},
		{name: "group6 secondary Savanna", word: 0x0194, cost: 31},
		{name: "group12 primary Road", word: 0x0311, cost: 37},
		{name: "group12 secondary Land", word: 0x0314, cost: 3},
		{name: "water ignores CostWater241", word: 0x0200, cost: 8, static: 1},
		{name: "water Land exception stays raw-water blocked", word: 0x0214, cost: 8, static: 1},
		{name: "water group9", word: 0x0240, cost: 8, static: 1},
		{name: "water group10", word: 0x0280, cost: 8, static: 1},
		{name: "water group11", word: 0x02c0, cost: 8, static: 1},
		{name: "land subcell14 rejects", word: 0x000e, cost: 255},
		{name: "land subcell15 rejects", word: 0x000f, cost: 255},
		{name: "water subcell8 rejects but stays blocked", word: 0x0208, cost: 255, static: 1},
		{name: "water subcell15 rejects but stays blocked", word: 0x020f, cost: 255, static: 1},
		{name: "group13 subcell14 rejects before pair read", word: 0x034e, cost: 255},
		{name: "group14 subcell15 rejects before pair read", word: 0x038f, cost: 255},
		{name: "group15 subcell14 rejects before pair read", word: 0x03ce, cost: 255},
		{name: "masked bit10", word: 0x0414, cost: 3},
		{name: "masked bit11", word: 0x0814, cost: 3},
		{name: "masked bit12", word: 0x1014, cost: 3},
		{name: "masked bit14", word: 0x4014, cost: 3},
		{name: "masked bit15", word: 0x8014, cost: 3},
		{name: "all ignored high bits", word: 0xdc14, cost: 3},
		{name: "bit13 still blocks", word: 0x2014, cost: 3, static: 1},
		{name: "bit13 among ignored high bits", word: 0xfc14, cost: 3, static: 1},
		{name: "scenery overrides Mountain", word: 0x01d1, overlay: 7, cost: 246, static: 5},
		{name: "scenery overrides raw water", word: 0x0214, overlay: 255, cost: 8, static: 5},
		{name: "scenery overrides bit13", word: 0x2014, overlay: 1, cost: 3, static: 5},
		{name: "scenery on otherwise open terrain", word: 0x0014, overlay: 128, cost: 3, static: 5},
	}
	m := originalTerrainMap1115(48, 48)
	for i := range cases {
		cases[i].x, cases[i].y = 8+i%24, 12+i/24
	}
	// Border stamps happen AFTER scenery, even on a rejected cost cell.
	cases = append(cases, originalTerrainCell1115{name: "border overwrites scenery", x: 2, y: 3, word: 0x0208, overlay: 255, cost: 255, static: 5})
	for _, c := range cases {
		i := c.y*m.Width + c.x
		m.Tiles[i], m.Overlay[i] = c.word, c.overlay
	}
	beforeTiles, beforeHeight, beforeOverlay := slices.Clone(m.Tiles), slices.Clone(m.Altitudes), slices.Clone(m.Overlay)
	got, err := mapload.OriginalTerrainPlanes(m, originalTerrainCosts1115())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		cell := c.y*256 + c.x
		wantStatic := c.static
		if originalTerrainBorder1115(m.Width, m.Height, c.x, c.y) {
			wantStatic = 0x1f
		}
		if got.Cost[cell] != c.cost || got.Static[cell] != wantStatic {
			t.Errorf("%s (%04x): cost/static = %d/%02x; want %d/%02x", c.name, c.word, got.Cost[cell], got.Static[cell], c.cost, wantStatic)
		}
	}
	assertOriginalTerrainPlanes1115(t, got, m.Width, m.Height, cases)
	if !slices.Equal(m.Tiles, beforeTiles) || !slices.Equal(m.Altitudes, beforeHeight) || !slices.Equal(m.Overlay, beforeOverlay) {
		t.Fatal("raw plane construction changed the decoded ALM")
	}
}

func TestOriginalTerrainPlanes1115RejectsUninitializedPairs(t *testing.T) {
	for group := 13; group <= 15; group++ {
		for column := 0; column < 4; column++ {
			for subcell := 0; subcell < 14; subcell++ {
				word := uint16(group<<6 | column<<4 | subcell)
				m := originalTerrainMap1115(24, 24)
				m.Tiles[9*m.Width+9] = word
				got, err := mapload.OriginalTerrainPlanes(m, originalTerrainCosts1115())
				if err == nil || got != nil {
					t.Fatalf("uninitialized group/column/subcell %d/%d/%d (%04x) returned planes=%t, err=%v", group, column, subcell, word, got != nil, err)
				}
			}
		}
	}
}

func TestOriginalTerrainPlanes1115RejectsIncompleteInputs(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(**alm.Map, *[11]byte)
	}{
		{"nil map", func(m **alm.Map, _ *[11]byte) { *m = nil }},
		{"zero width", func(m **alm.Map, _ *[11]byte) { (*m).Width = 0 }},
		{"negative height", func(m **alm.Map, _ *[11]byte) { (*m).Height = -1 }},
		{"width below border", func(m **alm.Map, _ *[11]byte) { (*m).Width = 7 }},
		{"height below border", func(m **alm.Map, _ *[11]byte) { (*m).Height = 7 }},
		{"width beyond original plane", func(m **alm.Map, _ *[11]byte) { (*m).Width = 257 }},
		{"height beyond original plane", func(m **alm.Map, _ *[11]byte) { (*m).Height = 257 }},
		{"short tile plane", func(m **alm.Map, _ *[11]byte) { (*m).Tiles = (*m).Tiles[:575] }},
		{"long tile plane", func(m **alm.Map, _ *[11]byte) { (*m).Tiles = append((*m).Tiles, 0) }},
		{"short height plane", func(m **alm.Map, _ *[11]byte) { (*m).Altitudes = (*m).Altitudes[:575] }},
		{"long height plane", func(m **alm.Map, _ *[11]byte) { (*m).Altitudes = append((*m).Altitudes, 0) }},
		{"short overlay plane", func(m **alm.Map, _ *[11]byte) { (*m).Overlay = (*m).Overlay[:575] }},
		{"long overlay plane", func(m **alm.Map, _ *[11]byte) { (*m).Overlay = append((*m).Overlay, 0) }},
		{"invalid reject cost", func(_ **alm.Map, c *[11]byte) { c[0] = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, costs := originalTerrainMap1115(24, 24), originalTerrainCosts1115()
			tc.edit(&m, &costs)
			if got, err := mapload.OriginalTerrainPlanes(m, costs); err == nil || got != nil {
				t.Fatalf("incomplete input returned planes=%t, err=%v", got != nil, err)
			}
		})
	}
}

package game_test

import (
	"image/color"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/pal"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
)

// The four tier states one class exercises, and the two further classes: one
// declaring no tiers at all, and one naming the SAME sheet and the same
// directory, which is what the loader's per-pair memo must answer with the very
// slices the first class got.
const (
	tierAll   = 41 // Palette 4: equal, differing, absent, refused
	tierNone  = 43 // Palette 0: no tiers at all
	tierShare = 47 // Palette 4, same File as tierAll
)

// tierSheetPalette is the fixture sheet's own colour table, and tierOtherPalette
// a table that differs from it at every entry a pixel of the fixture can carry.
// Both are generated: three channels that disagree with each other, so a
// conversion that swapped two of them cannot pass.
func tierSheetPalette() []color.RGBA {
	out := make([]color.RGBA, 256)
	for i := range out {
		out[i] = color.RGBA{R: uint8(i), G: uint8(255 - i), B: uint8(i * 3)}
	}
	return out
}

func tierOtherPalette() []color.RGBA {
	out := tierSheetPalette()
	for i := range out {
		out[i].G ^= 0x5a
	}
	return out
}

// tierPaletteFile builds one synthetic palette file: the two magic bytes, filler
// up to the table offset, the 1024-byte table, and a run of pixel data after it
// that the decoder must never read. Palette256 lays the table out in the same
// [B, G, R, reserved] order a sheet's own palette block uses, which is the whole
// reason a .pal table and a .256 palette can be compared for equality at all.
func tierPaletteFile(colors []color.RGBA) []byte {
	out := make([]byte, pal.TableOffset)
	for i := range out {
		out[i] = 0xa5
	}
	out[0], out[1] = 'B', 'M'
	out = append(out, synth.Palette256(colors)...)
	return append(out, make([]byte, 64)...) // pixel data, never opened
}

func tierRegistry() []byte {
	i := func(name string, v int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x02, Int: v}
	}
	return synth.UnitsReg([]string{`tiered\sprites`, `plain\sprites`},
		[]synth.RegNode{
			i("ID", tierAll), i("File", 0), i("Palette", 4),
			i("Width", 8), i("Height", 8), i("CenterX", 4), i("CenterY", 6),
		},
		[]synth.RegNode{
			i("ID", tierNone), i("File", 1), i("Palette", 0),
			i("Width", 8), i("Height", 8), i("CenterX", 4), i("CenterY", 6),
		},
		[]synth.RegNode{
			i("ID", tierShare), i("File", 0), i("Palette", 4),
			i("Width", 8), i("Height", 8), i("CenterX", 4), i("CenterY", 6),
		},
	)
}

func tierBundle(t *testing.T) *terrain.UnitSet {
	t.Helper()
	sheet := synth.Sheet256(synth.Sheet256Options{
		Palette: tierSheetPalette(),
		Frames: []synth.Frame256{
			{Width: 2, Height: 1, Pixels: []synth.Pixel256{{Index: 9, Opaque: true}, {Index: 200, Opaque: true}}},
			{Width: 1, Height: 2, Pixels: []synth.Pixel256{{Index: 0, Opaque: true}, {}}},
		},
	})
	files := []synth.File{
		{Path: graphicsEntry(t, game.UnitRegistry), Data: tierRegistry()},
		{Path: "units/tiered/sprites.256", Data: sheet},
		{Path: "units/plain/sprites.256", Data: sheet},
		// Tier 1: the sheet's own table, byte for byte — not a recolour.
		{Path: "units/tiered/palette.pal", Data: tierPaletteFile(tierSheetPalette())},
		// Tier 2: a table that genuinely differs.
		{Path: "units/tiered/palette2.pal", Data: tierPaletteFile(tierOtherPalette())},
		// Tier 3 is ABSENT — no entry at units/tiered/palette3.pal at all.
		// Tier 4 is present and REFUSED: the right length, the wrong magic,
		// which is the shape the shared-owner palette has.
		{Path: "units/tiered/palette4.pal", Data: append([]byte{0x00, 0x00},
			tierPaletteFile(tierOtherPalette())[2:]...)},
	}
	set, err := game.LoadUnits(openContainers(t, synth.Archive(files)))
	if err != nil {
		t.Fatalf("LoadUnits: %v", err)
	}
	return set
}

func sameSlice(a, b []*terrain.StaticFrame) bool {
	if len(a) != len(b) || len(a) == 0 {
		return len(a) == len(b)
	}
	return &a[0] == &b[0]
}

// AC-5. The four outcomes, and the two facts that separate a recolour from a
// copy: a differing tier's frames carry the new table over the SAME pixel
// memory, and an equal tier is the base slice itself.
func TestLoadUnitsBuildsOneSlicePerDeclaredTier(t *testing.T) {
	c := tierBundle(t).Classes[tierAll]
	if c == nil {
		t.Fatalf("Classes[%d] is nil", tierAll)
	}
	if len(c.Tiers) != 4 {
		t.Fatalf("Tiers holds %d entries, want 4 — one per declared tier", len(c.Tiers))
	}
	if len(c.Frames) != 2 {
		t.Fatalf("the sheet loaded %d frames, want 2", len(c.Frames))
	}

	// Tier 1: the base slice ITSELF, because its table is the sheet's own.
	if !sameSlice(c.Tiers[0], c.Frames) {
		t.Error("tier 1 is not the base slice by identity; an equal table is not a recolour")
	}

	// Tier 2: a slice of its own, the new table over shared pixels.
	tier2 := c.Tiers[1]
	if sameSlice(tier2, c.Frames) {
		t.Fatal("tier 2 answered the base slice; its table differs")
	}
	if len(tier2) != len(c.Frames) {
		t.Fatalf("tier 2 holds %d frames, want %d", len(tier2), len(c.Frames))
	}
	want := tierOtherPalette()
	for i, f := range tier2 {
		base := c.Frames[i]
		if f == base {
			t.Fatalf("tier 2 frame %d IS the base frame", i)
		}
		if f.Width != base.Width || f.Height != base.Height {
			t.Fatalf("tier 2 frame %d is %dx%d, want %dx%d", i, f.Width, f.Height, base.Width, base.Height)
		}
		// The pixel memory is SHARED, not copied: same backing array.
		if len(f.Pixels) != len(base.Pixels) || (len(f.Pixels) > 0 && &f.Pixels[0] != &base.Pixels[0]) {
			t.Fatalf("tier 2 frame %d does not share the base frame's pixels", i)
		}
		for e := 0; e < 256; e++ {
			w := color.RGBA{R: want[e].R, G: want[e].G, B: want[e].B, A: 0xff}
			if f.Palette[e] != w {
				t.Fatalf("tier 2 frame %d entry %d = %+v, want %+v", i, e, f.Palette[e], w)
			}
		}
	}

	// Tiers 3 and 4: absent and refused. Both are skips — nil, which the render
	// tier answers the sheet's own frames for.
	for _, tier := range []int{3, 4} {
		if got := c.Tiers[tier-1]; got != nil {
			t.Errorf("tier %d built %d frames; an absent or refused table is a skip", tier, len(got))
		}
		if !sameSlice(c.TierFrames(tier), c.Frames) {
			t.Errorf("tier %d does not fall back to the sheet's own frames", tier)
		}
	}

	// The fallback count is read off the slices themselves: the declared count
	// is the length, and a nil entry is one that fell back.
	fell := 0
	for _, s := range c.Tiers {
		if len(s) == 0 {
			fell++
		}
	}
	if fell != 2 {
		t.Errorf("%d tier(s) fell back, want 2", fell)
	}
}

// A class declaring no tiers carries none, and draws exactly what it drew
// before this story at every tier anyone can name (AC-11).
func TestLoadUnitsGivesAnUntieredClassNoTiers(t *testing.T) {
	c := tierBundle(t).Classes[tierNone]
	if c == nil {
		t.Fatalf("Classes[%d] is nil", tierNone)
	}
	if len(c.Tiers) != 0 {
		t.Fatalf("Tiers holds %d entries, want none", len(c.Tiers))
	}
	for tier := -1; tier <= 5; tier++ {
		if !sameSlice(c.TierFrames(tier), c.Frames) {
			t.Errorf("tier %d did not answer the sheet's own frames", tier)
		}
	}
}

func TestLoadUnitsSharesATierBetweenClassesNamingOneSheet(t *testing.T) {
	set := tierBundle(t)
	a, b := set.Classes[tierAll], set.Classes[tierShare]
	if a == nil || b == nil {
		t.Fatalf("Classes[%d] = %v, Classes[%d] = %v", tierAll, a, tierShare, b)
	}
	if !sameSlice(a.Frames, b.Frames) {
		t.Fatal("the two classes do not share the base sheet slice")
	}
	if len(a.Tiers) != len(b.Tiers) {
		t.Fatalf("%d tiers against %d", len(a.Tiers), len(b.Tiers))
	}
	for i := range a.Tiers {
		if !sameSlice(a.Tiers[i], b.Tiers[i]) {
			t.Errorf("tier %d is not shared between the two classes", i+1)
		}
	}
}

// A tier whose sheet did not load is the third skip, and it is a skip and not a
// failure: the registry loads, the class is in the bundle, and it has no frames
// at any tier.
func TestLoadUnitsSkipsATierWhoseSheetIsAbsent(t *testing.T) {
	i := func(name string, v int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x02, Int: v}
	}
	files := []synth.File{
		{Path: graphicsEntry(t, game.UnitRegistry), Data: synth.UnitsReg([]string{`gone\sprites`},
			[]synth.RegNode{i("ID", tierAll), i("File", 0), i("Palette", 2)})},
		// The sheet is NOT in the archive; the tables are.
		{Path: "units/gone/palette.pal", Data: tierPaletteFile(tierSheetPalette())},
		{Path: "units/gone/palette2.pal", Data: tierPaletteFile(tierOtherPalette())},
	}
	set, err := game.LoadUnits(openContainers(t, synth.Archive(files)))
	if err != nil {
		t.Fatalf("LoadUnits: %v", err)
	}
	c := set.Classes[tierAll]
	if c == nil {
		t.Fatalf("Classes[%d] is nil; a sheetless class is a skip, not a missing entry", tierAll)
	}
	if len(c.Frames) != 0 {
		t.Fatalf("the class carries %d frames, want none", len(c.Frames))
	}
	if len(c.Tiers) != 2 {
		t.Fatalf("Tiers holds %d entries, want 2 — the declared count stands", len(c.Tiers))
	}
	for tier, s := range c.Tiers {
		if s != nil {
			t.Errorf("tier %d built %d frames off a sheet that did not load", tier+1, len(s))
		}
	}
}

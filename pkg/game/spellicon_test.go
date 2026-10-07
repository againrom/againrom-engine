package game

// The spell icon strip: the cut, the extent check and the cache (0141;
// `MAGIC-ICON-024`). Fixtures are synthetic — synthBMP (portrait_test.go)
// builds the one bitmap shape this corpus ships.

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/bmp"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// atlasFixture is an atlas of the shipped extent whose every pixel encodes its
// OWN coordinate, so a cell taken from the wrong place is caught by what is in
// it rather than only by its size.
func atlasFixture(t *testing.T) *bmp.Image {
	t.Helper()
	im := &bmp.Image{
		Width:  data.SpellIconAtlasW,
		Height: data.SpellIconAtlasH,
		Pix:    make([]bmp.Color, data.SpellIconAtlasW*data.SpellIconAtlasH),
	}
	for y := 0; y < im.Height; y++ {
		for x := 0; x < im.Width; x++ {
			im.Pix[y*im.Width+x] = bmp.Color{R: uint8(x), G: uint8(x >> 8), B: uint8(y)}
		}
	}
	return im
}

func TestCutSpellIconTakesTheSlotsOwnSquare(t *testing.T) {
	im := atlasFixture(t)

	for _, slot := range []int{0, 1, 11, 12, 23} {
		x0, y0, ok := data.SpellIconCell(slot)
		if !ok {
			t.Fatalf("slot %d does not exist", slot)
		}
		pic := cutSpellIcon(im, slot)
		if pic == nil {
			t.Fatalf("slot %d cut nothing", slot)
		}
		if b := pic.Bounds(); b.Dx() != data.SpellIconSide || b.Dy() != data.SpellIconSide {
			t.Fatalf("slot %d cut %v, want %dx%d", slot, b, data.SpellIconSide, data.SpellIconSide)
		}
		// The corners say where in the atlas this came from, and the alpha says
		// the format has no fourth channel to carry.
		for _, c := range [][2]int{{0, 0}, {data.SpellIconSide - 1, 0}, {0, data.SpellIconSide - 1},
			{data.SpellIconSide - 1, data.SpellIconSide - 1}} {
			sx, sy := x0+c[0], y0+c[1]
			want := color.RGBA{R: uint8(sx), G: uint8(sx >> 8), B: uint8(sy), A: 0xff}
			if got := pic.RGBAAt(c[0], c[1]); got != want {
				t.Errorf("slot %d pixel (%d,%d) = %+v, want %+v — cut from the wrong place",
					slot, c[0], c[1], got, want)
			}
		}
	}

	// Two different slots are two different pictures, which is the property a
	// grid that collapsed to one origin would break silently.
	a, b := cutSpellIcon(im, 0), cutSpellIcon(im, 1)
	if a.RGBAAt(0, 0) == b.RGBAAt(0, 0) {
		t.Error("slots 0 and 1 cut the same pixels")
	}

	if cutSpellIcon(im, -1) != nil || cutSpellIcon(im, data.SpellIconSlots) != nil {
		t.Error("a slot outside the grid cut something")
	}
	if cutSpellIcon(nil, 0) != nil {
		t.Error("a nil atlas cut something")
	}
}

// The atlas is refused unless it is the extent the grid was measured against:
// cutting this grid out of a different picture would hand the window somebody
// else's pixels rather than fail.
func TestSpellAtlasRefusesAPictureOfTheWrongExtent(t *testing.T) {
	addr := graphicsPrefix + data.SpellIconAtlasPath

	for _, tc := range []struct {
		name string
		body []byte
		want bool
	}{
		{"the shipped extent", synthBMP(data.SpellIconAtlasW, data.SpellIconAtlasH, color.RGBA{A: 0xff}), true},
		{"a picture of some other size", synthBMP(64, 64, color.RGBA{A: 0xff}), false},
		{"not a bitmap at all", []byte("no"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mw := &mapWorld{mission: &missionNotices{src: missionSource{addr: tc.body}}}
			if got := mw.spellAtlas() != nil; got != tc.want {
				t.Errorf("atlas resolved = %v, want %v", got, tc.want)
			}
		})
	}

	// A MAP THE PICKER OPENED HAS NO MISSION AT ALL, and mw.mission is a nil
	// POINTER there — the case that made mw.archive() necessary. Both shapes
	// are driven, because the first draft of this file dereferenced the nil
	// one and panicked.
	t.Run("no mission at all", func(t *testing.T) {
		mw := &mapWorld{}
		if mw.spellAtlas() != nil {
			t.Error("a world with no mission resolved an atlas")
		}
		for id := uint16(1); id <= 3; id++ {
			if mw.spellIcon(id) != nil {
				t.Errorf("spell %d answered an icon with no mission", id)
			}
		}
	})

	t.Run("a mission carrying no source", func(t *testing.T) {
		mw := &mapWorld{mission: &missionNotices{}}
		if mw.spellAtlas() != nil {
			t.Error("a mission with no source resolved an atlas")
		}
	})
}

// Both caches hold their misses: the atlas is attempted once per mission
// whatever the answer, and a spell with no slot answers nil once.
func TestSpellIconCachesTheAtlasAndTheMisses(t *testing.T) {
	reads := 0
	addr := graphicsPrefix + data.SpellIconAtlasPath
	src := countingSource{
		missionSource: missionSource{
			addr: synthBMP(data.SpellIconAtlasW, data.SpellIconAtlasH, color.RGBA{B: 0xff, A: 0xff}),
		},
		reads: &reads,
	}
	mw := &mapWorld{mission: &missionNotices{src: src}}

	// Spell 1 is slot 0 and spell 18 is the last slot; both have a picture.
	if mw.spellIcon(1) == nil || mw.spellIcon(18) == nil {
		t.Fatal("a spell with a slot answered no icon")
	}
	// Ids 11, 17, 27 and 28 are in no slot at all — 24 cells serve 28 spells.
	for _, id := range []uint16{11, 17, 27, 28} {
		if pic := mw.spellIcon(id); pic != nil {
			t.Errorf("spell %d answered an icon, want none", id)
		}
	}
	if reads != 1 {
		t.Errorf("reads = %d over six spells, want 1 — the atlas is read once", reads)
	}

	// Asking again touches nothing.
	for _, id := range []uint16{1, 18, 11, 28} {
		mw.spellIcon(id)
	}
	if reads != 1 {
		t.Errorf("reads = %d after a second pass, want 1", reads)
	}

	// An install with no atlas: every spell answers nil, and it is attempted
	// once rather than once per spell.
	empty := 0
	none := countingSource{missionSource: missionSource{}, reads: &empty}
	mw2 := &mapWorld{mission: &missionNotices{src: none}}
	for id := uint16(1); id <= 28; id++ {
		if mw2.spellIcon(id) != nil {
			t.Fatalf("spell %d answered an icon with no atlas", id)
		}
	}
	if empty != 1 {
		t.Errorf("reads = %d over 28 spells with no atlas, want 1", empty)
	}
}

// spellbookOf carries the icon through and is unchanged for a caller that has
// none — which is what keeps every book this build composed before icons
// existed composing identically.
func TestSpellbookOfCarriesTheIconAndIsTotalWithout(t *testing.T) {
	icon := image.NewRGBA(image.Rect(0, 0, data.SpellIconSide, data.SpellIconSide))
	lookup := func(id uint16) *image.RGBA {
		if id == 2 {
			return icon
		}
		return nil
	}

	e := sim.Entity{KnownSpells: (1 << 1) | (1 << 2)}
	table := []sim.SpellRule{{ID: 1}, {ID: 2}}
	names := map[uint16]string{1: "Fire Arrow", 2: "Shield"}

	with := spellbookOf(sim.Rules{}, e, table, names, ui.Words{}, lookup)
	if len(with) != 2 {
		t.Fatalf("book of %d, want 2", len(with))
	}
	if with[0].Icon != nil {
		t.Error("spell 1 carries an icon its lookup does not give")
	}
	if with[1].Icon != icon {
		t.Error("spell 2's icon did not reach the entry")
	}

	without := spellbookOf(sim.Rules{}, e, table, names, ui.Words{}, nil)
	if len(without) != 2 || without[0].Icon != nil || without[1].Icon != nil {
		t.Error("a nil lookup put an icon on an entry")
	}
	for i := range without {
		if without[i].ID != with[i].ID || without[i].Name != with[i].Name {
			t.Errorf("entry %d differs beyond its icon", i)
		}
	}
}

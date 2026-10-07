package game

// The flat portrait: the decode, the cache and the decision behind the push
// (0141; `UNIT-PICT-035`, `UNIT-PICT-036`, `SPR256-PICT-043`).
//
// EVERY FIXTURE IS SYNTHETIC (AGENTS.md rule 2). synthBMP below builds the one
// bitmap shape this corpus ships at the format's own byte grammar — the same
// approach inventory_test.go takes to a `.16a` icon, and for the same reason:
// the shape is four header fields and a pixel run, and a builder for it is
// shorter than a fixture file could ever be.
//
// THE PUSH ITSELF IS NOT DRIVEN HERE. pushPortrait reads a selection that lives
// in pkg/ui and that this package cannot set, which is why the decision it
// makes is a function of its own — unitPicture — exactly as spellbookOf is
// split out of pushSpellbook (spell.go). What that split leaves untested is
// named at the end of this file.

import (
	"encoding/binary"
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

// synthBMP is a 24-bit uncompressed Windows bitmap of w x h, every pixel the
// same colour, carrying the two trailing bytes and the zero `imgSize` that 85
// of the 86 shipped portrait nodes have.
func synthBMP(w, h int, c color.RGBA) []byte {
	const header = 54
	stride := (w*3 + 3) &^ 3
	b := make([]byte, header+stride*h+2)
	b[0], b[1] = 'B', 'M'
	binary.LittleEndian.PutUint32(b[2:], uint32(len(b)))
	binary.LittleEndian.PutUint32(b[10:], header)
	binary.LittleEndian.PutUint32(b[14:], 40)
	binary.LittleEndian.PutUint32(b[18:], uint32(w))
	binary.LittleEndian.PutUint32(b[22:], uint32(h))
	binary.LittleEndian.PutUint16(b[26:], 1)
	binary.LittleEndian.PutUint16(b[28:], 24)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := header + y*stride + x*3
			b[p], b[p+1], b[p+2] = c.B, c.G, c.R
		}
	}
	return b
}

func TestLoadPortraitPreservesNonBlackPixels(t *testing.T) {
	want := color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xff}
	const addr = "graphics/infowindow/Bee.bmp"
	src := missionSource{addr: synthBMP(4, 3, want)}

	pic := loadPortrait(src, addr)
	if pic == nil {
		t.Fatal("loadPortrait answered nil for a well-formed node")
	}
	if b := pic.Bounds(); b.Dx() != 4 || b.Dy() != 3 {
		t.Fatalf("decoded %v, want 4x3", b)
	}
	// Colour keying must preserve every non-black source colour.
	for y := 0; y < 3; y++ {
		for x := 0; x < 4; x++ {
			if got := pic.RGBAAt(x, y); got != want {
				t.Fatalf("pixel (%d,%d) = %+v, want %+v", x, y, got, want)
			}
		}
	}
}

func TestLoadPortraitKeysOnlyPureBlack(t *testing.T) {
	for _, c := range []color.RGBA{{A: 255}, {R: 1, A: 255}, {G: 1, A: 255}, {B: 1, A: 255}} {
		src := missionSource{"portrait.bmp": synthBMP(3, 2, c)}
		pic := loadPortrait(src, "portrait.bmp")
		want := c
		if c.R|c.G|c.B == 0 {
			want.A = 0
		}
		if pic == nil {
			t.Fatal("portrait missing")
		}
		for y := 0; y < 2; y++ {
			for x := 0; x < 3; x++ {
				if got := pic.RGBAAt(x, y); got != want {
					t.Fatalf("source %v at %d,%d: got %v, want %v", c, x, y, got, want)
				}
			}
		}
	}
}

// EVERY FAILURE IS AN ABSENCE AND NONE IS AN ERROR — the contract the window
// above needs, because an install that does not carry a node is shipped
// reality (`REG-PICT-083`: the RU root names thirteen it has no entry for).
func TestLoadPortraitAnswersNilForEveryFailure(t *testing.T) {
	src := missionSource{"graphics/infowindow/good.bmp": synthBMP(2, 2, color.RGBA{A: 0xff})}

	for _, tc := range []struct {
		name string
		src  entrySource
		addr string
	}{
		{"no source at all", nil, "graphics/infowindow/good.bmp"},
		{"no address", src, ""},
		{"a node the archive does not carry", src, "graphics/infowindow/absent.bmp"},
		{"a node that is not a bitmap", missionSource{"x": []byte("not a bitmap")}, "x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if pic := loadPortrait(tc.src, tc.addr); pic != nil {
				t.Error("answered a picture, want nil")
			}
		})
	}
}

// The cache holds its MISSES as well as its hits, so an install without a node
// costs one read for the session rather than one per frame — and two classes
// naming one picture cost one read between them, which is the shipped case
// (`Bee` on ids 1 and 2, `Mage` on 23 and 24).
func TestClassPortraitCachesByNameIncludingItsMisses(t *testing.T) {
	reads := 0
	src := countingSource{
		missionSource: missionSource{
			"graphics/infowindow/Bee.bmp": synthBMP(2, 2, color.RGBA{R: 0xff, A: 0xff}),
		},
		reads: &reads,
	}
	cache := map[string]*image.RGBA{}

	bee1 := &terrain.UnitClass{Portrait: "Bee"}
	bee2 := &terrain.UnitClass{Portrait: "Bee"}
	gone := &terrain.UnitClass{Portrait: "absent"}
	unnamed := &terrain.UnitClass{}

	if classPortrait(src, bee1, 1, cache) == nil {
		t.Fatal("the first read answered no picture")
	}
	if classPortrait(src, bee2, 1, cache) == nil {
		t.Fatal("a second class naming the same picture answered none")
	}
	if reads != 1 {
		t.Errorf("reads = %d over two classes naming one picture, want 1", reads)
	}

	if classPortrait(src, gone, 1, cache) != nil {
		t.Error("a name the archive does not carry answered a picture")
	}
	if classPortrait(src, gone, 1, cache) != nil {
		t.Error("the cached miss answered a picture")
	}
	if reads != 2 {
		t.Errorf("reads = %d, want 2 — the miss was read a second time", reads)
	}

	if classPortrait(src, unnamed, 1, cache) != nil {
		t.Error("a class carrying no picture name answered a picture")
	}
	if classPortrait(src, nil, 1, cache) != nil {
		t.Error("a nil class answered a picture")
	}
	if reads != 2 {
		t.Errorf("reads = %d — a class naming nothing reached the archive", reads)
	}
}

// THE TIER REACHES THE ADDRESS (owner), and a class with only one picture
// falls back to it rather than showing none.
func TestClassPortraitAddressesTheTier(t *testing.T) {
	src := missionSource{
		"graphics/infowindow/Orc.bmp":  synthBMP(2, 2, color.RGBA{R: 1, A: 0xff}),
		"graphics/infowindow/Orc3.bmp": synthBMP(2, 2, color.RGBA{R: 3, A: 0xff}),
		"graphics/infowindow/One.bmp":  synthBMP(2, 2, color.RGBA{R: 9, A: 0xff}),
	}
	orc := &terrain.UnitClass{Portrait: "Orc"}
	single := &terrain.UnitClass{Portrait: "One"}
	cache := map[string]*image.RGBA{}

	// Tier 1 drops the digit, which is why the shipped set has no `<name>1`.
	if pic := classPortrait(src, orc, 1, cache); pic == nil || pic.RGBAAt(0, 0).R != 1 {
		t.Errorf("tier 1 answered %v", pic)
	}
	if pic := classPortrait(src, orc, 3, cache); pic == nil || pic.RGBAAt(0, 0).R != 3 {
		t.Errorf("tier 3 answered %v — the digit did not reach the address", pic)
	}
	// A tier this class has no file for falls back to its first picture rather
	// than to nothing: three shipped classes declare a single palette.
	if pic := classPortrait(src, orc, 4, cache); pic == nil || pic.RGBAAt(0, 0).R != 1 {
		t.Errorf("an absent tier answered %v, want the class's first picture", pic)
	}
	if pic := classPortrait(src, single, 2, cache); pic == nil || pic.RGBAAt(0, 0).R != 9 {
		t.Errorf("a single-picture class at tier 2 answered %v", pic)
	}
	// Tier 0 — what a placement stating no tier carries — is the first picture
	// too, and it is normalised rather than falling back, so it costs ONE read.
	reads := 0
	counted := countingSource{missionSource: src, reads: &reads}
	if pic := classPortrait(counted, orc, 0, map[string]*image.RGBA{}); pic == nil || pic.RGBAAt(0, 0).R != 1 {
		t.Errorf("tier 0 answered %v", pic)
	}
	if reads != 1 {
		t.Errorf("tier 0 cost %d reads, want 1 — it fell back instead of normalising", reads)
	}
}

// UNIT-PICT-035 from the wiring tier's side: the two kinds of picture are
// EXCLUSIVE and the class decides which — so a composing class never reaches the
// portrait tree at all, which is the half that matters for cost.
func TestUnitPictureAsksTheClassBeforeTheArchive(t *testing.T) {
	reads := 0
	src := countingSource{
		missionSource: missionSource{
			"graphics/infowindow/Ogre.bmp": synthBMP(4, 4, color.RGBA{G: 0xff, A: 0xff}),
		},
		reads: &reads,
	}
	// BOTH classes name the same picture, which is what the shipped registries
	// do on the composing side: the value is there and can never be formatted.
	mw := &mapWorld{
		mission: &missionNotices{src: src},
		units: &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{
			4:  {Portrait: "Ogre"},
			70: {Portrait: "Ogre"},
		}},
		portraits: map[string]*image.RGBA{},
	}

	if pic := mw.unitPicture(1, 4); pic != nil {
		t.Error("a composing class answered a flat portrait")
	}
	if reads != 0 {
		t.Errorf("reads = %d — the portrait tree was opened for a class that composes", reads)
	}

	pic := mw.unitPicture(1, 70)
	if pic == nil {
		t.Fatal("a flat-portrait class answered no picture")
	}
	if b := pic.Bounds(); b.Dx() != 4 || b.Dy() != 4 {
		t.Errorf("picture is %v, want 4x4", b)
	}
	if reads != 1 {
		t.Errorf("reads = %d, want 1", reads)
	}

	// A class this bundle does not hold, and no bundle at all: both answer no
	// picture rather than reaching for one.
	if mw.unitPicture(1, 71) != nil {
		t.Error("a class the bundle does not hold answered a portrait")
	}
	bare := &mapWorld{portraits: map[string]*image.RGBA{}}
	if bare.unitPicture(1, 70) != nil {
		t.Error("a world with no bundle answered a portrait")
	}
}

// WHAT IS NOT COVERED HERE, named rather than left to be discovered: pushPortrait
// itself — that it asks the viewer for the selection on every call and that it
// pushes owner 0 rather than an id with a nil picture. Both are one statement
// each over a selection this package cannot set, and the decision they carry is
// the function above. The same boundary already applies to pushSpellbook.

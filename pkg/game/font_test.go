package game_test

import (
	"encoding/binary"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/game"
)

// fontFixture is a shipped font's shape at a scale a test can read: a blank
// record 0, then records whose ink and advance differ record by record, so a
// loader that mixed two of them up could not pass.
func fontFixture(n int) []synth.Font16Glyph {
	glyphs := make([]synth.Font16Glyph, n)
	for i := range glyphs {
		g := synth.Font16Glyph{Width: 8, Height: 6, Advance: uint32(i % 7)}
		if i > 0 {
			k := i
			g.Ink = func(x, y int) (uint8, bool) { return uint8(1 + (x+y+k)%15), (x+y+k)%3 != 0 }
		}
		glyphs[i] = g
	}
	return glyphs
}

// fontArchiveFiles keys the two nodes of base under the entries a container
// holds, derived from the loader's own addresses rather than spelled again.
func fontArchiveFiles(t *testing.T, base string, atlas, advances []byte) []synth.File {
	t.Helper()
	var files []synth.File
	if atlas != nil {
		files = append(files, synth.File{Path: graphicsEntry(t, game.FontAtlasPath(base)), Data: atlas})
	}
	if advances != nil {
		files = append(files, synth.File{Path: graphicsEntry(t, game.FontAdvancePath(base)), Data: advances})
	}
	return files
}

// AC-9: the two nodes become one font, glyph for glyph.
func TestLoadFont(t *testing.T) {
	glyphs := fontFixture(24)
	atlas, advances := synth.Font16(glyphs)
	src := openContainers(t, synth.Archive(fontArchiveFiles(t, game.DefaultFont, atlas, advances)))

	font, err := game.LoadFont(src, game.DefaultFont)
	if err != nil {
		t.Fatalf("LoadFont: %v", err)
	}
	if len(font.Glyphs) != len(glyphs) {
		t.Fatalf("loaded %d glyphs, want %d", len(font.Glyphs), len(glyphs))
	}
	if font.Spacing != game.FontSpacing {
		t.Fatalf("letter spacing %d, want %d", font.Spacing, game.FontSpacing)
	}
	for i, want := range glyphs {
		got := font.Glyphs[i]
		if got.Width != want.Width || got.Height != want.Height {
			t.Fatalf("glyph %d is %dx%d, want %dx%d", i, got.Width, got.Height, want.Width, want.Height)
		}
		if got.Advance != int(want.Advance) {
			t.Fatalf("glyph %d advance %d, want %d", i, got.Advance, want.Advance)
		}
		if len(got.Pixels) != want.Width*want.Height {
			t.Fatalf("glyph %d holds %d pixels, want %d", i, len(got.Pixels), want.Width*want.Height)
		}
		// The levels survive both decodes and the conversion, level 0 included:
		// a painted 0 must not arrive as a hole.
		for y := 0; y < want.Height; y++ {
			for x := 0; x < want.Width; x++ {
				var lv uint8
				var on bool
				if want.Ink != nil {
					lv, on = want.Ink(x, y)
				}
				p := got.Pixels[y*want.Width+x]
				if p.Painted != on || (on && p.Level != lv&0x0F) {
					t.Fatalf("glyph %d pixel (%d,%d) = %+v, want level %d painted %v",
						i, x, y, p, lv&0x0F, on)
				}
			}
		}
	}
	for _, p := range font.Glyphs[0].Pixels {
		if p.Painted {
			t.Fatal("record 0 of the fixture came back with ink")
		}
	}
}

// The two addresses are built in one place and each carries the archive identity
// segment, so a caller need not know which container a font lives in.
func TestFontPaths(t *testing.T) {
	if got := game.FontAtlasPath("font2"); got != "graphics/font2/font2.16" {
		t.Fatalf("FontAtlasPath = %q", got)
	}
	if got := game.FontAdvancePath("font2"); got != "graphics/font2/font2.dat" {
		t.Fatalf("FontAdvancePath = %q", got)
	}
	if game.FontAtlasPath(game.DefaultFont) == game.FontAdvancePath(game.DefaultFont) {
		t.Fatal("the two nodes of a font resolved to one address")
	}
}

// AC-10, AC-15: every way the pair can fail, each with its own message and no
// font at all.
func TestLoadFontFailures(t *testing.T) {
	atlas, advances := synth.Font16(fontFixture(8))
	short, _ := synth.Font16(fontFixture(4))
	// A zero-record atlas and the zero-entry sidecar that agrees with it. The
	// sidecar node must EXIST and be empty, or the refusal under test would be
	// the absent-node one instead.
	empty, _ := synth.Font16(nil)
	emptyAdv := []byte{}

	cases := []struct {
		name       string
		atlas, adv []byte
		want       string
		notExist   bool
	}{
		{name: "no atlas node", adv: advances, want: "font1.16", notExist: true},
		{name: "no sidecar node", atlas: atlas, want: "font1.dat", notExist: true},
		{name: "atlas will not decode", atlas: []byte{1, 2, 3, 4, 5, 6}, adv: advances, want: "font1.16"},
		{name: "sidecar will not decode", atlas: atlas, adv: []byte{1, 2, 3}, want: "font1.dat"},
		{name: "counts differ", atlas: short, adv: advances, want: "advances"},
		{name: "atlas holds no record", atlas: empty, adv: emptyAdv, want: "no glyph record"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := openContainers(t, synth.Archive(fontArchiveFiles(t, game.DefaultFont, tc.atlas, tc.adv)))
			font, err := game.LoadFont(src, game.DefaultFont)
			if err == nil {
				t.Fatalf("loaded %d glyphs, want an error", len(font.Glyphs))
			}
			if font != nil {
				t.Fatalf("a font came back beside the error: %v", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not name %q", err, tc.want)
			}
			if tc.notExist && !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("an absent node's error %q is not testable with errors.Is", err)
			}
		})
	}

	// The mismatch names BOTH counts, since neither node alone is at fault.
	src := openContainers(t, synth.Archive(fontArchiveFiles(t, game.DefaultFont, short, advances)))
	_, err := game.LoadFont(src, game.DefaultFont)
	if err == nil || !strings.Contains(err.Error(), "8 advances") || !strings.Contains(err.Error(), "4 glyph") {
		t.Fatalf("count mismatch reported as %v; want both counts", err)
	}

	if _, err := game.LoadFont(nil, game.DefaultFont); err == nil {
		t.Fatal("a nil source loaded a font")
	}
}

// The sidecar is one u32 per record in record order, which is the fact the
// loader's count check rests on.
func TestFontSidecarShape(t *testing.T) {
	glyphs := fontFixture(9)
	_, advances := synth.Font16(glyphs)
	if len(advances) != 4*len(glyphs) {
		t.Fatalf("sidecar is %d bytes for %d records, want %d", len(advances), len(glyphs), 4*len(glyphs))
	}
	for i, g := range glyphs {
		if got := binary.LittleEndian.Uint32(advances[i*4:]); got != g.Advance {
			t.Fatalf("sidecar entry %d = %d, want %d", i, got, g.Advance)
		}
	}
}

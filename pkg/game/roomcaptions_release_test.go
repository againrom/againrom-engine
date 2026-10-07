package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"testing"

	"againrom/pkg/formats/bmp"
	"againrom/pkg/formats/spr16"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// These readers deliberately use literal archive addresses and low-level
// decoders. Expected words, language selection, font and plaques never come
// from Words, LoadFont, SplitTextTable or the production room-art cache.
func roomCaptionRaw(t *testing.T, f *FrontEnd, path string) []byte {
	t.Helper()
	raw, err := f.Archives.Containers.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func roomCaptionRawLine(t *testing.T, raw []byte, index int) string {
	t.Helper()
	for line, start := 0, 0; start < len(raw); line++ {
		span := bytes.IndexByte(raw[start:], '\r')
		if span < 0 {
			break
		}
		end := start + span
		if line == index {
			if span == 0 {
				t.Fatalf("raw main.txt line %d is empty", index)
			}
			return string(raw[start:end])
		}
		start = end + 2
	}
	t.Fatalf("raw main.txt has no CR-terminated line %d", index)
	return ""
}

func roomCaptionRawFont(t *testing.T, f *FrontEnd) *text.Font {
	t.Helper()
	frames, err := spr16.DecodeG(roomCaptionRaw(t, f, "graphics/font1/font1.16"))
	if err != nil {
		t.Fatal(err)
	}
	advances, err := spr16.Advances(roomCaptionRaw(t, f, "graphics/font1/font1.dat"))
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) == 0 || len(advances) != len(frames) {
		t.Fatalf("raw font1: %d frames, %d advances", len(frames), len(advances))
	}
	id := roomCaptionRaw(t, f, "main/id")
	if len(id) == 0 || id[len(id)-1] != '0' && id[len(id)-1] != '1' {
		t.Fatalf("lawful install has an unexpected language selector: %q", id)
	}
	font := &text.Font{Spacing: 2, Selector: int(id[len(id)-1] - '0'), Glyphs: make([]text.Glyph, len(frames))}
	for i, frame := range frames {
		glyph := text.Glyph{Width: frame.Width, Height: frame.Height, Advance: advances[i],
			Pixels: make([]text.Pixel, len(frame.Pixels))}
		for j, pixel := range frame.Pixels {
			glyph.Pixels[j] = text.Pixel{Level: pixel.Value, Painted: pixel.Painted}
		}
		font.Glyphs[i] = glyph
	}
	return font
}

func roomCaptionRawBMP(t *testing.T, f *FrontEnd, path string) *image.RGBA {
	t.Helper()
	source, err := bmp.Decode(roomCaptionRaw(t, f, path))
	if err != nil {
		t.Fatal(err)
	}
	picture := image.NewRGBA(image.Rect(0, 0, source.Width, source.Height))
	for y := 0; y < source.Height; y++ {
		for x := 0; x < source.Width; x++ {
			p := source.At(x, y)
			picture.SetRGBA(x, y, color.RGBA{R: p.R, G: p.G, B: p.B, A: 255})
		}
	}
	return picture
}

// TestReleaseSchoolCaptionsMatchInstalledWordsAndPixels follows the real
// TownScreen entry and selection route. TOWN-383 supplies only the two literal
// text indices; the existing plaque layout/value policy is not ROM1-certified
// by this test. Composition is CPU-side, not a physical original-game witness.
func TestReleaseSchoolCaptionsMatchInstalledWordsAndPixels(t *testing.T) {
	f := releaseFront(t)
	rawText := roomCaptionRaw(t, f, "main/text/main.txt")
	captions := [2]string{roomCaptionRawLine(t, rawText, 231), roomCaptionRawLine(t, rawText, 232)}
	if captions[0] == captions[1] {
		t.Fatal("the two installed school captions must be distinct")
	}
	font := roomCaptionRawFont(t, f)
	wells := [2]image.Rectangle{image.Rect(484, 71, 624, 117), image.Rect(484, 117, 624, 163)}
	var plaques [2][2]*image.RGBA
	for button := range plaques {
		for state, suffix := range []string{"off", "on"} {
			path := fmt.Sprintf("graphics/interface/training/buttons/b%d%s.bmp", button+1, suffix)
			plaques[button][state] = roomCaptionRawBMP(t, f, path)
			if plaques[button][state].Bounds().Size() != image.Pt(140, 46) {
				t.Fatalf("%s: raw plaque size %v, want 140x46", path, plaques[button][state].Bounds().Size())
			}
		}
	}

	f.Carried = f.NextParty()
	f.Town.gold = 100000
	s := f.TownScreen().(*townScreen)
	s.Choose(2)
	for i := 0; s.room == roomTalk && i < 64; i++ {
		s.AdvanceTownDialogue()
	}
	if s.room != roomSchool || len(f.Carried) == 0 {
		t.Fatal("production school entry has no training subject")
	}
	s.CloseTip()
	// Use the second untrained skill, not the hero's trained starting skill.
	// Level zero costs exactly 200 under the existing school-price policy.
	if f.Carried[0].Hero.Skill[2] != 0 {
		t.Fatal("the installed starting hero has no level-zero second-skill subject")
	}
	cell := 1
	if f.Carried[0].Mage {
		cell += 5
	}

	for phase := 0; phase < 2; phase++ {
		if phase == 1 {
			s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: cell}, false)
		}
		v := s.TownSurface()
		expected := [2]ui.TownSurfaceButton{
			{Label: captions[0], Value: "0", Enabled: false},
			{Label: captions[1], Value: "100,000", Enabled: true},
		}
		if phase == 1 {
			expected[0].Value, expected[0].Enabled = "200", true
		}
		if len(v.Buttons) != 2 || v.Buttons[0] != expected[0] || v.Buttons[1] != expected[1] {
			t.Fatalf("phase %d: production buttons %+v, want independently sourced %+v", phase, v.Buttons, expected)
		}
		for pressed := -1; pressed < 2; pressed++ {
			actual := v
			if pressed >= 0 {
				actual.Press = ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: pressed}
			}
			got := ui.ComposeTownSurface(actual)
			var wants [2]*image.RGBA
			for button, well := range wells {
				state := 0
				if pressed == button {
					state = 1
				}
				wants[button] = oracleTownButtonWell(nil, image.Point{}, plaques[button][state], well, font, expected[button], pressed == button)
				if n, points := diffWells(got, wants[button], well); n != 0 {
					t.Fatalf("phase %d press %d button %d: %d pixel differences, first %v", phase, pressed, button, n, points)
				}
			}

			// Change only in-memory views, never production data or source.
			// These controls prove the whole-well oracle rejects the named bugs.
			for _, defect := range []string{"english", "swapped", "blank-train", "blank-exit", "wrong-selector"} {
				if defect == "wrong-selector" && font.Selector != 1 {
					continue // Only the Russian install converts these glyph bytes.
				}
				bad := actual
				bad.Buttons = append([]ui.TownSurfaceButton(nil), actual.Buttons...)
				switch defect {
				case "english":
					bad.Buttons[0].Label, bad.Buttons[1].Label = "Train", "EXIT"
				case "swapped":
					bad.Buttons[0].Label, bad.Buttons[1].Label = captions[1], captions[0]
				case "blank-train":
					bad.Buttons[0].Label = ""
				case "blank-exit":
					bad.Buttons[1].Label = ""
				case "wrong-selector":
					wrongFont := *actual.Font
					wrongFont.Selector = 0
					bad.Font = &wrongFont
				}
				badPixels := ui.ComposeTownSurface(bad)
				differences := 0
				for button, well := range wells {
					n, _ := diffWells(badPixels, wants[button], well)
					differences += n
				}
				if differences == 0 {
					t.Fatalf("phase %d press %d: oracle failed to reject %s", phase, pressed, defect)
				}
			}
		}
	}
	if f.Town.Gold() != 100000 || f.Carried[0].Hero.Skill[2] != 0 || s.schoolCell != cell {
		t.Fatal("caption composition changed purse, skill or school selection")
	}
	t.Logf("selector %d: idle/trainable school, both released/pressed plaques, 12 full-well comparisons; caption and language negative controls rejected", font.Selector)
}

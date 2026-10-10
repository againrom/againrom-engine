package game

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
)

type chargenSource map[string][]byte

func (s chargenSource) ReadFile(name string) ([]byte, error) {
	b, ok := s[name]
	if !ok {
		return nil, errors.New(name + ": missing")
	}
	return b, nil
}

func TestChargenTextRowsRetainRawBytes(t *testing.T) {
	rows := SplitTextTable([]byte("one\r\ntwo\r\n"))
	if got, ok := rows.At(1); !ok || got != "two" {
		t.Fatalf("line 1 = %q, %v", got, ok)
	}
	if _, err := chargenTextAt(rows, chargenTextPath, 3); err == nil {
		t.Fatal("missing slot accepted")
	}
}

func TestLoadChargenAssets(t *testing.T) {
	src := chargenSource{}
	for level, size := range [3]image.Point{{60, 74}, {76, 112}, {100, 152}} {
		for _, suffix := range []string{"on", "l", "lon"} {
			src[fmt.Sprintf("%slevels/level%d%s.bmp", chargenPrecreatePath, level, suffix)] = synthBMP(size.X, size.Y, color.RGBA{R: 0x44, A: 0xff})
		}
	}
	putBMP := func(addr string) { src[addr] = synthBMP(2, 2, color.RGBA{R: 0x44, A: 0xff}) }
	src[chargenPrecreatePath+"mainarea.bmp"] = synthBMP(640, 480, color.RGBA{R: 0x44, A: 0xff})
	src[chargenPrecreatePath+"mask.bmp"] = synthBMP8(640, 480, chargenPreMaskCodes[:]...)
	src[chargenPrecreatePath+"amulet.bmp"] = synthBMP(112, 204, color.RGBA{R: 0x44, A: 0xff})
	src[chargenPrecreatePath+"buttonok.bmp"] = synthBMP(100, 56, color.RGBA{R: 0x44, A: 0xff})
	src[chargenPlatePath] = synthBMP(160, 238, color.RGBA{R: 0x44, A: 0xff})
	src[graphicsPrefix+"interface/inn/buttonsarea.bmp"] = synthBMP(160, 238, color.RGBA{G: 0x44, A: 0xff})
	for _, name := range []string{"button1off", "button1on", "button2off", "button2on", "button3off", "button3on"} {
		src[graphicsPrefix+"interface/inn/"+name+".bmp"] = synthBMP(140, 46, color.RGBA{R: 0x33, A: 0xff})
	}
	src[graphicsPrefix+"interface/inn/ruover.bmp"] = synthBMP(16, 238, color.RGBA{B: 0x44, A: 0xff})
	src[graphicsPrefix+"interface/chrgen/rollstatsr.bmp"] = synthBMP(16, 238, color.RGBA{B: 0x44, A: 0xff})
	src[graphicsPrefix+"interface/humanbackr.bmp"] = synthBMP(160, 242, color.RGBA{B: 0x44, A: 0xff})
	src[graphicsPrefix+"interface/humanbackl.bmp"] = synthBMP(16, 242, color.RGBA{B: 0x44, A: 0xff})
	src[graphicsPrefix+"interface/chrgen/fullstatsl.bmp"] = synthBMP(160, 242, color.RGBA{G: 0x44, A: 0xff})
	src[graphicsPrefix+"interface/chrgen/fullstatsr.bmp"] = synthBMP(16, 242, color.RGBA{G: 0x44, A: 0xff})
	for _, hero := range []string{"mf", "mm", "ff", "fm"} {
		for _, state := range []string{"on", "l", "lon"} {
			putBMP(chargenPrecreatePath + "heroes/" + hero + state + ".bmp")
		}
	}
	for class, classDir := range []string{"fighter", "mag"} {
		src[graphicsPrefix+"interface/chrgen/"+classDir+"/column.bmp"] = synthBMP(320, 480, color.RGBA{R: 0x44, A: 0xff})
		src[graphicsPrefix+"interface/chrgen/"+classDir+"/mask.bmp"] = synthBMP8(320, 480, chargenDetailedMaskCodes[class][:]...)
	}
	for class, skills := range [][]string{{"sword", "axe", "mace", "pike", "bow"}, {"fire", "water", "air", "earth", "astral"}} {
		classDir := []string{"fighter", "mag"}[class]
		for _, skill := range skills {
			for _, state := range []string{"on", "shine_off", "shine_on"} {
				putBMP(graphicsPrefix + "interface/chrgen/" + classDir + "/" + skill + "/" + state + ".bmp")
			}
		}
	}
	for _, name := range []string{"mnloff", "mloff", "mlon", "mnlon", "mdisable", "pnloff", "ploff", "plon", "pnlon", "pdisable"} {
		src[graphicsPrefix+"interface/chrgen/buttons/"+name+".bmp"] = synthBMP(20, 20, color.RGBA{R: 0x44, A: 0xff})
	}
	glyphs := []synth.Font16Glyph{{Width: 8, Height: 10, Advance: 8}}
	atlas, advances := synth.Font16(glyphs)
	src[FontAtlasPath("font2")], src[FontAdvancePath("font2")] = atlas, advances
	src[FontAtlasPathA(DocumentFont)], src[FontAdvancePath(DocumentFont)] = synthFont16A(), advances
	names := make([][]byte, 24)
	for i := range names {
		names[i] = []byte(fmt.Sprintf("npc%d", i))
	}
	src[chargenNamesPath] = stringJoinBytes(names)
	rows := make([][]byte, chargenBackSlot+1)
	for i := range rows {
		rows[i] = []byte("x")
	}
	rows[chargenPromptSlot] = []byte("Character name:")
	rows[chargenPlaySlot], rows[chargenResetSlot], rows[chargenBackSlot] = []byte("Accept"), []byte("Reset"), []byte("Back")
	src[chargenTextPath] = stringJoinBytes(rows)
	src[LanguagePath] = []byte("english 0")

	got, err := LoadChargenAssets(src)
	if err != nil {
		t.Fatalf("LoadChargenAssets: %v", err)
	}
	if got.Presentation == nil || got.Presentation.Choices[3][2] == nil || got.Presentation.PreMask == nil || got.Presentation.Amulet == nil {
		t.Fatal("source choice states were not retained")
	}
	if got.Prompt != "Character name:" || got.Play != "Accept" || got.Reset != "Reset" || got.Back != "Back" {
		t.Fatalf("raw source words = %q / %q / %q / %q", got.Prompt, got.Play, got.Reset, got.Back)
	}
	if got.Presentation.Skills[1][4][2] == nil || got.Presentation.Columns[1] == nil || got.Presentation.ColumnMask[1] == nil || got.Presentation.StatButtons[1][4] == nil || got.SkillHover[1][4] != "x" {
		t.Fatal("detailed skill presentation was not retained")
	}
	// Picture order is male fighter, male mage, female fighter, female mage.
	if want := [4]string{"npc20", "npc22", "npc21", "npc23"}; got.HeroNames != want {
		t.Fatalf("HeroNames = %q, want %q", got.HeroNames, want)
	}
	if got.Presentation.NameFont == nil || len(got.Presentation.NameFont.Glyphs) != 1 {
		t.Fatal("the prompt and name font was not retained")
	}
	src[chargenNamesPath] = stringJoinBytes(names[:23])
	if _, err := LoadChargenAssets(src); err == nil || !strings.Contains(err.Error(), chargenNamesPath+" slot 23") {
		t.Fatalf("missing name entry error = %v, want %s slot 23", err, chargenNamesPath)
	}
	src[chargenNamesPath] = stringJoinBytes(names)
	delete(src, FontAtlasPathA(DocumentFont))
	if _, err := LoadChargenAssets(src); err == nil {
		t.Fatal("missing prompt and name font was accepted")
	}
	src[FontAtlasPathA(DocumentFont)] = synthFont16A()
	src[chargenPrecreatePath+"mask.bmp"] = synthBMP8(640, 480)
	if _, err := LoadChargenAssets(src); err == nil || !strings.Contains(err.Error(), "missing required mask index 20") {
		t.Fatalf("blank pre-create mask error = %v, want missing required mask index", err)
	}
	src[chargenPrecreatePath+"mask.bmp"] = synthBMP8(640, 480, chargenPreMaskCodes[:]...)
	delete(src, chargenPrecreatePath+"heroes/mfon.bmp")
	if _, err := LoadChargenAssets(src); err == nil {
		t.Fatal("missing required picture was accepted")
	}
}

func synthBMP8(w, h int, hot ...uint8) []byte {
	const header, palette = 54, 256 * 4
	stride := (w + 3) &^ 3
	b := make([]byte, header+palette+stride*h)
	b[0], b[1] = 'B', 'M'
	binary.LittleEndian.PutUint32(b[2:], uint32(len(b)))
	binary.LittleEndian.PutUint32(b[10:], header+palette)
	binary.LittleEndian.PutUint32(b[14:], 40)
	binary.LittleEndian.PutUint32(b[18:], uint32(w))
	binary.LittleEndian.PutUint32(b[22:], uint32(h))
	binary.LittleEndian.PutUint16(b[26:], 1)
	binary.LittleEndian.PutUint16(b[28:], 8)
	binary.LittleEndian.PutUint32(b[46:], 256)
	for i := 0; i < 256; i++ {
		b[header+i*4], b[header+i*4+1], b[header+i*4+2] = byte(i), byte(i), byte(i)
	}
	for i, index := range hot {
		b[header+palette+i] = index
	}
	return b
}

// synthFont16A is a one-record `.16a` atlas: the declared 1024-byte palette,
// one 1x1 frame whose single literal pixel is level 15 at palette index 255,
// and the record-count trailer.
func synthFont16A() []byte {
	out := make([]byte, 1024, 1024+16+4)
	out = binary.LittleEndian.AppendUint32(out, 1) // width
	out = binary.LittleEndian.AppendUint32(out, 1) // height
	out = binary.LittleEndian.AppendUint32(out, 4) // block bytes
	out = binary.LittleEndian.AppendUint16(out, 1) // a literal run of one pixel
	out = binary.LittleEndian.AppendUint16(out, 15<<9|255<<1)
	return binary.LittleEndian.AppendUint32(out, 1)
}

func TestAlphaAtlasSmoothingBlendsInkWithBackground(t *testing.T) {
	ink := color.RGBA{64, 48, 20, 255}
	back := color.RGBA{200, 180, 140, 255}
	for _, alphaAtlas := range []bool{false, true} {
		t.Run(fmt.Sprintf("alpha-atlas=%v", alphaAtlas), func(t *testing.T) {
			var font *text.Font
			var err error
			if alphaAtlas {
				atlas := synthFont16A()
				binary.LittleEndian.PutUint16(atlas[1038:], 3<<9|255<<1)
				font, err = LoadFontA(chargenSource{
					FontAtlasPathA(DocumentFont):  atlas,
					FontAdvancePath(DocumentFont): binary.LittleEndian.AppendUint32(nil, 1),
				}, DocumentFont)
			} else {
				atlas, advances := synth.Font16([]synth.Font16Glyph{{Width: 1, Height: 1, Advance: 1,
					Ink: func(_, _ int) (uint8, bool) { return 3, true },
				}})
				font, err = LoadFont(chargenSource{
					FontAtlasPath(DefaultFont):   atlas,
					FontAdvancePath(DefaultFont): advances,
				}, DefaultFont)
			}
			if err != nil {
				t.Fatal(err)
			}
			dst := image.NewRGBA(image.Rect(0, 0, 3, 3))
			for y := 0; y < 3; y++ {
				for x := 0; x < 3; x++ {
					dst.SetRGBA(x, y, back)
				}
			}
			texts := []text.DrawCall{{Glyph: &font.Glyphs[0], Color: ink}}
			textsmooth.Composite(dst, texts, 3, 0, 0)
			want := color.RGBA{12, 9, 4, 255}
			if alphaAtlas {
				want = color.RGBA{166, 147, 110, 255}
			}
			for y := 0; y < 3; y++ {
				for x := 0; x < 3; x++ {
					if got := dst.RGBAAt(x, y); got != want {
						t.Fatalf("pixel (%d,%d) = %v, want %v", x, y, got, want)
					}
				}
			}
			texts[0].Flat = true
			dst = image.NewRGBA(dst.Rect)
			textsmooth.Composite(dst, texts, 3, 0, 0)
			if got := dst.RGBAAt(1, 1); got != ink {
				t.Fatalf("flat ink = %v, want %v", got, ink)
			}
		})
	}
}

// stringJoinBytes builds a synthetic install text file: every row terminated by
// CRLF, which is the shipped form and what the decoded loader walks
// (TEXT-STRTAB-023). Before 0168 this joined rows with a single NUL, which only
// the splitter this package used to carry could read.
func stringJoinBytes(rows [][]byte) []byte {
	out := make([]byte, 0, len(rows)*3)
	for _, row := range rows {
		out = append(out, row...)
		out = append(out, '\r', '\n')
	}
	return out
}

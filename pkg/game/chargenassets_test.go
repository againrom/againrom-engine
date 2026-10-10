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
	"againrom/pkg/ui"
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
	if _, err := chargenTextAt(rows, "main/text/main.txt", 3); err == nil {
		t.Fatal("missing slot accepted")
	}
}

// synthGeneratorSource is an archive holding every picture, mask, font and
// text line description l names, at the sizes it states.
func synthGeneratorSource(l *ui.GeneratorDescription) chargenSource {
	src := chargenSource{}
	fill := color.RGBA{R: 0x44, A: 0xff}
	put := func(key string, size *ui.GeneratorPoint) {
		w, h := 2, 2
		if size != nil {
			w, h = size[0], size[1]
		}
		src[key] = synthBMP(w, h, fill)
	}
	pre, d := &l.PreCreate, &l.Detail
	for _, h := range pre.Heroes {
		for _, a := range h.Art {
			put(a.Key, a.Size)
		}
	}
	for _, v := range pre.Levels {
		for _, a := range v.Art {
			put(a.Key, a.Size)
		}
	}
	put(pre.Background.Key, pre.Background.Size)
	var preCodes []uint8
	for _, c := range pre.Mask.Required {
		preCodes = append(preCodes, uint8(c))
	}
	src[pre.Mask.Key] = synthBMP8(pre.Mask.Size[0], pre.Mask.Size[1], preCodes...)
	for _, b := range []ui.GeneratorArt{pre.Back.Art, pre.Forward.Art} {
		if b.Key != "" {
			put(b.Key, b.Size)
		}
	}
	for _, loop := range pre.Loops {
		for k := 0; k < loop.Count; k++ {
			put(fmt.Sprintf(loop.Key, loop.First+k), &loop.Size)
		}
	}
	for _, pane := range []*ui.GeneratorPane{&d.Plate, d.PlateSeam, &d.Nav, d.NavSeam, &d.Doll, d.DollSeam, &d.Card, d.CardSeam} {
		if pane != nil && pane.Key != "" {
			put(pane.Key, &pane.Size)
		}
	}
	for _, c := range d.Commands {
		put(c.Off, &c.Size)
		put(c.On, &c.Size)
	}
	for _, c := range d.Classes {
		put(c.Column.Key, c.Column.Size)
		var codes []uint8
		for _, skill := range c.Skills {
			codes = append(codes, uint8(skill.Mask))
			for _, name := range []string{d.SkillStates.SelectedAtRest, d.SkillStates.Hover, d.SkillStates.Selected} {
				put(skill.Dir+name, nil)
			}
		}
		size := c.Column.Size
		if c.Mask.Size != nil {
			size = c.Mask.Size
		}
		src[c.Mask.Key] = synthBMP8(size[0], size[1], codes...)
	}
	st := &d.Stats
	for _, b := range []ui.GeneratorStatButtons{st.MinusArt, st.PlusArt} {
		for _, name := range []string{b.Rest, b.Hover, b.Down, b.Unused, b.Disabled} {
			put(st.ButtonDir+name, &st.ButtonSize)
		}
	}
	glyphs := []synth.Font16Glyph{{Width: 8, Height: 10, Advance: 8}}
	atlas, advances := synth.Font16(glyphs)
	for _, f := range []ui.GeneratorFont{l.Words.TextFont, l.Words.NameFont} {
		if f.Atlas == "16a" {
			src[FontAtlasPathA(f.Name)], src[FontAdvancePath(f.Name)] = synthFont16A(), advances
		} else {
			src[FontAtlasPath(f.Name)], src[FontAdvancePath(f.Name)] = atlas, advances
		}
	}
	names := make([][]byte, 40)
	for i := range names {
		names[i] = []byte(fmt.Sprintf("npc%d", i))
	}
	src[l.Words.Names] = stringJoinBytes(names)
	rows := make([][]byte, 400)
	for i := range rows {
		rows[i] = []byte("x")
	}
	rows[pre.Name.Prompt.Slot] = []byte("Character name:")
	for i, word := range []string{"Accept", "Reset", "Back"} {
		rows[d.Commands[i].Slot] = []byte(word)
	}
	src[l.Words.Table] = stringJoinBytes(rows)
	src[LanguagePath] = []byte("english 0")
	return src
}

func TestLoadChargenAssets(t *testing.T) {
	l := generatorDescriptions["rom1"]
	src := synthGeneratorSource(l)
	got, err := LoadChargenAssets(src, l)
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
	if want := [4]string{"npc20", "npc22", "npc21", "npc23"}; got.HeroNames != want || got.EnterName != "npc20" {
		t.Fatalf("HeroNames = %q, enter %q; want %q, npc20", got.HeroNames, got.EnterName, want)
	}
	if got.Presentation.NameFont == nil || len(got.Presentation.NameFont.Glyphs) != 1 {
		t.Fatal("the prompt and name font was not retained")
	}
	names := l.Words.Names
	full := src[names]
	src[names] = stringJoinBytes(bytesRows(23))
	if _, err := LoadChargenAssets(src, l); err == nil || !strings.Contains(err.Error(), names+" slot 2") {
		t.Fatalf("missing name entry error = %v, want %s slot 2x", err, names)
	}
	src[names] = full
	nameFont := FontAtlasPathA(l.Words.NameFont.Name)
	atlas := src[nameFont]
	delete(src, nameFont)
	if _, err := LoadChargenAssets(src, l); err == nil {
		t.Fatal("missing prompt and name font was accepted")
	}
	src[nameFont] = atlas
	mask := src[l.PreCreate.Mask.Key]
	src[l.PreCreate.Mask.Key] = synthBMP8(640, 480)
	if _, err := LoadChargenAssets(src, l); err == nil || !strings.Contains(err.Error(), "missing required mask index 20") {
		t.Fatalf("blank pre-create mask error = %v, want missing required mask index", err)
	}
	src[l.PreCreate.Mask.Key] = mask
	delete(src, l.PreCreate.Heroes[0].Art[0].Key)
	if _, err := LoadChargenAssets(src, l); err == nil {
		t.Fatal("missing required picture was accepted")
	}
}

func bytesRows(n int) [][]byte {
	rows := make([][]byte, n)
	for i := range rows {
		rows[i] = []byte(fmt.Sprintf("npc%d", i))
	}
	return rows
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

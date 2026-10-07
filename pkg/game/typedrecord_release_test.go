package game

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/formats/textinput"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// typedRecord is the TEXT-108 and TEXT-110 rule, not the build converters.
func typedRecord(b int, selector int) int {
	switch {
	case selector != 1:
		return b - 0x20
	case b >= 0xc0 && b <= 0xef:
		return b - 0x30
	default:
		return b - 0x20
	}
}

func typedName(f *FrontEnd, r rune) string {
	c := ui.NewChargen(f.ChargenSetup())
	for c.NameText() != "" {
		c.EditName("", true)
	}
	c.EditName(string(r), false)
	return c.NameText()
}

func glyphsEqual(a, b *text.Glyph) bool {
	return a.Width == b.Width && a.Height == b.Height && reflect.DeepEqual(a.Pixels, b.Pixels)
}

// Each typed letter reaches the claimed record in three fonts.
func TestReleaseTypedNameReachesTheRecordTheClaimNames(t *testing.T) {
	f := releaseFront(t)
	if f.ChargenAssets == nil || f.ChargenAssets.Presentation == nil {
		t.Skip("no generator assets")
	}
	selector := f.textSelector()
	pres := f.ChargenAssets.Presentation
	fonts := map[string]*text.Font{"name font": pres.NameFont, "generator font": pres.Font, "default font": f.Font.Value()}
	for label, font := range fonts {
		if font == nil || font.Selector != selector || len(font.Glyphs) < 224 {
			t.Fatalf("%s: selector or record count unusable", label)
		}
	}
	for b := 0xc0; b <= 0xff; b++ {
		r := typedRune(t, byte(b))
		name := typedName(f, r)
		if len(name) != 1 {
			t.Fatalf("typed %#02x: name field holds %d bytes", b, len(name))
		}
		want, ok := textinput.EncodeRune(r, selector)
		if !ok || name[0] != want {
			t.Fatalf("typed %#02x: field stored %#02x, encoder %#02x", b, name[0], want)
		}
		rec := typedRecord(b, selector)
		for label, font := range fonts {
			if got := font.GlyphFor(name[0]); got != &font.Glyphs[rec] {
				t.Fatalf("%s: typed %#02x reaches a record other than %d", label, b, rec)
			}
		}
	}
}

// TEXT-109 and TEXT-110: 17 of 18 homoglyph pairs match on RU, 4 on EN.
func TestReleaseTypedHomoglyphsMatchTheClaimCounts(t *testing.T) {
	f := releaseFront(t)
	if f.ChargenAssets == nil || f.ChargenAssets.Presentation == nil {
		t.Skip("no generator assets")
	}
	selector := f.textSelector()
	pairs := []struct {
		latin rune
		typed byte
	}{
		{'A', 0xc0}, {'B', 0xc2}, {'E', 0xc5}, {'K', 0xca}, {'M', 0xcc}, {'H', 0xcd}, {'O', 0xce}, {'P', 0xd0},
		{'C', 0xd1}, {'T', 0xd2}, {'X', 0xd5}, {'a', 0xe0}, {'c', 0xf1}, {'e', 0xe5}, {'o', 0xee}, {'p', 0xf0},
		{'x', 0xf5}, {'y', 0xf3},
	}
	wantRU, wantEN := 17, 4
	for label, font := range map[string]*text.Font{"font2": f.ChargenAssets.Presentation.Font, "font1": f.Font.Value()} {
		same := 0
		for _, p := range pairs {
			name := typedName(f, typedRune(t, p.typed))
			if len(name) != 1 {
				t.Fatalf("%s: typed %#02x refused", label, p.typed)
			}
			if glyphsEqual(font.GlyphFor(name[0]), font.GlyphFor(byte(p.latin))) {
				same++
			}
		}
		want := wantEN
		if selector == 1 {
			want = wantRU
		}
		if same != want {
			t.Errorf("%s at selector %d: %d of 18 pairs pixel-identical, want %d", label, selector, same, want)
		}
	}
}

// A shipped string byte keeps its display-conversion record.
func TestReleaseShippedStringByteKeepsItsRecord(t *testing.T) {
	f := releaseFront(t)
	font := f.Font.Value()
	if font == nil || len(font.Glyphs) < 224 {
		t.Fatal("no default font")
	}
	for b := 0x20; b <= 0xff; b++ {
		want := b - 0x20
		if font.Selector == 1 {
			switch {
			case b >= 0x80 && b <= 0xaf:
				want = b + 0x10
			case b >= 0xe0 && b <= 0xef:
				want = b - 0x10
			}
		}
		if font.GlyphFor(byte(b)) != &font.Glyphs[want] {
			t.Fatalf("shipped byte %#02x: not record %d at selector %d", b, want, font.Selector)
		}
	}
}

// A SAV label of typed letters keeps the rule's bytes.
func TestReleaseSavedTypedLabelKeepsItsBytes(t *testing.T) {
	f := releaseFront(t)
	selector := f.textSelector()
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Town traveler", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 30, 30}})
	f.arriveInTown()
	f.addChapterCompanions(f.Town.Chapter())
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	if _, err := save(false); err != nil {
		t.Fatal(err)
	}
	app := f.App("typed label")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("@first"); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("LOAD to town", err)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	for _, block := range [][2]int{{0xc0, 0xdf}, {0xe0, 0xff}} {
		var label string
		var wantBytes []byte
		for b := block[0]; b <= block[1]; b++ {
			r := typedRune(t, byte(b))
			label += string(r)
			stored, ok := textinput.EncodeRune(r, selector)
			if !ok {
				t.Fatalf("typed %#02x refused", b)
			}
			wantBytes = append(wantBytes, stored)
		}
		dir := filepath.Join(store.Dir, fmt.Sprintf("Typed %d", block[0]))
		if err := app.HeadlessGameMenuAction("save"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessSaveEdit(dir, label, ui.SaveSAV); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessSaveAction("save"); err != nil {
			t.Fatal("save with the typed letters", err)
		}
		raw, err := ReadSaveFile(filepath.Join(dir, label+".sav"))
		if err != nil {
			t.Fatal(err)
		}
		sf, err := sav.Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		if string(sf.Label) != string(wantBytes) {
			t.Fatalf("label bytes % x, want % x", sf.Label, wantBytes)
		}
		want := label + " - between missions"
		got, err := OriginalSaveLabel(raw, selector)
		if err != nil || got != want {
			t.Fatalf("reloaded label %q, %v; want %q", got, err, want)
		}
	}
}

package game

import (
	"bytes"
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"

	"golang.org/x/text/encoding/charmap"
)

// rawTextLine is line index of an installed text file as stored, empty lines
// included.
func rawTextLine(t *testing.T, f *FrontEnd, path string, index int) string {
	t.Helper()
	raw := roomCaptionRaw(t, f, path)
	lines := strings.Split(string(raw), "\r\n")
	if index >= len(lines) {
		t.Fatalf("%s has %d lines, want line %d", path, len(lines), index)
	}
	return lines[index]
}

// The name line of a book is the item's installed name followed by the item
// formatter's fragment: main.txt line 90, the item spell caption, and main.txt
// line 91. The Russian Drain Life item caption has an owner-directed alias.
func TestReleaseBookCardIsBuiltFromTheInstalledFormatterWords(t *testing.T) {
	f := releaseFront(t)
	of := rawTextLine(t, f, "main/text/main.txt", 90)
	suffix := rawTextLine(t, f, "main/text/main.txt", 91)
	if of == "" {
		t.Fatal("main.txt line 90 is empty")
	}
	rows := strings.Split(strings.TrimRight(string(roomCaptionRaw(t, f, "main/text/spell.txt")), "\r\n"), "\r\n")
	if len(rows) != 28 {
		t.Fatalf("spell.txt has %d lines, want 28", len(rows))
	}
	for id := uint32(1); id <= 28; id++ {
		book := gameSpellBook(id)
		lines := itemInstanceInfoLines(book, f.Table, f.Words)
		name := itemName(data.ItemCode(0x0e17), f.Table)
		spellName := rows[id-1]
		if id == 11 && f.textSelector() == 1 {
			rawName, err := charmap.CodePage866.NewEncoder().String("Высасывание жизни")
			if err != nil || rows[id-1] != rawName {
				t.Fatalf("raw Russian Drain Life name changed: %x/%v", rows[id-1], err)
			}
			spellName, err = charmap.CodePage866.NewEncoder().String("Вампиризм")
			if err != nil {
				t.Fatal(err)
			}
		}
		want := name + " " + of + " " + spellName + suffix
		if len(lines) != 1 || lines[0] != want {
			t.Fatalf("book of spell %d: %q, want the single line %q", id, lines, want)
		}
	}
	if of != f.Words.ItemSpellOf || suffix != f.Words.ItemSpellOfSuffix {
		t.Fatalf("words hold %q/%q, the install states %q/%q", f.Words.ItemSpellOf, f.Words.ItemSpellOfSuffix, of, suffix)
	}
	// A class-0xe00 item with a cast-spell effect, a scroll, takes the same fragment.
	scroll := sim.ItemInstance{Code: 0x0e0e, Kind: 4, Effects: []sim.ItemEffect{{Kind: 41, Operand: 12 | 3<<16}}}
	if got, want := itemInstanceInfoLines(scroll, f.Table, f.Words), []string{itemName(data.ItemCode(0x0e0e), f.Table) + " " + of + " " + rows[11] + suffix}; !slices.Equal(got, want) {
		t.Fatalf("scroll card %q, want %q", got, want)
	}
	// A non-book item carrying a teach-spell effect takes the heading, with the
	// fragment continuing it, and a book takes no heading.
	staff := sim.ItemInstance{Code: 0x812d, Effects: []sim.ItemEffect{{Kind: 42, Operand: 2}}}
	got := itemInstanceInfoLines(staff, f.Table, f.Words)
	if len(got) < 2 || !strings.HasPrefix(got[len(got)-1], f.Words.ItemMagic) || !strings.HasSuffix(got[len(got)-1], of+" "+rows[1]+suffix) {
		t.Fatalf("non-book teach-spell item: %q", got)
	}
	if lines := itemInstanceInfoLines(gameSpellBook(2), f.Table, f.Words); linesContain(lines, f.Words.ItemMagic) {
		t.Fatalf("book carries the %q heading: %q", f.Words.ItemMagic, lines)
	}
}

func linesContain(lines []string, s string) bool {
	for _, l := range lines {
		if l == s {
			return true
		}
	}
	return false
}

// A generated book on the shop's fourth shelf, hovered through App input, shows
// that name line in the popup. With AGAINROM_TOOLTIP_ARTIFACTS set the popup
// image is written as a PNG.
func TestReleaseShopBookShelfPopupShowsTheBookCard(t *testing.T) {
	app, s := releaseShopApp(t)
	f := frontOf(s)
	s.CloseTip()
	app.SetTooltipDelayPreference(0, nil)
	click := func(kind string, index int) {
		t.Helper()
		x, y, err := app.HeadlessShopPoint(kind, index)
		if err != nil {
			t.Fatal(err)
		}
		for _, edge := range []string{"press", "release"} {
			if err := app.HeadlessPointer(edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
	}
	// The default stock has no readable book unless the party holds a mage, so
	// the shelf is given two books and a scroll (class 0xe00 with a cast-spell
	// effect, which the formatter writes the same way).
	scroll := sim.ItemInstance{Code: 0x0e0e, Kind: 4, Effects: []sim.ItemEffect{{Kind: 41, Operand: 12 | 3<<16}}}
	f.Shop.shelves[ShelfBooks] = []ShopItem{shopItemFromInstance(gameSpellBook(2), 1), shopItemFromInstance(gameSpellBook(26), 1),
		shopItemFromInstance(scroll, 1)}
	click("shelf_pick", 3)
	view := s.ShopScreen()
	items := f.Shop.Shelf(ShelfBooks)
	of := rawTextLine(t, f, "main/text/main.txt", 90)
	shown := 0
	for c := range view.Shelf {
		k := s.shelfBase + c
		if k >= len(items) || !view.Shelf[c].Occupied() {
			continue
		}
		spell, ok := items[k].Instance().BookSpell()
		if !ok {
			continue
		}
		shown++
		if len(view.Shelf[c].Info) != 1 || !strings.Contains(view.Shelf[c].Info[0], " "+of+" "+f.Words.ItemSpellNames[spell]) {
			t.Fatalf("book cell info %q does not read name, %q, spell", view.Shelf[c].Info, of)
		}
		x, y, err := app.HeadlessShopPoint("shelf", c)
		if err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		state, pic := app.HeadlessTooltip()
		drawn, _, ok := ui.ComposeTooltipHint(view.Shelf[c].Info, f.tipFont(), image.Pt(x, y), image.Rect(0, 0, 640, 480))
		if !state.Visible || pic == nil || !ok || !bytes.Equal(pic.Pix, drawn.Pix) {
			t.Fatalf("popup (visible %v) is not the drawn card %q", state.Visible, view.Shelf[c].Info)
		}
		tooltipReleasePNG(t, fmt.Sprintf("book-card-%d", shown), pic)
	}
	if shown != 2 {
		t.Fatalf("the book shelf shows %d books, want the 2 it was given", shown)
	}
}

// An Armor or Shield writes its base defence as a pair of the same kind as a
// Defence Effect, and the item formatter prints one line per pair without
// merging: base defence 1 and a Defence +1 Effect give two lines carrying the
// installed label of stats.txt line 15, base pair first, then the heading, then
// the Effect pair. Two Effects give two further lines, and absorption has a line
// only above zero. The population is every armour code whose base defence is 1.
func TestReleaseArmourCardPrintsOneLinePerStoredPairWithoutMerging(t *testing.T) {
	f := releaseFront(t)
	label := rawTextLine(t, f, "main/text/stats.txt", 15)
	absorption := rawTextLine(t, f, "main/text/stats.txt", 16)
	if label == "" || label != f.Words.ItemStats[15] || absorption != f.Words.ItemStats[16] {
		t.Fatalf("stats.txt lines 15 and 16 are %q and %q, words hold %q and %q", label, absorption, f.Words.ItemStats[15], f.Words.ItemStats[16])
	}
	checked, withAbsorption := 0, 0
	for code := 0; code <= 0xffff; code++ {
		a, err := data.ArmorFromCode(data.ItemCode(code), f.Table.Shapes, f.Table.Materials, f.Table.Armors)
		if err != nil || a.Defence != 1 {
			continue
		}
		name := itemName(data.ItemCode(code), f.Table)
		base := []string{name, "#" + label + " +1"}
		if a.Absorption > 0 {
			base = append(base, fmt.Sprintf("#%s %+d", absorption, uint8(a.Absorption)))
			withAbsorption++
		}
		one := sim.ItemInstance{Code: uint16(code), Effects: []sim.ItemEffect{{Kind: 15, Operand: 1}}}
		want := append(slices.Clone(base), f.Words.ItemMagic, "#"+label+" +1")
		if got := itemInstanceInfoLines(one, f.Table, f.Words); !slices.Equal(got, want) {
			t.Fatalf("code %#04x with one Defence Effect: %q, want %q", code, got, want)
		}
		two := sim.ItemInstance{Code: uint16(code), Effects: []sim.ItemEffect{{Kind: 15, Operand: 1}, {Kind: 15, Operand: 1}}}
		want = append(slices.Clone(base), f.Words.ItemMagic, "#"+label+" +1", "#"+label+" +1")
		if got := itemInstanceInfoLines(two, f.Table, f.Words); !slices.Equal(got, want) {
			t.Fatalf("code %#04x with two Defence Effects: %q, want %q", code, got, want)
		}
		if got := itemInstanceInfoLines(sim.ItemInstance{Code: uint16(code)}, f.Table, f.Words); !slices.Equal(got, base) {
			t.Fatalf("code %#04x with no Effect: %q, want %q", code, got, base)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no armour code carries base defence 1")
	}
	t.Logf("%d armour codes with base defence 1 checked, %d with absorption above zero", checked, withAbsorption)
}

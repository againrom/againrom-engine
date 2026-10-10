package game

import (
	"testing"

	"golang.org/x/text/encoding/charmap"
)

// secondGameShippedRows is one installed text table of the second game's
// install, split by the loader's walk, with its bytes as shipped.
func secondGameShippedRows(t *testing.T, f *FrontEnd, addr string) *TextTable {
	t.Helper()
	b, err := f.Archives.Containers.ReadFile(addr)
	if err != nil {
		t.Fatalf("%s: %v", addr, err)
	}
	return SplitTextTable(b)
}

// secondGameWordReads reports whether drawn, a word the engine hands to the
// install's font, reads as shipped, the installed row: on a Russian install
// the shipped Windows Cyrillic row and the drawn DOS Cyrillic word decode to
// the same text (DIV-2844); on any other install the bytes are equal.
func secondGameWordReads(selector int, shipped, drawn string) bool {
	if selector != 1 {
		return shipped == drawn
	}
	want, err := charmap.Windows1251.NewDecoder().String(shipped)
	if err != nil {
		return false
	}
	got, err := charmap.CodePage866.NewDecoder().String(drawn)
	return err == nil && got == want
}

// Every word of the second game's install text tables reaches the font
// readable: the generator's tooltips (each control's main.txt row), the
// generator's own words, and every line of the eight tables and the help
// text the install words read, the item names and the room tips
// (R2-ENGINE-052, R2-ENGINE-093, DIV-2844).
func TestReleaseSecondGameInstallWordsReadInTheFont(t *testing.T) {
	f := secondGameFront(t)
	src := f.Archives.Containers
	selector := LanguageSelector(src)
	main := secondGameShippedRows(t, f, MainTextPath)
	hover := func(what string, slot int) {
		t.Helper()
		shipped, ok := main.At(slot)
		if !ok {
			return
		}
		if drawn := f.Words.Hover[slot]; !secondGameWordReads(selector, shipped, drawn) {
			t.Errorf("%s: main.txt row %d draws %q from %q", what, slot, drawn, shipped)
		}
	}
	l := f.generator()
	if l == nil {
		t.Fatal("the second game's install loaded no generator description")
	}
	pre, d := &l.PreCreate, &l.Detail
	for _, p := range []*int{pre.Name.Tooltip, pre.Forward.Tooltip, pre.Back.Tooltip} {
		if p != nil {
			hover("pre-create control", *p)
		}
	}
	for _, level := range pre.Levels {
		if level.Tooltip != nil {
			hover("level", *level.Tooltip)
		}
	}
	for _, hero := range pre.Heroes {
		if hero.Tooltip != nil {
			hover("picture", *hero.Tooltip)
		}
	}
	for class, c := range d.Classes {
		for skill := 0; skill < c.Selectable; skill++ {
			hover("skill", c.Tooltip+skill)
			shipped, _ := main.At(c.Tooltip + skill)
			if got := f.ChargenAssets.SkillHover[class][skill]; !secondGameWordReads(selector, shipped, got) {
				t.Errorf("generator skill %d/%d reads %q from %q", class, skill, got, shipped)
			}
		}
	}
	for i := range d.Stats.Value {
		hover("statistic label", d.Stats.LabelTooltip+i)
		hover("statistic value", d.Stats.ValueLabel+i)
	}
	hover("pool", d.Stats.PoolTooltip)
	setup := f.ChargenSetup()
	for _, w := range []struct {
		what         string
		slot         int
		drawn, setup string
	}{
		{"tip close", mainSlotTipClose, f.Words.TipClose, setup.TipClose},
		{"tip show next", mainSlotTipShowNext, f.Words.TipShowNext, setup.TipToggle},
		{"mission won", mainSlotMissionWon, f.Words.MissionWon, f.Words.MissionWon},
	} {
		shipped, ok := main.At(w.slot)
		if !ok {
			continue
		}
		if !secondGameWordReads(selector, shipped, w.drawn) || !secondGameWordReads(selector, shipped, w.setup) {
			t.Errorf("%s: words %q, generator %q from %q", w.what, w.drawn, w.setup, shipped)
		}
	}

	words := LoadInstallWords(src, f.textCode())
	for _, table := range []struct {
		addr string
		read func(int) (string, bool)
	}{
		{MainTextPath, words.Global},
		{DialogsTextPath, words.Dialogs},
		{StatsTextPath, words.Stats},
		{UnitNameTextPath, words.UnitName},
		{SitesTextPath, words.Site},
		{BuildingTextPath, words.buildingNames.At},
		{SpellNamesTextPath, words.spellNames.At},
		{SpellBookNamesTextPath, words.spellBookNames.At},
	} {
		shipped := secondGameShippedRows(t, f, table.addr)
		for i := range shipped.Lines() {
			s, ok := shipped.At(i)
			if !ok {
				continue
			}
			if got, _ := table.read(i); !secondGameWordReads(selector, s, got) {
				t.Errorf("%s row %d reads %q from %q", table.addr, i, got, s)
			}
		}
	}
	if help, err := src.ReadFile(HelpTextPath); err == nil && !secondGameWordReads(selector, string(help), words.help) {
		t.Errorf("%s does not read whole", HelpTextPath)
	}

	// An install without the item-name pair has no names to read.
	shippedNames, _ := ReadItemNames(src, TextCode{})
	for code, s := range shippedNames {
		if got := f.Table.Names[code]; !secondGameWordReads(selector, s, got) {
			t.Errorf("item %#x reads %q from %q", code, got, s)
		}
	}
	// The second game names the first game's room description (Edition.Rooms).
	for _, room := range []townRoom{roomShop, roomTavern, roomSchool} {
		addr := roomTipIn(ROM1TownDescription(), room).Text
		shipped, ok := ReadShopTip(src, addr, TextCode{})
		if !ok {
			continue
		}
		if got, _ := ReadShopTip(src, addr, f.textCode()); !secondGameWordReads(selector, shipped, got) {
			t.Errorf("%s does not read whole", addr)
		}
	}
}

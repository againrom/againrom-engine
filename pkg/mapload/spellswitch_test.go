package mapload

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
)

func spellSwitchFixture(t *testing.T) mod.SpellData {
	t.Helper()
	src, err := os.ReadFile("../modrt/testdata/mods/spell-switch/data/spells.toml")
	if err != nil {
		t.Fatal(err)
	}
	d, err := mod.ParseSpells("spell-switch", mod.SpellsFile, src, nil)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// A map unit casts what the map gives it: the fixture turns Fire Ball off and
// the unit's book, mask and cast list stay the map's.
func TestSpellSwitchLeavesAMapUnitsCastList(t *testing.T) {
	var rows []sim.SpellRule
	for id := 1; id <= 29; id++ {
		rows = append(rows, sim.SpellRule{ID: uint16(id)})
	}
	params := make([]int32, 64)
	params[48], params[49], params[50], params[51] = 2, 70, 3, 30
	for _, spells := range []string{"rom1", "rom2"} {
		plain := &Table{builtSpellRules: rows, Edition: &base.Edition{Spells: spells}}
		modded := &Table{builtSpellRules: rows, Edition: &base.Edition{Spells: spells}}
		modded.Mods.Spells = spellSwitchFixture(t)
		if p := SpellGameProfileOf(modded); p.ShopBook(2) || p.ShopScroll(2) || p.BookRows()[1] != 0 || p.BookRows()[0] != 3 {
			t.Fatalf("%s: the fixture leaves Fire Ball in the book or a shop: %v", spells, p.BookRows())
		}
		wantKnown, wantBook, wantSlots := unitSpellbook(params, plain, 0x46)
		known, book, slots := unitSpellbook(params, modded, 0x46)
		if known != wantKnown || known&(1<<2) == 0 || !reflect.DeepEqual(book, wantBook) || slots != wantSlots || slots[0].ID != 2 {
			t.Errorf("%s: the unit's book moved under the mod: %#x %+v", spells, known, slots)
		}
	}
}

func TestModSpellProfileRefusalNamesTheRow(t *testing.T) {
	tab := &Table{Edition: &base.Edition{Spells: "rom1"}}
	d := spellSwitchFixture(t)
	d.Rows = append(d.Rows, mod.SpellRow{Mod: "late", File: mod.SpellsFile, Line: 40, Target: "Bless",
		BookPosition: mod.SpellInt{Set: true, Val: 24, Line: 42}})
	_, at, msg := ModSpellProfile(tab, d)
	if at.Mod != "late" || at.Line != 42 || msg != "book_position of Bless is 24, outside 1..23" {
		t.Errorf("refusal %q at %s:%d", msg, at.Mod, at.Line)
	}
	d = spellSwitchFixture(t)
	d.Global.BookSize.Val = 30
	_, at, msg = ModSpellProfile(tab, d)
	if at.Mod != "spell-switch" || at.File != mod.SpellsFile || at.Line != 4 || !strings.Contains(msg, "book_size is 30, outside 1..24") {
		t.Errorf("size refusal %q at %+v", msg, at)
	}
	if p, _, msg := ModSpellProfile(tab, mod.SpellData{}); msg != "" || p.BookSize != 24 {
		t.Errorf("no mod: %q %d", msg, p.BookSize)
	}
}

// A move and an explicit on of one spell load in one row, in two rows, and in
// two mods; the on is judged against the cell the moves leave, and a true
// overflow still names its own row and the limit.
func TestAnExplicitOnFollowsTheMovedCell(t *testing.T) {
	size := mod.SpellInt{Set: true, Val: 10, Line: 2}
	move := func(m, name string) mod.SpellRow {
		return mod.SpellRow{Mod: m, File: mod.SpellsFile, Line: 5, Target: name, BookPosition: mod.SpellInt{Set: true, Val: 1, Line: 6}}
	}
	on := func(m, name string) mod.SpellRow {
		return mod.SpellRow{Mod: m, File: mod.SpellsFile, Line: 8, Target: name, Off: mod.SpellBool{Set: true, Line: 9}}
	}
	for _, g := range []string{"rom1", "rom2"} {
		tab := &Table{Edition: &base.Edition{Spells: g}}
		var name, first string
		for _, s := range SpellGameProfileOf(tab).Spells {
			switch s.Slot {
			case 1:
				first = s.Name
			case 20:
				name = s.Name
			}
		}
		clear := mod.SpellRow{Mod: "one", File: mod.SpellsFile, Line: 3, Target: first, Off: mod.SpellBool{Set: true, Val: true, Line: 4}}
		if name == "" {
			t.Fatalf("%s: no spell at cell 20", g)
		}
		both := move("one", name)
		both.Off = mod.SpellBool{Set: true, Line: 7}
		for label, rows := range map[string][]mod.SpellRow{
			"one row":    {clear, both},
			"two rows":   {clear, move("one", name), on("one", name)},
			"two mods":   {clear, move("a", name), on("b", name)},
			"two mods 2": {clear, on("b", name), move("a", name)},
		} {
			d := mod.SpellData{Rows: rows}
			d.Global.BookSize, d.Global.BookSizeMod, d.Global.BookSizeFile = size, "one", mod.SpellsFile
			p, at, msg := ModSpellProfile(tab, d)
			if msg != "" || p.BookRows()[0] == 0 {
				t.Errorf("%s %s: refused %q at %s:%d, rows %v", g, label, msg, at.Mod, at.Line, p.BookRows())
			}
		}
		d := mod.SpellData{Rows: []mod.SpellRow{on("b", name)}}
		d.Global.BookSize, d.Global.BookSizeMod, d.Global.BookSizeFile = size, "one", mod.SpellsFile
		_, at, msg := ModSpellProfile(tab, d)
		if at.Mod != "b" || at.Line != 9 || msg != name+" is on, but its book position 20 is past book_size 10" {
			t.Errorf("%s overflow: %q at %s:%d", g, msg, at.Mod, at.Line)
		}
	}
}

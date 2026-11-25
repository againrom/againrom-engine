package mapload

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

func mapSpellRows() []sim.SpellRule {
	var rows []sim.SpellRule
	for id := 1; id <= 29; id++ {
		rows = append(rows, sim.SpellRule{ID: uint16(id), MaxRange: 5, ManaCost: 10})
	}
	return rows
}

// mapSpellMap places one unit with id 7 and gives it the enchantment records.
func mapSpellMap(records ...alm.Enchantment) *alm.Map {
	return &alm.Map{Width: 72, Height: 68,
		Units:        []alm.Unit{{X: 0x1780, Y: 0x1580, ClassID: 7, UnitID: 7}},
		Enchantments: records}
}

// linkRecordToItem makes the sack of the map's loot link to record index 0.
func linkRecordToItem(m *alm.Map) {
	body := make([]byte, 0, 26)
	body = binary.LittleEndian.AppendUint32(body, 1) // one element
	body = binary.LittleEndian.AppendUint32(body, 0)
	body = binary.LittleEndian.AppendUint32(body, 0x100)
	body = binary.LittleEndian.AppendUint32(body, 0x200)
	body = binary.LittleEndian.AppendUint32(body, 0x0205)
	body = binary.LittleEndian.AppendUint16(body, 3)
	body = binary.LittleEndian.AppendUint32(body, 1) // tile marker index 1 is record 0
	m.LootSection = alm.LootSection{Body: body}
	m.Meta.Word2C = 1
}

func loadMapSpellUnit(t *testing.T, m *alm.Map) sim.Entity {
	t.Helper()
	w, err := FromALMWith(m, &Table{builtSpellRules: mapSpellRows()}, DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	return w.Entities()[0]
}

// A placed unit whose map holds a unit spell record for a spell its kind
// lacks loads with that spell in its book, and the book survives the saved
// form.
func TestPlacedUnitKnowsTheSpellItsMapRecordGives(t *testing.T) {
	t.Parallel()
	w, err := FromALMWith(mapSpellMap(alm.Enchantment{Tag: 1, A: 8, B: 7, SpellRaw: 9}),
		&Table{builtSpellRules: mapSpellRows()}, DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	e := w.Entities()[0]
	if e.KnownSpells != 1<<9 || e.Book.State != sim.BookPresent || *e.Book.Slot(9) != (sim.BookSpell{Range: 5, ManaCost: 10}) {
		t.Fatalf("known %#x book %+v, want spell 9 only in a present book", e.KnownSpells, e.Book.State)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back sim.World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	b := back.Entities()[0]
	if b.KnownSpells != e.KnownSpells || b.Book != e.Book || back.Hash() != w.Hash() {
		t.Fatalf("saved form lost the spell: known %#x, book %+v", b.KnownSpells, b.Book.State)
	}
}

// A record an item links to takes the item arm and gives no book entry; a
// record keyed to another unit and a record without the unit bit give none.
func TestMapRecordsThatAreNotTheUnitArmGiveNoSpell(t *testing.T) {
	t.Parallel()
	linked := mapSpellMap(alm.Enchantment{Tag: 1, A: 8, B: 7, SpellRaw: 9})
	linkRecordToItem(linked)
	for name, m := range map[string]*alm.Map{
		"item-linked": linked,
		"other unit":  mapSpellMap(alm.Enchantment{Tag: 1, A: 8, B: 8, SpellRaw: 9}),
		"no unit bit": mapSpellMap(alm.Enchantment{Tag: 1, A: 4, B: 7, SpellRaw: 9}),
		"no records":  mapSpellMap(),
	} {
		e := loadMapSpellUnit(t, m)
		if e.KnownSpells != 0 || e.Book.State != sim.BookLegacy {
			t.Errorf("%s: known %#x book state %d, want the kind's own empty book", name, e.KnownSpells, e.Book.State)
		}
	}
}

// A table-backed book keeps its state and gains the membership bit.
func TestMapSpellKeepsATableBackedBook(t *testing.T) {
	t.Parallel()
	b := spawnBlock{knownSpells: 1 << 2}
	b.giveMapSpells(mapSpellMap(alm.Enchantment{Tag: 1, A: 8, B: 7, SpellRaw: 9}), 7, &Table{builtSpellRules: mapSpellRows()})
	if b.knownSpells != 1<<2|1<<9 || b.book.State != sim.BookLegacy {
		t.Fatalf("known %#x state %d", b.knownSpells, b.book.State)
	}
}

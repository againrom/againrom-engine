package mapload

import (
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// The fixed books and skill words per Dragon face and for the Daemon, as the
// original's saves hold them; a face outside 1..4 and every other class get
// none.
func TestFixedCreatureBooks(t *testing.T) {
	cases := []struct {
		typeID, face int32
		spells       []int32
		skill        [data.SkillSlots]int32
	}{
		{0x47, 1, []int32{2, 3, 13}, [6]int32{7, 30, 0, 30, 0, 0}},
		{0x47, 2, []int32{2, 3, 13, 11, 27}, [6]int32{7, 50, 60, 50, 0, 20}},
		{0x47, 3, []int32{2, 8, 11, 14, 27}, [6]int32{7, 50, 50, 50, 0, 50}},
		{0x47, 4, []int32{2, 8, 11, 14, 17, 20}, [6]int32{7, 90, 90, 90, 0, 30}},
		{0x48, 1, []int32{2, 8, 11, 14, 17, 20}, [6]int32{7, 99, 99, 99, 20, 0}},
		{0x47, 5, nil, [6]int32{7, 0, 0, 0, 0, 0}},
		{0x46, 4, nil, [6]int32{7, 30, 30, 30, 30, 30}},
	}
	for _, c := range cases {
		fixed, ok := FirstGameCreatureBooks.rowFor(c.typeID, c.face)
		if ok != (c.spells != nil) || !reflect.DeepEqual(fixed.Spells, c.spells) {
			t.Fatalf("type %#x face %d: %v %v, want %v", c.typeID, c.face, fixed.Spells, ok, c.spells)
		}
		// The spawn's own order: the face-4 arm first, then the fixed book.
		skill := [data.SkillSlots]int32{7}
		if c.face == 4 {
			for i := 1; i < len(skill); i++ {
				skill[i] = 30
			}
		}
		var known uint32
		var book sim.Spellbook
		// A table without spell rows inserts nothing and still writes the skills.
		FirstGameCreatureBooks.add(c.typeID, c.face, nil, &known, &book, &skill)
		if skill != c.skill || known != 0 || book != (sim.Spellbook{}) {
			t.Fatalf("type %#x face %d skills %v known %b, want %v", c.typeID, c.face, skill, known, c.skill)
		}
	}
}

// todaysCreatureBook is the fixed book each class and face got before the
// rows became profile data, written apart from them: the first game's Daemon
// whatever its face and Dragon by face 1..4, nothing on the second game.
func todaysCreatureBook(second bool, typeID, face int32) ([]int32, [6]int32, bool) {
	dragon := [4]struct {
		spells []int32
		skill  [6]int32
	}{
		{[]int32{2, 3, 13}, [6]int32{-1, 30, -1, 30, -1, -1}},
		{[]int32{2, 3, 13, 11, 27}, [6]int32{-1, 50, 60, 50, -1, 20}},
		{[]int32{2, 8, 11, 14, 27}, [6]int32{-1, 50, 50, 50, -1, 50}},
		{[]int32{2, 8, 11, 14, 17, 20}, [6]int32{-1, 90, 90, 90, 0, -1}},
	}
	switch {
	case second:
	case typeID == 0x48:
		return []int32{2, 8, 11, 14, 17, 20}, [6]int32{-1, 99, 99, 99, 20, -1}, true
	case typeID == 0x47 && face >= 1 && face <= 4:
		return dragon[face-1].spells, dragon[face-1].skill, true
	}
	return nil, [6]int32{}, false
}

// Every class and face, on the table each game reads (nil and an unset field
// are the first game's), gets today's fixed book and skill words through the
// spawn's call, over a spell table holding every spell id.
func TestEveryCreatureGetsTodaysBookOnBothGames(t *testing.T) {
	spells := make(itemTestCollection, 29)
	for id := 1; id < len(spells); id++ {
		spells[id].params = make([]int32, 21)
		for i := range spells[id].params {
			spells[id].params[i] = int32(id + i)
		}
	}
	for _, tc := range []struct {
		second bool
		table  *Table
	}{
		{false, &Table{Spells: spells}},
		{false, &Table{Spells: spells, CreatureBooks: FirstGameCreatureBooks}},
		{true, &Table{Spells: spells, CreatureBooks: ClassCellBooks}},
	} {
		rules := SpellRules(tc.table)
		if len(rules) != 28 {
			t.Fatalf("%d spell rules", len(rules))
		}
		for typeID := int32(-1); typeID <= 0x100; typeID++ {
			for face := int32(-1); face <= 6; face++ {
				spellsWant, skillWant, ok := todaysCreatureBook(tc.second, typeID, face)
				if !tc.second {
					if row, got := (*Table)(nil).creatureBooks().rowFor(typeID, face); got != ok || !reflect.DeepEqual(row.Spells, spellsWant) {
						t.Fatalf("nil table type %#x face %d: row %+v %v", typeID, face, row, got)
					}
				}
				row, got := tc.table.creatureBooks().rowFor(typeID, face)
				if got != ok || !reflect.DeepEqual(row.Spells, spellsWant) || (ok && row.Skill != skillWant) {
					t.Fatalf("second %v type %#x face %d: row %+v %v", tc.second, typeID, face, row, got)
				}
				var known, knownWant uint32
				var book, bookWant sim.Spellbook
				skill := [data.SkillSlots]int32{7, 1, 2, 3, 4, 5}
				want := skill
				for _, id := range spellsWant {
					bookWant.Slots[id-1] = bookSpell(rules[id-1])
					knownWant |= 1 << uint(id)
				}
				for i, v := range skillWant {
					if ok && v >= 0 {
						want[i] = v
					}
				}
				tc.table.creatureBooks().add(typeID, face, tc.table, &known, &book, &skill)
				if known != knownWant || book != bookWant || skill != want {
					t.Fatalf("second %v type %#x face %d: known %b book %v skill %v, want %b %v", tc.second, typeID, face, known, book, skill, knownWant, want)
				}
			}
		}
	}
}

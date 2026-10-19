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
		fixed, ok := creatureBookFor(c.typeID, c.face)
		if ok != (c.spells != nil) || !reflect.DeepEqual(fixed.spells, c.spells) {
			t.Fatalf("type %#x face %d: %v %v, want %v", c.typeID, c.face, fixed.spells, ok, c.spells)
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
		addFixedCreatureBook(c.typeID, c.face, nil, &known, &book, &skill)
		if skill != c.skill || known != 0 || book != (sim.Spellbook{}) {
			t.Fatalf("type %#x face %d skills %v known %b, want %v", c.typeID, c.face, skill, known, c.skill)
		}
	}
}

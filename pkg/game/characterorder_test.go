package game

import (
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mod"
)

func TestCharacterEditsKeepBaseOutcomes(t *testing.T) {
	cases := []struct {
		name  string
		rows  []mod.CharacterRow
		order mod.Precedence
		faces [2]int32
		kind  bool
		strip bool
	}{
		{"unordered_disjoint_fields", []mod.CharacterRow{
			{Mod: "a", Target: "NPC06", Face: 5},
			{Mod: "b", Target: "NPC06", Strip: []string{mod.StripArmour}},
		}, nil, [2]int32{5, 5}, false, true},
		{"ordered_later_kind_resets_earlier_face", []mod.CharacterRow{
			{Mod: "a", Target: "NPC06", Face: 5},
			{Mod: "b", Target: "NPC06", Kind: "U_Woman"},
		}, mod.Precedence{"b": {"a"}}, [2]int32{12, 12}, true, false},
		{"ordered_interleaved_family_and_row", []mod.CharacterRow{
			{Mod: "a", Target: "NPC06", Face: 5},
			{Mod: "b", Target: "NPC06_1", Face: 6},
			{Mod: "b", Target: "NPC06", Face: 7},
		}, mod.Precedence{"b": {"a"}}, [2]int32{7, 7}, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			before, after := characterFront(), characterFront()
			for i := range c.rows {
				c.rows[i].File, c.rows[i].Line = mod.CharactersFile, i*4+1
			}
			if err := after.SetModCharacters(mod.CharacterData{Rows: c.rows, Order: c.order}); err != nil {
				t.Fatalf("base accepted these operations: %v", err)
			}
			for i := 1; i < before.Humans.Len(); i++ {
				params := slices.Clone(before.Humans.EntryParams(i))
				cells := slices.Clone(before.Humans.EntryStrings(i))
				if i == 1 || i == 2 {
					params[data.HumanSlotFace] = c.faces[i-1]
					if c.kind {
						params[data.HumanSlotTypeID], params[data.HumanSlotGender] = 1, 1
					}
					if c.strip {
						clear(cells[data.HumanCellArmour:])
					}
				}
				if !slices.Equal(params, after.Humans.EntryParams(i)) || !slices.Equal(cells, after.Humans.EntryStrings(i)) {
					t.Errorf("%s: params %v, cells %q; base outcome params %v, cells %q", before.Humans.EntryName(i), after.Humans.EntryParams(i), after.Humans.EntryStrings(i), params, cells)
				}
			}
		})
	}
}

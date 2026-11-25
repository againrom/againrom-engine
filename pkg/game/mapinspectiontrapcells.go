package game

import (
	"slices"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// trapCellRows are the type-9 fields a teleport trap shows as map cells: its
// own cell as Source and its destination as Target, each an X field whose
// Y field the map pick sets with it.
var trapCellRows = []struct {
	x, y, word string
	own        bool
}{
	{x: "triggerx", y: "triggery", word: "editor.cast.teleportsource", own: true},
	{x: "targetx", y: "targety", word: "editor.cast.teleporttarget"},
}

// trapCellFields marks the cell rows of out, the fields of type-9 record
// index, when the record holds the game's cell move: the source cell fields
// the move does not read are left out.
func (s *mapInspectionEdits) trapCellFields(out []ui.InspectionField, index int) []ui.InspectionField {
	move := int64(cellMoveOf(s.model))
	if spell, err := s.model.FieldValue(mapedit.KindEnchantment, index, "spell"); err != nil || move == 0 || spell != move {
		return out
	}
	if !slices.ContainsFunc(out, func(f ui.InspectionField) bool { return f.Key == "targetx" }) {
		return out
	}
	out = slices.DeleteFunc(out, func(f ui.InspectionField) bool { return f.Key == "sourcex" || f.Key == "sourcey" })
	for i := range out {
		for _, r := range trapCellRows {
			switch out[i].Key {
			case r.x:
				out[i].CellPair, out[i].CellOwn = r.y, r.own
				out[i].Label = s.inspector.word(r.word) + " X"
			case r.y:
				out[i].Label = s.inspector.word(r.word) + " Y"
			}
		}
	}
	return out
}

package mapedit

import "slices"

// cellMoveRow is the one row of the format dictionary for the cell event that
// moves a unit to its target: the fields a record of that event holds and
// which of them are map cells. The event reads no source cell, so the source
// fields are not held; its own cell and its target are each an X field whose
// Y field the map pick sets with it (CellPair), under the word CellWord.
var cellMoveRow = struct {
	drop  []string
	cells []struct {
		x, y, word string
		own        bool
	}
}{
	drop: []string{"sourcex", "sourcey"},
	cells: []struct {
		x, y, word string
		own        bool
	}{
		{x: "triggerx", y: "triggery", word: "editor.cast.teleportsource", own: true},
		{x: "targetx", y: "targety", word: "editor.cast.teleporttarget"},
	},
}

// cellMoveFields applies the cell move's row to fields, a type-9 record's
// fields of either format.
func cellMoveFields(fields []Field) []Field {
	out := slices.DeleteFunc(slices.Clone(fields), func(f Field) bool { return slices.Contains(cellMoveRow.drop, f.Key) })
	for i := range out {
		for _, c := range cellMoveRow.cells {
			switch out[i].Key {
			case c.x:
				out[i].CellPair, out[i].CellOwn, out[i].CellWord = c.y, c.own, c.word
			case c.y:
				out[i].CellWord = c.word
			}
		}
	}
	return out
}

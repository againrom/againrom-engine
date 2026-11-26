package mapedit

import "slices"

// cellMoveRow is the one row of the format dictionary for the cell event that
// moves a unit to its target: the roles of the fields a record of that event
// does not hold and which of the others are map cells. The event reads no
// source cell, so the source fields are not held; its own cell and its target
// are each an X field whose Y field the map pick sets with it (CellPair),
// under the word CellWord.
var cellMoveRow = struct {
	drop  []CellRole
	cells []struct {
		x, y CellRole
		word string
		own  bool
	}
}{
	drop: []CellRole{CellSourceX, CellSourceY},
	cells: []struct {
		x, y CellRole
		word string
		own  bool
	}{
		{x: CellTriggerX, y: CellTriggerY, word: "editor.cast.teleportsource", own: true},
		{x: CellTargetX, y: CellTargetY, word: "editor.cast.teleporttarget"},
	},
}

// cellMoveFields applies the cell move's row to fields, a type-9 record's
// fields of either format.
func cellMoveFields(fields []Field) []Field {
	out := slices.DeleteFunc(slices.Clone(fields), func(f Field) bool { return slices.Contains(cellMoveRow.drop, f.Cell) })
	for i := range out {
		for _, c := range cellMoveRow.cells {
			switch out[i].Cell {
			case c.x:
				out[i].CellPair, out[i].CellOwn, out[i].CellWord = c.y.Key(), c.own, c.word
			case c.y:
				out[i].CellWord = c.word
			}
		}
	}
	return out
}

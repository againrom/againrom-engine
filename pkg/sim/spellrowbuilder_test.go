package sim

import "testing"

// Every route to a spell rule reaches the one table: the World's own game,
// the game a wrapper decodes under, and an unknown game, which is the first.
func TestSpellRowsComeFromTheOneTable(t *testing.T) {
	for _, c := range []struct {
		name         string
		dialect      ScriptDialect
		wholeColumns bool
		armRows      bool
	}{
		{"first", ScriptROM1, true, false},
		{"second", ScriptROM2, false, true},
	} {
		w := spWorld(t, 1, nil, spEnt(1, 0, 0))
		w.game = newGameProfile(c.dialect)
		row := w.spellRow()
		if row != spellRowOf(w.game.spellGame) || row != &spellRows[w.game.spellGame] {
			t.Errorf("%s: the World's row is not the table's", c.name)
		}
		if row.wholeColumns != c.wholeColumns || row.armRows != c.armRows {
			t.Errorf("%s: row marks wholeColumns %v armRows %v", c.name, row.wholeColumns, row.armRows)
		}
		if row.damage == nil || row.heal == nil || row.columns == nil || row.poison == nil || row.drain.amount == nil || row.drain.credit == nil {
			t.Errorf("%s: the row leaves a rule unset", c.name)
		}
	}
	if spellRowOf(testSecondGame) == spellRowOf(testFirstGame) {
		t.Error("the two games share one row")
	}
	if spellRowOf(testSecondGame+1) != spellRowOf(testFirstGame) {
		t.Error("an unknown game does not take the first game's row")
	}
	w := spWorld(t, 1, nil, spEnt(1, 0, 0))
	if w.formSpellRow() != spellRowOf(testFirstGame) {
		t.Error("a wrapper decodes under the first game before the World has its game")
	}
}

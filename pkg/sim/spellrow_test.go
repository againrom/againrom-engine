package sim

import "testing"

var (
	testFirstGame  = newGameProfile(ScriptROM1).spellGame
	testSecondGame = newGameProfile(ScriptROM2).spellGame
)

// spSecondWorld is spWorld running the second game.
func spSecondWorld(t *testing.T, seed uint64, spells []SpellRule, ents ...Entity) *World {
	t.Helper()
	w := spWorld(t, seed, spells, ents...)
	w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	return w
}

// setTestGame makes w run the second game, or the first.
func setTestGame(w *World, second bool) {
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
		return
	}
	w.rom2, w.game = nil, newGameProfile(ScriptROM1)
}

// testRowOf is the spell row of the game a test names with a second flag.
func testRowOf(second bool) *spellRow {
	if second {
		return spellRowOf(testSecondGame)
	}
	return spellRowOf(testFirstGame)
}

package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

// secondKnower builds a knower of a second-game row: the row carries the arm
// the second game's table assigns its id.
func secondKnower(id uint16, p int32) spellKnower {
	rows := []sim.SpellRule{{ID: id}}
	sim.AssignSecondGameArms(rows)
	return spellKnower{rule: rows[0], c: sim.SpellCharacteristics{Power: p}}
}

// TestSecondGamePopupStatesItsOwnArmsCaption pins that a second-game row's
// popup caption follows the arm the row runs, not the first-game spell its
// id equals.
func TestSecondGamePopupStatesItsOwnArmsCaption(t *testing.T) {
	for _, tc := range []struct {
		name  string
		id    uint16
		power int32
		want  string
	}{
		{"Heal (24) states no Haste caption", 24, 45, ""},
		{"Darkness (14) is outside the second game's book rows and states none", 14, 60, ""},
		{"Haste (21) states its arm's caption", 21, 45, "C182: 4"},
		{"Blizzard (7) runs the second game's arm 21 and states none", 7, 45, ""},
		{"Shield (27) is inside the second game's book", 27, 100, "C217: 13"},
		{"Poison Cloud (6) states none", 6, 80, ""},
	} {
		if got := captionLineIn(t, data.SpellPresentationOf(rules.SpellGameSecond).BookRows, secondKnower(tc.id, tc.power)); got != tc.want {
			t.Errorf("%s: caption %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestFirstGamePopupKeepsItsIDKeyedCaption pins one first-game popup of each
// kept caption kind next to the second-game rows above.
func TestFirstGamePopupKeepsItsIDKeyedCaption(t *testing.T) {
	for _, tc := range []struct {
		id    uint16
		power int32
		want  string
	}{
		{24, 45, "C182: 4"}, {14, 30, "C186: 3"}, {12, 59, "C184: -2"}, {5, 51, "C183: +25"},
		{23, 50, "C185: +60%"}, {18, 100, "C217: 13"}, {27, 80, ""},
	} {
		if got := captionLine(t, knower(tc.id, tc.power)); got != tc.want {
			t.Errorf("first game id %d: caption %q, want %q", tc.id, got, tc.want)
		}
	}
}

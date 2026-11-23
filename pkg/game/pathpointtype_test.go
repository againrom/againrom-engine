package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/rules"
)

// A second-game Lightning point stores type 28 and a chain point the victim
// index mod 7 (R2-ENGINE-1516); the first game's Lightning stores 34.
func TestThePathPointTypeIsTheIdentifierOrTheVictimIndexModSeven(t *testing.T) {
	second := data.SpellPresentationOf(rules.SpellGameSecond)
	if got := boltTag(28, 9, second.Effect(28).Chain); got != 28 {
		t.Errorf("second-game Lightning point type %d, want 28", got)
	}
	if got := boltTag(30, 9, second.Effect(30).Chain); got != 9%7 {
		t.Errorf("second-game chain point type %d, want %d", got, 9%7)
	}
	first := data.SpellPresentationOf(rules.SpellGameFirst)
	if got := boltTag(34, 9, first.Effect(34).Chain); got != 34 {
		t.Errorf("first-game Lightning point type %d, want 34", got)
	}
}

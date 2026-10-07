package game

import (
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func captionWords() ui.Words {
	w := testSpellPopupWords()
	for _, n := range []int{182, 183, 184, 185, 186, 187, 217} {
		w.Hover[n] = fmt.Sprintf("C%d", n)
	}
	return w
}

// knower builds a knower at spell power p for spell id.
func knower(id uint16, p int32) spellKnower {
	return spellKnower{rule: sim.SpellRule{ID: id}, c: sim.SpellCharacteristics{Power: p}}
}

func captionLine(t *testing.T, ks ...spellKnower) string {
	t.Helper()
	w := captionWords()
	lines := spellPopupLines(ks, "n", &w)
	// name and the mana line come first; the caption, when present, is last.
	if len(lines) <= 2 {
		return ""
	}
	return lines[len(lines)-1]
}

// C182 is main[182], C183 is [183] and so on.
func TestSpellCaptionFormulasAtTheirLevels(t *testing.T) {
	for _, tc := range []struct {
		name  string
		id    uint16
		power int32
		want  string
	}{
		{"182 on id 24", 24, 45, "C182: 4"},
		{"182 on id 7 is negative", 7, 45, "C182: -4"},
		{"183 on id 5", 5, 51, "C183: +25"},
		{"183 on id 16", 16, 100, "C183: +50"},
		{"183 on id 10", 10, 4, "C183: +2"},
		{"183 on id 22", 22, 2, "C183: +1"},
		{"183 is absent at level 0 and 1", 22, 1, ""},
		{"184 on id 12", 12, 59, "C184: -2"},
		{"185 on id 23", 23, 50, "C185: +60%"},
		{"186 on id 14", 14, 30, "C186: 3"},
		{"186 caps at 7", 14, 100, "C186: 7"},
		{"217 on id 18", 18, 100, "C217: 13"},
		{"a spell with no caption", 1, 80, ""},
		{"id 27 is outside the book", 27, 80, ""},
		{"id 17 is outside the book", 17, 80, ""},
		{"id 28 is outside the book", 28, 80, ""},
	} {
		if got := captionLine(t, knower(tc.id, tc.power)); got != tc.want {
			t.Errorf("%s: caption %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestSpellCaptionFoldsMinimumAndMaximumOverSelectedKnowers(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   uint16
		p    []int32
		want string
	}{
		{"182 range", 24, []int32{15, 45}, "C182: 2...4"},
		{"183 range carries a plus on both ends", 5, []int32{20, 60}, "C183: +10...+30"},
		{"184 range", 12, []int32{0, 90}, "C184: -4...-1"},
		{"185 range closes with one percent sign", 23, []int32{0, 100}, "C185: +20...+100%"},
		{"186 range uses a hyphen", 14, []int32{0, 100}, "C186: 2-7"},
		{"217 range uses a hyphen", 18, []int32{0, 100}, "C217: 3-13"},
		{"equal values collapse", 23, []int32{50, 50}, "C185: +60%"},
	} {
		var ks []spellKnower
		for _, p := range tc.p {
			ks = append(ks, knower(tc.id, p))
		}
		if got := captionLine(t, ks...); got != tc.want {
			t.Errorf("%s: caption %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestSpellPopupFoldsDamageRangeDurationAndMana(t *testing.T) {
	w := testSpellPopupWords()
	rule := sim.SpellRule{ID: 1, Damaging: true}
	ruleA, ruleB := rule, rule
	ruleA.DamageMin, ruleA.DamageMax = 2, 4
	ruleB.DamageMin, ruleB.DamageMax = 3, 4
	a := spellKnower{ruleA, sim.SpellCharacteristics{ManaCost: 5, Range: 6, Duration: 16}}
	b := spellKnower{ruleB, sim.SpellCharacteristics{ManaCost: 9, Range: 8, Duration: 32}}
	got := spellPopupLines([]spellKnower{a, b}, "Fire", &w)
	want := []string{"Fire", "Mana cost: 9", "Damage: 5-8", "Range: 6-8", "Duration:   1.0-  2.0"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("fold = %q, want %q", got, want)
	}
	got = spellPopupLines([]spellKnower{a, a}, "Fire", &w)
	want = []string{"Fire", "Mana cost: 5", "Damage: 4-8", "Range: 6", "Duration:   1.0"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("equal knowers = %q, want %q", got, want)
	}
	zero := spellKnower{rule, sim.SpellCharacteristics{ManaCost: 5}}
	got = spellPopupLines([]spellKnower{zero}, "Fire", &w)
	if want := []string{"Fire", "Mana cost: 5"}; !reflect.DeepEqual(got, want) {
		t.Errorf("zero maxima = %q, want %q", got, want)
	}
}

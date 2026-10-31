package game

import (
	"testing"

	"againrom/pkg/sim"
)

func TestShopBookAndCaptionSpellListsKeepTheirAnswers(t *testing.T) {
	for id := -5; id <= 70; id++ {
		want := id >= 2 && id <= 27 && id != 6 && id != 11 && id != 12 && id != 17 && id != 18 && id != 27
		if got := shopBookSpellAdmitted(id); got != want {
			t.Errorf("shop book spell %d admitted = %v, today %v", id, got, want)
		}
	}
	for id := uint16(0); id <= 70; id++ {
		for _, c := range []sim.SpellCharacteristics{{Power: 47}, {Power: 47, HasMagnitude: true, Magnitude: -3}} {
			got, gotOK := spellCaptions[0].value(sim.SpellRule{ID: id}, c)
			want, wantOK := int64(0), false
			switch {
			case id != 24 && id != 7 && id != 28:
			case c.HasMagnitude:
				want, wantOK = int64(c.Magnitude), true
			case id == 24:
				want, wantOK = 47/15+1, true
			default:
				want, wantOK = -(47/15 + 1), true
			}
			if got != want || gotOK != wantOK {
				t.Errorf("caption 182 of spell %d = %d %v, today %d %v", id, got, gotOK, want, wantOK)
			}
		}
	}
}

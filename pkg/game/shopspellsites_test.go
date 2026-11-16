package game

import (
	"testing"

	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

func TestShopBookAndCaptionSpellListsKeepTheirAnswers(t *testing.T) {
	for _, game := range []rules.SpellGame{rules.SpellGameFirst, rules.SpellGameSecond} {
		for id := -5; id <= 70; id++ {
			want := id >= 2 && id <= 27 && id != 6 && id != 11 && id != 12 && id != 17 && id != 18 && id != 27
			if got := shopBookSpellAdmitted(rules.GameProfile(game), id); got != want {
				t.Errorf("game %d shop book spell %d admitted = %v, today %v", game, id, got, want)
			}
			if got, want := rules.SpellShopScroll(game, id), id >= 1 && id <= 27; got != want {
				t.Errorf("game %d shop scroll spell %d admitted = %v, today %v", game, id, got, want)
			}
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

package secondshop

import (
	"testing"

	"againrom/pkg/shopstock"
)

type recordedZeroDraws struct {
	calls []int
}

func (d *recordedZeroDraws) Intn(n int) int {
	d.calls = append(d.calls, n)
	return 0
}

// TestSecondShopMageWeightedCastTakesAHundredTimesThePrice pins the second
// game's weighted cast budget (DIV-4329, SHOP-156): 100x price on a mage's
// item (powers 1..5 at price 10, base 50) and 10x on a fighter's (no power).
func TestSecondShopMageWeightedCastTakesAHundredTimesThePrice(t *testing.T) {
	spells := map[int][]int32{}
	for _, id := range castSpells[0] {
		p := make([]int32, 22)
		p[colEnchant] = 50
		spells[int(id)] = p
	}
	tb := Tables{Magic: fixtureRows{n: 50, params: map[int][]int32{tagCastSpell: effectRow(1, 1, 1, 100, nil)}},
		Spells: fixtureRows{n: 30, params: spells}}
	for _, fighter := range []bool{false, true} {
		base := shopstock.Base{Price: 10, Capacity: 100, Ceiling: 100000, Slot: 3, Fighter: fighter}
		draws := &recordedZeroDraws{}
		effects, ok := tb.enchant().Apply(base, draws)
		if fighter {
			if ok {
				t.Errorf("fighter's item: weighted cast at ten times the price gave %+v, want no effect", effects)
			}
			continue
		}
		if !ok || len(effects) != 1 || effects[0].Kind != tagCastSpell || effects[0].Operand != 1|1<<16 {
			t.Fatalf("mage's item: Apply = %+v, %v, want one cast of spell 1 at power 1", effects, ok)
		}
		if len(draws.calls) != 3 || draws.calls[1] != len(castSpells[0]) || draws.calls[2] != 5 {
			t.Errorf("mage's item: draw calls = %v, want [100 %d 5] (power drawn in 1..5)", draws.calls, len(castSpells[0]))
		}
	}
}

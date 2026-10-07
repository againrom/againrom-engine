package mapload

import (
	"testing"

	"againrom/pkg/sim"
)

func TestStoredItemPriceUsesThePublishedEffectControls(t *testing.T) {
	spells := make(itemTestCollection, 14)
	for id, scalar := range map[int]int32{1: 50, 13: 500} {
		spells[id].params = make([]int32, 21)
		spells[id].params[20] = scalar
	}
	magic := make(itemTestCollection, 3)
	magic[2].params = []int32{150}
	table := &Table{Spells: spells, Magic: magic}
	cast := func(id, power uint32) sim.ItemEffect {
		return sim.ItemEffect{Kind: 41, Operand: id | power<<16}
	}
	for _, tc := range []struct {
		name    string
		base    int32
		effects []sim.ItemEffect
		want    int32
	}{
		{"base only", 167, nil, 167},
		{"missing suffix", 167, []sim.ItemEffect{cast(1, 0)}, 667},
		{"fire low", 167, []sim.ItemEffect{cast(1, 1)}, 733},
		{"fire starting", 167, []sim.ItemEffect{cast(1, 10)}, 1659},
		{"fire high", 167, []sim.ItemEffect{cast(1, 100)}, 132001},
		{"lightning low", 167, []sim.ItemEffect{cast(13, 1)}, 5830},
		{"truncate each cast", 167, []sim.ItemEffect{cast(1, 1), cast(1, 1)}, 1299},
		{"ordinary aggregate", 167, []sim.ItemEffect{{Kind: 2, Operand: 1}}, 25548},
		{"combined", 167, []sim.ItemEffect{{Kind: 2, Operand: 1}, cast(1, 10)}, 27040},
		{"upper clamp", 9_999_900, []sim.ItemEffect{cast(1, 10)}, 9_999_999},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := storedItemPrice(tc.base, tc.effects, table); got != tc.want {
				t.Fatalf("base %d effects %+v stored price = %d, want %d", tc.base, tc.effects, got, tc.want)
			}
		})
	}
}

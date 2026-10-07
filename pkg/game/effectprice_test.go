package game

import (
	"testing"

	"againrom/pkg/mapload"
)

func TestShopCastPriceUsesThePublishedScalarPowerControls(t *testing.T) {
	for _, tc := range []struct {
		name                             string
		spell, scalar, power, draw, want int32
	}{
		{"fire low", 1, 50, 1, 0, 566},
		{"fire starting", 1, 50, 10, 0, 1492},
		{"fire high", 1, 50, 100, 0, 131834},
		{"lightning low", 13, 500, 1, 1, 5663},
	} {
		t.Run(tc.name, func(t *testing.T) {
			table := &mapload.Table{Spells: enchantSpells{int(tc.spell): tc.scalar}}
			draws := &fixedDraws{values: []int{int(tc.draw), int(tc.power - 1)}}
			got, ok := shopCastEffect(false, 10_000_000, table, draws)
			if !ok || uint16(got.effect.Operand) != uint16(tc.spell) ||
				int16(got.effect.Operand>>16) != int16(tc.power) || got.effect.Kind != 41 || got.effect.Mode != 0 {
				t.Fatalf("cast setup = %+v ok=%v", got, ok)
			}
			if got.cast != tc.want {
				t.Fatalf("scalar %d power %d addition = %d, want %d", tc.scalar, tc.power, got.cast, tc.want)
			}
		})
	}
}

func TestShopNonCastPriceUsesThePublishedPointControls(t *testing.T) {
	for _, tc := range []struct{ points, want int32 }{
		{0, 0}, {1, 100}, {10, 1029}, {70, 8750}, {150, 25381}, {700, 2053276},
	} {
		if got := shopNonCastPrice(tc.points); got != tc.want {
			t.Errorf("points %d addition = %d, want %d", tc.points, got, tc.want)
		}
	}
}

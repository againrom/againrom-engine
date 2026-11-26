package shopstock

import "testing"

func TestEffectsPriceMatchesBothGamesRows(t *testing.T) {
	magic := make(castPriceRows, 50)
	magic[5] = []int32{1}
	magic[41] = []int32{1}
	magic[44] = []int32{5}

	spells := make(castPriceRows, 20)
	spells[1] = make([]int32, 21)
	spells[1][20] = 50
	spells[18] = make([]int32, 21)
	spells[18][20] = 10

	mixed := []Effect{
		{Kind: 5, Operand: 20},
		{Kind: 44, Operand: 5 | 7<<8},
		{Kind: 41, Operand: 1 | 10<<16},
	}
	withFixed := append(append([]Effect(nil), mixed...),
		Effect{Kind: 1, Operand: 500},
		Effect{Kind: 1, Operand: 250},
	)
	neg := int16(-5)
	negPower := []Effect{
		{Kind: 5, Operand: 70},
		{Kind: 41, Operand: 1 | uint32(uint16(neg))<<16},
	}
	midCap := []Effect{{Kind: 5, Operand: 1000}}
	overBothCaps := []Effect{{Kind: 5, Operand: 1200}}

	for _, tc := range []struct {
		name               string
		base               int64
		class              ItemClass
		effects            []Effect
		wantFirst, wantSec int64
	}{
		{"mixed weapon", 1000, ItemWeapon, mixed, 12849, 12849},
		{"mixed armour", 1000, ItemArmor, mixed, 13079, 13079},
		{"mixed shield", 1000, ItemShield, mixed, 13079, 13079},
		{"fixed effect add vs replace weapon", 1000, ItemWeapon, withFixed, 13599, 500},
		{"fixed effect add vs replace armour", 1000, ItemArmor, withFixed, 13829, 500},
		{"negative cast power skipped vs unsigned", 1000, ItemWeapon, negPower, 9750, 19_999_999},
		{"between caps 9999999 and 19999999", 1000, ItemWeapon, midCap, 9_999_999, 16_440_262},
		{"above both caps", 1000, ItemWeapon, overBothCaps, 9_999_999, 19_999_999},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := EffectsPrice(tc.base, tc.effects, tc.class, magic, spells, FirstItemPrice); got != tc.wantFirst {
				t.Errorf("FirstItemPrice = %d, want %d", got, tc.wantFirst)
			}
			if got := EffectsPrice(tc.base, tc.effects, tc.class, magic, spells, SecondItemPrice); got != tc.wantSec {
				t.Errorf("SecondItemPrice = %d, want %d", got, tc.wantSec)
			}
		})
	}
}

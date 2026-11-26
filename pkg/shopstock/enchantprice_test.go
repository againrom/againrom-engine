package shopstock

import "testing"

// The effects an enchant draws price through EffectsPrice, in each game's row,
// to the prices the enchant's own running price gave before it kept none. The
// table is the base output of 144 items: both rows, a weapon and an armour,
// a mage's item, a fighter's and a forced cast, on twelve fixed sources; it
// holds ordinary effects, damage effects, casts and three-effect items.
func TestEnchantedItemsPriceThroughEffectsPrice(t *testing.T) {
	magic := make(castPriceRows, 49)
	kind := func(cost, min, max, weight int32) []int32 {
		r := []int32{cost, min, max, 3}
		for i := 0; i < 24; i++ {
			r = append(r, weight)
		}
		return r
	}
	magic[2] = kind(2, 1, 40, 40)
	magic[3] = kind(3, 1, 30, 70)
	magic[4] = kind(1, 1, 60, 100)
	magic[castSpell] = kind(1, 0, 0, 115)
	magic[44] = kind(2, 1, 60, 130)
	spells := make(castPriceRows, 3)
	for i := 1; i < 3; i++ {
		spells[i] = make([]int32, 21)
		spells[i][20] = int32(30 * i)
	}
	source := func(i int) *seq {
		x := uint32(i*7919 + 13)
		v := make([]int, 40)
		for k := range v {
			x = x*1664525 + 1013904223
			v[k] = int(x >> 8 % 1000)
		}
		return &seq{values: v}
	}
	want := [][]int32{
		{10039, 12857, 3110, 70229, 13024, 11880, 18401, 8891, 4849, 6270, 9172, 4467},
		{10039, 12857, 3110, 8105, 13024, 11880, 18401, 8891, 4849, 6270, 9172, 4467},
		{77073, 2883, 60847, 68594, 74882, 2839, 59020, 66590, 3700, 32571, 25187, 19235},
		{10039, 12857, 3110, 11880, 13024, 11880, 18401, 8891, 4849, 6270, 9172, 3636},
		{10039, 12857, 3110, 5078, 13024, 11880, 18401, 8891, 4849, 6270, 9172, 3636},
		{16044, 2701, 14579, 15301, 15856, 2600, 14401, 15118, 3110, 8891, 8069, 7278},
		{10039, 12857, 3110, 98520, 13024, 11880, 18401, 8891, 4849, 6270, 9172, 4467},
		{10039, 12857, 3110, 8105, 13024, 11880, 18401, 8891, 4849, 6270, 9172, 4467},
		{4291, 3578, 31673, 25558, 19681, 15594, 3313, 2931, 3700, 32571, 25187, 19235},
		{10039, 12857, 3110, 13532, 13024, 11880, 18401, 8891, 4849, 6270, 9172, 3636},
		{10039, 12857, 3110, 5078, 13024, 11880, 18401, 8891, 4849, 6270, 9172, 3636},
		{4398, 3743, 11250, 10336, 9314, 8476, 3424, 2802, 3110, 8891, 8069, 7278},
	}
	line := 0
	for _, row := range []ItemPriceRow{FirstItemPrice, SecondItemPrice} {
		for _, class := range []ItemClass{ItemWeapon, ItemArmor} {
			for mode := 0; mode < 3; mode++ {
				for i := 0; i < 12; i++ {
					e := Enchant{Effects: magic, Spells: spells, CastSpells: [2][]uint16{{1, 2}, {2}},
						CastPriceTimes: [2]int64{100, 10}, Row: row, CastInCapacity: row.Fixed == FixedReplace,
						Repeat: RepeatReplace}
					b := Base{Price: 2500, Capacity: 90, Ceiling: 60000, Slot: 3, Fighter: mode == 1, Forced: mode == 2, Class: class}
					got, ok := e.Apply(b, source(i))
					if !ok {
						t.Fatalf("row %d class %d mode %d source %d: no effects", line, class, mode, i)
					}
					if price := int32(EffectsPrice(int64(b.Price), got, class, magic, spells, row)); price != want[line][i] {
						t.Errorf("row %d class %d mode %d source %d: price %d, want %d", line, class, mode, i, price, want[line][i])
					}
				}
				line++
			}
		}
	}
}

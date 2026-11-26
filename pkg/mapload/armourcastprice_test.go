package mapload

import (
	"fmt"
	"math"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// A castSpell effect of spell 1 (enchant base 50) at power 10 on an item of
// base price 0. A weapon prices the cast apart; an armour and a shield sum
// its power times Magic row 41's cost 1 into the points and pay the one
// surcharge trunc((1.5^(N/70)+1) x N x 50) (SHOP-158).
func TestArmourAndShieldSumACastsPowerIntoThePoints(t *testing.T) {
	table := castPriceTable()
	effects := []sim.ItemEffect{{Kind: 41, Operand: 1 | 10<<16}}
	weapon := int32(10 * 50 * math.Pow(2, math.Log(1+10.0/30)/math.Log(1.2)))
	summed := int32((math.Pow(1.5, 10.0/70) + 1) * 10 * 50)
	for _, c := range []struct {
		name  string
		class data.ItemCode
		want  int32
	}{
		{"weapon", 1, weapon},
		{"shield", 2, summed},
		{"armour", 3, summed},
	} {
		if got := equipmentInstance(c.class<<8|1, effects, nil, table).Price; got != c.want {
			t.Errorf("%s: placed price %d, want %d", c.name, got, c.want)
		}
	}
}

func castPriceTable() *Table {
	spells := itemTestCollection{{}, {name: "Fire Arrow", params: make([]int32, 21)}}
	spells[1].params[20] = 50
	magic := make(itemTestCollection, 42)
	magic[41].params = []int32{1}
	return &Table{
		Shapes: itemTestScale(1), Materials: itemTestScale(16),
		Shields: itemTestCollection{{}, {name: "Buckler", params: []int32{0, 0, 12}}},
		Armors:  itemTestCollection{{}, {name: "Coat", params: []int32{0, 0, 50, 0, 7}}},
		Spells:  spells, Magic: magic,
	}
}

func TestArmourAndShieldCastPriceByEdition(t *testing.T) {
	for _, tc := range []struct {
		game          base.Game
		power         int
		buckler, coat int32
	}{
		{base.GameROM2, 10, 1504, 1542},
		{base.GameROM2, 100, 131846, 131884},
		{base.GameROM1, 10, 1041, 1079},
		{base.GameROM1, 6000, 9_999_999, 9_999_999},
		{base.GameROM1, 32767, 9_999_999, 9_999_999},
	} {
		t.Run(fmt.Sprintf("%s/%d", tc.game, tc.power), func(t *testing.T) {
			edition := tc.game.Edition()
			table := castPriceTable()
			table.Edition = &edition
			for _, piece := range []struct {
				name string
				want int32
			}{{"Buckler", tc.buckler}, {"Coat", tc.coat}} {
				cell := fmt.Sprintf("%s {castSpell=Fire_Arrow:%d}", piece.name, tc.power)
				item, ok := CheatItem(cell, table)
				if !ok {
					t.Fatalf("could not create %s", cell)
				}
				created := item.Price
				item.Price = -1
				if repriced := RepriceItemInstance(item, table); created != piece.want || repriced != piece.want {
					t.Errorf("%s: created %d, repriced %d, want %d", cell, created, repriced, piece.want)
				}
			}
		})
	}
}

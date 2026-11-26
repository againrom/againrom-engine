package shopstock

import (
	"math"
	"testing"
)

type castPriceRows [][]int32

func (r castPriceRows) Len() int                  { return len(r) }
func (r castPriceRows) EntryParams(i int) []int32 { return r[i] }

// A forced cast of spell 1 (enchant base 50) on an item of base price 1000.
// A weapon adds the cast's own price; an armour or a shield sums the cast's
// power times the cast row's cost 1 into the points and pays the one
// surcharge on them (SHOP-158).
func TestShopArmourCastEntersThePointsAndAWeaponCastIsPricedApart(t *testing.T) {
	effects := make(castPriceRows, castSpell+1)
	effects[castSpell] = []int32{1}
	spells := make(castPriceRows, 2)
	spells[1] = make([]int32, 21)
	spells[1][20] = 50
	e := Enchant{
		Effects: effects, Spells: spells, CastSpells: [2][]uint16{{1}, {1}}, Row: FirstItemPrice,
	}
	for _, armour := range []bool{false, true} {
		class := ItemWeapon
		if armour {
			class = ItemArmor
		}
		b := Base{Price: 1000, Capacity: 100, Ceiling: 100000, Slot: 1, Forced: true, Class: class}
		got, ok := e.Apply(b, &seq{values: []int{0, 9}})
		if !ok || len(got) != 1 || got[0].Kind != castSpell {
			t.Fatalf("armour %v: effects %+v ok %v, want one cast", armour, got, ok)
		}
		price := int32(EffectsPrice(int64(b.Price), got, class, effects, spells, e.Row))
		power := float64(int16(got[0].Operand >> 16))
		want := int32(1000 + int64(10*50*math.Pow(2, math.Log(1+power/30)/math.Log(1.2))))
		if armour {
			want = int32(1000 + int64((math.Pow(1.5, power/70)+1)*power*50))
		}
		if price != want {
			t.Errorf("armour %v: cast power %v price %d, want %d", armour, power, price, want)
		}
	}
}

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
		Effects: effects, Spells: spells, CastSpells: [2][]uint16{{1}, {1}}, PriceCap: 9_999_999,
		PointsPrice: func(points []int32) int64 {
			n := int32(0)
			for _, p := range points {
				n += p
			}
			if n <= 0 {
				return 0
			}
			return int64((math.Pow(1.5, float64(n)/70) + 1) * float64(n) * 50)
		},
	}
	for _, armour := range []bool{false, true} {
		b := Base{Price: 1000, Capacity: 100, Ceiling: 100000, Slot: 1, Forced: true, CastInPoints: armour}
		got, price, ok := e.Apply(b, &seq{values: []int{0, 9}})
		if !ok || len(got) != 1 || got[0].Kind != castSpell {
			t.Fatalf("armour %v: effects %+v ok %v, want one cast", armour, got, ok)
		}
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

package secondshop

import (
	"testing"

	"againrom/pkg/sim"
)

// An equipment price adds one surcharge on the sum of its ordinary effects'
// sizes, not one per effect, and the surcharge is trunc((1.5^(S/70)+1)*S*50)
// (R2-ENGINE-1764, R2-ENGINE-1824). The numbers are the claim's.
func TestSecondShopPriceAddsOneSurchargeOnTheSummedSize(t *testing.T) {
	tb := enchantTables()
	effect := func(n int) sim.ItemEffect { return sim.ItemEffect{Kind: 5, Operand: uint32(n)} }
	plain := tb.Price(Item{Table: Weapon, Row: 1})
	for _, c := range []struct {
		name    string
		effects []sim.ItemEffect
		want    int64
	}{
		{"one of 70", []sim.ItemEffect{effect(70)}, 8750},
		{"two of 70", []sim.ItemEffect{effect(70), effect(70)}, 22750},
		{"two of 35", []sim.ItemEffect{effect(35), effect(35)}, 8750},
		{"one of 10", []sim.ItemEffect{effect(10)}, 1029},
		{"three of 70", []sim.ItemEffect{effect(70), effect(70), effect(70)}, 45937},
	} {
		if got := tb.Price(Item{Table: Weapon, Row: 1, Effects: c.effects}) - plain; got != c.want {
			t.Errorf("%s: effects price %d, want %d", c.name, got, c.want)
		}
	}
	for s, want := range map[int]int64{35: 3893, 105: 14894, 140: 22750, 210: 45937, 280: 84875} {
		if got := EffectPrice(s); got != want {
			t.Errorf("surcharge of size %d is %d, want %d", s, got, want)
		}
	}
	it := Item{Table: Weapon, Row: 1, Effects: []sim.ItemEffect{effect(1 << 30), effect(1 << 30)}}
	if got := tb.Price(it); got != priceCap {
		t.Errorf("huge sum: price %d, want the cap %d", got, priceCap)
	}
}

// A weapon's cast adds its spell's own price apart from the surcharge; an
// armour's or a shield's cast enters the summed size with its level word as
// its value (R2-ENGINE-1764, R2-ENGINE-1824).
func TestSecondShopArmourCastEntersTheSummedSize(t *testing.T) {
	tb := enchantTables()
	tb.Armors = tb.Weapons
	tb.Shields = tb.Weapons
	cast := []sim.ItemEffect{{Kind: tagCastSpell, Operand: 18 | 70<<16}}
	weapon := tb.Price(Item{Table: Weapon, Row: 1, Effects: cast}) - tb.Price(Item{Table: Weapon, Row: 1})
	if want := SpellEffectPrice(10, 70); weapon != want {
		t.Errorf("weapon cast adds %d, want its spell price %d", weapon, want)
	}
	for _, k := range []Kind{Armor, Shield} {
		got := tb.Price(Item{Table: k, Row: 1, Effects: cast}) - tb.Price(Item{Table: k, Row: 1})
		if want := EffectPrice(70); got != want {
			t.Errorf("table %d: cast of level 70 adds %d, want the surcharge of size 70, %d", k, got, want)
		}
	}
}

// The price walk sets an equipment price to the value of its first tag 1
// effect and returns at once, whatever the base and the other effects
// (R2-ENGINE-1764).
func TestSecondShopTagOneEffectReplacesTheEquipmentPrice(t *testing.T) {
	tb := enchantTables()
	tb.Armors = tb.Weapons
	tb.Shields = tb.Weapons
	effects := []sim.ItemEffect{{Kind: tagCastSpell, Operand: 18 | 70<<16}, {Kind: 1, Operand: 1234}, {Kind: 1, Operand: 99}}
	for _, k := range []Kind{Armor, Shield, Weapon} {
		if got := tb.Price(Item{Table: k, Row: 1, Effects: effects}); got != 1234 {
			t.Errorf("table %d: price %d, want 1234", k, got)
		}
	}
}

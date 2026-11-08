package secondshop

import (
	"reflect"
	"testing"

	"againrom/pkg/random"
	"againrom/pkg/shopstock"
)

func rollsOf(factor int) Rolls {
	return Rolls{Factor: factor, Source: func() shopstock.Source { return random.NewGo(77) }}
}

func multiplierRecord() Record {
	worn := Category{Min: 0, Max: 100000, Draws: 8, Mask: kindArmor | kindShield | bitEnchanted | 1<<classShift | 1<<1}
	held := Category{Min: 0, Max: 100000, Draws: 6, Mask: kindWeapon | bitEnchanted | 1<<classShift | 1<<1}
	return Record{worn, held}
}

// With a factor above one, every rolled category keeps its first roll and
// adds the further rolls' items, and the shared draw source is consumed as
// with one roll.
func TestSecondShopMultiplierRollsEveryCategoryAgain(t *testing.T) {
	tb := enchantTables()
	for seed := uint32(1); seed <= 10; seed++ {
		g, h := &random.MSVC{State: seed}, &random.MSVC{State: seed}
		base := tb.Fill(multiplierRecord(), func() int { return int(g.Rand()) })
		tb.Rolls = rollsOf(3)
		got := tb.Fill(multiplierRecord(), func() int { return int(h.Rand()) })
		tb.Rolls = Rolls{}
		if g.State != h.State {
			t.Fatalf("seed %d: the shared source ended at %d, with one roll %d", seed, h.State, g.State)
		}
		for k := 0; k < 2; k++ {
			if len(base[k]) == 0 {
				t.Fatalf("seed %d category %d: empty first roll", seed, k)
			}
			if len(got[k]) != 3*len(base[k]) {
				t.Fatalf("seed %d category %d: %d items, want %d (3 rolls of %d)", seed, k, len(got[k]), 3*len(base[k]), len(base[k]))
			}
			for _, it := range base[k] {
				found := false
				for _, o := range got[k] {
					found = found || reflect.DeepEqual(o, it)
				}
				if !found {
					t.Fatalf("seed %d category %d: first-roll item %+v lost", seed, k, it)
				}
			}
		}
	}
}

func TestSecondShopFillKeyingIsLinearInTheFactor(t *testing.T) {
	tb := enchantTables()
	count := func(factor int) int {
		calls := 0
		key := Stacking.Key
		Stacking.Key = func(it Item) stackKey { calls++; return key(it) }
		defer func() { Stacking.Key = key }()
		g := &random.MSVC{State: 9}
		tb.Rolls = rollsOf(factor)
		tb.Fill(multiplierRecord(), func() int { return int(g.Rand()) })
		return calls
	}
	small, large := count(40), count(80)
	t.Logf("keys asked: %d at factor 40, %d at factor 80", small, large)
	if large > 5*small/2 {
		t.Fatalf("keys asked grew from %d at factor 40 to %d at factor 80, want at most 2.5 times", small, large)
	}
}

// A factor of one or none is today's stock, and a death's item list is never
// multiplied.
func TestSecondShopMultiplierOffAndDeathItems(t *testing.T) {
	tb := enchantTables()
	mask := uint32(kindWeapon | bitEnchanted | 1<<classShift | 1<<1)
	for seed := uint32(1); seed <= 10; seed++ {
		g, h, d, e := &random.MSVC{State: seed}, &random.MSVC{State: seed}, &random.MSVC{State: seed}, &random.MSVC{State: seed}
		base := tb.Fill(multiplierRecord(), func() int { return int(g.Rand()) })
		tb.Rolls = rollsOf(1)
		one := tb.Fill(multiplierRecord(), func() int { return int(h.Rand()) })
		if !reflect.DeepEqual(base, one) || g.State != h.State {
			t.Fatalf("seed %d: a factor of one changed the stock", seed)
		}
		plain := tb.DeathItems(0, 100000, mask, func() int { return int(d.Rand()) })
		tb.Rolls = rollsOf(3)
		death := tb.DeathItems(0, 100000, mask, func() int { return int(e.Rand()) })
		tb.Rolls = Rolls{}
		if !reflect.DeepEqual(plain, death) || d.State != e.State {
			t.Fatalf("seed %d: the death's list changed with a factor of three", seed)
		}
	}
}

// The magic category's books at quantity 2 and six potions are taken first
// and stay once; the further rolls add only drawn items.
func TestSecondShopMultiplierKeepsTheLeadOnce(t *testing.T) {
	tb := enchantTables()
	tb.MagicItems = fixtureRows{n: 80, params: map[int][]int32{69: {10}, 70: {10}, 71: {10}, 72: {10}, 73: {10}, 74: {10}, 60: {10}}}
	rec := Record{{Min: 0, Max: 100000, Draws: 4, Bound: 1, Mask: kindMagic}}
	count := func(s Stock) (lead, all int) {
		for _, it := range s[0] {
			all += int(it.Count)
			if it.Row >= potionFirst && it.Row <= potionLast && it.Count >= potionLeast {
				lead++
			}
		}
		return
	}
	g, h := &random.MSVC{State: 5}, &random.MSVC{State: 5}
	base := tb.Fill(rec, func() int { return int(g.Rand()) })
	tb.Rolls = rollsOf(3)
	got := tb.Fill(rec, func() int { return int(h.Rand()) })
	bl, bu := count(base)
	gl, gu := count(got)
	if bl != 6 || gl != 6 {
		t.Fatalf("lead potions %d with one roll and %d with three, want 6 and 6", bl, gl)
	}
	if gu <= bu {
		t.Fatalf("the magic category held %d units with three rolls, %d with one", gu, bu)
	}
}

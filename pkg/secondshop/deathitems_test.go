package secondshop

import (
	"reflect"
	"testing"

	"againrom/pkg/random"
	"againrom/pkg/shopstock"
	"againrom/pkg/sim"
)

// A death's item list is the shop draw of the one category {min, max, 100
// draws, bound 1, mask} (R2-ENGINE-941): the same entries, as bag units, on
// the same draws as a shop record holding that category alone.
func TestDeathItemsAreTheShopDrawOfOneCategory(t *testing.T) {
	tb := enchantTables()
	for _, mask := range []uint32{kindArmor | kindShield | 1<<classShift | 1<<1,
		kindWeapon | bitEnchanted | 1<<classShift | 1<<1} {
		for seed := uint32(1); seed <= 10; seed++ {
			g, h := &random.MSVC{State: seed}, &random.MSVC{State: seed}
			got := tb.DeathItems(0, 100000, mask, func() int { return int(g.Rand()) })
			shelf := tb.Fill(Record{{Min: 0, Max: 100000, Draws: deathDraws, Bound: deathBound, Mask: mask}},
				func() int { return int(h.Rand()) })[0]
			if len(got) == 0 || len(got) != len(shelf) || g.State != h.State {
				t.Fatalf("mask %#x seed %d: %d entries, shelf %d; states %d and %d", mask, seed, len(got), len(shelf), g.State, h.State)
			}
			for i, it := range shelf {
				if want, _ := tb.Instance(it); !reflect.DeepEqual(got[i], want) {
					t.Fatalf("mask %#x seed %d entry %d: %+v, want %+v", mask, seed, i, got[i], want)
				}
			}
		}
	}
}

// R2-ENGINE-941: no candidate leaves the list empty, before the magic lead.
func TestDeathItemsOfNoCandidate(t *testing.T) {
	tb := enchantTables()
	tb.MagicItems = fixtureRows{n: 80, params: map[int][]int32{69: {10}, 70: {10}, 71: {10}, 72: {10}, 73: {10}, 74: {10}}}
	for _, mask := range []uint32{kindMagic, kindMagic | bitEnchanted, 0x64, 0} {
		g := &random.MSVC{State: 3}
		got := tb.DeathItems(1, 2, mask, func() int { return int(g.Rand()) })
		if len(got) != 0 {
			t.Fatalf("mask %#x: %d entries, want none", mask, len(got))
		}
	}
}

func TestDeathItemsInstalledOgreFailureBound(t *testing.T) {
	tb := installTables(t)
	for _, tc := range []struct {
		min, max         int32
		entries          int
		generator, death uint32
		gold             int32
	}{
		{300, 1500, 5, 2347574015, 3265786310, 573},
		{2100, 10050, 99, 3268641339, 1827980306, 3882},
	} {
		g := &random.MSVC{State: 1}
		if g.Range(100) >= 90 {
			t.Fatal("gold chance failed")
		}
		gold := tc.min/2 + g.Range((tc.max+1)/2)
		if g.Range(100) >= 40 {
			t.Fatal("item chance failed")
		}
		cat := Category{Min: tc.min, Max: tc.max, Draws: 100, Bound: 1, Mask: 0x21c7ffff}
		p := tb.Profile(Record{cat}, func() int { return int(g.Rand()) })
		p.Categories, p.Place = p.Categories[:1], nil
		failed := 0
		makeItem := p.Categories[0].Make
		p.Categories[0].Make = func(it Item, src shopstock.Source) (Item, bool) {
			it, ok := makeItem(it, src)
			if !ok {
				failed++
			}
			return it, ok
		}
		list := shopstock.Generate(p, picks(func() int { return int(g.Rand()) }))[0]
		if failed != 1001 || len(list) != tc.entries || g.State != tc.generator || gold != tc.gold {
			t.Fatalf("R2-ENGINE-941 window %d..%d: failures %d entries %d RNG %d gold %d", tc.min, tc.max, failed, len(list), g.State, gold)
		}
		script, err := sim.NewROM2Script(nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		w, err := sim.NewStockedWorld(1, sim.Bounds{Width: 12, Height: 12}, sim.ModeCanonical, sim.Terrain{},
			[]sim.Entity{{ID: 1, X: 4, Y: 5, HP: 10, MaxHP: 10, TypeID: 66, GoldChance: 90,
				TreasureMin: tc.min / 2, TreasureMax: (tc.max + 1) / 2,
				DeathItem: sim.DeathItemRoll{Chance: 40, Min: tc.min, Max: tc.max, Mask: cat.Mask}}}, script, sim.Relations{}, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		w.SetRandom(random.Original, 1)
		w.SetDeathItemDraw(func(r sim.DeathItemRoll, rnd func() int) []sim.ItemInstance {
			return tb.DeathItems(r.Min, r.Max, r.Mask, rnd)
		})
		sim.Step(w, []sim.Command{sim.TerminalKill(1)})
		if w.RandomState() != uint64(tc.death) {
			t.Fatalf("death RNG %d, want %d", w.RandomState(), tc.death)
		}
		if sacks := w.Sacks(); len(sacks) != 1 || sacks[0].Gold != uint32(tc.gold) || len(sacks[0].ItemInstances) != 1 {
			t.Fatalf("sacks %+v, want gold %d and one item", sacks, tc.gold)
		}
	}
}

// With bit 28 beside bit 29 a drawn unit is enchanted only on a rand(100)
// above 50; a plain unit is kept inside the price range too.
func TestHalfEnchantLeavesSomeUnitsPlain(t *testing.T) {
	tb := enchantTables()
	cat := Category{Min: 0, Max: 100000, Draws: 50, Mask: kindWeapon | bitEnchanted | bitHalfEnchant | 1<<classShift | 1<<1}
	plain, enchanted := 0, 0
	for seed := uint32(1); seed <= 20; seed++ {
		g := &random.MSVC{State: seed}
		for _, it := range tb.Fill(Record{cat}, func() int { return int(g.Rand()) })[0] {
			if it.Price < cat.Min || it.Price > cat.Max {
				t.Fatalf("seed %d: %+v outside the range", seed, it)
			}
			if len(it.Effects) == 0 {
				plain++
			} else {
				enchanted++
			}
		}
	}
	if plain == 0 || enchanted == 0 {
		t.Fatalf("%d plain and %d enchanted units, want both", plain, enchanted)
	}
	full := cat
	full.Mask &^= bitHalfEnchant
	g := &random.MSVC{State: 1}
	for _, it := range tb.Fill(Record{full}, func() int { return int(g.Rand()) })[0] {
		if len(it.Effects) == 0 {
			t.Fatalf("bit 29 alone left %+v plain", it)
		}
	}
}

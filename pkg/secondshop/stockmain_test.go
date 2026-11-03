package secondshop

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/random"
	"againrom/pkg/shopstock"
)

// The frozen digest checks stock contents and draws through the old stable
// server-order projection. CRT permutations are checked on planted lists.
func TestReleaseSecondShopStockMatchesMain(t *testing.T) {
	const mainDigest = "a20725ff6df410ca78237840959cae97c807d5b7556f6eb97a2e5cbeb8cb28f2"
	if got := stockDigest(t, installTables(t)); got != mainDigest {
		t.Fatalf("stock digest %s, want main %s", got, mainDigest)
	}
}

func stockDigest(t *testing.T, tb Tables) string {
	t.Helper()
	h := sha256.New()
	for _, loc := range []int{1, 2} {
		for seed := uint32(1); seed <= 100; seed++ {
			g := &random.MSVC{State: seed}
			rnd := func() int { return int(g.Rand()) }
			actual := tb.Fill(NewGameRecords[loc], rnd)
			next := rnd()
			before := &random.MSVC{State: seed}
			oldRand := func() int { return int(before.Rand()) }
			p := tb.Profile(NewGameRecords[loc], oldRand)
			for i := range p.Categories {
				p.Categories[i].Finish = nil
			}
			var projected Stock
			copy(projected[:], shopstock.Generate(p, picks(oldRand)))
			if next != oldRand() || stockContents(actual) != stockContents(projected) {
				t.Fatalf("town %d seed %d: server sort changed stock contents or draws", loc, seed)
			}
			for _, list := range projected {
				shopstock.SortBy(list, func(it Item) []int64 {
					if it.Table == Armor || it.Table == Shield || it.Table == Weapon {
						flag := int64(1)
						if tb.fighter(it) {
							flag = 0
						}
						return []int64{flag, int64(it.Row), int64(it.Price)}
					}
					return []int64{1, 0, int64(it.Price)}
				})
			}
			fmt.Fprintf(h, "%d %d %s %d\n", loc, seed, stockText(projected), next)
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func stockContents(s Stock) string {
	var out strings.Builder
	for k, list := range s {
		rows := make([]string, len(list))
		for i, it := range list {
			rows[i] = stockText(Stock{[]Item{it}})
		}
		slices.Sort(rows)
		fmt.Fprintf(&out, "%d:%q;", k, rows)
	}
	return out.String()
}

// stockText is s as the digest was taken: each item's table identity,
// effects, quantity and price.
func stockText(s Stock) string {
	var b strings.Builder
	b.WriteByte('[')
	for k, list := range s {
		if k > 0 {
			b.WriteByte(' ')
		}
		b.WriteByte('[')
		for i, it := range list {
			if i > 0 {
				b.WriteByte(' ')
			}
			fmt.Fprintf(&b, "{Kind:%d Row:%d Class:%d Material:%d Spell:%d Effects:%v Quantity:%d Price:%d}",
				it.Table, it.Row, it.Class, it.Material, it.Spell, it.Effects, it.Count, it.Price)
		}
		b.WriteByte(']')
	}
	b.WriteByte(']')
	return b.String()
}

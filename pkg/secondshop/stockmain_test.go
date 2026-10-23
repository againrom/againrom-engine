package secondshop

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/random"
)

// TestReleaseSecondShopStockMatchesMain holds the town shops' stock and the
// draws it takes to the values the second game's own fill drew before the
// shared generator: records 1 and 2, seeds 1..100 of the original's rand(),
// each stock in order and the next draw after it, hashed.
func TestReleaseSecondShopStockMatchesMain(t *testing.T) {
	tb := installTables(t)
	h := sha256.New()
	for _, loc := range []int{1, 2} {
		for seed := uint32(1); seed <= 100; seed++ {
			g := &random.MSVC{State: seed}
			rnd := func() int { return int(g.Rand()) }
			s := tb.Fill(NewGameRecords[loc], rnd)
			fmt.Fprintf(h, "%d %d %s %d\n", loc, seed, stockText(s), rnd())
		}
	}
	const mainDigest = "c21b9ef87d5730eedae94525391c0a1705b0d98bb1c081f800571ed2ee532f8d"
	if got := fmt.Sprintf("%x", h.Sum(nil)); got != mainDigest {
		t.Fatalf("stock digest %s, want main %s", got, mainDigest)
	}
}

// stockText is s as the digest was taken: each item's table identity,
// quantity and price, in the form fmt's %+v gave the second game's own item.
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
			fmt.Fprintf(&b, "{Kind:%d Row:%d Class:%d Material:%d Spell:%d Level:%d Quantity:%d Price:%d}",
				it.Table, it.Row, it.Class, it.Material, it.Spell, it.Level, it.Count, it.Price)
		}
		b.WriteByte(']')
	}
	b.WriteByte(']')
	return b.String()
}

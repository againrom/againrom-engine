package secondshop

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/random"
)

// TestReleaseSecondShopStockMatchesMain holds the town shops' stock and the
// draws it takes to a recorded digest of the fill with the claims' enchant,
// price and shelf order (R2-ENGINE-710, R2-ENGINE-711).
func TestReleaseSecondShopStockMatchesMain(t *testing.T) {
	const mainDigest = "a20725ff6df410ca78237840959cae97c807d5b7556f6eb97a2e5cbeb8cb28f2"
	if got := stockDigest(installTables(t)); got != mainDigest {
		t.Fatalf("stock digest %s, want main %s", got, mainDigest)
	}
}

// stockDigest hashes records 1 and 2 over seeds 1..100 of the original's
// rand(): each stock in order and the next draw after it.
func stockDigest(tb Tables) string {
	h := sha256.New()
	for _, loc := range []int{1, 2} {
		for seed := uint32(1); seed <= 100; seed++ {
			g := &random.MSVC{State: seed}
			rnd := func() int { return int(g.Rand()) }
			s := tb.Fill(NewGameRecords[loc], rnd)
			fmt.Fprintf(h, "%d %d %s %d\n", loc, seed, stockText(s), rnd())
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))
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

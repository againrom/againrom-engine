package secondshop

import (
	"testing"

	"againrom/pkg/random"
)

func TestReleaseNoFilledStockHoldsABookOfSlow(t *testing.T) {
	tb := installTables(t)
	for _, r := range NewGameRecords {
		for seed := uint32(1); seed <= 200; seed++ {
			g := &random.MSVC{State: seed}
			for _, list := range tb.Fill(r, func() int { return int(g.Rand()) }) {
				for _, it := range list {
					if it.Table == Book && it.Spell == 29 {
						t.Fatalf("seed %d stocks a book of Slow: %+v", seed, it)
					}
				}
			}
		}
	}
}

func TestNoShopStockOffersABookOfSlow(t *testing.T) {
	cands := Tables{}.Candidates(Category{Min: -1, Max: 1 << 30, Mask: kindMagic}, nil)
	books, scrolls := 0, 0
	for _, it := range cands {
		if it.Table == Book {
			books++
			if it.Spell == 29 {
				t.Fatalf("candidate %+v is a book of Slow", it)
			}
		}
		if it.Table == Magic && it.Row == 29+scrollOffset {
			scrolls++
		}
	}
	if books == 0 || scrolls != 1 {
		t.Fatalf("books %d, scrolls of Slow %d; want books and one scroll of Slow", books, scrolls)
	}
}

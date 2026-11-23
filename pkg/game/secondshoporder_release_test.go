package game

import (
	"testing"

	"againrom/pkg/random"
	"againrom/pkg/secondshop"
)

// shownKeyGroup is the weapon row group of a kind-1 item; every other kind
// has group 1.
func shownKeyGroup(it secondshop.Item, kind int) int {
	if kind != 1 {
		return 1
	}
	switch r := it.Row & 0x1f; {
	case r >= 1 && r <= 6:
		return 1
	case r >= 18 && r <= 19:
		return 2
	case r >= 15 && r <= 17:
		return 3
	case r >= 7 && r <= 12:
		return 5
	case r >= 20 && r <= 22:
		return 6
	}
	return 7
}

// TestReleaseSecondShopShownOrderFollowsTheClientKeys fills the NewGame
// shops of towns 1 and 2 on the installed root and checks each shown shelf
// is ordered by flag 0x2 set first, kind, weapon group and shown price
// (R2-ENGINE-798, and its RU bodies R2-ENGINE-1382).
func TestReleaseSecondShopShownOrderFollowsTheClientKeys(t *testing.T) {
	f := secondGameFront(t)
	tb, ok := f.ShopTables()
	if !ok {
		t.Fatal("the root has no shop tables")
	}
	flagged, shelves, items := 0, 0, 0
	for _, loc := range []int{1, 2} {
		for seed := uint32(1); seed <= 40; seed++ {
			g := &random.MSVC{State: seed}
			stock := tb.Fill(secondshop.NewGameRecords[loc], func() int { return int(g.Rand()) })
			for k, list := range stock {
				shown := append([]secondshop.Item(nil), list...)
				tb.Sort(shown)
				if len(shown) != len(list) {
					t.Fatalf("town %d seed %d tab %d: sort changed the count", loc, seed, k)
				}
				shelves++
				items += len(shown)
				for i := 1; i < len(shown); i++ {
					a, b := shown[i-1], shown[i]
					ka, kb := tb.ShownKind(a), tb.ShownKind(b)
					fa, fb := tb.ShownFlag(a), tb.ShownFlag(b)
					if fb {
						flagged++
					}
					switch {
					case fa != fb:
						if !fa {
							t.Fatalf("town %d seed %d tab %d cell %d: unflagged item before a flagged one", loc, seed, k, i)
						}
					case ka != kb:
						if ka > kb {
							t.Fatalf("town %d seed %d tab %d cell %d: kind %d before %d", loc, seed, k, i, ka, kb)
						}
					case shownKeyGroup(a, ka) != shownKeyGroup(b, kb):
						if shownKeyGroup(a, ka) > shownKeyGroup(b, kb) {
							t.Fatalf("town %d seed %d tab %d cell %d: weapon group order", loc, seed, k, i)
						}
					case tb.ShownPrice(a) > tb.ShownPrice(b):
						t.Fatalf("town %d seed %d tab %d cell %d: price %d before %d", loc, seed, k, i, tb.ShownPrice(a), tb.ShownPrice(b))
					}
				}
			}
		}
	}
	if shelves == 0 || items == 0 {
		t.Fatal("no shelf was checked")
	}
	t.Logf("%d shelves, %d items, %d flagged pairs", shelves, items, flagged)
}

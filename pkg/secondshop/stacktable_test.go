package secondshop

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

// stackTableRecord is the second game's stock on the scroll fixture at a
// seed and roll factor, with or without a mod's stocked scroll: a digest of
// the stream state after the fill, every row but the scroll rows, and each
// scroll row with its quantity. A change of the stack table's potion, scroll
// or other stacking row moves it.
var stackTableRecord = []struct {
	seed    uint32
	factor  int
	stocked bool
	digest  string
}{
	{1, 1, false, "5d402c8d667dee7f"}, {2, 1, false, "6a3aae9298b57d78"}, {3, 1, false, "35c96dcb3be15a22"}, {4, 1, false, "377f8ba468dc438a"},
	{5, 1, false, "fb7401bec14863fb"}, {6, 1, false, "b8ac27bdab36c6b2"}, {7, 1, false, "38b7bcdd8ef1893e"}, {8, 1, false, "e8d9f0aebbed646e"},
	{1, 3, false, "23eabfdea413f5c3"}, {2, 3, false, "ce6fdd875aacfaea"}, {3, 3, false, "4b943410d618722a"}, {4, 3, false, "791f6c67897b009e"},
	{5, 3, false, "14a040ca6537cc7b"}, {6, 3, false, "a71e3a16d6745566"}, {7, 3, false, "4c2dedcd5df395dd"}, {8, 3, false, "cd65211bd87eb912"},
	{1, 1, true, "5d3a8383cda06990"}, {2, 1, true, "3351ac617e91539a"}, {3, 3, true, "5cbb9431b0c3807d"}, {4, 3, true, "2cc88e6def463e16"},
}

func TestSecondShopStockFollowsTheStackTable(t *testing.T) {
	for _, w := range stackTableRecord {
		tb := scrollStackTables()
		if w.stocked {
			tb.Stocked = []Item{{Table: Magic, Row: scrollOffset + 1, Count: 1, Price: 60}}
		}
		line, _ := scrollStackLine(tb, w.seed, w.factor)
		if got := fmt.Sprintf("%x", sha256.Sum256([]byte(line)))[:16]; got != w.digest {
			t.Errorf("seed %d x%d stocked %t: digest %s, want %s: %s", w.seed, w.factor, w.stocked, got, w.digest, line)
		}
	}
}

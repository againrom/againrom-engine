package game

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

// shopStackTableRecord is the first game's stock on the scroll fixture at a
// seed and stock multiplier: a digest of the stream position after the fill,
// its next number, every row but the scroll rows, and each scroll row with
// its quantity. A change of the stack table's potion or scroll row moves it.
var shopStackTableRecord = []struct {
	seed   int64
	m      int32
	digest string
}{
	{1, 1, "f677e654d3a71693"}, {2, 1, "a952cde626e02b63"}, {3, 1, "c7c57b8fcefff672"}, {4, 1, "b253b2bdd5192089"},
	{5, 1, "3aeb92386b1ca6bf"}, {6, 1, "398e3ac9d0b455ea"}, {7, 1, "c7cb26bebd8336e6"}, {8, 1, "042d8aebe43e4bce"},
	{9, 1, "e42ae4b244485244"}, {10, 1, "bc0415f10796baf7"}, {11, 1, "eb8b0b2d84ce81ae"}, {12, 1, "82dc4cc7549edea4"},
	{1, 2, "4c8650ab3ef8c61f"}, {2, 2, "82da579bedfbb498"}, {3, 2, "1f5df9bca49286da"}, {4, 2, "189ab642b191f11f"},
	{5, 2, "64f57703e1b5f925"}, {6, 2, "b578bbf21bd213eb"}, {7, 2, "0202c3bed819c196"}, {8, 2, "880d6edc121eaffa"},
	{9, 2, "51c4898845ab7f56"}, {10, 2, "f4a0a51f7e2a75b2"}, {11, 2, "9fe2911662a4ae0e"}, {12, 2, "a5752368cb1990cf"},
}

func TestShopStockFollowsTheStackTable(t *testing.T) {
	table := shopScrollStackTable()
	for _, w := range shopStackTableRecord {
		line, _ := shopScrollStackRecordLine(table, w.seed, w.m)
		if got := fmt.Sprintf("%x", sha256.Sum256([]byte(line)))[:16]; got != w.digest {
			t.Errorf("seed %d x%d: digest %s, want %s: %s", w.seed, w.m, got, w.digest, line)
		}
	}
}

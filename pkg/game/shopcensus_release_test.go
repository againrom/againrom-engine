package game

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"os"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/data"
)

// shopCensusSeeds is the number of town-arrival stocks the census draws.
const shopCensusSeeds = 64

// shippedShopCensus is the digest of the census on each root with no mod
// loaded. The two roots ship one item table, so the digests agree.
var shippedShopCensus = map[string]string{
	"rom1-en": "ad00c6b7b2d8a8b73ce4b757d0ce529c08cc12ce1e042c22f89f3ef9c9e927c6",
	"rom1-ru": "ad00c6b7b2d8a8b73ce4b757d0ce529c08cc12ce1e042c22f89f3ef9c9e927c6",
}

func censusWrite(h hash.Hash, v ...int64) {
	var b [8]byte
	for _, x := range v {
		binary.LittleEndian.PutUint64(b[:], uint64(x))
		h.Write(b[:])
	}
}

// shopCensus digests the first game's candidate pools at every ceiling the
// census walks and the stock of every shelf over shopCensusSeeds seeds.
func shopCensus(t *testing.T, f *FrontEnd) string {
	t.Helper()
	h := sha256.New()
	for _, ceiling := range []int32{0, 100, 1000, 10000, 1 << 30} {
		for _, pool := range [][]data.ShopCandidate{shopArmourPool(f.Table, ceiling), shopWeaponPool(f.Table, ceiling)} {
			censusWrite(h, int64(len(pool)))
			for _, c := range pool {
				censusWrite(h, int64(c.Code), int64(c.Price), int64(c.MagCap))
			}
		}
	}
	units := 0
	for seed := int64(1); seed <= shopCensusSeeds; seed++ {
		var s Shop
		s.Generate(f.Table, seed)
		for _, shelf := range []ShopShelf{ShelfWeapons, ShelfArmour, ShelfMagic, ShelfBooks} {
			items := s.Shelf(shelf)
			censusWrite(h, int64(shelf), int64(len(items)))
			for _, it := range items {
				units++
				censusWrite(h, int64(it.Code), int64(it.Price), int64(it.Count), int64(it.Kind), int64(len(it.Effects)))
				for _, e := range it.Effects {
					censusWrite(h, int64(e.Kind), int64(e.Mode), int64(e.Operand))
				}
			}
		}
	}
	if units == 0 {
		t.Fatal("the census drew no shelf item")
	}
	return hex.EncodeToString(h.Sum(nil))
}

func TestReleaseShopStockCensusWithoutModsIsShipped(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	base := BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS")))
	got := shopCensus(t, f)
	want, ok := shippedShopCensus[base]
	if !ok {
		t.Fatalf("no recorded census for %s; this run's digest is %s", base, got)
	}
	if got != want {
		t.Fatalf("the shop census of %s is %s, want the shipped %s", base, got, want)
	}
}

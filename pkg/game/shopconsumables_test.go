package game

import (
	"math/rand"
	"testing"

	"againrom/pkg/mapload"
)

type potionShopRow struct{ name, effects string }
type potionShopRows []potionShopRow

func (r potionShopRows) Len() int                    { return len(r) }
func (r potionShopRows) EntryName(i int) string      { return r[i].name }
func (r potionShopRows) EntryParams(i int) []int32   { return []int32{37, 1} }
func (r potionShopRows) EntryStrings(i int) []string { return []string{r[i].effects} }

func TestPotionShelfResolvesNamesAndSuppliesSixStacksWithoutBooks(t *testing.T) {
	rows := make(potionShopRows, 256)
	// Deliberately shuffled, non-shipped row identities; production must look
	// up names and cannot accidentally pass with the shipped 6..11 range.
	positions := []int{255, 32, 21, 128, 22, 4}
	for i, name := range shopPotionNames {
		rows[positions[i]] = potionShopRow{name, "health=30"}
	}
	table := &mapload.Table{MagicItems: rows}
	for seed := int64(0); seed < 12; seed++ {
		shop := NewShop(1) // literal potion append is not the spell price window
		shop.generate(table, rand.New(rand.NewSource(seed)))
		stock := shop.Shelf(ShelfBooks)
		if len(stock) != 6 {
			t.Fatalf("seed %d: stock=%+v", seed, stock)
		}
		seen := map[uint16]bool{}
		for _, item := range stock {
			if item.Kind != 3 || item.Count < 51 || item.Count > 100 || item.Price != 37 || len(item.Effects) != 1 || item.Effects[0].Operand != 30 {
				t.Fatalf("seed %d: item=%+v", seed, item)
			}
			seen[uint16(item.Code)] = true
		}
		for _, row := range positions {
			if !seen[uint16(0x0e00|row)] {
				t.Fatalf("seed %d: absent row %d", seed, row)
			}
		}
	}
}

func TestPotionShelfRejectsMissingAndMalformedPayload(t *testing.T) {
	table := &mapload.Table{MagicItems: potionShopRows{{}, {shopPotionNames[0], "health=bad"}, {shopPotionNames[1], ""}}}
	if got := shopPotionStock(table, rand.New(rand.NewSource(1))); len(got) != 0 {
		t.Fatalf("bad potion stock=%+v", got)
	}
}

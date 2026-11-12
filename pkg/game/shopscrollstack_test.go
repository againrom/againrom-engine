package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/shopstock"
)

// shopScrollStackTable is shopTable with the five schools' book and scroll
// templates, the six potions and 27 spells, each school-bearing and priced so
// every scroll and book enters a ceiling of 1000.
func shopScrollStackTable() *mapload.Table {
	t := shopTable()
	items := make(dbCollection, 20)
	for i, school := range shopSchoolNames[1:] {
		items[1+i] = dbEntry{name: "Book_" + school, params: []int32{100}}
		items[6+i] = dbEntry{name: "Scroll_" + school, params: []int32{10}}
	}
	for i, name := range shopPotionNames {
		items[11+i] = dbEntry{name: name, params: []int32{37, 1}, strings: []string{"health=30"}}
	}
	t.MagicItems = items
	spells := make(dbCollection, 28)
	for s := 1; s < len(spells); s++ {
		p := make([]int32, 22)
		p[2], p[20], p[21] = int32(1+s%5), int32(20*s), int32(300+10*s)
		spells[s] = dbEntry{name: "spell " + strconv.Itoa(s), params: p}
	}
	t.Spells = spells
	return t
}

// countedDraws is a shop draw source that counts the numbers it gives.
type countedDraws struct {
	src shopDrawSource
	n   int
}

func (c *countedDraws) Intn(n int) int { c.n++; return c.src.Intn(n) }

func shopScrollRow(it ShopItem) bool {
	return it.Kind == 4 && len(it.Effects) == 1 && it.Effects[0].Kind == 41
}

// shopScrollStackRecordLine is one fill at seed and stock multiplier m: the
// shop stream's position after it and its next number, a digest of every row
// but the scroll rows, and each scroll row's identity and quantity.
func shopScrollStackRecordLine(table *mapload.Table, seed int64, m int32) (string, []ShopItem) {
	s := NewShop(1000)
	s.stockMultiplier = m
	src := &countedDraws{src: random.NewGo(seed)}
	s.drawSeed = seed
	s.generate(table, src)
	pos := src.n
	next := src.Intn(1 << 30)
	h := sha256.New()
	var scrolls []string
	var rows []ShopItem
	for k := ShopShelf(0); k < shopShelfCount; k++ {
		for _, it := range s.Shelf(k) {
			if shopScrollRow(it) {
				scrolls = append(scrolls, fmt.Sprintf("%#04x:%d:%d:%d", uint16(it.Code), it.Price, it.Effects[0].Operand, it.Count))
				rows = append(rows, it)
				continue
			}
			fmt.Fprintf(h, "%d %#04x %d %d %d %v;", k, uint16(it.Code), it.Kind, it.Price, it.Count, it.Effects)
		}
	}
	return fmt.Sprintf("seed %d x%d pos %d next %d rows %x scrolls %s", seed, m, pos, next, h.Sum(nil)[:8], strings.Join(scrolls, " ")), rows
}

// The record testdata/shop_scroll_stack_record.txt is the first game's fill
// as drawn before the scroll stack range: a scroll row's quantity there is
// its number of picks. Each fill now ends at the same stream position with
// the same next number and the same other rows, every scroll row holds
// 51..100, rows of two or more picks included, and the rows reach both ends.
func TestShopScrollRowsStackFiftyOneToHundred(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("testdata", "shop_scroll_stack_record.txt"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(want)), "\n")
	table := shopScrollStackTable()
	least, most, repeated := 1000, 0, 0
	for i, line := range lines {
		var seed int64
		var m int32
		if _, err := fmt.Sscanf(line, "seed %d x%d", &seed, &m); err != nil {
			t.Fatalf("record line %d: %v", i, err)
		}
		got, scrolls := shopScrollStackRecordLine(table, seed, m)
		head, wantScrolls, _ := strings.Cut(line, " scrolls ")
		gotHead, _, _ := strings.Cut(got, " scrolls ")
		if gotHead != head {
			t.Fatalf("seed %d x%d: stream or other rows moved:\n got %s\nwant %s", seed, m, gotHead, head)
		}
		before := strings.Fields(wantScrolls)
		if len(before) != len(scrolls) {
			t.Fatalf("seed %d x%d: %d scroll rows, want %d", seed, m, len(scrolls), len(before))
		}
		for j, it := range scrolls {
			id := fmt.Sprintf("%#04x:%d:%d:", uint16(it.Code), it.Price, it.Effects[0].Operand)
			picks, err := strconv.Atoi(strings.TrimPrefix(before[j], id))
			if err != nil || !strings.HasPrefix(before[j], id) {
				t.Fatalf("seed %d x%d: scroll row %d is %s, want %s", seed, m, j, id, before[j])
			}
			if it.Count < 51 || it.Count > 100 {
				t.Fatalf("seed %d x%d: scroll %s of %d picks holds %d", seed, m, id, picks, it.Count)
			}
			if picks >= 2 {
				repeated++
			}
			least, most = min(least, int(it.Count)), max(most, int(it.Count))
		}
	}
	if least != 51 || most != 100 || repeated == 0 {
		t.Fatalf("scroll rows reach %d..%d, %d of them repeated picks, want 51..100", least, most, repeated)
	}
}

// A books shelf whose one candidate is a scroll picks it 1..8 times; its row
// holds one draw of 51..100 however many picks it took.
func TestShopRepeatedScrollPicksKeepOneStack(t *testing.T) {
	table := shopScrollStackTable()
	spells := make(dbCollection, 28)
	p := make([]int32, 22)
	p[2], p[20] = 1, 20
	spells[1] = dbEntry{name: "spell 1", params: p}
	table.Spells = spells
	repeated := 0
	for seed := int64(1); seed <= 60; seed++ {
		s := NewShop(1000)
		s.drawSeed = seed
		src := random.NewGo(seed)
		profile := s.stockProfile(table, src)
		picks := 0
		books := &profile.Categories[ShelfBooks]
		mk := books.Make
		books.Make = func(c shopStockCandidate, src shopstock.Source) (ShopItem, bool) {
			picks++
			return mk(c, src)
		}
		var rows []ShopItem
		for _, it := range shopstock.Generate(profile, src)[ShelfBooks] {
			if shopScrollRow(it) {
				rows = append(rows, it)
			}
		}
		if len(rows) != 1 || rows[0].Count < 51 || rows[0].Count > 100 {
			t.Fatalf("seed %d: %d picks gave scroll rows %+v, want one of 51..100", seed, picks, rows)
		}
		if picks >= 2 {
			repeated++
		}
	}
	if repeated < 20 {
		t.Fatalf("%d of 60 fills picked the scroll twice or more", repeated)
	}
}

// A mod's stocked item keeps the one unit its stock field places, on its own
// shelf, beside scroll rows drawn at 51..100.
func TestShopScrollStacksKeepModStockUnits(t *testing.T) {
	tab := modStockRecordTable(t)
	scrolls := shopScrollStackTable()
	tab.Spells, tab.MagicItems = scrolls.Spells, scrolls.MagicItems
	mods := make(map[uint16]bool)
	for _, m := range tab.Mods.Items {
		mods[m.Code] = m.Stock
	}
	placed, drawn := 0, 0
	for seed := int64(1); seed <= 8; seed++ {
		s := NewShop(5000)
		s.Generate(tab, seed)
		for k := ShopShelf(0); k < shopShelfCount; k++ {
			for _, it := range s.Shelf(k) {
				switch {
				case mods[uint16(it.Code)]:
					placed++
					if it.Count != 1 {
						t.Fatalf("seed %d: mod item %#04x holds %d units, want 1", seed, uint16(it.Code), it.Count)
					}
				case shopScrollRow(it):
					drawn++
					if it.Count < 51 {
						t.Fatalf("seed %d: scroll %#04x holds %d units", seed, uint16(it.Code), it.Count)
					}
				}
			}
		}
	}
	if placed == 0 || drawn == 0 {
		t.Fatalf("%d mod units and %d scroll rows placed", placed, drawn)
	}
}

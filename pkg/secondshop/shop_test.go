package secondshop

import (
	"os"
	"reflect"
	"sort"
	"testing"

	"againrom/pkg/formats/databin"
	"againrom/pkg/random"
	"againrom/pkg/vfs"
)

// installTables reads the second game's data.bin from AGAINROM_ASSETS.
func installTables(t *testing.T) Tables {
	t.Helper()
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the shop populations need a lawful install")
	}
	if _, err := os.Stat(root + "/allods2.exe"); err != nil {
		t.Skip("the root holds the first game")
	}
	fs, err := vfs.Open([]string{root + "/world.res"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := fs.ReadFile("world/data/data.bin")
	if err != nil {
		t.Fatal(err)
	}
	f, err := databin.ParseWith(b, databin.ROM2Layout)
	if err != nil {
		t.Fatal(err)
	}
	return Tables{Shapes: f.Collection(databin.Shapes), Materials: f.Collection(databin.Materials),
		Armors: f.Collection(databin.Armors), Shields: f.Collection(databin.Shields), Weapons: f.Collection(databin.Weapons),
		MagicItems: f.Collection(databin.MagicItems), Spells: f.Collection(databin.Spells)}
}

// swapped reads factor slot 3 where slot 2 is asked: the loss control of the
// price column.
type swapped struct{ Collection }

func (s swapped) EntryDoubles(i int) []float64 {
	d := append([]float64(nil), s.Collection.EntryDoubles(i)...)
	d[factorPrice], d[factorPrice+1] = d[factorPrice+1], d[factorPrice]
	return d
}

func count(items []Item) map[Kind]int {
	n := map[Kind]int{}
	for _, it := range items {
		n[it.Kind]++
	}
	return n
}

// enchantVariants counts the (base, spell, level) rows of an enchanted
// category whose final price lies in its range (R2-ENGINE-265).
func enchantVariants(tb Tables, cat Category) (n int, lo, hi int64) {
	lo = 1 << 62
	for _, base := range tb.Candidates(cat, nil) {
		for _, s := range enchantSpells {
			for level := 1; level <= tb.LevelCap(base, s, cat); level++ {
				it := base
				it.Spell, it.Level = s, level
				if p := tb.Price(it); p >= int64(cat.Min) && p <= int64(cat.Max) {
					n++
					lo, hi = min(lo, p), max(hi, p)
				}
			}
		}
	}
	return n, lo, hi
}

func TestReleaseSecondShopTownOnePopulations(t *testing.T) {
	tb := installTables(t)
	r := NewGameRecords[1]
	if got := count(tb.Candidates(r[0], nil)); got[Armor] != 57 || got[Shield] != 13 || len(got) != 2 {
		t.Errorf("category 0 admits %v, want 57 armour and 13 shield rows", got)
	}
	weapons := tb.Candidates(r[1], nil)
	if len(weapons) != 44 {
		t.Errorf("category 1 admits %d weapon rows, want 44", len(weapons))
	}
	for _, it := range weapons {
		if it.Kind != Weapon || it.Class > 1 || it.Material != 0 && it.Material != 1 && it.Material != 8 {
			t.Errorf("category 1 admits %+v outside classes 0..1 and materials 0, 1, 8", it)
		}
	}
	staffs := tb.Candidates(r[2], nil)
	var prices []int64
	for _, it := range staffs {
		if it.Kind != Weapon || it.Material != 8 || it.Row != 13 && it.Row != 14 {
			t.Errorf("category 2 admits %+v, want rows 13 and 14 of material 8", it)
		}
		prices = append(prices, tb.Price(it))
	}
	sort.Slice(prices, func(i, j int) bool { return prices[i] < prices[j] })
	if !reflect.DeepEqual(prices, []int64{83, 167, 167, 333}) {
		t.Errorf("category 2 weapon prices %v, want 83, 167, 167, 333", prices)
	}
	if n, lo, hi := enchantVariants(tb, r[2]); n != 160 || lo != 649 || hi != 1442 {
		t.Errorf("category 2 enchant variants %d priced %d..%d, want 160 priced 649..1442", n, lo, hi)
	}
	var books []int
	for _, it := range tb.Candidates(r[3], nil) {
		if it.Kind == Book {
			books = append(books, it.Spell)
			if p := tb.Price(it); p != 1000 {
				t.Errorf("book of spell %d priced %d, want 1000", it.Spell, p)
			}
		}
	}
	if !reflect.DeepEqual(books, []int{1, 5, 10, 16, 26}) {
		t.Errorf("books for spells %v, want 1, 5, 10, 16, 26", books)
	}
	admissible := 0
	for row := scrollOffset + 1; row <= superOffset+bookSpells; row++ {
		if p := tb.Price(Item{Kind: Magic, Row: row}); p >= 0 && p <= 1500 {
			admissible++
		}
	}
	if admissible != 23 {
		t.Errorf("%d phase-two rows admissible, want 23", admissible)
	}

	// Loss control: price factor slot 3 in place of slot 2 breaks the populations.
	alt := tb
	alt.Shapes, alt.Materials = swapped{tb.Shapes}, swapped{tb.Materials}
	if count(alt.Candidates(r[0], nil))[Armor] == 57 && len(alt.Candidates(r[1], nil)) == 44 {
		t.Error("factor slot 3 reproduces the populations; slot 2 is not discriminated")
	}
}

func TestReleaseSecondShopTownOneFill(t *testing.T) {
	tb := installTables(t)
	r := NewGameRecords[1]
	fill := func(seed uint32) Stock {
		g := &random.MSVC{State: seed}
		return tb.Fill(r, func() int { return int(g.Rand()) })
	}
	s := fill(1)
	if !reflect.DeepEqual(s, fill(1)) {
		t.Fatal("one seed fills two different stocks")
	}
	if reflect.DeepEqual(s, fill(2)) {
		t.Error("two seeds fill one stock")
	}
	books, potions := 0, 0
	for _, it := range s[3] {
		switch {
		case it.Kind == Book:
			books++
			if it.Quantity != 2 {
				t.Errorf("book %+v, want quantity 2", it)
			}
		case it.Row >= potionFirst && it.Row <= potionLast:
			potions++
			if it.Quantity < 51 || it.Quantity > 100 {
				t.Errorf("potion %+v, want quantity 51..100", it)
			}
		}
	}
	if books != 5 || potions != 6 {
		t.Errorf("category 3 holds %d books and %d potions, want 5 and 6", books, potions)
	}
	for _, it := range s[2] {
		if it.Level < 1 || it.Price < 649 || it.Price > 1442 {
			t.Errorf("category 2 item %+v outside the enchanted population", it)
		}
	}
	for c := 0; c < 2; c++ {
		total := 0
		for _, it := range s[c] {
			total += it.Quantity
			if it.Price != tb.Price(it) {
				t.Errorf("category %d item %+v carries price %d, want %d", c, it, it.Price, tb.Price(it))
			}
		}
		if total < 100 || total > 300 {
			t.Errorf("category %d holds %d units from 100 draws of quantity 1..3", c, total)
		}
	}
}

func TestSecondShopSettlement(t *testing.T) {
	if got := SaleCredit(83, 3); got != 125 {
		t.Errorf("sale credit of 3 at 83 = %d, want floor(249/2+0.5) = 125", got)
	}
	if got := SaleCell(83) * 3; got != 126 {
		t.Errorf("client sale total of 3 at 83 = %d, want ((83+1)/2)*3 = 126", got)
	}
	if got := SaleCredit(84, 3); got != 126 {
		t.Errorf("sale credit of 3 at 84 = %d, want 126", got)
	}
	pending := []Item{{Kind: Weapon, Row: 2, Price: 100, Quantity: 2}, {Kind: Armor, Row: 3, Price: 500, Quantity: 1}, {Kind: Magic, Row: 69, Price: 1, Quantity: 1}}
	bought, left, gold := Buy(pending, 600)
	if len(bought) != 1 || len(left) != 2 || gold != 400 {
		t.Errorf("buy with 600: bought %d, left %d, gold %d; want 1, 2, 400", len(bought), len(left), gold)
	}
	if bought, left, gold = Buy(pending, 701); len(bought) != 3 || left != nil || gold != 0 {
		t.Errorf("buy with 701: bought %d, left %d, gold %d; want 3, 0, 0", len(bought), len(left), gold)
	}
	s := Stock{{{Kind: Armor, Row: 3, Quantity: 3, Price: 10}}}
	s, taken, ok := Take(s, 0, 0, 2)
	if !ok || taken.Quantity != 2 || s[0][0].Quantity != 1 {
		t.Errorf("take 2 of 3: taken %d, shelf %d", taken.Quantity, s[0][0].Quantity)
	}
	s, gold = Sell(s, NewGameRecords[1], []Item{taken}, 0)
	if s[0][0].Quantity != 3 || len(s[0]) != 1 || gold != 10 {
		t.Errorf("sale of 2 at 10 back: shelf %+v, gold %d; want merged 3 and 10", s[0], gold)
	}
	enchanted := Item{Kind: Weapon, Row: 13, Material: 8, Spell: 1, Level: 3}
	if c := Place(NewGameRecords[1], enchanted); c != 2 {
		t.Errorf("enchanted weapon placed in category %d, want 2", c)
	}
	if c := Place(NewGameRecords[1], Item{Kind: Weapon, Row: 2}); c != 1 {
		t.Errorf("plain weapon placed in category %d, want 1", c)
	}
	if c := Place(NewGameRecords[1], Item{Kind: Book, Spell: 1}); c != 3 {
		t.Errorf("book placed in category %d, want 3", c)
	}
	if c := Place(NewGameRecords[2], Item{Kind: Weapon, Row: 2}); c != 3 {
		t.Errorf("record 2 plain weapon placed in category %d, want 3 (R2-ENGINE-333)", c)
	}
	if got := Leave(Stock{{{Quantity: 0}, {Quantity: 1}}}); len(got[0]) != 1 {
		t.Errorf("leave keeps %d entries, want 1", len(got[0]))
	}
}

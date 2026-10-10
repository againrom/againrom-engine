package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/spr16"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/render/text"
	"io/fs"
)

// itemTable is a table in the shape of the shipped item tables: a base
// collection per class with reserved row 0, a few named rows and unnamed
// filler rows up to a length, and one-entry scale tables of the reference
// shape and material.
type fixtureCollection struct {
	names  []string
	params [][]int32
	raw    [][]byte
}

func (c *fixtureCollection) Len() int                    { return len(c.names) }
func (c *fixtureCollection) EntryName(i int) string      { return c.names[i] }
func (c *fixtureCollection) EntryParams(i int) []int32   { return c.params[i] }
func (c *fixtureCollection) EntryStrings(i int) []string { return nil }
func (c *fixtureCollection) EntryRaw(i int) []byte       { return c.raw[i] }

type fixtureScale struct {
	names   []string
	doubles [][]float64
}

func (s fixtureScale) Len() int                     { return len(s.names) }
func (s fixtureScale) EntryName(i int) string       { return s.names[i] }
func (s fixtureScale) EntryDoubles(i int) []float64 { return s.doubles[i] }

func mask(bits ...int) []byte {
	var m uint16
	for _, b := range bits {
		m |= 1 << uint(b)
	}
	out := make([]byte, 10)
	binary.LittleEndian.PutUint16(out, m)
	return out
}

// fixtureItemTable has armour rows 1 and 2 named, filler up to length armors,
// and shields and weapons likewise.
func fixtureItemTable(armors, shields, weapons int) *mapload.Table {
	row := func(slot, price, weight, defence int32) []int32 {
		return []int32{0, 0, price, weight, slot, -1, 0, 0, 0, defence, 0, -1, -1, -1, -1, 3, -1}
	}
	build := func(n int, named map[int]string, params map[int][]int32, raw map[int][]byte) *fixtureCollection {
		c := &fixtureCollection{names: make([]string, n), params: make([][]int32, n), raw: make([][]byte, n)}
		for i := 1; i < n; i++ {
			c.params[i] = row(7, 10, 10, 10)
			c.raw[i] = make([]byte, 10)
		}
		for i, s := range named {
			c.names[i] = s
		}
		for i, p := range params {
			c.params[i] = p
		}
		for i, r := range raw {
			c.raw[i] = r
		}
		return c
	}
	t := &mapload.Table{
		Shapes:    fixtureScale{names: []string{"Common", "Rare"}, doubles: [][]float64{{0, 0, 1, 1, 0.2, 1, 0.25, 1, 1}, {0, 0, 4, 1.2, 0.3, 1.4, 0.36, 1.2, 3}}},
		Materials: fixtureScale{names: []string{"Iron", "Leather", "Hard Leather"}, doubles: [][]float64{{0, 0, 1, 1, 1, 1, 1, 1, 0}, {0, 0, 0.5, 0.2, 0.3, 1, 0.3, 1, 0}, {0, 0, 0.6, 0.4, 0.6, 1, 0.6, 1, 1}}},
		Armors: build(armors, map[int]string{1: "Soft Mail", 2: "Cap", 3: "Chain Mail"},
			map[int][]int32{2: row(6, 10, 15, 10), 3: row(7, 170, 40, 26)},
			map[int][]byte{1: mask(1, 2), 2: mask(0), 3: mask(0)}),
		Shields: build(shields, map[int]string{1: "Buckler"},
			map[int][]int32{1: row(2, 12, 10, 20)}, map[int][]byte{1: mask(0)}),
		Weapons: build(weapons, map[int]string{1: "Dagger"},
			map[int][]int32{1: row(1, 100, 5, 0)}, map[int][]byte{1: mask(0)}),
		Names: data.ItemNames{},
	}
	return t
}

func itemFront(t *mapload.Table) *FrontEnd {
	f := &FrontEnd{}
	f.Table = t
	return f
}

func testRow(key, slot, standIn string) mod.ItemRow {
	return mod.ItemRow{Mod: "m", File: "data/items.toml", Line: 4, Key: key, Name: "Name " + key, Slot: slot, SlotNo: mod.ItemSlots[slot],
		Defence: 6, Absorption: 1, Weight: 6, Price: 120, StandIn: standIn, StandInLine: 9, Suit: mod.SuitAny}
}

func TestSetModItemsTakesTheLowestFreeRowsInLoadOrder(t *testing.T) {
	tab := fixtureItemTable(30, 5, 5)
	f := itemFront(tab)
	err := f.SetModItems(mod.ItemData{Rows: []mod.ItemRow{
		testRow("a", "body", "Soft Mail"), testRow("b", "head", "Cap"), testRow("s", "shield", "Buckler"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	items := tab.Mods.Items
	if len(items) != 3 || items[0].Row != 30 || items[1].Row != 31 || items[2].Row != 5 {
		t.Fatalf("%+v", items)
	}
	if code := data.ItemCode(items[0].Code); code != data.ComposeItemCode(0, 7, 0, 30) || items[0].StandInCode != uint16(data.ComposeItemCode(1, 7, 0, 1)) || items[0].StandInRow != 1 {
		t.Fatalf("%+v", items[0])
	}
	if items[2].Code != uint16(data.ComposeItemCode(0, 2, 0, 5)) {
		t.Fatalf("%+v", items[2])
	}
	if name, ok := tab.Names.NameFor(data.ItemCode(items[0].Code)); !ok || name != "Name a" {
		t.Fatalf("name %q %v", name, ok)
	}
	if tab.Armors.Len() != 32 || tab.Armors.EntryName(30) != "a" || tab.Shields.Len() != 6 || tab.Weapons.Len() != 5 {
		t.Fatalf("lengths %d %d %d", tab.Armors.Len(), tab.Shields.Len(), tab.Weapons.Len())
	}
	// The item resolves through the production readers with the stated numbers.
	a, err := data.ArmorFromCode(data.ItemCode(items[0].Code), tab.Shapes, tab.Materials, tab.Armors)
	if err != nil || a.Defence != 6 || a.Absorption != 1 || a.Weight != 6 || a.Slot != 7 {
		t.Fatalf("%+v %v", a, err)
	}
	s, err := data.ShieldFromCode(data.ItemCode(items[2].Code), tab.Shapes, tab.Materials, tab.Shields)
	if err != nil || s.Defence != 6 || s.Weight != 6 {
		t.Fatalf("%+v %v", s, err)
	}
	if items[0].Price != 120 {
		t.Fatalf("price %d", items[0].Price)
	}
	// The shipped rows are unchanged.
	if p := tab.Armors.EntryParams(3); p[3] != 40 {
		t.Fatalf("row 3 %v", p)
	}
	// A row of the mod is never drawn at random: its masks are empty.
	pool := data.ShopPool(maskTable(tab.Armors), data.ArmorShopClass, tab.Shapes, tab.Materials, 1<<20)
	for _, c := range pool {
		if c.Row >= 30 {
			t.Fatalf("the shop pool draws mod row %d", c.Row)
		}
	}
}

func TestSetModItemsRefusesWhenTheCodeSpaceIsFull(t *testing.T) {
	f := itemFront(fixtureItemTable(31, 5, 5))
	err := f.SetModItems(mod.ItemData{Rows: []mod.ItemRow{testRow("a", "body", "Soft Mail"), testRow("b", "body", "Soft Mail")}})
	want := `mod "m": data/items.toml:4: no free row: the armour table holds 32 rows, the most an item code names, and every one is in use`
	if err == nil || err.Error() != want {
		t.Fatalf("%v", err)
	}
	if f.Table.Mods.Items != nil && len(f.Table.Mods.Items) != 0 {
		t.Fatal("a refused launch left items behind")
	}
}

func TestSetModItemsStandInForms(t *testing.T) {
	cases := []struct {
		standIn string
		code    data.ItemCode
		err     string
	}{
		{"Soft Mail", data.ComposeItemCode(1, 7, 0, 1), ""},
		{"Hard Leather Soft Mail", data.ComposeItemCode(2, 7, 0, 1), ""},
		{"Leather Soft Mail", data.ComposeItemCode(1, 7, 0, 1), ""},
		{"Common Hard Leather Soft Mail", data.ComposeItemCode(2, 7, 0, 1), ""},
		{"  Soft   Mail ", data.ComposeItemCode(1, 7, 0, 1), ""},
		{"Rare Soft Mail", 0, "never makes"},
		{"Iron Soft Mail", 0, "never makes"},
		{"Bone Soft Mail", 0, `no row named "Bone Soft Mail"`},
		{"Cap", 0, `"Cap" is not an item of the body slot`},
		{"Nothing", 0, `no row named "Nothing"`},
	}
	for _, c := range cases {
		tab := fixtureItemTable(10, 5, 5)
		err := itemFront(tab).SetModItems(mod.ItemData{Rows: []mod.ItemRow{testRow("a", "body", c.standIn)}})
		if c.err != "" {
			if err == nil || !strings.Contains(err.Error(), c.err) || !strings.Contains(err.Error(), "data/items.toml:9: stand-in") {
				t.Errorf("%q: %v", c.standIn, err)
			}
			continue
		}
		if err != nil || tab.Mods.Items[0].StandInCode != uint16(c.code) {
			t.Errorf("%q: %v %+v", c.standIn, err, tab.Mods.Items)
		}
	}
}

func TestSetModItemsChangesExistingRows(t *testing.T) {
	tab := fixtureItemTable(10, 5, 5)
	w, d := int32(20), int32(8)
	err := itemFront(tab).SetModItems(mod.ItemData{Changes: []mod.ItemChange{
		{Mod: "m", File: "data/items.toml", Line: 2, Item: "Chain Mail", Weight: &w, Defence: &d},
		{Mod: "m", File: "data/items.toml", Line: 6, Item: "Dagger", Price: &w},
	}})
	if err != nil {
		t.Fatal(err)
	}
	a, err := data.ArmorFromCode(data.ComposeItemCode(0, 7, 0, 3), tab.Shapes, tab.Materials, tab.Armors)
	if err != nil || a.Weight != 20 || a.Defence != 8 {
		t.Fatalf("%+v %v", a, err)
	}
	if p := tab.Weapons.EntryParams(1); p[2] != 20 || p[3] != 5 {
		t.Fatalf("dagger %v", p)
	}
	if p := tab.Armors.EntryParams(1); p[3] != 10 {
		t.Fatalf("an untouched row changed: %v", p)
	}
	if len(tab.Mods.Items) != 0 {
		t.Fatal("a change added an item")
	}
}

func TestSetModItemsChangeRefusals(t *testing.T) {
	n := int32(1)
	for _, c := range []struct {
		change mod.ItemChange
		want   string
	}{
		{mod.ItemChange{Item: "Gown", Weight: &n}, `no row named "Gown"`},
		{mod.ItemChange{Item: "Dagger", Defence: &n}, `"Dagger" is a weapon; a change can set its price and weight only`},
	} {
		c.change.Mod, c.change.File, c.change.Line = "m", "data/items.toml", 12
		err := itemFront(fixtureItemTable(10, 5, 5)).SetModItems(mod.ItemData{Changes: []mod.ItemChange{c.change}})
		if err == nil || !strings.HasPrefix(err.Error(), `mod "m": data/items.toml:12: `) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.want, err)
		}
	}
}

func TestSetModItemsRefusesNumbersTheTableCannotState(t *testing.T) {
	tab := fixtureItemTable(10, 5, 5)
	// A shape factor of 3 for defence makes odd reference numbers unreachable.
	tab.Shapes = fixtureScale{names: []string{"Common"}, doubles: [][]float64{{0, 0, 1, 1, 0.2, 1, 3, 1, 1}}}
	r := testRow("a", "body", "Soft Mail")
	r.Defence = 2
	err := itemFront(tab).SetModItems(mod.ItemData{Rows: []mod.ItemRow{r}})
	if err == nil || !strings.Contains(err.Error(), "cannot state defence 2") {
		t.Fatalf("%v", err)
	}
}

func TestSetModItemsWithoutItemsChangesNothing(t *testing.T) {
	tab := fixtureItemTable(10, 5, 5)
	armors := tab.Armors
	if err := itemFront(tab).SetModItems(mod.ItemData{}); err != nil || tab.Armors != armors || tab.Names == nil || len(tab.Mods.Items) != 0 {
		t.Fatal("an empty item set changed the table")
	}
	if err := itemFront(&mapload.Table{}).SetModItems(mod.ItemData{Rows: []mod.ItemRow{testRow("a", "body", "x")}}); err == nil {
		t.Fatal("a table without item collections accepted an item")
	}
}

func TestModShopStockListsTheItemsThatAskForAPlace(t *testing.T) {
	tab := fixtureItemTable(10, 5, 5)
	a, b := testRow("a", "body", "Soft Mail"), testRow("b", "head", "Cap")
	a.Stock = true
	if err := itemFront(tab).SetModItems(mod.ItemData{Rows: []mod.ItemRow{a, b}}); err != nil {
		t.Fatal(err)
	}
	stock := modShopStock(tab, 0, 0, 100)
	if len(stock) != 1 || stock[0].Code != data.ComposeItemCode(0, 7, 0, 10) || stock[0].Price != 120 || stock[0].Count != 1 || stock[0].Kind != 1 {
		t.Fatalf("%+v", stock)
	}
	if modShopStock(fixtureItemTable(10, 5, 5), 0, 0, 100) != nil {
		t.Fatal("an unmodded table has mod stock")
	}
}

func TestShopGenerateAddsTheModStockWithoutTakingADraw(t *testing.T) {
	plain := fixtureItemTable(10, 5, 5)
	moddedTab := fixtureItemTable(10, 5, 5)
	a := testRow("a", "body", "Soft Mail")
	a.Stock = true
	if err := itemFront(moddedTab).SetModItems(mod.ItemData{Rows: []mod.ItemRow{a}}); err != nil {
		t.Fatal(err)
	}
	for _, tab := range []*mapload.Table{plain, moddedTab} {
		tab.Spells = nil
	}
	s1, s2 := NewShop(5000), NewShop(5000)
	s1.Generate(plain, 7)
	s2.Generate(moddedTab, 7)
	base, withMod := s1.Shelf(ShelfArmour), s2.Shelf(ShelfArmour)
	if len(withMod) != len(base)+1 {
		t.Fatalf("shelf %d, with the mod %d", len(base), len(withMod))
	}
	found := 0
	rest := 0
	for _, it := range withMod {
		if it.Code == data.ComposeItemCode(0, 7, 0, 10) {
			found++
			if it.Price != 120 {
				t.Fatalf("price %d", it.Price)
			}
			continue
		}
		rest++
	}
	if found != 1 || rest != len(base) {
		t.Fatalf("found %d, rest %d of %d", found, rest, len(base))
	}
	for _, shelf := range []ShopShelf{ShelfWeapons, ShelfMagic, ShelfBooks} {
		if len(s1.Shelf(shelf)) != len(s2.Shelf(shelf)) {
			t.Fatalf("shelf %v differs", shelf)
		}
	}
}

func fixturePNG(t *testing.T, w, h int, f func(x, y int) color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, f(x, y))
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestModIconSpriteRoundTripsThroughTheGamesDecoder(t *testing.T) {
	src := fixturePNG(t, 6, 4, func(x, y int) color.NRGBA {
		switch {
		case x == 0:
			return color.NRGBA{}
		case x == 5 && y == 3:
			return color.NRGBA{R: 255, A: 128}
		}
		return color.NRGBA{R: uint8(40 * x), G: uint8(60 * y), B: 9, A: 255}
	})
	b, err := modIconSprite(src)
	if err != nil {
		t.Fatal(err)
	}
	s, err := spr16.DecodeA(b, true)
	if err != nil || len(s.Frames) != 1 {
		t.Fatalf("%v", err)
	}
	f := s.Frames[0]
	if f.Width != 6 || f.Height != 4 {
		t.Fatalf("%dx%d", f.Width, f.Height)
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 6; x++ {
			p := f.Pixels[y*6+x]
			switch {
			case x == 0:
				if p.Painted {
					t.Fatalf("(%d,%d) painted", x, y)
				}
			case x == 5 && y == 3:
				if !p.Painted || p.Level != 7 || s.Palette[p.Index].R != 255 {
					t.Fatalf("(%d,%d) %+v", x, y, p)
				}
			default:
				c := s.Palette[p.Index]
				if !p.Painted || p.Level != 15 || c.R != uint8(40*x) || c.G != uint8(60*y) || c.B != 9 {
					t.Fatalf("(%d,%d) %+v %+v", x, y, p, c)
				}
			}
		}
	}
	// The game's own reader turns it into a picture.
	img, err := loadItemIcon(memorySource{"a.16a": b}, "a.16a")
	if err != nil || img.Bounds().Dx() != 6 || img.Pix[3] != 0 {
		t.Fatalf("%v", err)
	}
}

type memorySource map[string][]byte

func (m memorySource) ReadFile(name string) ([]byte, error) {
	if b, ok := m[name]; ok {
		return b, nil
	}
	return nil, fs.ErrNotExist
}

func TestModIconSpriteReducesManyColoursAndRunsLongBlankSpans(t *testing.T) {
	src := fixturePNG(t, 80, 80, func(x, y int) color.NRGBA {
		if y < 70 && x < 70 {
			return color.NRGBA{R: uint8(x * 3), G: uint8(y * 3), B: uint8((x * y) % 256), A: 255}
		}
		if x == 79 && y == 79 {
			return color.NRGBA{R: 1, G: 2, B: 3, A: 255}
		}
		return color.NRGBA{}
	})
	b, err := modIconSprite(src)
	if err != nil {
		t.Fatal(err)
	}
	s, err := spr16.DecodeA(b, true)
	if err != nil || !s.Frames[0].Pixels[80*80-1].Painted || s.Frames[0].Pixels[80*80-2].Painted {
		t.Fatalf("%v", err)
	}
}

func TestModIconSpriteRefusals(t *testing.T) {
	big := fixturePNG(t, 81, 10, func(x, y int) color.NRGBA { return color.NRGBA{A: 255} })
	empty := fixturePNG(t, 4, 4, func(x, y int) color.NRGBA { return color.NRGBA{} })
	for _, c := range []struct {
		name string
		src  []byte
		want string
	}{
		{"not a png", []byte("hello"), "not a readable PNG"},
		{"too wide", big, "81x10; an item picture is 1 to 80"},
		{"nothing visible", empty, "no visible pixel"},
	} {
		if _, err := modIconSprite(c.src); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

func TestSetModItemsStoresTheNameInTheInstallAlphabet(t *testing.T) {
	const name = "Поддоспешник"
	for _, selector := range []int{0, 1} {
		tab := fixtureItemTable(30, 5, 5)
		f := itemFront(tab)
		f.Font = resolved(&text.Font{Selector: selector}, nil)
		r := testRow("a", "body", "Soft Mail")
		r.Name = name
		if err := f.SetModItems(mod.ItemData{Rows: []mod.ItemRow{r}}); err != nil {
			t.Fatal(err)
		}
		want, err := encodeSaveLabel(name, selector)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := tab.Names.NameFor(data.ItemCode(tab.Mods.Items[0].Code))
		if !ok || got != want {
			t.Errorf("selector %d: stored %q, want the install bytes %q", selector, got, want)
		}
		if got == name {
			t.Errorf("selector %d: the name is stored as UTF-8", selector)
		}
	}
}

func TestSetModItemsRefusesANameTheInstallCannotDraw(t *testing.T) {
	tab := fixtureItemTable(30, 5, 5)
	r := testRow("a", "body", "Soft Mail")
	r.Name = "Rüstung 漢"
	err := itemFront(tab).SetModItems(mod.ItemData{Rows: []mod.ItemRow{r}})
	if err == nil || !strings.Contains(err.Error(), `mod "m": data/items.toml:4: the name "Rüstung 漢" cannot be drawn by this install`) {
		t.Fatalf("got %v", err)
	}
}

func TestModShopStockFollowsTheShippedChanceOnARestock(t *testing.T) {
	if c := modStockChance(1000, 100); c < 0.094 || c > 0.096 {
		t.Fatalf("chance %v", c)
	}
	if modStockChance(0, 100) != 1 || modStockChance(1, 100) != 1 {
		t.Fatal("a pool of one row always offers it")
	}
	for seed := int64(0); seed < 50; seed++ {
		if !modStockOffered(seed, 0, 0x071f, 1000) {
			t.Fatalf("the town-arrival stock lacks the item at seed %d", seed)
		}
	}
	hits := 0
	const n = 4000
	for seed := int64(0); seed < n; seed++ {
		first := modStockOffered(seed, 1, 0x071f, 1000)
		if first != modStockOffered(seed, 1, 0x071f, 1000) {
			t.Fatal("the draw is not reproducible")
		}
		if first {
			hits++
		}
	}
	if hits < n*7/100 || hits > n*12/100 {
		t.Fatalf("restocks offered the item %d times of %d, shipped chance 0.095", hits, n)
	}
}

func TestShopRestockOffersTheModItemOnlyByChance(t *testing.T) {
	tab := fixtureItemTable(10, 5, 5)
	tab.Spells = nil
	a := testRow("a", "body", "Soft Mail")
	a.Stock = true
	if err := itemFront(tab).SetModItems(mod.ItemData{Rows: []mod.ItemRow{a}}); err != nil {
		t.Fatal(err)
	}
	code := data.ComposeItemCode(0, 7, 0, 10)
	count := func(s *Shop) int {
		n := 0
		for _, it := range s.Shelf(ShelfArmour) {
			if it.Code == code {
				n += int(it.Count)
			}
		}
		return n
	}
	s := NewShop(5000)
	s.Generate(tab, 7)
	if count(s) != 1 {
		t.Fatalf("arrival stock holds %d", count(s))
	}
	pool := len(shopPlainShelfPool(shopArmourPool(tab, 5000)))
	for i := 1; i <= 60; i++ {
		s.Restock(tab, 11)
		want := 0
		if modStockOffered(shopRestockSeed(11, uint64(i)), uint64(i), uint16(code), pool) {
			want = 1
		}
		if count(s) != want {
			t.Fatalf("restock %d holds %d units, the chance rule says %d (pool %d)", i, count(s), want, pool)
		}
	}
}

package game

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/databin"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// itemNameSurfaceDigests are the SHA-256 digests of every surface that shows
// an item's name, over the fixture tables below, as the engine wrote them
// before the item name had one resolver. Setting
// AGAINROM_ITEMNAME_SURFACES to a directory writes each surface's text there.
var itemNameSurfaceDigests = map[string]string{
	"first/editorCatalog":       "dc8c266da9cc614ea1579dfef8227eafef81368160befbebf994628ff246f2f5",
	"first/editorItemList":      "64171d6e476d93d6195fdc9e4ecd0ae812e5781cf29c00b4322ef2f635827e38",
	"first/editorRecipeItems":   "f7714fb5e9b6164a4959df2ac474374f220e176df3c136ee74d7c489617e82f2",
	"first/editorSack":          "13d70dfcf205f1f11ac27f9f315d45298bd669dca2be8ec20116863d98681455",
	"first/itemInfoLines":       "5c0c7b5f47568d8728aac8e1c4617b8e40fab4ee81ebe2165e3368a242bc5d1a",
	"first/itemName":            "2ad0c7549c608b5b395dbb00d8192f56492248f7cecc87a4218748d7dc9441a0",
	"first/pickup":              "4d3bdf7ec1375f315b445479cc0abc5510fbb8a70fec8b18a05c011f5f517ab0",
	"first/shopBookLabel":       "909cb81127c5e1944b16dc5944ade59facb822c7a3f250d1f0256293bfd3c055",
	"first/weapon":              "61c2eab5f43b2fc6e7dcdec0c5cc9ae5ce86ea85fb20c3590ac467f9e696d9c3",
	"none/itemInfoLines":        "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	"none/itemName":             "e5d96da823d08b119e1825172454e0c7a3d2294ffdb72ccef88ab0fb32d81899",
	"none/pickup":               "0b6c9f0f45b616e0af2b7d773c3f8568b86138c5e0b4ab2b9349ff0ab5da2e3c",
	"none/shopBookLabel":        "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	"none/shopItemName":         "ed9c405a6d25943f689722fc6ce8a1a4e4d54a33bc825e665b97933fb9fa3d66",
	"none/weapon":               "50b1801fa2ee152388041320db20bab832ae4472fb2c8c49e3e781286c96c7b3",
	"second/itemInfoLines":      "5d6846cda0e9800d6673d1cecd87bd5b6df16af6476f43190033604fda0d9fc9",
	"second/itemName":           "da3e60ff41327c1a138a7e175a2205e2b5aa4289910cde4ba7cf58e47b44e44d",
	"second/pickup":             "f97c484f6c5315bc31b5f89339f369542a978a78aa70cc8ae5bc7e9dce67f4c5",
	"second/shopBookLabel":      "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	"second/shopItemName":       "e5c3e4ddb277d7aa5c8ec07c45a9da7367b9684cce005acd53fd03ef56397ea5",
	"second/weapon":             "382ac5eb2bd14aeff0bea5605c6db453a43d861d1d2a798307bd8ffbdb992bd8",
	"secondNoShop/shopItemName": "fca0903d14ec28bf762f141c5f289f1eef97d1062200a62294eac7d00a1c1c2c",
	"secondRaw/shopItemName":    "e5c3e4ddb277d7aa5c8ec07c45a9da7367b9684cce005acd53fd03ef56397ea5",
}

// surfaceRows is a data.bin collection that answers every reader the two
// games' item names and shop rows use.
type surfaceRows []surfaceRow

type surfaceRow struct {
	name    string
	params  []int32
	doubles []float64
}

func (c surfaceRows) Len() int                     { return len(c) }
func (c surfaceRows) EntryName(i int) string       { return c[i].name }
func (c surfaceRows) EntryParams(i int) []int32    { return c[i].params }
func (c surfaceRows) EntryStrings(int) []string    { return nil }
func (c surfaceRows) EntryRaw(int) []byte          { return nil }
func (c surfaceRows) EntryDoubles(i int) []float64 { return c[i].doubles }

func surfaceNamedRows(params int, names ...string) surfaceRows {
	out := make(surfaceRows, len(names))
	for i, n := range names {
		out[i] = surfaceRow{name: n, params: make([]int32, params), doubles: []float64{0, 0, 0, 0, 1, 1, 1, 1, 0}}
	}
	return out
}

// firstSurfaceTable is a first-game table: two shapes, two materials, three
// weapons, a shield, an armour, four magic items and shipped names for a
// weapon, a magic item and an unknown code, one of them empty.
func firstSurfaceTable(t *testing.T) *mapload.Table {
	t.Helper()
	scale := func(name string) synth.DataBinRow {
		r := eqScaleRow(1, 1, 1)
		r.Name = name
		return r
	}
	armour := eqShieldRow("Mail", 5, 2)
	f, err := databin.Parse(synth.DataBin{
		Rows: [synth.DataBinCollections][]synth.DataBinRow{
			synth.DataBinShapes:    {scale("Short"), scale("Long")},
			synth.DataBinMaterials: {scale("Iron"), scale("Steel")},
			synth.DataBinWeapons: {
				eqWeaponRowHands("Sword", data.SkillBlade, 10, 20, 5, 3, 2, 7, 4, 0),
				eqWeaponRowHands("Mace", data.SkillBludgen, 10, 20, 500, 3, 2, 1, 1, 0),
				eqWeaponRowHands("Bow", 11, 6, 12, 8, 1, 6, 8, 5, 0),
			},
			synth.DataBinShields: {eqShieldRow("Ward", 7, 3)},
			synth.DataBinArmors:  {armour},
		},
	}.Bytes())
	if err != nil {
		t.Fatalf("databin.Parse: %v", err)
	}
	items := make(dbCollection, 5)
	items[1] = dbEntry{name: "Book_Fire", params: []int32{1000}}
	items[2] = dbEntry{name: "Potion", params: []int32{50}}
	items[3] = dbEntry{name: "Scroll_Fire_Arrow", params: []int32{100}}
	tab := &mapload.Table{
		Shapes: f.Collection(databin.Shapes), Materials: f.Collection(databin.Materials),
		Weapons: f.Collection(databin.Weapons), Shields: f.Collection(databin.Shields), Armors: f.Collection(databin.Armors),
		MagicItems: items,
		Names:      data.ItemNames{0x0e01: "Book", 0x0e02: "Potion of Life", 0x7777: "Odd", 0x0e04: ""},
	}
	for code := range 65536 {
		if w, err := data.WeaponFromCode(data.ItemCode(code), tab.Shapes, tab.Materials, tab.Weapons); err == nil && w.Name != "" {
			tab.Names[data.ItemCode(code)] = "Shipped " + w.Name
			break
		}
	}
	return readUnder(base.GameROM1, tab)
}

// secondSurfaceTable is a second-game table whose collections answer the
// shop's readers. The shipped names name one armour and one magic item and
// hold an empty line for another armour's key; the
// Magic effect table is left out when shop is false, so the shop's tables do
// not resolve.
func secondSurfaceTable(shop bool) *mapload.Table {
	armors := surfaceNamedRows(30, "Cloak", "Helm", "Boots")
	armors[0].params[4], armors[1].params[4], armors[2].params[4] = 6, 1, 4
	shields := surfaceNamedRows(30, "Buckler", "Tower")
	weapons := surfaceNamedRows(30, "Sword", "Axe", "Staff")
	spells := surfaceNamedRows(22, "", "Fire Arrow", "Heal", "")
	tab := &mapload.Table{
		Shapes: surfaceNamedRows(9, "Common", "Fine", "Rare"), Materials: surfaceNamedRows(9, "Iron", "Steel", ""),
		Armors: armors, Shields: shields, Weapons: weapons, Spells: spells,
		MagicItems: surfaceNamedRows(4, "", "Potion", "Scroll", "Wand"),
		Names:      data.ItemNames{0x6600: "Shipped Cloak", 0x0e02: "Shipped Scroll", 0x0101: ""},
	}
	if shop {
		tab.Magic = surfaceNamedRows(4, "", "castSpell", "", "")
	}
	return tab
}

// surfaceItems is every shop item the second-game tables can produce, a row,
// material and class past each table's end, with and without a cast spell.
func surfaceItems(t *mapload.Table) []secondshop.Item {
	var out []secondshop.Item
	cast := []sim.ItemEffect{{Kind: 41, Operand: 1 | 3<<16}}
	rows := map[secondshop.Kind]data.Collection{secondshop.Armor: t.Armors, secondshop.Shield: t.Shields, secondshop.Weapon: t.Weapons}
	for _, k := range []secondshop.Kind{secondshop.Armor, secondshop.Shield, secondshop.Weapon} {
		for row := 0; row <= rows[k].Len(); row++ {
			for m := 0; m <= t.Materials.Len(); m++ {
				for c := 0; c <= t.Shapes.Len(); c++ {
					out = append(out, secondshop.Item{Table: k, Row: row, Material: m, Class: c},
						secondshop.Item{Table: k, Row: row, Material: m, Class: c, Effects: cast})
				}
			}
		}
	}
	for row := 0; row <= t.MagicItems.Len(); row++ {
		out = append(out, secondshop.Item{Table: secondshop.Magic, Row: row}, secondshop.Item{Table: secondshop.Magic, Row: row, Effects: cast})
	}
	for spell := 0; spell <= t.Spells.Len(); spell++ {
		out = append(out, secondshop.Item{Table: secondshop.Book, Spell: spell})
	}
	return out
}

// surfaceSack is the editor's view of a map whose ground sacks each hold one
// of codes: the sack records' details and each sack's recipe item choices.
func surfaceSack(t *testing.T, tab *mapload.Table, codes []uint16) (details, choices []string) {
	t.Helper()
	m, err := mapedit.New(synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12)}))
	if err != nil {
		t.Fatal(err)
	}
	for i, code := range codes {
		if _, err := m.Place(mapedit.KindLoot, m.GroundSack(uint32(2+i)*256+128, 5*256+128, code)); err != nil {
			t.Fatal(err)
		}
	}
	x := &MapInspector{tiles: &terrain.Tileset{}, table: tab, structures: &terrain.StructureSet{}}
	doc, err := x.inspectModel(m, "source.alm", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range doc.Records {
		if r.Kind == "Sack" {
			details = append(details, fmt.Sprintf("%s | %s", strings.Join(r.Details, " / "), r.Warning))
		}
	}
	s := &mapInspectionEdits{inspector: x, model: m}
	for i := range codes {
		for _, c := range recipeItemChoices(s, fieldChoiceAsk{index: i}) {
			choices = append(choices, fmt.Sprintf("%d %s", c.Value, c.Label))
		}
	}
	return details, choices
}

// itemNameSurfaces is each surface's text over the fixture tables.
func itemNameSurfaces(t *testing.T) map[string][]string {
	t.Helper()
	first, second, secondRaw, secondNoShop := firstSurfaceTable(t), readUnder(base.GameROM2, secondSurfaceTable(true)), secondSurfaceTable(true), readUnder(base.GameROM2, secondSurfaceTable(false))
	tables := []struct {
		name string
		t    *mapload.Table
	}{{"first", first}, {"second", second}, {"none", nil}}
	out := map[string][]string{}
	words := ui.AuthoredWords()
	for _, tab := range tables {
		var names, info, pickup []string
		for code := range 65536 {
			c := data.ItemCode(code)
			names = append(names, fmt.Sprintf("%#04x %q", code, itemName(c, tab.t)))
			if tab.t != nil && (code < 0x1000 || code&0xff == 0x77 || code>>8 == 0x66) {
				info = append(info, fmt.Sprintf("%#04x %q", code, itemInfoLines(c, tab.t)))
			}
		}
		for _, code := range []uint16{0x0e01, 0x0e02, 0x0e03, 0x0e04, 0x0101, 0x7777, 0x6600, 0x1234} {
			pickup = append(pickup, pickupLinesForItems([]namedStack{{stack: sim.ItemStack{Code: code, Count: 2}, name: "Hero"}}, tab.t, &words)...)
			pickup = append(pickup, grantLinesForItems([]namedStack{{stack: sim.ItemStack{Code: code, Count: 1}, name: "Hero"}}, tab.t, &words)...)
		}
		out[tab.name+"/itemName"], out[tab.name+"/itemInfoLines"], out[tab.name+"/pickup"] = names, info, pickup
		out[tab.name+"/shopBookLabel"] = []string{shopBookLabel(tab.t)}

		var weapons []string
		for _, code := range []uint16{0x0101, 0x0102, 0x0201, 0x0e01, 0x1234, 0x7777, 0x6600} {
			plain := mapload.ItemInstanceFromCode(code, tab.t)
			source := sim.ItemInstance{Code: code, SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon, Definition: sim.SourceWeaponDefinition{Present: true}}}
			for _, item := range []sim.ItemInstance{plain, source} {
				for _, saved := range []string{"", "Saved Blade"} {
					fb := &data.Weapon{Code: data.ItemCode(code), Name: saved}
					p := mapload.PartyMember{Weapon: fb}
					p.WornItems[0] = item
					w := mapload.MemberWeapon(p, tab.t)
					r := currentPartyWeapon{NameCode: code, Name: []byte(saved)}.restore(item, false, tab.t)
					weapons = append(weapons, fmt.Sprintf("%#04x %d %q member %s restore %s", code, item.SourceEquipment.Class, saved, surfaceWeaponName(w), surfaceWeaponName(r)))
				}
			}
		}
		out[tab.name+"/weapon"] = weapons
	}

	for _, tab := range []struct {
		name string
		t    *mapload.Table
	}{{"second", second}, {"secondRaw", secondRaw}, {"secondNoShop", secondNoShop}, {"none", nil}} {
		s := &townScreen{shopController: shopController{in: &InstallResources{Table: tab.t}}}
		var lines []string
		items := surfaceItems(secondSurfaceTable(true))
		for _, it := range items {
			lines = append(lines, fmt.Sprintf("%+v %q", it, s.shopItemName(it)))
		}
		out[tab.name+"/shopItemName"] = lines
	}

	x := &MapInspector{table: first}
	var catalog []string
	for _, c := range x.recordCatalog("item") {
		catalog = append(catalog, fmt.Sprintf("%d %s", c.Value, c.Label))
	}
	out["first/editorCatalog"] = catalog
	var list []string
	for _, code := range []uint16{0x0e01, 0x0e02, 0x0e04, 0x0101, 0x7777, 0x1234} {
		list = append(list, x.itemList([]sim.ItemInstance{sim.PlainItem(code)}))
	}
	list = append(list, x.itemList([]sim.ItemInstance{sim.PlainItem(0x0e01), sim.PlainItem(0x1234), sim.PlainItem(0x0101)}))
	out["first/editorItemList"] = list
	details, choices := surfaceSack(t, first, []uint16{0x0e01, 0x0e04, 0x0101, 0x7777, 0x1234})
	out["first/editorSack"], out["first/editorRecipeItems"] = details, choices
	return out
}

func surfaceWeaponName(w *data.Weapon) string {
	if w == nil {
		return "nil"
	}
	return fmt.Sprintf("%q", w.Name)
}

// TestItemNameSurfacesKeepTheirText holds every item name the shop cells,
// sheets, pickup lines, item tips, current weapons and the editor's lists
// and sack records show, over both games' fixture tables, to the text the
// engine wrote before the item name had one resolver.
func TestItemNameSurfacesKeepTheirText(t *testing.T) {
	got := itemNameSurfaces(t)
	dump := os.Getenv("AGAINROM_ITEMNAME_SURFACES")
	keys := make([]string, 0, len(got))
	for k := range got {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		text := strings.Join(got[k], "\n")
		sum := sha256.Sum256([]byte(text))
		digest := hex.EncodeToString(sum[:])
		if dump != "" {
			if err := os.WriteFile(filepath.Join(dump, strings.ReplaceAll(k, "/", "-")+".txt"), []byte(text), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		if want := itemNameSurfaceDigests[k]; digest != want {
			t.Errorf("%q: %q,", k, digest)
		}
	}
	if len(got) != len(itemNameSurfaceDigests) {
		t.Errorf("%d surfaces, %d digests", len(got), len(itemNameSurfaceDigests))
	}
}

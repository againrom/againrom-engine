package game

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/vfs"
)

func testMaterial(modID, item, tier, material string, allow bool) mod.ItemMaterial {
	return mod.ItemMaterial{Mod: modID, File: "data/items.toml", Line: 7, AllowLine: 11, Item: item, Tier: tier, Material: material, Allow: allow}
}

func testWeapon(modID, key, base string, params map[string]int32) mod.WeaponRow {
	w := mod.WeaponRow{Mod: modID, File: "data/items.toml", Line: 20, Key: key, Name: "Name " + key, NameLine: 22, Base: base, BaseLine: 23}
	for _, p := range mod.WeaponParams {
		if v, ok := params[p.Key]; ok {
			w.Params = append(w.Params, mod.WeaponParamValue{WeaponParam: p, Value: v, Line: 24})
		}
	}
	return w
}

func weaponPoolCodes(tab *mapload.Table) []data.ItemCode {
	var out []data.ItemCode
	for _, c := range shopWeaponPool(tab, 1<<30) {
		out = append(out, c.Code)
	}
	return out
}

func armourPoolCodes(tab *mapload.Table) []data.ItemCode {
	var out []data.ItemCode
	for _, c := range shopArmourPool(tab, 1<<30) {
		out = append(out, c.Code)
	}
	return out
}

// itemFrontWithIcons is itemFront over a container filesystem holding the
// inventory icon of each code.
func itemFrontWithIcons(t *testing.T, tab *mapload.Table, codes ...data.ItemCode) *FrontEnd {
	t.Helper()
	fs, err := vfs.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range codes {
		fs.Overlay(graphicsPrefix+data.ItemIconPath(c), func() ([]byte, error) { return []byte{1}, nil })
	}
	f := itemFront(tab)
	f.Archives = &Archives{Containers: fs}
	return f
}

func TestSetModItemsAllowedMaterialRollsInTheShop(t *testing.T) {
	tab := fixtureItemTable(10, 5, 5)
	leatherDagger := data.ComposeItemCode(1, 1, 0, 1)
	if slices.Contains(weaponPoolCodes(tab), leatherDagger) {
		t.Fatal("the fixture already sells a leather dagger")
	}
	f := itemFrontWithIcons(t, tab, leatherDagger)
	m := testMaterial("m", "Dagger", "Common", "Leather", true)
	m.NameKey, m.Name, m.NameLine = "n", "Hide Knife", 12
	if err := f.SetModItems(mod.ItemData{Materials: []mod.ItemMaterial{m}}); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(weaponPoolCodes(tab), leatherDagger) || !slices.Contains(weaponPoolCodes(tab), data.ComposeItemCode(0, 1, 0, 1)) {
		t.Fatalf("pool %v", weaponPoolCodes(tab))
	}
	if name, ok := tab.Names.NameFor(leatherDagger); !ok || name != "Hide Knife" {
		t.Fatalf("name %q %v", name, ok)
	}
	if w, err := data.WeaponFromCode(leatherDagger, tab.Shapes, tab.Materials, tab.Weapons); err != nil || w.Row != 1 {
		t.Fatalf("%+v %v", w, err)
	}
	if len(tab.Mods.Items) != 0 {
		t.Fatal("a pair of a shipped row became a mod item")
	}
}

func TestSetModItemsForbiddenMaterialNeverRolls(t *testing.T) {
	tab := fixtureItemTable(10, 5, 5)
	leatherMail := data.ComposeItemCode(1, 7, 0, 1)
	if !slices.Contains(armourPoolCodes(tab), leatherMail) {
		t.Fatal("the fixture sells no leather soft mail")
	}
	if err := itemFront(tab).SetModItems(mod.ItemData{Materials: []mod.ItemMaterial{testMaterial("m", "Soft Mail", "Common", "Leather", false)}}); err != nil {
		t.Fatal(err)
	}
	pool := armourPoolCodes(tab)
	if slices.Contains(pool, leatherMail) || !slices.Contains(pool, data.ComposeItemCode(2, 7, 0, 1)) {
		t.Fatalf("pool %v", pool)
	}
	// The forbidden pair still resolves: a save or a script may hold it.
	if _, err := data.ArmorFromCode(leatherMail, tab.Shapes, tab.Materials, tab.Armors); err != nil {
		t.Fatal(err)
	}
}

func TestSetModItemsRefusesAnAllowedPairWithoutAPicture(t *testing.T) {
	tab := fixtureItemTable(10, 5, 5)
	err := itemFrontWithIcons(t, tab).SetModItems(mod.ItemData{Materials: []mod.ItemMaterial{testMaterial("m", "Dagger", "Rare", "Leather", true)}})
	want := `mod "m": data/items.toml:7: weapon row "Dagger" tier "Rare" material "Leather" has no inventory picture`
	if err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("%v", err)
	}
}

func TestSetModItemsMaterialRefusals(t *testing.T) {
	for _, c := range []struct {
		m    mod.ItemMaterial
		want string
	}{
		{testMaterial("m", "Gown", "Common", "Iron", true), `data/items.toml:7: "Gown" is neither a row`},
		{testMaterial("m", "Dagger", "Epic", "Iron", true), `data/items.toml:7: tier "Epic" is not one of "Common", "Rare"`},
		{testMaterial("m", "Dagger", "Common", "Mithril", true), `data/items.toml:7: material "Mithril" is not one of`},
		{testMaterial("m", "star", "Common", "Iron", true), `"star" is neither a row`},
	} {
		err := itemFront(fixtureItemTable(10, 5, 5)).SetModItems(mod.ItemData{Materials: []mod.ItemMaterial{c.m}})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.want, err)
		}
	}
}

func TestSetModItemsAddsAWeaponRowFromABaseRow(t *testing.T) {
	tab := fixtureItemTable(10, 5, 5)
	base := tab.Weapons.EntryParams(1)
	w := testWeapon("m", "star", "Dagger", map[string]int32{"damage-min": 9, "damage-max": 14, "range": 3, "price": 300})
	if err := itemFront(tab).SetModItems(mod.ItemData{Weapons: []mod.WeaponRow{w}}); err != nil {
		t.Fatal(err)
	}
	if tab.Weapons.Len() != 6 || tab.Weapons.EntryName(5) != "star" {
		t.Fatalf("weapons %d %q", tab.Weapons.Len(), tab.Weapons.EntryName(5))
	}
	p := tab.Weapons.EntryParams(5)
	for col, v := range map[int]int32{2: 300, 6: 9, 7: 14, 11: 3} {
		if p[col] != v {
			t.Fatalf("column %d = %d", col, p[col])
		}
	}
	for _, col := range []int{0, 1, 3, 4, 5, 8, 9, 10, 12, 13, 14, 15, 16} {
		if p[col] != base[col] {
			t.Fatalf("column %d = %d, base %d", col, p[col], base[col])
		}
	}
	code := data.ComposeItemCode(0, 1, 0, 5)
	if !slices.Contains(weaponPoolCodes(tab), code) || slices.Contains(weaponPoolCodes(tab), data.ComposeItemCode(1, 1, 0, 5)) {
		t.Fatalf("pool %v: the row takes the base row's masks", weaponPoolCodes(tab))
	}
	got, err := data.WeaponFromCode(code, tab.Shapes, tab.Materials, tab.Weapons)
	// The fixture's Common Iron damage factor is 0.2: 9 and 14 make 2 and 1.
	if err != nil || got.Row != 5 || got.DamageBase != 2 || got.DamageSpread != 1 || got.Range != 3 {
		t.Fatalf("%+v %v", got, err)
	}
	if name, ok := tab.Names.NameFor(code); !ok || name != "Name star" {
		t.Fatalf("name %q %v", name, ok)
	}
	var item mapload.ModItem
	for _, m := range tab.Mods.Items {
		if m.Code == uint16(code) {
			item = m
		}
	}
	if item.Row != 5 || item.StandInRow != 1 || item.StandInCode != uint16(data.ComposeItemCode(0, 1, 0, 1)) || item.Key != "star" || item.Price != 300 {
		t.Fatalf("%+v", item)
	}
	// One mod item per pair: two shapes and three materials in the fixture scales.
	if len(tab.Mods.Items) != 2*3 {
		t.Fatalf("%d mod items", len(tab.Mods.Items))
	}
}

func TestSetModItemsAddedWeaponRowIsDrawnWithItsBaseRowsBody(t *testing.T) {
	tab := fixtureItemTable(10, 5, 5)
	f := itemFrontWithIcons(t, tab, data.ComposeItemCode(0, 1, 0, 1))
	f.Bodies = data.NewBodyList("swordsman", "axeman", "spearman", "bowman")
	if err := f.SetModItems(mod.ItemData{Weapons: []mod.WeaponRow{testWeapon("m", "star", "Dagger", nil)}}); err != nil {
		t.Fatal(err)
	}
	if b := f.Bodies.RowEntry(5); b != "swordsman" {
		t.Fatalf("row 5 answers %q", b)
	}
	var eq data.Equipment
	eq.SetCode(1, data.ComposeItemCode(0, 1, 0, 5))
	if b, ok := data.HeroBodyFor(f.Bodies, eq); !ok || b != "swordsman" {
		t.Fatalf("%q %v", b, ok)
	}
	// Every reader of the list through the front end's files sees the row.
	list, ok := ReadBodyList(mapBodySource{f.Archives.Containers, data.NewBodyList("swordsman", "axeman", "spearman", "bowman")})
	if !ok || list.RowEntry(5) != "swordsman" || len(list.AddedRows()) != 1 {
		t.Fatalf("%v %v", list.AddedRows(), ok)
	}
}

// mapBodySource serves a shipped body list beside a front end's files.
type mapBodySource struct {
	fs   *vfs.FS
	list data.BodyList
}

func (s mapBodySource) ReadFile(address string) ([]byte, error) {
	if address == HeroPictureAddress {
		var b strings.Builder
		for i := 0; i < s.list.Len(); i++ {
			b.WriteString(string(s.list.Entry(i)) + "\n")
		}
		return []byte(b.String()), nil
	}
	return s.fs.ReadFile(address)
}

func TestSetModItemsWeaponRowTakesMaterialsAndPictures(t *testing.T) {
	tab := fixtureItemTable(10, 5, 5)
	base := data.ComposeItemCode(0, 1, 0, 1)
	f := itemFrontWithIcons(t, tab, base)
	err := f.SetModItems(mod.ItemData{Weapons: []mod.WeaponRow{testWeapon("m", "star", "Dagger", nil)},
		Materials: []mod.ItemMaterial{testMaterial("m", "star", "Common", "Leather", true)}})
	want := `mod "m": data/items.toml:7: weapon row "star" tier "Common" material "Leather" has no inventory picture`
	if err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("%v", err)
	}
	tab = fixtureItemTable(10, 5, 5)
	f = itemFrontWithIcons(t, tab, base, data.ComposeItemCode(1, 1, 0, 1))
	if err := f.SetModItems(mod.ItemData{Weapons: []mod.WeaponRow{testWeapon("m", "star", "Dagger", nil)},
		Materials: []mod.ItemMaterial{testMaterial("m", "star", "Common", "Leather", true)}}); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(weaponPoolCodes(tab), data.ComposeItemCode(1, 1, 0, 5)) {
		t.Fatalf("pool %v", weaponPoolCodes(tab))
	}
	// The added row's icon is the base row's icon of the same pair.
	b, err := f.Archives.Containers.ReadFile(graphicsPrefix + data.ItemIconPath(data.ComposeItemCode(1, 1, 0, 5)))
	if err != nil || len(b) != 1 {
		t.Fatalf("%v %v", b, err)
	}
}

func TestSetModItemsWeaponRowRefusals(t *testing.T) {
	for _, c := range []struct {
		items mod.ItemData
		tab   *mapload.Table
		want  string
	}{
		{mod.ItemData{Weapons: []mod.WeaponRow{testWeapon("m", "star", "Halberd", nil)}}, fixtureItemTable(10, 5, 5),
			`mod "m": data/items.toml:23: base "Halberd": the weapon table has no row named "Halberd"`},
		{mod.ItemData{Weapons: []mod.WeaponRow{testWeapon("m", "star", "Dagger", nil)}}, fixtureItemTable(10, 5, 32),
			`mod "m": data/items.toml:20: no free row: the weapon table holds 32 rows`},
		{mod.ItemData{Weapons: []mod.WeaponRow{testWeapon("a", "star", "Dagger", nil), testWeapon("b", "star", "Dagger", nil)}}, fixtureItemTable(10, 5, 5),
			`item star is added by mod "a" and named by mod "b", and neither loads after the other`},
		{mod.ItemData{Weapons: []mod.WeaponRow{testWeapon("a", "star", "Dagger", nil)}, Materials: []mod.ItemMaterial{testMaterial("b", "star", "Common", "Iron", false)}}, fixtureItemTable(10, 5, 5),
			`item star is added by mod "a" and named by mod "b"`},
		{mod.ItemData{Materials: []mod.ItemMaterial{testMaterial("a", "Dagger", "Common", "Iron", false), testMaterial("b", "Dagger", "Common", "Iron", true)}}, fixtureItemTable(10, 5, 5),
			`item weapon Dagger Common Iron is set by mod "a" and by mod "b", and neither loads after the other`},
	} {
		err := itemFront(c.tab).SetModItems(c.items)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.want, err)
		}
	}
}

func TestSetModItemsLaterModWinsAMaskBitWithLoadAfter(t *testing.T) {
	tab := fixtureItemTable(10, 5, 5)
	items := mod.ItemData{
		Materials: []mod.ItemMaterial{testMaterial("a", "Soft Mail", "Common", "Leather", true), testMaterial("b", "Soft Mail", "Common", "Leather", false)},
		Mods:      []string{"a", "b"}, Order: mod.Precedence{"b": {"a"}},
	}
	overrides, err := itemFront(tab).setModItems(items)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(armourPoolCodes(tab), data.ComposeItemCode(1, 7, 0, 1)) {
		t.Fatal("the earlier mod's bit won")
	}
	if len(overrides) != 1 || overrides[0].String() != `item armour Soft Mail Common Leather: mod "b" (data/items.toml:11) overrides mod "a"` {
		t.Fatalf("%v", overrides)
	}
}

func TestSetModItemsTwoChangesOfOneFieldNeedLoadAfter(t *testing.T) {
	w1, w2 := int32(20), int32(30)
	changes := []mod.ItemChange{
		{Mod: "a", File: "data/items.toml", Line: 2, Item: "Chain Mail", Weight: &w1},
		{Mod: "b", File: "data/items.toml", Line: 3, Item: "Chain Mail", Weight: &w2},
	}
	err := itemFront(fixtureItemTable(10, 5, 5)).SetModItems(mod.ItemData{Changes: changes})
	if err == nil || !strings.Contains(err.Error(), `item armour Chain Mail weight is set by mod "a" and by mod "b"`) {
		t.Fatalf("%v", err)
	}
	tab := fixtureItemTable(10, 5, 5)
	if err := itemFront(tab).SetModItems(mod.ItemData{Changes: changes, Order: mod.Precedence{"b": {"a"}}}); err != nil {
		t.Fatal(err)
	}
	if a, err := data.ArmorFromCode(data.ComposeItemCode(0, 7, 0, 3), tab.Shapes, tab.Materials, tab.Armors); err != nil || a.Weight != 30 {
		t.Fatalf("%+v %v", a, err)
	}
}

func TestSetModItemsWeaponRowRefusesDamageTheByteWouldWrap(t *testing.T) {
	// The fixture's tier and material damage factors are below 1; scale the
	// Common tier to 2 so a pair can pass 255.
	big := func() *mapload.Table {
		tab := fixtureItemTable(10, 5, 5)
		tab.Shapes = fixtureScale{names: []string{"Common", "Rare"}, doubles: [][]float64{{0, 0, 1, 1, 2, 1, 0.25, 1, 1}, {0, 0, 4, 1.2, 2, 1.4, 0.36, 1.2, 3}}}
		tab.Materials = fixtureScale{names: []string{"Iron", "Leather", "Hard Leather"}, doubles: [][]float64{{0, 0, 1, 1, 1, 1, 1, 1, 0}, {0, 0, 0.5, 0.2, 1, 1, 0.3, 1, 0}, {0, 0, 0.6, 0.4, 1, 1, 0.6, 1, 1}}}
		return tab
	}
	for _, c := range []struct {
		params map[string]int32
		want   string
	}{
		{map[string]int32{"damage-min": 100}, `mod "m": data/items.toml:24: weapon row "star": damage-min is 100 and damage-max is 0`},
		{map[string]int32{"damage-min": 80, "damage-max": 40}, `damage-min is 80 and damage-max is 40`},
		{map[string]int32{"damage-min": 130, "damage-max": 140}, `damage-min 130 and damage-max 140 scale to base 260 and spread 20 at tier "Common" material "Iron"`},
		{map[string]int32{"damage-min": 0, "damage-max": 130}, `scale to base 0 and spread 260 at tier "Common" material "Iron"`},
	} {
		err := itemFront(big()).SetModItems(mod.ItemData{Weapons: []mod.WeaponRow{testWeapon("m", "star", "Dagger", c.params)}})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%v: %v; want %q", c.params, err, c.want)
		}
	}
	// In order and within a byte on every pair the masks allow: 127 * 2 = 254.
	if err := itemFront(big()).SetModItems(mod.ItemData{Weapons: []mod.WeaponRow{testWeapon("m", "star", "Dagger", map[string]int32{"damage-min": 100, "damage-max": 127})}}); err != nil {
		t.Errorf("an in-order row within a byte is refused: %v", err)
	}
}

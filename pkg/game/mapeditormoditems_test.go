package game

import (
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/secondshop"
	"againrom/pkg/ui"
)

// itemsPage is W with a new mod folder open over the fixture item tables.
func itemsPage(t *testing.T) (*MapInspector, *mapInspectionEdits, string) {
	t.Helper()
	x, s := contextEditor(t)
	x.setEditorMod(nil)
	x.table = fixtureItemTable(10, 5, 5)
	// The install holds the picture of each item's first pair, as the
	// shipped installs do; an added weapon is served its base's picture.
	for i, it := range x.installItems() {
		icon := packIconStream(byte(20+i), 30, 40)
		x.archives.Containers.Overlay(graphicsPrefix+data.ItemIconPath(data.ItemCode(it.code)), func() ([]byte, error) { return icon, nil })
	}
	dir := filepath.Join(t.TempDir(), "authored-map")
	d, err := s.SetWorkspaceField("package", "folder", dir)
	if err != nil || d == nil {
		t.Fatalf("make %s: %v", dir, err)
	}
	return x, d.Edits.(*mapInspectionEdits), dir
}

// itemsField is the items page's field key; false when the page has none.
func itemsField(s *mapInspectionEdits, key string) (ui.InspectionField, bool) {
	for _, f := range s.WorkspaceFields(ui.ModItemsScope) {
		if f.Key == key {
			return f, true
		}
	}
	return ui.InspectionField{}, false
}

// setItems makes one edit on the items page, failing the test on a refusal.
func setItems(t *testing.T, s *mapInspectionEdits, key, value string) {
	t.Helper()
	if _, err := s.SetWorkspaceField(ui.ModItemsScope, key, value); err != nil {
		t.Fatalf("%s = %q: %v", key, value, err)
	}
}

// installValue is the choice value the page's item choices give the install
// item named name: its place among the install items, counted from 1.
func installValue(t *testing.T, x *MapInspector, name string) string {
	t.Helper()
	i := slices.IndexFunc(x.installItems(), func(it installItem) bool { return it.name == name })
	if i < 0 {
		t.Fatalf("no install item %q in %+v", name, x.installItems())
	}
	return strconv.Itoa(i + 1)
}

// modItemOf is the mod item key of the context the editor reads by.
func modItemOf(x *MapInspector, key string) (mapload.ModItem, bool) {
	if x.mod == nil || x.mod.context == nil {
		return mapload.ModItem{}, false
	}
	for _, m := range x.mod.context.table.Mods.Items {
		if m.Key == key {
			return m, true
		}
	}
	return mapload.ModItem{}, false
}

// The page adds an armour and a weapon row from install items, changes and
// switches off a shipped row, allows a material pair and switches its own
// row off; each edit is one draft step the context reads, undone and redone,
// and an undo past a save takes the saved files out again.
func TestItemsPageAddsChangesAndSwitchesOffRows(t *testing.T) {
	x, s, dir := itemsPage(t)
	made := tree(t, dir)
	steps := len(s.undoWorkspace)
	setItems(t, s, itemsNewArmour, installValue(t, x, "Soft Mail"))
	setItems(t, s, "item.armour.price", "55")
	setItems(t, s, "item.armour.shop", "2")
	setItems(t, s, itemsNewWeapon, installValue(t, x, "Dagger"))
	setItems(t, s, "weapon.weapon.price", "300")
	setItems(t, s, itemsNewChange, installValue(t, x, "Cap"))
	setItems(t, s, "change.0.defence", "12")
	setItems(t, s, "change.0.off", "")
	setItems(t, s, itemsNewMaterial, installValue(t, x, "Soft Mail"))
	pair, _ := assetFile(t, "pair.png", assetPNG(t, 24, 24, color.RGBA{90, 90, 90, 255}))
	setItems(t, s, "material.0.sprite", pair)
	if len(s.undoWorkspace) != steps+10 {
		t.Fatalf("steps %d, want 10", len(s.undoWorkspace)-steps)
	}
	if m, ok := modItemOf(x, "armour"); !ok || m.Price != 55 || !m.Stock {
		t.Fatalf("the context's armour %+v, %v", m, ok)
	}
	if _, ok := modItemOf(x, "weapon"); !ok {
		t.Fatal("the context has no mod weapon")
	}
	tab := x.mod.context.table
	if data.ItemMasks(tab.Armors.(data.MaskTable), 2) != ([data.ShopShapes]uint16{}) || tab.Armors.EntryParams(2)[9] == 12 {
		t.Fatalf("the shipped row switched off keeps a pair or takes the change: %v", tab.Armors.EntryParams(2))
	}
	src := string(s.workspace.Project.Files[mod.ItemsFile])
	for _, want := range []string{"[[item]]\nkey = \"armour\"", "Soft Mail", "price = 55", "[[weapon]]\nkey = \"weapon\"", "Dagger", "price = 300",
		"[[change]]\nitem = \"Cap\"", "defence = 12", "off = true", "[[material]]\nitem = \"Soft Mail\"", "Common", "Iron", "allow = true", "sprite = \"assets/items/pair.png\""} {
		if !strings.Contains(src, want) {
			t.Fatalf("the items file lacks %q:\n%s", want, src)
		}
	}
	if f, ok := itemsField(s, "material.0.tier"); !ok || f.Value != 1 || len(f.Choices) != 2 {
		t.Fatalf("the material's tier choice %+v", f)
	}
	setItems(t, s, "item.armour.off", "")
	if _, ok := itemsField(s, "item.armour.state"); !ok {
		t.Fatal("the switched-off row shows no state")
	}
	if _, ok := modItemOf(x, "armour"); ok {
		t.Fatal("the switched-off armour keeps a code")
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if _, ok := modItemOf(x, "armour"); !ok {
		t.Fatal("the undo of the switch left the armour off")
	}
	if _, err := s.Redo(); err != nil {
		t.Fatal(err)
	}
	if _, ok := modItemOf(x, "armour"); ok {
		t.Fatal("the redo did not switch the armour off")
	}
	if !slices.Equal(keysOf(made), keysOf(tree(t, dir))) {
		t.Fatal("the page wrote into the mod folder before a save")
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if got := tree(t, dir)[mod.ItemsFile]; got != string(s.workspace.Project.Files[mod.ItemsFile]) {
		t.Fatalf("the saved items file:\n%s", got)
	}
	for range 11 {
		if _, err := s.Undo(); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := modItemOf(x, "weapon"); ok {
		t.Fatal("the undo of every step keeps the weapon")
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if after := tree(t, dir); !slices.Equal(keysOf(made), keysOf(after)) {
		t.Fatalf("the save after the undo past a save holds %v, want %v", keysOf(after), keysOf(made))
	}
}

// A choice value the page does not offer is refused and makes no step, the
// prompt of a choice makes none either, and a number the reader refuses is
// refused under the field's label.
func TestItemsPageRefusesAFieldByTheReadersRule(t *testing.T) {
	x, s, _ := itemsPage(t)
	steps := len(s.undoWorkspace)
	for _, c := range []struct{ key, value string }{
		{itemsNewArmour, "99"}, {itemsNewArmour, installValue(t, x, "Dagger")}, {itemsNewWeapon, installValue(t, x, "Cap")},
		{itemsNewChange, "No Such Row"}, {itemsNewMaterial, "-1"},
	} {
		if _, err := s.SetWorkspaceField(ui.ModItemsScope, c.key, c.value); err == nil {
			t.Fatalf("%s = %q was taken", c.key, c.value)
		}
	}
	for _, key := range []string{itemsNewArmour, itemsNewWeapon, itemsNewChange, itemsNewMaterial} {
		if _, err := s.SetWorkspaceField(ui.ModItemsScope, key, "0"); err != nil {
			t.Fatalf("the prompt of %s: %v", key, err)
		}
	}
	if len(s.undoWorkspace) != steps {
		t.Fatalf("a refused field made %d steps", len(s.undoWorkspace)-steps)
	}
	setItems(t, s, itemsNewArmour, installValue(t, x, "Cap"))
	steps = len(s.undoWorkspace)
	if _, err := s.SetWorkspaceField(ui.ModItemsScope, "item.armour.price", "a lot"); err == nil || !strings.Contains(err.Error(), x.word("editor.items.price")) {
		t.Fatalf("a price in words: %v", err)
	}
	if _, err := s.SetWorkspaceField(ui.ModItemsScope, "item.armour.stand-in", installValue(t, x, "Dagger")); err == nil {
		t.Fatal("a weapon took the stand-in of a helmet")
	}
	if len(s.undoWorkspace) != steps {
		t.Fatalf("a refused field made %d steps", len(s.undoWorkspace)-steps)
	}
}

// Every field the page shows a modder to type into holds a name or a plain
// number: a row key, game row name, file path, tier, material or mask is a
// choice, the file choice, or a picture. Each choice of install items shows
// their pictures, and every editable field has a label and a help line.
func TestItemsPageTypesOnlyNamesAndNumbers(t *testing.T) {
	x, s, _ := itemsPage(t)
	setItems(t, s, itemsNewArmour, installValue(t, x, "Soft Mail"))
	setItems(t, s, itemsNewWeapon, installValue(t, x, "Dagger"))
	setItems(t, s, itemsNewChange, installValue(t, x, "Cap"))
	setItems(t, s, itemsNewMaterial, installValue(t, x, "Soft Mail"))
	numbers := []string{"defence", "absorption", "weight", "price"}
	for _, p := range mod.WeaponParams {
		numbers = append(numbers, p.Key)
	}
	typed, choices := 0, 0
	for _, f := range s.WorkspaceFields(ui.ModItemsScope) {
		if f.ReadOnly || f.Action {
			continue
		}
		if f.Label == "" || f.Meaning == "" {
			t.Errorf("field %s: label %q, help %q", f.Key, f.Label, f.Meaning)
		}
		last := f.Key[strings.LastIndex(f.Key, ".")+1:]
		switch {
		case !f.Text:
			choices++
			if len(f.Choices) == 0 {
				t.Errorf("choice %s offers nothing", f.Key)
			}
			if strings.HasPrefix(f.Key, "new.") || strings.HasSuffix(f.Key, ".stand-in") || strings.HasSuffix(f.Key, ".base") || strings.HasSuffix(f.Key, ".item") {
				if len(f.Icons) == 0 {
					t.Errorf("item choice %s shows no picture", f.Key)
				}
			}
		case len(f.Pick) != 0:
			if last != "sprite" {
				t.Errorf("file choice %s", f.Key)
			}
		case strings.Contains(f.Key, ".name."):
			typed++
		case slices.Contains(numbers, last):
			typed++
			if strings.Contains(f.Label, last) && f.Label != x.word("editor.items.price") {
				t.Errorf("number %s is labelled with its key: %q", f.Key, f.Label)
			}
		default:
			t.Errorf("free text field %s (%q) holds neither a name nor a number", f.Key, f.Label)
		}
	}
	if typed == 0 || choices < 10 {
		t.Fatalf("typed %d, choices %d", typed, choices)
	}
	for _, key := range []string{"item.armour.stand-in", "weapon.weapon.base", "change.0.item", "material.0.item", "material.0.tier", "material.0.material"} {
		if f, ok := itemsField(s, key); !ok || f.Text || f.Value == 0 {
			t.Errorf("%s is no chosen choice: %+v", key, f)
		}
	}
}

// A picture the item layer's rule takes is imported under assets/items as
// the row's sprite in one step and shown on the page; a picture over the
// rule's side is refused in its words.
func TestItemsPageImportsASpriteByTheLayersRule(t *testing.T) {
	x, s, _ := itemsPage(t)
	setItems(t, s, itemsNewArmour, installValue(t, x, "Soft Mail"))
	steps := len(s.undoWorkspace)
	big, _ := assetFile(t, "big.png", assetPNG(t, maxModIconSide+1, 8, color.RGBA{200, 40, 40, 255}))
	if _, err := s.SetWorkspaceField(ui.ModItemsScope, "item.armour.sprite", big); err == nil || !strings.Contains(err.Error(), "pixels on each side") {
		t.Fatalf("an oversized picture: %v", err)
	}
	source, png := assetFile(t, "bone.png", assetPNG(t, 32, 32, color.RGBA{200, 40, 40, 255}))
	setItems(t, s, "item.armour.sprite", source)
	rel := mod.AssetsFolder + "/items/bone.png"
	if string(s.workspace.Project.Files[rel]) != string(png) || len(s.undoWorkspace) != steps+1 {
		t.Fatalf("the sprite's draft: files %v, steps %d", keysOf(s.workspace.Project.Files), len(s.undoWorkspace)-steps)
	}
	if f, ok := itemsField(s, "item.armour.sprite"); !ok || f.TextValue != rel || len(f.Pick) != 1 {
		t.Fatalf("the sprite field %+v", f)
	}
	f, ok := itemsField(s, "item.armour.look")
	if !ok || f.Picture == nil || f.Picture.Bounds().Dx() != 32 || f.Picture.RGBAAt(4, 4) != (color.RGBA{200, 40, 40, 255}) {
		t.Fatalf("the sprite is not shown: %+v", f)
	}
}

// The party of a mission offers a weapon the page adds before a save, by the
// name the page gives it.
func TestItemsPageWeaponIsOfferedToTheParty(t *testing.T) {
	x, n, _ := itemsPage(t)
	code := int64(data.ComposeItemCode(0, 1, 0, 5))
	setItems(t, n, itemsNewWeapon, installValue(t, x, "Dagger"))
	setItems(t, n, "weapon.weapon.name.en", "Star Blade")
	if _, err := n.SetWorkspaceField("package", "landing", "1"); err != nil {
		t.Fatal(err)
	}
	for _, f := range n.WorkspaceFields("package") {
		if f.Key == "party.0.weapon" {
			if !slices.ContainsFunc(f.Choices, func(c ui.InspectionChoice) bool { return c.Value == code }) {
				t.Fatalf("weapon choices %+v", f.Choices)
			}
			return
		}
	}
	t.Fatal("no party weapon field")
}

// An armour and a weapon added on the page by choices and typed names save
// into the items file and the language files, publish and load with the
// names in both languages, and the launch's data check takes them.
func TestItemsPageAddedRowsRoundTrip(t *testing.T) {
	x, s, dir := itemsPage(t)
	setItems(t, s, itemsNewArmour, installValue(t, x, "Chain Mail"))
	setItems(t, s, "item.armour.name.en", "Bone Mail")
	setItems(t, s, "item.armour.name.ru", "Костяная броня")
	source, _ := assetFile(t, "bone.png", assetPNG(t, 32, 32, color.RGBA{200, 40, 40, 255}))
	setItems(t, s, "item.armour.sprite", source)
	setItems(t, s, itemsNewWeapon, installValue(t, x, "Dagger"))
	setItems(t, s, "weapon.weapon.name.en", "Star Blade")
	setItems(t, s, "weapon.weapon.name.ru", "Звёздный клинок")
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	saved := tree(t, dir)
	for file, wants := range map[string][]string{
		mod.ItemsFile:         {"key = \"armour\"", "Chain Mail", "key = \"weapon\"", "Dagger"},
		mod.StringsFile("en"): {"Bone Mail", "Star Blade"},
		mod.StringsFile("ru"): {"Костяная броня", "Звёздный клинок"},
	} {
		for _, want := range wants {
			if !strings.Contains(saved[file], want) {
				t.Fatalf("the saved %s lacks %q:\n%s", file, want, saved[file])
			}
		}
	}
	out := t.TempDir()
	if _, err := s.SetWorkspaceField("package", "output", out); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetWorkspaceField("package", "publish", ""); err != nil {
		t.Fatal(err)
	}
	entries, err := mod.Resolve(out, []string{"authored-map"})
	if err != nil || len(entries) != 1 {
		t.Fatal(entries, err)
	}
	for game, want := range map[string][2]string{"rom1-en": {"Bone Mail", "Star Blade"}, "rom1-ru": {"Костяная броня", "Звёздный клинок"}} {
		res, err := modrt.Load(entries, game, nil, modrt.Options{})
		if err != nil {
			t.Fatalf("%s: %v", game, err)
		}
		if len(res.Items.Rows) != 1 || len(res.Items.Weapons) != 1 {
			t.Fatalf("%s: the published items %+v", game, res.Items)
		}
		armour, weapon := res.Items.Rows[0], res.Items.Weapons[0]
		if armour.Key != "armour" || armour.StandIn != "Chain Mail" || weapon.Key != "weapon" || weapon.Base != "Dagger" {
			t.Fatalf("%s: the published rows %+v, %+v", game, armour, weapon)
		}
		if armour.Name != want[0] || weapon.Name != want[1] {
			t.Fatalf("%s: the published names %q, %q, want %q", game, armour.Name, weapon.Name, want)
		}
		tab := fixtureItemTable(10, 5, 5)
		tab.Spells = nil
		res.Data.Missions = mod.MissionData{}
		if _, err := itemFront(tab).SetModData(res.Data); err != nil {
			t.Fatalf("%s: the launch's check refused the published mod: %v", game, err)
		}
	}
}

// A mod made on the page alone, applying to both games, saves through the
// project writer with no entry-file edit, publishes, loads and passes the
// launch's data check; the first game's stock generator and the second
// game's shop both offer its armour.
func TestItemsPageModPublishesAndShopsOfferIt(t *testing.T) {
	x, s, dir := itemsPage(t)
	made := tree(t, dir)
	if _, err := s.SetWorkspaceField("package", "applies-to", "0"); err != nil {
		t.Fatal(err)
	}
	setItems(t, s, itemsNewArmour, installValue(t, x, "Soft Mail"))
	setItems(t, s, "item.armour.price", "55")
	setItems(t, s, "item.armour.shop", "2")
	source, _ := assetFile(t, "bone.png", assetPNG(t, 32, 32, color.RGBA{200, 40, 40, 255}))
	setItems(t, s, "item.armour.sprite", source)
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	saved := tree(t, dir)
	if saved[mod.EntryName] != made[mod.EntryName] {
		t.Fatalf("the save changed the entry file:\n%s", saved[mod.EntryName])
	}
	if _, ok := saved[mod.AssetsFolder+"/items/bone.png"]; !ok {
		t.Fatalf("the saved folder %v", keysOf(saved))
	}
	out := t.TempDir()
	if _, err := s.SetWorkspaceField("package", "output", out); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetWorkspaceField("package", "publish", ""); err != nil {
		t.Fatal(err)
	}
	entries, err := mod.Resolve(out, []string{"authored-map"})
	if err != nil || len(entries) != 1 {
		t.Fatal(entries, err)
	}
	for _, game := range []string{"rom1-en", "rom2-en"} {
		res, err := modrt.Load(entries, game, nil, modrt.Options{})
		if err != nil {
			t.Fatalf("%s: %v", game, err)
		}
		if len(res.Items.Rows) != 1 || res.Items.Rows[0].Sprite == "" {
			t.Fatalf("%s: the published items %+v", game, res.Items)
		}
		if _, _, err := mod.ItemSprite.ReadIn(entries[0].Dir, res.Items.Rows[0].Sprite); err != nil {
			t.Fatalf("%s: the packed sprite: %v", game, err)
		}
		tab := fixtureItemTable(10, 5, 5)
		tab.Spells = nil
		res.Data.Missions = mod.MissionData{}
		if _, err := itemFront(tab).SetModData(res.Data); err != nil {
			t.Fatalf("%s: the launch's check refused the published mod: %v", game, err)
		}
		i := slices.IndexFunc(tab.Mods.Items, func(m mapload.ModItem) bool { return m.Key == "armour" })
		if i < 0 {
			t.Fatalf("%s: no mod item %+v", game, tab.Mods.Items)
		}
		item := tab.Mods.Items[i]
		if game == "rom1-en" {
			shop := NewShop(5000)
			shop.Generate(tab, 7)
			if !slices.ContainsFunc(shop.Shelf(ShelfArmour), func(it ShopItem) bool { return it.Code == data.ItemCode(item.Code) && it.Price == 55 }) {
				t.Fatalf("the first game's shop %+v", shop.Shelf(ShelfArmour))
			}
			continue
		}
		tables, _ := secondShopTables(tab)
		if !slices.ContainsFunc(tables.Stocked, func(it secondshop.Item) bool { return it.Price == 55 }) {
			t.Fatalf("the second game's stock %+v", tables.Stocked)
		}
	}
	if b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(mod.ItemsFile))); err != nil || !strings.Contains(string(b), "shop = true") {
		t.Fatalf("the saved items file %q, %v", b, err)
	}
}

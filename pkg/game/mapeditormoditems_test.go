package game

import (
	"image/color"
	"os"
	"path/filepath"
	"slices"
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

// starPicture is the host path of a sprite named for the weapon "star".
func starPicture(t *testing.T) string {
	t.Helper()
	source, _ := assetFile(t, "star.png", assetPNG(t, 24, 24, color.RGBA{220, 220, 60, 255}))
	return source
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

// The page adds an armour and a weapon row, changes and switches off a
// shipped row by its key, sets a material mask and switches its own row off;
// each edit is one draft step the context reads, undone and redone, and an
// undo past a save takes the saved file out again.
func TestItemsPageAddsChangesAndSwitchesOffRows(t *testing.T) {
	x, s, dir := itemsPage(t)
	made := tree(t, dir)
	steps := len(s.undoWorkspace)
	setItems(t, s, itemsNewArmour, "bone_mail")
	setItems(t, s, "item.bone_mail.price", "55")
	setItems(t, s, "item.bone_mail.shop", "2")
	setItems(t, s, itemsNewWeapon, starPicture(t))
	setItems(t, s, "weapon.star.price", "300")
	setItems(t, s, itemsNewChange, "Cap")
	setItems(t, s, "change.0.defence", "12")
	setItems(t, s, "change.0.off", "")
	setItems(t, s, itemsNewMaterial, "Soft Mail, Common, Iron")
	pair, _ := assetFile(t, "pair.png", assetPNG(t, 24, 24, color.RGBA{90, 90, 90, 255}))
	setItems(t, s, "material.0.sprite", pair)
	if len(s.undoWorkspace) != steps+10 {
		t.Fatalf("steps %d, want 10", len(s.undoWorkspace)-steps)
	}
	if m, ok := modItemOf(x, "bone_mail"); !ok || m.Price != 55 || !m.Stock {
		t.Fatalf("the context's armour %+v, %v", m, ok)
	}
	if _, ok := modItemOf(x, "star"); !ok {
		t.Fatal("the context has no mod weapon")
	}
	tab := x.mod.context.table
	if data.ItemMasks(tab.Armors.(data.MaskTable), 2) != ([data.ShopShapes]uint16{}) || tab.Armors.EntryParams(2)[9] == 12 {
		t.Fatalf("the shipped row switched off keeps a pair or takes the change: %v", tab.Armors.EntryParams(2))
	}
	src := string(s.workspace.Project.Files[mod.ItemsFile])
	for _, want := range []string{"[[item]]\nkey = \"bone_mail\"", "price = 55", "[[weapon]]\nkey = \"star\"", "price = 300",
		"[[change]]\nitem = \"Cap\"", "defence = 12", "off = true", "[[material]]\nitem = \"Soft Mail\"", "allow = true", "sprite = \"assets/items/pair.png\""} {
		if !strings.Contains(src, want) {
			t.Fatalf("the items file lacks %q:\n%s", want, src)
		}
	}
	setItems(t, s, "item.bone_mail.off", "")
	if _, ok := itemsField(s, "item.bone_mail.state"); !ok {
		t.Fatal("the switched-off row shows no state")
	}
	if _, ok := modItemOf(x, "bone_mail"); ok {
		t.Fatal("the switched-off armour keeps a code")
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if _, ok := modItemOf(x, "bone_mail"); !ok {
		t.Fatal("the undo of the switch left the armour off")
	}
	if _, err := s.Redo(); err != nil {
		t.Fatal(err)
	}
	if _, ok := modItemOf(x, "bone_mail"); ok {
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
	if _, ok := modItemOf(x, "star"); ok {
		t.Fatal("the undo of every step keeps the weapon")
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if after := tree(t, dir); !slices.Equal(keysOf(made), keysOf(after)) {
		t.Fatalf("the save after the undo past a save holds %v, want %v", keysOf(after), keysOf(made))
	}
}

// A value the items reader refuses is refused in its words and makes no
// step; a change of a row no install table holds is refused by the layer.
func TestItemsPageRefusesAFieldByTheReadersRule(t *testing.T) {
	_, s, _ := itemsPage(t)
	steps := len(s.undoWorkspace)
	_, err := s.SetWorkspaceField(ui.ModItemsScope, itemsNewArmour, "Bone Mail")
	if err == nil || !strings.Contains(err.Error(), "must be 1 to 32 of a-z") {
		t.Fatalf("a key with a space: %v", err)
	}
	if _, err := s.SetWorkspaceField(ui.ModItemsScope, itemsNewChange, "No Such Row"); err == nil || !strings.Contains(err.Error(), "No Such Row") {
		t.Fatalf("a change of no row: %v", err)
	}
	if len(s.undoWorkspace) != steps {
		t.Fatalf("a refused field made %d steps", len(s.undoWorkspace)-steps)
	}
}

// A picture the item layer's rule takes is imported under assets/items as
// the row's sprite in one step; a picture over the rule's side is refused in
// its words.
func TestItemsPageImportsASpriteByTheLayersRule(t *testing.T) {
	_, s, _ := itemsPage(t)
	setItems(t, s, itemsNewArmour, "bone_mail")
	steps := len(s.undoWorkspace)
	big, _ := assetFile(t, "big.png", assetPNG(t, maxModIconSide+1, 8, color.RGBA{200, 40, 40, 255}))
	if _, err := s.SetWorkspaceField(ui.ModItemsScope, "item.bone_mail.sprite", big); err == nil || !strings.Contains(err.Error(), "pixels on each side") {
		t.Fatalf("an oversized picture: %v", err)
	}
	source, png := assetFile(t, "bone.png", assetPNG(t, 32, 32, color.RGBA{200, 40, 40, 255}))
	setItems(t, s, "item.bone_mail.sprite", source)
	rel := mod.AssetsFolder + "/items/bone.png"
	if string(s.workspace.Project.Files[rel]) != string(png) || len(s.undoWorkspace) != steps+1 {
		t.Fatalf("the sprite's draft: files %v, steps %d", keysOf(s.workspace.Project.Files), len(s.undoWorkspace)-steps)
	}
	if f, ok := itemsField(s, "item.bone_mail.sprite"); !ok || f.TextValue != rel || f.Meaning != rel || len(f.Pick) != 1 {
		t.Fatalf("the sprite field %+v", f)
	}
}

// The party of a mission offers a weapon the page adds before a save.
func TestItemsPageWeaponIsOfferedToTheParty(t *testing.T) {
	_, n, _ := itemsPage(t)
	code := int64(data.ComposeItemCode(0, 1, 0, 5))
	setItems(t, n, itemsNewWeapon, starPicture(t))
	if _, err := n.SetWorkspaceField("package", "landing", "1"); err != nil {
		t.Fatal(err)
	}
	for _, f := range n.WorkspaceFields("package") {
		if f.Key == "party.0.weapon" {
			if !slices.ContainsFunc(f.Choices, func(c ui.InspectionChoice) bool { return c.Value == code && strings.Contains(c.Label, "star") }) {
				t.Fatalf("weapon choices %+v", f.Choices)
			}
			return
		}
	}
	t.Fatal("no party weapon field")
}

// A mod made on the page alone, applying to both games, saves through the
// project writer with no entry-file edit, publishes, loads and passes the
// launch's data check; the first game's stock generator and the second
// game's shop both offer its armour.
func TestItemsPageModPublishesAndShopsOfferIt(t *testing.T) {
	_, s, dir := itemsPage(t)
	made := tree(t, dir)
	if _, err := s.SetWorkspaceField("package", "applies-to", "0"); err != nil {
		t.Fatal(err)
	}
	setItems(t, s, itemsNewArmour, "bone_mail")
	setItems(t, s, "item.bone_mail.price", "55")
	setItems(t, s, "item.bone_mail.shop", "2")
	source, _ := assetFile(t, "bone.png", assetPNG(t, 32, 32, color.RGBA{200, 40, 40, 255}))
	setItems(t, s, "item.bone_mail.sprite", source)
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
		i := slices.IndexFunc(tab.Mods.Items, func(m mapload.ModItem) bool { return m.Key == "bone_mail" })
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

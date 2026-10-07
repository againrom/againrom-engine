package game

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// slotModDir writes a mod whose one item is worn in the body slot: the armour
// of the heavy-armour example, without the layer key.
func slotModDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"plate-slot/mod.toml": "id = \"plate-slot\"\ntitle = \"Plate\"\nversion = \"1.0.0\"\nauthor = \"Againrom\"\n" +
			"description = \"A body armour worn in the body slot.\"\napi = 1\napplies-to = [\"rom1\"]\nentry = \"main.star\"\n",
		"plate-slot/main.star": "def init(game, settings):\n    game.data.add(\"data/items.toml\")\n",
		"plate-slot/data/items.toml": "[[item]]\nkey = \"plate\"\nname = \"item.plate\"\nslot = \"body\"\ndefence = 6\nabsorption = 1\n" +
			"weight = 6\nprice = 120\nstand-in = \"Soft Mail\"\nshop = true\n",
		"plate-slot/text/en/strings.toml": "\"item.plate\" = \"Plate\"\n",
	}
	for name, body := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestReleaseModSlotItemSurvivesAMissionSaveAndColdLoad: a mod item worn in the
// body slot keeps its stated defence, absorption and weight through a mission
// SAVE and a cold LOAD under the mod. The SAV's ordinary fields hold the
// stand-in, whose own numbers differ, so a reader that skipped the mod mark
// would see other numbers; a LOAD without the mod refuses.
func TestReleaseModSlotItemSurvivesAMissionSaveAndColdLoad(t *testing.T) {
	dir := slotModDir(t)
	mk := func() *FrontEnd { return modItemFrontOf(t, dir, "plate-slot") }
	f := modItemTownOf(t, mk)
	if len(f.Table.Mods.Items) != 1 || f.Table.Mods.Items[0].Layer != 0 {
		t.Fatalf("the mod items are %+v", f.Table.Mods.Items)
	}
	item := f.Table.Mods.Items[0]
	code := item.Code
	standIn, err := data.ArmorFromCode(data.ItemCode(item.StandInCode), f.Table.Shapes, f.Table.Materials, f.Table.Armors)
	if err != nil || standIn.Defence == 6 || standIn.Weight == 6 {
		t.Fatalf("the stand-in %+v (%v) states the mod item's numbers, so the loss control sees nothing", standIn, err)
	}
	currentTownBuy(t, currentTownShop(f, "hero"), roomArmour, code)

	g := mk()
	{
		dirSav := t.TempDir()
		if err := os.WriteFile(filepath.Join(dirSav, "town.sav"), currentTownSave(t, f), 0o644); err != nil {
			t.Fatal(err)
		}
		_, _, load := g.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dirSav}, nil)
		if _, town, err := load("town.sav"); err != nil || !town {
			t.Fatalf("town LOAD: town=%v %v", town, err)
		}
	}
	app := g.App("mod slot item")
	if err := app.OpenMission(g.MissionOpenerWith(30, g.NextParty())); err != nil {
		t.Fatalf("open mission 30: %v", err)
	}
	hero := equipmentReturnHero(t, g)
	before := releaseEntity(t, g.live, hero)
	worn, _ := g.live.world.EquippedItems(hero)
	var old data.Armor
	if c := worn[6].Code; c != 0 {
		a, err := data.ArmorFromCode(data.ItemCode(c), g.Table.Shapes, g.Table.Materials, g.Table.Armors)
		if err != nil {
			t.Fatal(err)
		}
		old = a
	}
	carried, _ := g.live.world.CarriedItems(hero)
	i := slices.IndexFunc(carried, func(it sim.ItemInstance) bool { return it.Code == code })
	if i < 0 {
		t.Fatalf("the pack holds the plate as %+v", carried)
	}
	g.live.enqueueEquip(i)
	g.live.tick()
	g.live.refreshEquipment()
	after := releaseEntity(t, g.live, hero)
	if got, want := after.Defence-before.Defence, 6-old.Defence; got != want {
		t.Fatalf("defence moved by %d, want %d", got, want)
	}
	if got, want := after.Absorption-before.Absorption, 1-old.Absorption; got != want {
		t.Fatalf("absorption moved by %d, want %d", got, want)
	}
	if after.Load == before.Load {
		t.Fatalf("the load stayed %d after the armour changed to a 6-weight plate", before.Load)
	}
	if worn, _ = g.live.world.EquippedItems(hero); worn[6].Code != code {
		t.Fatalf("the body slot holds %+v", worn[6])
	}

	snapshot, label, err := g.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := g.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	mark, present, err := readModMark(doc)
	if err != nil || !present || len(mark.Layers) != 0 {
		t.Fatalf("the mark: %+v %v %v", mark, present, err)
	}
	listed := 0
	for _, m := range mark.Items {
		if m.Code != code {
			continue
		}
		listed++
		obj := &doc.Objects[m.Object]
		if c, _ := savedStructureValue(obj, "F40"); uint16(c) != item.StandInCode {
			t.Fatalf("a reader that skips the mark sees %#x, want the stand-in %#x", c, item.StandInCode)
		}
		if row, _ := savedStructureValue(obj, "T0C"); uint8(row) != item.StandInRow {
			t.Fatalf("the object holds row %d, want the stand-in row %d", row, item.StandInRow)
		}
	}
	if listed == 0 {
		t.Fatalf("the mark lists no plate: %+v", mark.Items)
	}
	for i := range doc.Objects {
		if r := &doc.Objects[i]; savedItemClass(r.Class) {
			if c, err := savedStructureValue(r, "F40"); err == nil && uint16(c) == code {
				t.Fatalf("object %d still holds the mod code", i)
			}
		}
	}

	cold := mk()
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("mission LOAD: town=%v %v", town, err)
	}
	if err := cold.App("mod slot cold").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	var restored sim.Entity
	for _, e := range cold.live.world.Entities() {
		if w, _ := cold.live.world.EquippedItems(e.ID); w[6].Code == code {
			restored = e
		}
	}
	if restored.ID == 0 || restored.Defence != after.Defence || restored.Absorption != after.Absorption || restored.Load != after.Load {
		t.Fatalf("the cold LOAD restored entity %d defence %d absorption %d load %d, want %d %d %d",
			restored.ID, restored.Defence, restored.Absorption, restored.Load, after.Defence, after.Absorption, after.Load)
	}
	// Loss control: the stand-in the SAV holds would give the hero other numbers.
	if standIn.Defence == 6 || after.Defence-before.Defence == standIn.Defence-old.Defence {
		t.Fatal("the stand-in's numbers equal the mod item's, so the witness proves nothing")
	}
	if _, _, err := releaseFront(t).RestoreOriginal(raw); err == nil || !errors.Is(err, ErrModMark) {
		t.Fatalf("a mission LOAD without the mod: %v", err)
	}
}

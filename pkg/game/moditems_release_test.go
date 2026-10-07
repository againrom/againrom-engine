package game

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const modItemsDir = "../modrt/testdata/mods"

// modItemCode is the code of the example mod's gambeson: the iron material, the
// body slot, the common shape and the first free armour row.
var modItemCode = uint16(data.ComposeItemCode(0, 7, 0, 31))

// modItemFront is a front end on the lawful install running under the example
// mod, started through the same steps the launcher takes: the mod's script and
// data are read, the rules and the mod set are applied, then the items.
func modItemFront(t *testing.T) *FrontEnd {
	t.Helper()
	return modItemFrontOf(t, modItemsDir, "heavy-armor")
}

// modItemFrontOf is modItemFront for the mod id found in the folder modsDir.
func modItemFrontOf(t *testing.T, modsDir, id string) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	entries, err := mod.Resolve(modsDir, []string{id})
	if err != nil {
		t.Fatal(err)
	}
	res, err := modrt.Load(entries, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.SetMods(res.Rules, res.Set, false); err != nil {
		t.Fatal(err)
	}
	if err := f.SetModItems(res.Items); err != nil {
		t.Fatal(err)
	}
	return f
}

// modItemTown is the player's route to a town under the example mod: chargen,
// mission 10, a mission SAVE in mission 20 and its cold LOAD, then mission 20's
// finish.
func modItemTown(t *testing.T) *FrontEnd {
	t.Helper()
	return modItemTownOf(t, func() *FrontEnd { return modItemFront(t) })
}

// modItemTownOf is modItemTown over front ends the mk function starts.
func modItemTownOf(t *testing.T, mk func() *FrontEnd) *FrontEnd {
	t.Helper()
	f := mk()
	party := f.ChargenParty(ui.ChargenResult{Name: "Mod items", Choices: []int{1, 0, 0}, Stats: []int{31, 27, 24, 29}})
	app := f.App("mod items")
	if err := app.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatalf("open mission 10: %v", err)
	}
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatalf("finish mission 10: %v", err)
	}
	if err := app.OpenMission(f.MissionOpenerWith(20, f.NextParty())); err != nil {
		t.Fatalf("open mission 20: %v", err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("mission SAVE in mission 20 wrote %q: %v", name, err)
	}
	g := mk()
	_, _, load := agsSaveSeams(g, store, OriginalStore{}, nil)
	open, town, err := load(localOriginalSaveToken(name))
	if err != nil || town {
		t.Fatalf("LOAD of the mission SAVE: town=%v err=%v", town, err)
	}
	if err := g.App("mod items loaded").OpenMission(open); err != nil {
		t.Fatalf("reopen mission 20: %v", err)
	}
	if _, _, err := g.LiveCompleteCampaign(); err != nil {
		t.Fatalf("finish mission 20: %v", err)
	}
	return g
}

// modItemReload is a cold LOAD of a town SAV in a fresh front end that runs
// under the mod, or under no mod when plain is set.
func modItemReload(t *testing.T, raw []byte, plain bool) (*FrontEnd, error) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "town.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	g := releaseFront(t)
	if !plain {
		g = modItemFront(t)
	}
	_, _, load := g.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	_, town, err := load("town.sav")
	if err == nil && !town {
		t.Fatal("the town SAV did not open a town")
	}
	return g, err
}

func modItemPackHolds(f *FrontEnd, code uint16) bool {
	_, pack := currentTownMemberNoFail(f, "hero")
	return slices.Contains(pack, code)
}

func currentTownMemberNoFail(f *FrontEnd, id string) (worn [sim.EquipSlots]uint16, pack []uint16) {
	for _, p := range f.Carried {
		if p.ID == id {
			return memberItemCodes(p)
		}
	}
	return worn, pack
}

func modItemObjects(t *testing.T, raw []byte) (holding, standIn int) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	items := modItemFrontItems(t)
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if !savedItemClass(r.Class) {
			continue
		}
		code, err := savedStructureValue(r, "F40")
		if err != nil {
			continue
		}
		if uint16(code) == modItemCode {
			holding++
		}
		if uint16(code) == items.StandInCode {
			standIn++
		}
	}
	return holding, standIn
}

func modItemFrontItems(t *testing.T) (item struct{ StandInCode uint16 }) {
	t.Helper()
	f := modItemFront(t)
	items := f.Table.Mods.Items
	if len(items) != 3 || items[0].Code != modItemCode || items[0].Key != "gambeson" {
		t.Fatalf("the mod items are %+v", items)
	}
	item.StandInCode = items[0].StandInCode
	return item
}

func TestReleaseModItemShopEquipSaveAndLoad(t *testing.T) {
	f := modItemTown(t)
	item := f.Table.Mods.Items[0]
	wantName := map[string]string{"rom1-en": "Gambeson", "rom1-ru": "Поддоспешник"}[f.ModSet().Base]
	if wantName == "" {
		t.Fatalf("base %q", f.ModSet().Base)
	}
	wantName, err := encodeSaveLabel(wantName, f.textSelector())
	if err != nil {
		t.Fatal(err)
	}
	if got := itemName(data.ItemCode(modItemCode), f.Table); got != wantName {
		t.Fatalf("the item is named %q, want %q", got, wantName)
	}

	// The item is on the armour shelf at the price the mod set, once.
	s := currentTownShop(f, "hero")
	s.chooseRoomShelf(roomArmour)
	shelf := f.Shop.Shelf(s.shopShelf)
	count := 0
	for _, it := range shelf {
		if uint16(it.Code) == modItemCode {
			count++
			if it.Price != 120 || it.Kind != 1 {
				t.Fatalf("the shelf lists %+v", it)
			}
		}
	}
	if count != 1 {
		t.Fatalf("the armour shelf lists the gambeson %d times", count)
	}
	for _, other := range []ShopShelf{ShelfWeapons, ShelfMagic, ShelfBooks} {
		for _, it := range f.Shop.Shelf(other) {
			if uint16(it.Code) == modItemCode {
				t.Fatalf("the %v shelf lists the gambeson", other)
			}
		}
	}

	// Its picture is the mod's: the icon decodes to the PNG's 80x80 and the worn
	// figure layers are those of the original item the mod names as its figure.
	icon, err := loadItemIcon(f.Archives.Containers, graphicsPrefix+data.ItemIconPath(data.ItemCode(modItemCode)))
	if err != nil || icon.Bounds().Dx() != 80 || icon.Bounds().Dy() != 80 {
		t.Fatalf("icon %v", err)
	}
	if px := icon.RGBAAt(40, 40); px.A != 255 || px.R < 120 {
		t.Fatalf("the icon's centre %+v is not the mod's picture", px)
	}
	plain := releaseFront(t)
	soft, err := resolveFigure(mod.ItemRow{Mod: "heavy-armor", Figure: "Soft Mail"},
		&itemClass{name: "armour", base: plain.Table.Armors}, plain.Table)
	if err != nil {
		t.Fatal(err)
	}
	read := func(dir data.FigureDir, code data.ItemCode) ([]byte, error) {
		return f.Archives.Containers.ReadFile(graphicsPrefix + data.ItemFigureLayerPath(dir, code))
	}
	// Fighters have the sheet; the mage directories have none of their own and
	// borrow the fighter's of the same sex.
	for _, c := range []struct{ dir, from data.FigureDir }{
		{data.FigureDirManFighter, data.FigureDirManFighter},
		{data.FigureDirWomanFighter, data.FigureDirWomanFighter},
		{data.FigureDirManMage, data.FigureDirManFighter},
		{data.FigureDirWomanMage, data.FigureDirWomanFighter},
	} {
		got, gerr := read(c.dir, data.ItemCode(modItemCode))
		want, werr := read(c.from, soft)
		if gerr != nil || werr != nil || len(want) == 0 || !bytes.Equal(got, want) {
			t.Fatalf("figure layer for %s: %v %v", c.dir, gerr, werr)
		}
	}

	// Buy it.
	gold := f.Town.Gold()
	currentTownBuy(t, s, roomArmour, modItemCode)
	if f.Town.Gold() != gold-120 || !modItemPackHolds(f, modItemCode) {
		t.Fatalf("gold %d -> %d, pack holds it %v", gold, f.Town.Gold(), modItemPackHolds(f, modItemCode))
	}

	// SAVE in the town: the ordinary fields hold the stand-in, the mark the item.
	raw := currentTownSave(t, f)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	mark, present, err := readModMark(doc)
	if err != nil || !present || len(mark.Items) != 1 || mark.Items[0].Code != modItemCode {
		t.Fatalf("the save's mark: %+v %v %v", mark, present, err)
	}
	obj := &doc.Objects[mark.Items[0].Object]
	if code, _ := savedStructureValue(obj, "F40"); uint16(code) != item.StandInCode {
		t.Fatalf("the item object holds %#x, want the stand-in %#x", code, item.StandInCode)
	}
	if row, _ := savedStructureValue(obj, "T0C"); uint8(row) != item.StandInRow || obj.Class != "Armor" {
		t.Fatalf("the item object holds row %d class %s", row, obj.Class)
	}
	if holding, _ := modItemObjects(t, raw); holding != 0 {
		t.Fatalf("%d item objects still hold the mod item's code", holding)
	}

	// A cold LOAD under the same mod restores the item, the purse and the
	// name; the next action, wearing it, puts it on as a layer.
	g, err := modItemReload(t, raw, false)
	if err != nil {
		t.Fatal(err)
	}
	if !modItemPackHolds(g, modItemCode) || g.Town.Gold() != gold-120 {
		t.Fatalf("after the LOAD: pack %v gold %d", modItemPackHolds(g, modItemCode), g.Town.Gold())
	}
	gs := currentTownShop(g, "hero")
	wornBefore, _ := currentTownMember(t, g, "hero")
	gs.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: currentTownPackCell(t, gs, modItemCode)}, ui.ShopControl{Kind: ui.ShopControlDoll})
	if worn, _ := currentTownMember(t, g, "hero"); worn != wornBefore || !slices.Equal(g.Carried[0].Layers, []uint16{modItemCode}) {
		t.Fatalf("after wearing the gambeson the slots are %v (were %v) and the layers %v", worn, wornBefore, g.Carried[0].Layers)
	}
	raw2 := currentTownSave(t, g)
	h, err := modItemReload(t, raw2, false)
	if err != nil {
		t.Fatal(err)
	}
	if worn, _ := currentTownMember(t, h, "hero"); worn != wornBefore || !slices.Equal(h.Carried[0].Layers, []uint16{modItemCode}) {
		t.Fatalf("after the second LOAD the slots are %v and the layers %v", worn, h.Carried[0].Layers)
	}

	// A LOAD without the mod refuses and names it; so does a LOAD under another
	// mod set.
	if _, err := modItemReload(t, raw2, true); err == nil || !errors.Is(err, ErrModMark) || !strings.Contains(err.Error(), "heavy-armor") {
		t.Fatalf("a load without the mod: %v", err)
	}
}

func TestReleaseModItemChangeAppliesToTheShippedRow(t *testing.T) {
	f := modItemFront(t)
	plain := releaseFront(t)
	// Chain Mail is armour row 16 in the shipped table; the mod sets its weight
	// column to 20, so the common iron item weighs 20.
	code := data.ComposeItemCode(0, 7, 0, 16)
	got, err := data.ArmorFromCode(code, f.Table.Shapes, f.Table.Materials, f.Table.Armors)
	if err != nil || got.Weight != 20 {
		t.Fatalf("modded %+v %v", got, err)
	}
	shipped, err := data.ArmorFromCode(code, plain.Table.Shapes, plain.Table.Materials, plain.Table.Armors)
	if err != nil || shipped.Weight != 40 {
		t.Fatalf("shipped %+v %v", shipped, err)
	}
	if got.Defence != shipped.Defence || got.Absorption != shipped.Absorption {
		t.Fatalf("the change touched other numbers: %+v vs %+v", got, shipped)
	}
	if plain.Table.Armors.Len() != 31 || f.Table.Armors.Len() != 32 {
		t.Fatalf("lengths %d %d", plain.Table.Armors.Len(), f.Table.Armors.Len())
	}
}

func TestReleaseUnmoddedTownSaveHasNoModLeaf(t *testing.T) {
	f := currentTown(t, nil, nil)
	if raw := currentTownSave(t, f); bytes.Contains(raw, modMarkName) {
		t.Fatal("an unmodded town save carries the mod leaf")
	}
}

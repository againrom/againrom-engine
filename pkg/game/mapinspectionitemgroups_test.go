package game

import (
	"bytes"
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func TestEditorSackPaletteKeepsOneBlockPerItemKind(t *testing.T) {
	s, _ := viewsMap(t)
	e := viewsWindow(t, s, false, ui.KindSack)
	x := s.inspector
	x.archives.Base.Profile.Language = "russian"
	x.table.Armors = dbCollection{{}, {name: "Plate Boots"}, {name: "Chain Boots"}, {name: "Soft Boots"}}
	x.table.Materials = fixtureScale{names: []string{"Iron", "Meteoric"}}
	x.table.Names = data.ItemNames{0x901: "iron plate boots", 0x902: "chain boots", 0x903: "soft boots", 0x1901: "meteoric plate boots"}
	x.recordOptions = nil
	e.Layout(1920, 1080)
	dirty, raw := e.Dirty(), bytes.Clone(e.Document().SourceBytes)
	if err := e.Key("sack"); err != nil {
		t.Fatal(err)
	}
	var cells [4]image.Rectangle
	for i := range cells {
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPlaceGridCell, N: i})
		if !ok {
			t.Fatalf("catalog cell %d is missing from Sacks", i)
		}
		cells[i] = r
	}
	if cells[0].Min.Y != cells[3].Min.Y || cells[0].Min.X >= cells[3].Min.X {
		t.Fatalf("plate boots do not share one ordered block: iron at %v, meteoric at %v", cells[0], cells[3])
	}
	if cells[1].Min.Y <= cells[3].Min.Y || cells[2].Min.Y <= cells[1].Min.Y {
		t.Fatalf("chain and soft boots need separate blocks in catalog order: %v", cells)
	}
	if e.Dirty() != dirty || !bytes.Equal(raw, e.Document().SourceBytes) {
		t.Fatal("opening the palette edited the map")
	}
}

func TestEditorItemGroupsPartitionCodesByKind(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		s, _ := viewsMap(t)
		x := s.inspector
		x.archives.Base.Profile.Language = lang
		x.table.Materials = fixtureScale{names: []string{"Iron", "Meteoric"}}
		x.table.Weapons = dbCollection{{}, {name: "Two Handed Sword"}, {name: "Shaman Staff"}, {name: "custom sword"}, {name: "custom sword"}}
		x.table.Names = data.ItemNames{
			0x101: "sword", 0x121: "rare sword", 0x1101: "meteorite sword", 0x102: "staff",
			0x103: "first custom sword", 0x1103: "meteorite custom sword", 0x104: "second custom sword", 0xe08: "potion",
		}
		x.recordOptions = nil
		offer := s.fieldChoiceOffer(fieldChoiceAsk{kind: ui.KindSack, index: groundSack(t, s), key: "item1"})
		if offer.Choices[0].Value != 0 || offer.Choices[1].Value != stockDuplicate {
			t.Fatal("Remove and Duplicate are no longer first", offer.Choices)
		}
		want := [][]int64{{0x101, 0x121, 0x1101}, {0x102}, {0x103, 0x1103}, {0x104}, {0xe08}}
		if len(offer.Groups) != len(want) {
			t.Fatalf("groups=%+v", offer.Groups)
		}
		seen := map[int64]int{}
		for i, g := range offer.Groups {
			if g.Head != "" || !slices.Equal(g.Values, want[i]) {
				t.Fatalf("group %d=%+v want=%v", i, g, want[i])
			}
			for _, code := range g.Values {
				seen[code]++
			}
		}
		for _, c := range offer.Choices[2:] {
			if seen[c.Value] != 1 {
				t.Fatalf("code %d grouped %d times", c.Value, seen[c.Value])
			}
		}
		if offer.Groups[0].Block != x.itemGroupWord("kind", "Two Handed Sword", "twohandedsword") {
			t.Fatal("kind caption did not follow the table")
		}
		if lang == "russian" && offer.Groups[0].Block != x.inspectionUTF8("Двуручные мечи") {
			t.Fatal("Russian kind caption is missing")
		}
	}
}

func TestEditorItemChoicePicturesUseInventoryCache(t *testing.T) {
	s, _ := viewsMap(t)
	x := s.inspector
	reads := map[data.ItemCode]int{}
	for i, code := range []data.ItemCode{0x101, 0xe08} {
		x.archives.Containers.Overlay(graphicsPrefix+data.ItemIconPath(code), func() ([]byte, error) {
			reads[code]++
			return packIconStream(byte(20+i), 30, 40), nil
		})
	}
	x.recordPictures = nil
	offer := s.fieldChoiceOffer(fieldChoiceAsk{kind: ui.KindSack, index: groundSack(t, s), key: "item1"})
	if len(reads) != 0 {
		t.Fatal("opening the catalogue loaded pictures before a group opened")
	}
	for _, code := range []data.ItemCode{0x101, 0xe08, 0x303} {
		choices := []ui.InspectionChoice{{Value: int64(code)}}
		pic := offer.Pictures(choices)[int64(code)]
		want, _ := itemIcon(x.archives.artSource(), x.table, uint16(code), map[uint16]*image.RGBA{})
		if (pic == nil) != (want == nil) || pic != nil && (!bytes.Equal(pic.Pix, want.Pix) || pic.Rect != want.Rect) {
			t.Fatalf("%x: choice picture differs from itemIcon", code)
		}
		before := reads[code]
		if again := offer.Pictures(choices)[int64(code)]; again != pic || reads[code] != before {
			t.Fatalf("%x: reopening the group reread the picture", code)
		}
		_, same := s.ItemLook(int64(code))
		if same != pic || reads[code] != before {
			t.Fatal("the item row and choice keep separate picture caches")
		}
	}
}

func TestEditorGroupedItemPicksWriteTheFlatChoicesBytes(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		for _, plain := range []bool{false, true} {
			for _, key := range []string{"item1", "additem", "carryadd", "carry1", "ground.item1", "ground.additem"} {
				if plain && strings.HasPrefix(key, "carry") {
					continue
				}
				t.Run(fmt.Sprintf("%s/%v/%s", lang, plain, key), func(t *testing.T) {
					s, stock, _ := sackRecipeMap(t, lang, false)
					twin, _, _ := sackRecipeMap(t, lang, false)
					kind, index := ui.KindStock, stock
					if strings.HasPrefix(key, "ground.") {
						kind, index = ui.KindSack, groundSack(t, s)
						key = strings.TrimPrefix(key, "ground.")
					}
					if strings.HasPrefix(key, "carry") {
						kind, index = ui.KindUnit, 0
					}
					var e *ui.MapEditor
					c := &editorClicker{t: t}
					if kind == ui.KindStock {
						e, c = sackRecipeWindow(t, s, lang, plain)
					} else if kind == ui.KindSack {
						e = viewsWindow(t, s, plain, ui.KindSack)
						e.Layout(1920, 1080)
						if plain {
							openBlocks(t, e, c)
						}
					} else {
						e = viewsWindow(t, s, plain, ui.KindUnit)
						e.Layout(1920, 1080)
						r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlObjectSection, Key: "inventory"})
						if ok {
							c.click(e, r, true, "Inventory")
						}
					}
					chooseField(t, e, c, key, 0xe08)
					if _, err := twin.SetFields(kind, index, []ui.InspectionFieldValue{{Key: key, Value: 0xe08, Step: 1}}); err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(s.model.Bytes(), twin.model.Bytes()) {
						t.Fatal("grouped pick wrote other bytes than the existing flat-choice writer")
					}
				})
			}
		}
	}
}

func TestEditorInstalledItemGroups(t *testing.T) {
	x := installedInspector(t)
	choices := x.recordCatalog("item")
	seen := map[int64]int{}
	decode := func(value string) string { return x.messageUTF8([]byte(value)) }
	for _, g := range x.itemGroups {
		for _, code := range g.Values {
			seen[code]++
		}
		t.Logf("GROUP\t%s\t%s\t%d", decode(g.Block), decode(g.Head), len(g.Values))
	}
	for _, c := range choices {
		if seen[c.Value] != 1 {
			t.Fatalf("code %d grouped %d times", c.Value, seen[c.Value])
		}
	}
	if len(choices) != len(seen) || len(choices) != len(mapload.ShippedItemCodes(x.table)) {
		t.Fatal("the group population differs from the installed item-name table")
	}
	t.Logf("TOTAL codes=%d groups=%d", len(choices), len(x.itemGroups))
}

func TestEditorInstalledItemChoiceFirstOpenCost(t *testing.T) {
	x := installedInspector(t)
	largest := MapEntry{}
	for _, entry := range x.Maps {
		if entry.Err == nil && entry.Width*entry.Height > largest.Width*largest.Height {
			largest = entry
		}
	}
	if largest.Width == 0 {
		t.Fatal("no readable installed map")
	}
	key := "loose/" + largest.Source
	if largest.FromArchive {
		key = "scenario/" + largest.Source
	}
	e := x.Editor()
	e.SetPanelProfile([]byte(`{"right":{"minimized":true,"view":"technical","width":520}}`), nil, nil)
	e.Layout(1920, 1080)
	if err := e.Open(key); err != nil {
		t.Fatal(err)
	}
	if err := e.Key("sack"); err != nil {
		t.Fatal(err)
	}
	if e.PlaceGridLen() == 0 || e.Dirty() {
		t.Fatal("Sacks did not open its item grid without editing the map")
	}
	t.Logf("COST map=%s grid_cells=%d", key, e.PlaceGridLen())
}

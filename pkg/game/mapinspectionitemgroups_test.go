package game

import (
	"bytes"
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func TestEditorItemGroupsPartitionCodesByTableRowAndMaterial(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		s, _ := viewsMap(t)
		x := s.inspector
		x.archives.Base.Profile.Language = lang
		x.table.Materials = fixtureScale{names: []string{"Iron", "Meteoric"}}
		x.table.Weapons = dbCollection{{}, {name: "Two Handed Sword"}, {name: "Shaman Staff"}}
		x.table.Names = data.ItemNames{0x101: "sword", 0x121: "rare sword", 0x1101: "meteorite sword", 0x102: "staff", 0xe08: "potion"}
		x.recordOptions = nil
		offer := s.fieldChoiceOffer(fieldChoiceAsk{kind: ui.KindSack, index: groundSack(t, s), key: "item1"})
		if offer.Choices[0].Value != 0 || offer.Choices[1].Value != stockDuplicate {
			t.Fatal("Remove and Duplicate are no longer first", offer.Choices)
		}
		want := [][]int64{{0x101, 0x121}, {0x102}, {0xe08}, {0x1101}}
		if len(offer.Groups) != len(want) {
			t.Fatalf("groups=%+v", offer.Groups)
		}
		seen := map[int64]int{}
		for i, g := range offer.Groups {
			if !slices.Equal(g.Values, want[i]) {
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
		if offer.Groups[0].Block != x.itemGroupWord("kind", "Two Handed Sword", "twohandedsword") || offer.Groups[3].Head != x.itemGroupWord("material", "Meteoric", "meteoric") {
			t.Fatal("kind or material caption did not follow the table")
		}
		if lang == "russian" && (offer.Groups[0].Block != x.inspectionUTF8("Двуручные мечи") || offer.Groups[3].Head != x.inspectionUTF8("Метеорит")) {
			t.Fatal("Russian category captions are missing")
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
	_ = e.Key("clear")
	e.TextInput("32,32")
	start := time.Now()
	_ = e.Key("enter")
	if step, note, ok := e.AddStep(); !ok || step != "item" {
		t.Fatalf("item step did not open: %s %s %v", step, note, ok)
	}
	rootCost := time.Since(start)
	groups := x.itemGroups
	if len(groups) == 0 {
		t.Fatal("no item groups")
	}
	largestGroup := slices.MaxFunc(groups, func(a, b ui.InspectionChoiceGroup) int { return len(a.Values) - len(b.Values) })
	start = time.Now()
	var blocks, materials []string
	for _, group := range groups {
		if !slices.Contains(blocks, group.Block) {
			blocks = append(blocks, group.Block)
		}
		if group.Block == largestGroup.Block && !slices.Contains(materials, group.Head) {
			materials = append(materials, group.Head)
		}
	}
	for _, row := range []int{slices.Index(blocks, largestGroup.Block), 1 + slices.Index(materials, largestGroup.Head)} {
		for range row {
			_ = e.Key("down")
		}
		_ = e.Key("enter")
	}
	groupCost := time.Since(start)
	if len(x.recordPictures["item"]) != len(largestGroup.Values) || e.Dirty() {
		t.Fatal("the group did not resolve exactly its own pictures, or navigation wrote the map")
	}
	t.Logf("COST map=%s dimensions=%dx%d maps=%d codes=%d groups=%d first_open=%s largest_group_open=%s group=%s/%s items=%d cached=%d", key, largest.Width, largest.Height, len(x.Maps), len(x.recordCatalog("item")), len(groups), rootCost, groupCost, largestGroup.Block, largestGroup.Head, len(largestGroup.Values), len(x.recordPictures["item"]))
}

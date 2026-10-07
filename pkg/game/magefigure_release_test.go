package game

import (
	"image"
	"image/color"
	"math"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The Mage's Gloves and the Silver Ring, which the Russian install names a
// bracelet, on the installed definition rows that EN and RU share. Their
// sheets overlap on the hands of a female mage.
const (
	mageWitnessGloves   = uint16(0xfa17)
	mageWitnessBracelet = uint16(0x3421)
)

// mageWitnessOrder is the mage half of the original's colour pass after the
// body (HERO-FIGURE-059), in paint order. An entry names an equipment slot
// (HERO-FIGURE-058) and whether it paints that slot's second sheet. Slot 8's
// first sheet stands behind the body and slot 9 is never blitted for a mage,
// so neither is listed.
var mageWitnessOrder = []struct {
	slot   int
	second bool
}{
	{12, false}, {10, false}, {10, true}, {4, false}, {4, true},
	{7, false}, {5, false}, {1, false}, {6, false}, {8, true},
}

type mageWitnessLayer struct {
	slot int
	pic  *image.RGBA
}

// mageWitnessLayers reads the installed sheet of every worn slot in
// mageWitnessOrder. It shares only the sheet decoder with the compositor.
func mageWitnessLayers(src entrySource, dir data.FigureDir, worn [sim.EquipSlots]uint16) []mageWitnessLayer {
	sheets := sheetCache{src: src, decoded: map[string]*spr256.Sprite{}, converted: map[string][]*terrain.StaticFrame{}}
	var out []mageWitnessLayer
	for _, step := range mageWitnessOrder {
		code := data.ItemCode(worn[step.slot-1])
		if code == 0 {
			continue
		}
		path := data.ItemFigureLayerPath(dir, code)
		if step.second {
			path = data.ItemFigureSecondaryLayerPath(dir, code)
		}
		if pic := figureRGBA(&sheets, path); pic != nil {
			out = append(out, mageWitnessLayer{step.slot, pic})
		}
	}
	return out
}

// mageWitnessTop is the slot and colour of the last layer that paints p.
func mageWitnessTop(layers []mageWitnessLayer, p image.Point) (int, color.RGBA) {
	for i := len(layers) - 1; i >= 0; i-- {
		if c := layers[i].pic.RGBAAt(p.X, p.Y); c.A != 0 {
			return layers[i].slot, c
		}
	}
	return 0, color.RGBA{}
}

// mageWitnessOverlap is every pixel that a sheet of the gloves and a sheet of
// the bracelet both paint.
func mageWitnessOverlap(layers []mageWitnessLayer) []image.Point {
	var out []image.Point
	for y := 0; y < 240; y++ {
		for x := 0; x < 160; x++ {
			var gloves, bracelet bool
			for _, l := range layers {
				if l.pic.RGBAAt(x, y).A != 0 {
					gloves, bracelet = gloves || l.slot == 10, bracelet || l.slot == 4
				}
			}
			if gloves && bracelet {
				out = append(out, image.Pt(x, y))
			}
		}
	}
	return out
}

// mageWitnessCheck compares pic and mask with the last layer at every overlap
// pixel. origin is where the figure's corner stands in pic and skip lists the
// rectangles of pic that controls paint over. It answers how many pixels each
// slot owns and the rectangle of the pixels it left out.
func mageWitnessCheck(t *testing.T, name string, pic *image.RGBA, origin image.Point, mask *ui.SlotMask,
	skip []image.Rectangle, layers []mageWitnessLayer, overlap []image.Point) (map[int]int, image.Rectangle) {
	t.Helper()
	owned, bad := map[int]int{}, 0
	var left image.Rectangle
	for _, p := range overlap {
		at := p.Add(origin)
		if slices.ContainsFunc(skip, func(r image.Rectangle) bool { return at.In(r) }) {
			left = left.Union(image.Rectangle{Min: p, Max: p.Add(image.Pt(1, 1))})
			continue
		}
		slot, want := mageWitnessTop(layers, p)
		owned[slot]++
		got := pic.RGBAAt(at.X, at.Y)
		n, marked := mask.At(p.X, p.Y)
		if got != want || !marked || n != slot {
			if bad == 0 {
				t.Errorf("%s at figure pixel %v: colour %+v and hit map slot %d (%v), want %+v and slot %d", name, p, got, n, marked, want, slot)
			}
			bad++
		}
	}
	if bad > 0 {
		t.Errorf("%s: %d of %d overlap pixels differ from the original's paint order", name, bad, len(overlap))
	}
	return owned, left
}

// TestReleaseTownMageGlovesAreDrawnUnderTheBraceletsOnReniesta dresses the
// companion npc:22 in the Mage's Gloves and the Silver Ring by dragging both
// off the merchant's armour shelf onto her doll. The original's mage colour
// pass paints the ring after the gloves (HERO-FIGURE-059, DIV-1541), so the
// ring is on top wherever the two sheets overlap. The oracle reads the installed
// sheets and that pass and never calls the compositor. It is compared pixel for
// pixel with the shop's frame and with the map figure, and slot for slot with
// their hit maps. The merchant's stock and purse are the fixture; the wearing
// is the player's drag.
func TestReleaseTownMageGlovesAreDrawnUnderTheBraceletsOnReniesta(t *testing.T) {
	f, app, _, shop := recoveredCapShop(t, currentTownSave(t, currentTown(t, nil, nil)))
	app.Layout(640, 480)
	recoveredCapShow(t, app, shop, "npc:22")
	reniesta := func() mapload.PartyMember {
		for _, p := range f.Carried {
			if p.ID == "npc:22" {
				return p
			}
		}
		t.Fatal("the town has no member npc:22")
		return mapload.PartyMember{}
	}
	dir, _ := memberFigure(reniesta())
	if !dir.Mage() {
		t.Fatalf("npc:22 is drawn from %q, want a mage figure", dir)
	}
	for code, slot := range map[uint16]int{mageWitnessGloves: 10, mageWitnessBracelet: 4} {
		if got, ok := EquipTarget(data.ItemCode(code), f.Table); !ok || got != slot {
			t.Fatalf("the installed rows put %#x in slot %d (%v), want %d", code, got, ok, slot)
		}
	}
	pool := shopArmourPool(f.Table, math.MaxInt32)
	var stock []ShopItem
	for _, code := range []uint16{mageWitnessGloves, mageWitnessBracelet} {
		i := slices.IndexFunc(pool, func(c data.ShopCandidate) bool { return uint16(c.Code) == code })
		if i < 0 {
			t.Fatalf("the installed armour pool has no %#x", code)
		}
		stock = append(stock, shopItemFromInstance(sim.ItemInstance{Code: code, Kind: pool[i].ItemKind, Price: pool[i].Price}, 1))
		t.Logf("%#x is %q", code, decodeInstallText(itemName(data.ItemCode(code), f.Table)))
	}
	f.Shop.shelves[ShelfArmour] = stock
	f.Town.gold = 100000
	cityPotionPointer(t, app, "shelf_pick", 0, "press", "release")
	for _, want := range []struct {
		slot int
		code uint16
	}{{10, mageWitnessGloves}, {4, mageWitnessBracelet}} {
		at := releaseShopBuyToPack(t, app, f, want.code)
		if err := app.HeadlessPointer("press", at.X, at.Y); err != nil {
			t.Fatal(err)
		}
		cityPotionPointer(t, app, "doll_box", 0, "move", "release")
		if worn, _ := currentTownMember(t, f, "npc:22"); worn[want.slot-1] != want.code {
			t.Fatalf("slot %d holds %#x after the drag, want %#x; message %q", want.slot, worn[want.slot-1], want.code, app.HeadlessMessage())
		}
	}
	x, y, err := app.HeadlessShopIdlePoint()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessPointer("hover", x, y); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30; i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	pic, _, err := app.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	if pic.Bounds() != image.Rect(0, 0, 640, 480) {
		t.Fatalf("the shop frame is %v, want 640x480", pic.Bounds())
	}
	worn, _ := currentTownMember(t, f, "npc:22")
	layers := mageWitnessLayers(f.Archives.Containers, dir, worn)
	overlap := mageWitnessOverlap(layers)
	if len(overlap) == 0 {
		t.Fatal("the sheets of the gloves and the bracelet share no pixel")
	}
	origin := ui.TownCharacterRegion.Min.Add(image.Pt(0, 2))
	shown, left := mageWitnessCheck(t, "shop frame", pic, origin, shop.ShopScreen().SlotMask, ui.TownCharacterPersistentControls(), layers, overlap)
	member := reniesta()
	unit, unitMask := composeUnitFigure(f.Archives.Containers, mapload.EquipmentFromParty(member), memberFigureID(member))
	if unit == nil {
		t.Fatal("the map figure composed nothing")
	}
	unitOwned, _ := mageWitnessCheck(t, "map figure", unit, image.Point{}, unitMask, nil, layers, overlap)
	if shown[4] == 0 || shown[10] != 0 {
		t.Fatalf("the bracelet owns %d overlap pixels and the gloves %d; the bracelet must own some and the gloves none", shown[4], shown[10])
	}
	t.Logf("npc:22 wears %#x", worn)
	t.Logf("%d overlap pixels; owners by slot in the shop frame %v, with controls over %v, and in the map figure %v", len(overlap), shown, left, unitOwned)
}

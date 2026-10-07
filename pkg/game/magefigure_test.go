package game

import (
	"fmt"
	"image/color"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/ui"
)

// The mage figure's layer order is the original's mage programme
// (HERO-FIGURE-059): slot 8's primary sheet, the body, then slot 12, slot 10
// and its second sheet, slot 4 and its second sheet, slots 7 and 5, slot 1,
// slot 6 and last slot 8's second sheet. Each row names two slots that are
// adjacent in that list; the later one must show where both paint. Slot 8
// stands for its second sheet, the last layer of the programme.
//
// Slot 10 is the mage's Gloves and slot 4 the ring that the Russian install
// names a bracelet, so the second row is the tester's report: the gloves were
// drawn over the bracelets.
var mageFigurePairs = []struct {
	name      string
	low, high int
}{
	{"boots under gloves", 12, 10},
	{"gloves under bracelets", 10, 4},
	{"bracelets under the robe", 4, 7},
	{"robe under the amulet", 7, 5},
	{"amulet under the staff", 5, 1},
	{"staff under the hat", 1, 6},
	{"hat under the cloak's second sheet", 6, 8},
}

// mageFigureFixture paints slot low in green and slot high in blue over a red
// base. A slot with a second sheet paints it on the pixel beside its primary
// in the same colour, so the pair is checked on both sides of a paired slot.
// A cloak's primary sheet stands behind the body and sits on a pixel of its
// own; its second sheet paints both pixels of the top row.
func mageFigureFixture(dir data.FigureDir, low, high int) (missionSource, data.Equipment) {
	src := missionSource{graphicsPrefix + data.ItemFigureBasePath(dir, 1): invBaseSheet()}
	var eq data.Equipment
	for _, layer := range []struct {
		slot int
		col  color.RGBA
	}{{low, color.RGBA{G: 0xff}}, {high, color.RGBA{B: 0xff}}} {
		code := invWornCode(layer.slot)
		eq.SetCode(layer.slot, code)
		primary, second := invSparseSheet(layer.col, 0), invSparseSheet(layer.col, 1)
		if layer.slot == 8 {
			primary, second = invSparseSheet(layer.col, 2), invSparseSheet(layer.col, 0, 1)
		}
		src[graphicsPrefix+data.ItemFigureLayerPath(dir, code)] = primary
		if data.HasItemFigureSecondaryLayer(layer.slot) {
			src[graphicsPrefix+data.ItemFigureSecondaryLayerPath(dir, code)] = second
		}
	}
	return src, eq
}

// Both composers paint the mage programme: the map figure and the doll of the
// inventory window and the shop. Pixel (0,0) is where the primary sheets meet
// and (1,0) where the second sheets do; the mask names the slot on top.
func TestMageFigureLayersFollowTheOriginalPaintOrderInBothComposers(t *testing.T) {
	for _, dir := range []data.FigureDir{data.FigureDirWomanMage, data.FigureDirManMage} {
		for _, tc := range mageFigurePairs {
			t.Run(fmt.Sprintf("%s/%s", dir, tc.name), func(t *testing.T) {
				src, eq := mageFigureFixture(dir, tc.low, tc.high)
				unit, unitMask := composeUnitFigure(src, eq, figureID{Dir: dir, Face: 1})
				inventory, _ := composeInventorySubject(src, 7, eq, dir, 1)
				if unit == nil || inventory.Figure == nil {
					t.Fatalf("a composer produced no figure: unit %v inventory %v", unit != nil, inventory.Figure != nil)
				}
				meeting := [][2]int{{0, 0}}
				if data.HasItemFigureSecondaryLayer(tc.low) && data.HasItemFigureSecondaryLayer(tc.high) {
					meeting = append(meeting, [2]int{1, 0})
				}
				pictures := map[string]color.RGBA{}
				masks := map[string]*ui.SlotMask{"unit": unitMask, "inventory": inventory.SlotMask}
				for _, p := range meeting {
					pictures["unit"] = unit.RGBAAt(p[0], p[1])
					pictures["inventory"] = inventory.Figure.RGBAAt(p[0], p[1])
					for name, got := range pictures {
						if got.R != 0 || got.G != 0 || got.B != 0xff {
							t.Errorf("%s (%d,%d) = %+v, want slot %d's blue over slot %d's green", name, p[0], p[1], got, tc.high, tc.low)
						}
					}
					for name, mask := range masks {
						if n, ok := mask.At(p[0], p[1]); !ok || n != tc.high {
							t.Errorf("%s mask (%d,%d) = (%d,%v), want slot %d", name, p[0], p[1], n, ok, tc.high)
						}
					}
				}
			})
		}
	}
}

package game

import (
	"image"
	"math"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// fighterWitnessStep is one step of an oracle pass: an equipment slot and
// whether it paints that slot's second sheet.
type fighterWitnessStep struct {
	slot   int
	second bool
}

// fighterWitnessOriginal is the original's non-mage colour pass in equipment
// slots (HERO-FIGURE-059 read by HERO-FIGURE-058's drawable index plus one),
// before the tail: slot 1 comes before slot 8 and the second sheets of 4, 9
// and 10 after it.
var fighterWitnessOriginal = []fighterWitnessStep{
	{12, false}, {11, false}, {7, false}, {4, false}, {5, false}, {9, false}, {10, false},
	{1, false}, {8, false}, {4, true}, {6, false}, {9, true}, {10, true},
}

// fighterWitnessEarlier is the order this build drew before the pass was
// ported: a primary-then-secondary run per slot, the held slots last.
var fighterWitnessEarlier = []fighterWitnessStep{
	{12, false}, {11, false}, {7, false}, {4, false}, {4, true}, {5, false},
	{9, false}, {9, true}, {10, false}, {10, true}, {8, false}, {6, false}, {3, false},
}

// fighterWitnessPaint builds a figure from the installed sheets alone: the
// base, then each step's sheet where the slot is worn, then the tail slots.
// It returns the picture and the slot that owns each pixel (0 for the base).
func fighterWitnessPaint(sheets *sheetCache, dir data.FigureDir, face int, eq data.Equipment,
	steps []fighterWitnessStep, tail []int) (*image.RGBA, []int) {
	base := figureRGBA(sheets, data.ItemFigureBasePath(dir, face))
	if base == nil {
		return nil, nil
	}
	w := base.Bounds().Dx()
	pic := image.NewRGBA(base.Bounds())
	owner := make([]int, w*base.Bounds().Dy())
	put := func(layer *image.RGBA, slot int) {
		b := layer.Bounds()
		for y := 0; y < b.Dy() && y < pic.Bounds().Dy(); y++ {
			for x := 0; x < b.Dx() && x < w; x++ {
				if c := layer.RGBAAt(b.Min.X+x, b.Min.Y+y); c.A != 0 {
					pic.SetRGBA(x, y, c)
					owner[y*w+x] = slot
				}
			}
		}
	}
	put(base, 0)
	all := append([]fighterWitnessStep(nil), steps...)
	for _, n := range tail {
		all = append(all, fighterWitnessStep{n, false})
	}
	for _, st := range all {
		if occupied, _ := eq.Occupied(st.slot); !occupied {
			continue
		}
		code, _ := eq.Code(st.slot)
		path := data.ItemFigureLayerPath(dir, code)
		if st.second {
			path = data.ItemFigureSecondaryLayerPath(dir, code)
		}
		if layer := figureRGBA(sheets, path); layer != nil {
			put(layer, st.slot)
		}
	}
	return pic, owner
}

// fighterWitnessDiff counts the pixels where two oracle pictures differ.
func fighterWitnessDiff(a, b *image.RGBA) int {
	n := 0
	for y := 0; y < a.Bounds().Dy(); y++ {
		for x := 0; x < a.Bounds().Dx(); x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				n++
			}
		}
	}
	return n
}

// fighterWitnessCompare checks every pixel an oracle layer paints against the
// compositor's picture and against its hit map.
func fighterWitnessCompare(t *testing.T, name string, got *image.RGBA, mask *ui.SlotMask, want *image.RGBA, owner []int) {
	t.Helper()
	w := want.Bounds().Dx()
	bad, painted := 0, 0
	for y := 0; y < want.Bounds().Dy(); y++ {
		for x := 0; x < w; x++ {
			wc := want.RGBAAt(x, y)
			if wc.A == 0 {
				continue
			}
			painted++
			gc := got.RGBAAt(x, y)
			slot, _ := mask.At(x, y)
			if gc != wc || slot != owner[y*w+x] {
				if bad == 0 {
					t.Errorf("%s pixel (%d,%d): colour %+v and slot %d, want %+v and slot %d", name, x, y, gc, slot, wc, owner[y*w+x])
				}
				bad++
			}
		}
	}
	if bad > 0 {
		t.Errorf("%s: %d of %d painted pixels differ from the original's pass", name, bad, painted)
	}
}

// TestReleaseFighterWornItemsFollowTheOriginalsColourPass dresses a man
// fighter in a weapon, armour, bracers, gauntlets, a ring, a helm and a shield
// out of the installed definition rows, for each outcome of the held-slot
// predicate (HERO-FIGURE-060). The oracle reads the installed sheets and
// HERO-FIGURE-059's non-mage pass and never calls a compositor. It is
// compared pixel for pixel, and slot for slot with the hit map, against the
// map figure and the inventory doll. A second oracle in the earlier order must
// differ from the first, so the witness fails if the order stops mattering.
func TestReleaseFighterWornItemsFollowTheOriginalsColourPass(t *testing.T) {
	f := releaseFront(t)
	src := f.Archives.Containers
	list, _ := ReadBodyList(src)
	sheets := sheetCache{src: src, decoded: map[string]*spr256.Sprite{}, converted: map[string][]*terrain.StaticFrame{}}
	dir := data.FigureDirManFighter
	face := -1
	for n := 0; n < 40 && face < 0; n++ {
		if figureRGBA(&sheets, data.ItemFigureBasePath(dir, n)) != nil {
			face = n
		}
	}
	if face < 0 {
		t.Fatal("no base sheet for the man fighter")
	}
	pool := append(shopArmourPool(f.Table, math.MaxInt32), shopWeaponPool(f.Table, math.MaxInt32)...)
	// first returns the first pool item that targets slot, ships a primary
	// sheet in dir and, when second is set, a second sheet too.
	first := func(slot int, second bool) data.ItemCode {
		for _, c := range pool {
			if got, found := EquipTarget(c.Code, f.Table); !found || got != slot {
				continue
			}
			if figureRGBA(&sheets, data.ItemFigureLayerPath(dir, c.Code)) == nil {
				continue
			}
			if second && figureRGBA(&sheets, data.ItemFigureSecondaryLayerPath(dir, c.Code)) == nil {
				continue
			}
			return c.Code
		}
		t.Fatalf("no installed item for slot %d", slot)
		return 0
	}
	worn := func(weapon, body data.ItemCode) data.Equipment {
		var eq data.Equipment
		eq.SetCode(1, weapon)
		eq.SetCode(2, first(2, false))
		eq.SetCode(4, first(4, true))
		eq.SetCode(6, first(6, false))
		eq.SetCode(8, body)
		eq.SetCode(9, first(9, true))
		eq.SetCode(10, first(10, true))
		return eq
	}
	// slotEight lists every installed slot 8 item that ships a primary sheet.
	var slotEight []data.ItemCode
	for _, c := range pool {
		if got, found := EquipTarget(c.Code, f.Table); found && got == 8 &&
			figureRGBA(&sheets, data.ItemFigureLayerPath(dir, c.Code)) != nil {
			slotEight = append(slotEight, c.Code)
		}
	}
	// outfitFor is the first weapon whose predicate answers heldLast, with the
	// first slot 8 item its sheet overlaps, so that the two orders draw
	// different pictures.
	outfitFor := func(heldLast int, tail, before []int) (weapon, body data.ItemCode) {
		for _, c := range pool {
			if got, found := EquipTarget(c.Code, f.Table); !found || got != 1 {
				continue
			}
			if figureRGBA(&sheets, data.ItemFigureLayerPath(dir, c.Code)) == nil {
				continue
			}
			for _, b := range slotEight {
				eq := worn(c.Code, b)
				if data.FigureHeldLast(list, eq) != heldLast {
					break
				}
				want, _ := fighterWitnessPaint(&sheets, dir, face, eq, fighterWitnessOriginal, tail)
				earlier, _ := fighterWitnessPaint(&sheets, dir, face, eq, fighterWitnessEarlier, before)
				if want != nil && fighterWitnessDiff(want, earlier) > 0 {
					return c.Code, b
				}
			}
		}
		t.Fatalf("no installed weapon with held-last slot %d separates the two orders", heldLast)
		return 0, 0
	}
	for _, tc := range []struct {
		name         string
		heldLast     int
		tail, before []int
	}{
		{"shield-last body", 2, []int{2}, []int{1, 2}},
		{"weapon-last body", 1, []int{1}, []int{2, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			weapon, body := outfitFor(tc.heldLast, tc.tail, tc.before)
			eq := worn(weapon, body)
			want, owner := fighterWitnessPaint(&sheets, dir, face, eq, fighterWitnessOriginal, tc.tail)
			if want == nil {
				t.Fatal("the oracle found no base sheet")
			}
			earlier, _ := fighterWitnessPaint(&sheets, dir, face, eq, fighterWitnessEarlier, tc.before)
			d := fighterWitnessDiff(want, earlier)
			if d == 0 {
				t.Fatal("the earlier order draws the same figure for this equipment; the witness cannot tell the orders apart")
			}
			t.Logf("weapon %#x, slot 8 item %#x: the original's pass differs from the earlier order on %d pixels", uint16(weapon), uint16(body), d)

			unit, unitMask := composeUnitFigure(src, eq, figureID{Dir: dir, Face: face})
			if unit == nil {
				t.Fatal("the map figure composed nothing")
			}
			fighterWitnessCompare(t, "map figure", unit, unitMask, want, owner)
			subject, unread := composeInventorySubject(src, 1, eq, dir, face)
			if subject.Figure == nil {
				t.Fatalf("the inventory doll composed nothing; unread %v", unread)
			}
			fighterWitnessCompare(t, "inventory doll", subject.Figure, subject.SlotMask, want, owner)
		})
	}
}

package data

import (
	"encoding/binary"
	"testing"
)

// maskEntry and maskCollection are a MaskTable built in test code:
// defsearch_test.go's testCollection with the raw block an armour, shield or
// weapon entry carries. Index 0 is the reserved empty entry a one-based
// collection allocates and never writes.
type maskEntry struct {
	name   string
	params []int32
	masks  [ShopShapes]uint16
}

type maskCollection []maskEntry

func (c maskCollection) Len() int                    { return len(c) }
func (c maskCollection) EntryName(i int) string      { return c[i].name }
func (c maskCollection) EntryParams(i int) []int32   { return c[i].params }
func (c maskCollection) EntryStrings(i int) []string { return nil }

func (c maskCollection) EntryRaw(i int) []byte {
	if c[i].name == "" {
		return nil
	}
	raw := make([]byte, shopMaskStride*ShopShapes)
	for s, m := range c[i].masks {
		binary.LittleEndian.PutUint16(raw[shopMaskStride*s:], m)
	}
	return raw
}

// factorTable is a ScaleTable whose price slot carries a chosen factor and whose
// other eight doubles are 1, so a test states the one number the window reads.
type factorTable []float64

func (t factorTable) Len() int               { return len(t) }
func (t factorTable) EntryName(i int) string { return "" }

func (t factorTable) EntryDoubles(i int) []float64 {
	d := make([]float64, 9)
	for j := range d {
		d[j] = 1
	}
	d[scalePriceSlot] = t[i]
	return d
}

// priceRow is a parameter array carrying the price cell and the armour slot
// column, long enough for both.
func priceRow(price, slot int32) []int32 {
	p := make([]int32, armorSlotColumn+1)
	for i := range p {
		p[i] = -1
	}
	p[itemPriceColumn] = price
	p[armorSlotColumn] = slot
	return p
}

// AC-1: the window's value and the item's price differ by the rounding addend
// alone, and a negative price cell is admitted at no ceiling.
func TestShopValueAndPriceDifferByTheRoundingAddend(t *testing.T) {
	shapes := factorTable{2.5}
	materials := factorTable{1.4}

	// 3 x 2.5 x 1.4 = 10.5: the window truncates to 10, the price rounds to 11.
	if got := ItemValue(3, shapes, materials, 0, 0); got != 10 {
		t.Errorf("ItemValue = %d, want 10 (trunc of 10.5)", got)
	}
	if got := ItemPrice(3, shapes, materials, 0, 0); got != 11 {
		t.Errorf("ItemPrice = %d, want 11 (trunc of 11.0)", got)
	}

	c := maskCollection{
		{},
		{name: "sentinel", params: priceRow(-1, 1), masks: [ShopShapes]uint16{1}},
	}
	for _, ceiling := range []int32{0, 1000, 1 << 30} {
		if pool := ShopPool(c, WeaponShopClass, shapes, materials, ceiling); len(pool) != 0 {
			t.Errorf("ceiling %d admitted the negative-price row: %+v", ceiling, pool)
		}
	}
}

// AC-2: a material is admitted only where the row's own mask for that shape has
// the bit, and a ceiling of zero admits only a zero value.
func TestShopPoolAdmitsOnlyTheMaskedMaterialsInsideTheCeiling(t *testing.T) {
	// Materials 0 and 3 at shape 1 only; every factor 1, so value == price cell.
	c := maskCollection{
		{},
		{name: "axe", params: priceRow(50, 1), masks: [ShopShapes]uint16{1: 1<<0 | 1<<3}},
	}
	shapes := factorTable{1, 1, 1, 1, 1}
	materials := make(factorTable, ShopMaterials)
	for i := range materials {
		materials[i] = 1
	}

	pool := ShopPool(c, WeaponShopClass, shapes, materials, 100)
	if len(pool) != 2 {
		t.Fatalf("pool = %+v, want exactly the two masked materials", pool)
	}
	for i, want := range []int{0, 3} {
		if pool[i].Material != want || pool[i].Shape != 1 || pool[i].Row != 1 {
			t.Errorf("pool[%d] = %+v, want material %d at shape 1 row 1", i, pool[i], want)
		}
		if pool[i].Price != 50 {
			t.Errorf("pool[%d] price = %d, want 50", i, pool[i].Price)
		}
		if got := pool[i].Code; got != ComposeItemCode(want, weaponItemClass, 1, 1) {
			t.Errorf("pool[%d] code = %04x, want the composed code", i, uint16(got))
		}
	}

	if pool := ShopPool(c, WeaponShopClass, shapes, materials, 49); len(pool) != 0 {
		t.Errorf("ceiling 49 admitted a 50-coin item: %+v", pool)
	}
	if pool := ShopPool(c, WeaponShopClass, shapes, materials, 0); len(pool) != 0 {
		t.Errorf("ceiling 0 admitted a 50-coin item: %+v", pool)
	}
}

// The armour class comes off the row's Slot column, and a cell that names no
// equipment place composes field B of zero rather than a slot the item cannot
// occupy.
func TestArmorShopClassReadsTheSlotColumn(t *testing.T) {
	for _, tc := range []struct {
		name string
		slot int32
		want int
	}{
		{"a real slot", 7, 7},
		{"zero", 0, 0},
		{"past the twelve", 13, 0},
		{"negative", -1, 0},
	} {
		if got := ArmorShopClass(priceRow(10, tc.slot)); got != tc.want {
			t.Errorf("%s: ArmorShopClass = %d, want %d", tc.name, got, tc.want)
		}
	}
	if got := ArmorShopClass([]int32{1, 2}); got != 0 {
		t.Errorf("short row: ArmorShopClass = %d, want 0", got)
	}
}

func TestShopPoolRefusesEveryTableItCannotRead(t *testing.T) {
	shapes := factorTable{1}
	materials := factorTable{1}
	full := maskCollection{{}, {name: "axe", params: priceRow(5, 1), masks: [ShopShapes]uint16{1}}}

	if pool := ShopPool(nil, WeaponShopClass, shapes, materials, 100); pool != nil {
		t.Errorf("nil collection admitted %+v", pool)
	}
	if pool := ShopPool(full, nil, shapes, materials, 100); pool != nil {
		t.Errorf("nil class admitted %+v", pool)
	}
	if pool := ShopPool(full, WeaponShopClass, shapes, materials, -1); pool != nil {
		t.Errorf("negative ceiling admitted %+v", pool)
	}

	// A row with no name is the reserved entry and is skipped; a row too short
	// to carry the price cell is skipped; a row whose masks are all clear names
	// no material at all.
	thin := maskCollection{
		{},
		{name: "", params: priceRow(5, 1), masks: [ShopShapes]uint16{0xffff}},
		{name: "short", params: []int32{1, 2}, masks: [ShopShapes]uint16{0xffff}},
		{name: "unmasked", params: priceRow(5, 1)},
	}
	if pool := ShopPool(thin, WeaponShopClass, shapes, materials, 100); len(pool) != 0 {
		t.Errorf("thin table admitted %+v", pool)
	}
}

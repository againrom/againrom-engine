package mapload_test

import (
	"fmt"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

// ALM-127/128: the walker selects Shop using the stored low16 kind,
// while constructor definition lookup receives its low8. Only Building
// tests the placement word for zero; nonzero is not an HP quantity.
func TestStructurePlacement1175ZeroOverrideAndClassWidths(t *testing.T) {
	tbl := bldTable(map[int][]int32{
		1:  bldRowHealth(2, 1, 1, 3, 73),
		33: bldRowHealth(2, 1, 1, 3, 73),
		34: bldRowHealth(2, 1, 1, 3, 73),
		35: bldRowHealth(2, 1, 1, 3, 73),
	})
	for _, kind := range []uint32{1, 33, 34, 35, 0x10022, 0x122} {
		for _, word := range []uint16{0, 1, 100, 0x7fff, 0x8000, 0xffff} {
			t.Run(fmt.Sprintf("kind_%x_word_%x", kind, word), func(t *testing.T) {
				o := alm.Object{Kind: kind, X: 10 << 8, Y: 10 << 8, Field0C: word}
				if kind == 33 {
					o.Ext = ext(2, 1)
				}
				got := mapload.Structures(structMap(o), tbl)
				want := uint16(73)
				if word == 0 && uint16(kind) != 34 && uint16(kind) != 35 {
					want = 0
				}
				if len(got) != 1 || got[0].Field42 != want || got[0].MaxHealth != 73 {
					t.Fatalf("placement kind=%#x word=%#x: %+v, want current/max %d/73", kind, word, got, want)
				}
				if got[0].Width != 2 || got[0].Height != 1 || got[0].Col != 10 || got[0].Row != 10 {
					t.Fatalf("zero override changed footprint: %+v", got[0])
				}
			})
		}
	}
}

package mapload

import (
	"fmt"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

func TestType9Head1175UsesByteResultWithoutNarrowingTail(t *testing.T) {
	for _, tc := range []struct {
		a    uint16
		kind uint8
	}{{212, 255}, {213, 0}, {65535, 42}} {
		t.Run(fmt.Sprint(tc.a), func(t *testing.T) {
			m := &alm.Map{Enchantments: []alm.Enchantment{{A: tc.a, B: 0x1234, C: 0xabcd}}}
			e := alm.LootElement{Code: 0x0e1f, TileMarkerIndex: 1}
			item, err := lootItemInstance(e, m, nil)
			if err != nil {
				t.Fatal(err)
			}
			want := sim.ItemEffect{Kind: tc.kind, Operand: 0xcd34}
			if len(item.Effects) != 1 || item.Effects[0] != want {
				t.Fatalf("head %+v, want %+v", item.Effects, want)
			}
			m.Enchantments[0].A = 0
			item, err = lootItemInstance(e, m, nil)
			if err != nil || len(item.Effects) != 0 {
				t.Fatalf("A=0 produced a head: %+v, %v", item, err)
			}
			m.Enchantments[0].A, m.Enchantments[0].X = tc.a, 1
			if _, err := lootItemInstance(e, m, nil); err == nil {
				t.Fatal("ineligible coordinates admitted")
			}
			m.Enchantments[0].X = 0
			m.Enchantments[0].Elements = []alm.EnchantmentElement{{Kind: 0x100}}
			if _, err := lootItemInstance(e, m, nil); err == nil {
				t.Fatal("head narrowing also narrowed the tail")
			}
		})
	}
}

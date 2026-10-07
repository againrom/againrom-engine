package mapload_test

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestType9Placement1175PreservesWrappedHeadInBothContainers(t *testing.T) {
	for _, tc := range []struct {
		a    uint16
		kind uint8
	}{{212, 255}, {213, 0}, {65535, 42}} {
		t.Run(fmt.Sprint(tc.a), func(t *testing.T) {
			m := &alm.Map{
				Width: 10, Height: 10, FormatVersion: 990,
				Meta: alm.Meta{Word2C: 2},
				LootSection: alm.LootSection{Body: llJoin(
					llHead(1, 0, llCell(2), llCell(3), 0), llElement(0x0e1f, 0, 1),
					llHead(1, 7, 0, 0, 0), llElement(0x0e1f, 0, 1),
				)},
				Units:        []alm.Unit{{X: 0x100, Y: 0x100, UnitID: 7}},
				Enchantments: []alm.Enchantment{{A: tc.a, B: 0x1234, C: 0xabcd}},
			}
			world, err := mapload.FromALMWith(m, nil, mapload.DifficultyNormal)
			if err != nil {
				t.Fatal(err)
			}
			check := func(w *sim.World) {
				t.Helper()
				ground := w.Sacks()
				stock, ok := w.CarriedItems(0)
				if len(ground) != 1 || len(ground[0].ItemInstances) != 1 || !ok || len(stock) != 1 {
					t.Fatalf("containers: ground=%+v stock=%+v present=%v", ground, stock, ok)
				}
				want := []sim.ItemEffect{{Kind: tc.kind, Operand: 0xcd34}}
				if !slices.Equal(ground[0].ItemInstances[0].Effects, want) || !slices.Equal(stock[0].Effects, want) {
					t.Fatalf("head lost in placement: ground=%+v stock=%+v", ground[0].ItemInstances, stock)
				}
			}
			check(world)
			form, err := world.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold sim.World
			if err := cold.UnmarshalBinary(form); err != nil {
				t.Fatal(err)
			}
			check(&cold)
		})
	}
}

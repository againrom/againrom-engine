package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"strings"
	"testing"
)

func TestNativeSaveKeepsInterleavedItemIdentities(t *testing.T) {
	first := ItemInstance{Code: 0x0e06, Kind: 3, Price: 125,
		Effects: []ItemEffect{{Kind: 8, Mode: 1, Operand: 0x03c00064}}}
	middle := ItemInstance{Code: 0x0102, Kind: 1, Price: 37}
	for _, property := range []string{"price", "weight", "effect", "kind"} {
		t.Run(property, func(t *testing.T) {
			last := first.Clone()
			switch property {
			case "price":
				last.Price++
			case "weight":
				last.WeightPresent = true
				last.Weight++
			case "effect":
				last.Effects[0].Operand++
			case "kind":
				last.Kind = 2
			}
			w := mustStockedWorld(t, 71,
				[]Entity{{ID: 1, X: 1, Y: 1, HP: 20, MaxHP: 20}, {ID: 2, X: 2, Y: 1, HP: 20, MaxHP: 20}},
				[]Stock{{ID: 1, ItemInstances: []ItemInstance{first, first, first, middle, last, last}}})
			want, _ := w.CarriedStacks(1)
			if len(want) != 3 || want[0].Count != 3 || want[1].Count != 1 || want[2].Count != 2 {
				t.Fatalf("fixture did not retain three distinct stacks: %+v", want)
			}
			wire, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var back World
			if err := back.UnmarshalBinary(wire); err != nil {
				t.Fatalf("fresh LOAD of ordinary native SAVE: %v", err)
			}
			got, _ := back.CarriedStacks(1)
			if !reflect.DeepEqual(got, want) || back.Hash() != w.Hash() {
				t.Fatalf("LOAD changed item identity, quantity or order: got=%+v want=%+v", got, want)
			}
			again, err := back.MarshalBinary()
			if err != nil || !bytes.Equal(again, wire) {
				t.Fatalf("SAVE after LOAD changed bytes: %v", err)
			}
			// A real disagreement between the two wire projections must still
			// refuse atomically. Locate the independently written flat list.
			needle := binary.LittleEndian.AppendUint32(nil, 6)
			for _, code := range []uint16{first.Code, first.Code, first.Code, middle.Code, last.Code, last.Code} {
				needle = binary.LittleEndian.AppendUint16(needle, code)
			}
			if bytes.Count(wire, needle) != 1 {
				t.Fatal("flat carried list is not uniquely located")
			}
			bad := append([]byte(nil), wire...)
			bad[bytes.Index(wire, needle)+4] ^= 1
			before := back.Hash()
			if err := back.UnmarshalBinary(bad); err == nil || !strings.Contains(err.Error(), "code projection disagrees") {
				t.Fatalf("corrupt code list was not refused: %v", err)
			}
			if back.Hash() != before {
				t.Fatal("refused projection replaced the receiver")
			}
			for _, world := range []*World{w, &back} {
				if err := world.MoveCarried(1, 2, first.Code, 1); err != nil {
					t.Fatal(err)
				}
			}
			moved, _ := back.CarriedItems(2)
			if len(moved) != 1 || !reflect.DeepEqual(moved[0], first) {
				t.Fatalf("next transfer lost the first stack's instance: %+v", moved)
			}
			if back.Hash() != w.Hash() {
				t.Fatal("next transfer differs across LOAD")
			}
		})
	}
}

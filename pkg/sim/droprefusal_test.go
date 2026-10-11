package sim

import (
	"encoding/binary"
	"testing"
)

func TestDropTargetRefusalKeepsItemAtFeet(t *testing.T) {
	for _, test := range []struct {
		name  string
		aim   CellPoint
		setup func(*World)
	}{
		{"outside window", CellPoint{X: 8, Y: 5}, nil},
		{"ground blocked", CellPoint{X: 6, Y: 5}, func(w *World) { i, _ := w.cellIndex(6, 5); w.grid[i] |= blockGround }},
		{"occupied", CellPoint{X: 6, Y: 5}, func(w *World) {
			w.entities[1].X, w.entities[1].Y, w.entities[1].TokenSize = 6, 5, 2
		}},
		{"structure", CellPoint{X: 6, Y: 5}, func(w *World) {
			w.DeclareStructures([]Structure{{ID: 1, Col: 6, Row: 5, Width: 1, Height: 1, Attach: 1}})
		}},
		{"outside map", CellPoint{X: -1, Y: 5}, nil},
		{"unbound sack slot", CellPoint{X: 6, Y: 5}, func(w *World) { w.SetSavedCellRecords([]SavedCellRecord{{Cell: cellKey(6, 5), Sack: 91}}) }},
		{"physical sack slot", CellPoint{X: 6, Y: 5}, func(w *World) {
			c := SavedActorCell{Cell: cellKey(6, 5)}
			binary.LittleEndian.PutUint32(c.Payload[16:], 91)
			w.savedMotion = &savedActorMotionState{Cells: []SavedActorCell{c}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, landing := range []bool{false, true} {
				for _, worn := range []bool{false, true} {
					w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5}, {ID: 2, X: 9, Y: 9}}, []Stock{{ID: 1, Items: []uint16{0x101}}})
					if worn {
						Step(w, []Command{Equip(1, 0, 1)})
					}
					if test.setup != nil {
						test.setup(w)
					}
					aim := test.aim
					if landing {
						var ok bool
						aim, ok = w.DropLanding(1, aim.X, aim.Y)
						if !ok || aim != (CellPoint{X: 5, Y: 5}) {
							t.Fatalf("landing = %v, %v, want own cell", aim, ok)
						}
					}
					command := DropCarried(1, 0, aim)
					if worn {
						command = DropWorn(1, 1, aim)
					}
					Step(w, []Command{command})
					sacks := w.Sacks()
					if len(sacks) != 1 || sacks[0].X != 5 || sacks[0].Y != 5 || !equalCodes(sacks[0].Items, []uint16{0x101}) {
						t.Fatalf("landing=%t worn=%t sacks = %+v, want item at 5,5", landing, worn, sacks)
					}
					pack, _ := w.Carried(1)
					slots, _ := w.Equipped(1)
					if len(pack) != 0 || slots != ([EquipSlots]uint16{}) {
						t.Fatal("item remains in inventory", pack, slots)
					}
					bytes, err := w.MarshalBinary()
					if err != nil {
						t.Fatal(err)
					}
					var cold World
					if err := cold.UnmarshalBinary(bytes); err != nil {
						t.Fatal(err)
					}
					if cold.Hash() != w.Hash() {
						t.Fatal("cold World hash differs")
					}
				}
			}
		})
	}
}

func TestDropRefusedTargetMergesBoundItemsAtFeet(t *testing.T) {
	for _, worn := range []bool{false, true} {
		w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5}}, []Stock{{ID: 1, Items: []uint16{0x101, 0x102}}})
		bindOperationsPack(t, w, 0)
		if worn {
			Step(w, []Command{Equip(1, 0, 1)})
		}
		index := ItemSlot(1)
		if worn {
			index = 0
		}
		Step(w, []Command{DropCarried(1, index, CellPoint{X: 5, Y: 5})})
		sackID := w.sacks[0].ObjectID
		w.DeclareStructures([]Structure{{ID: 1, Col: 6, Row: 5, Width: 1, Height: 1, Attach: 1}})
		command := DropCarried(1, 0, CellPoint{X: 6, Y: 5})
		if worn {
			command = DropWorn(1, 1, CellPoint{X: 6, Y: 5})
		}
		Step(w, []Command{command})
		if sacks := w.Sacks(); len(sacks) != 1 || sacks[0].ObjectID != sackID || !equalCodes(sacks[0].Items, []uint16{0x102, 0x101}) {
			t.Fatalf("worn=%t ground = %+v, want bound items merged into %d at feet", worn, sacks, sackID)
		}
		cold := reloadOperations(t, w)
		for range 16 {
			Step(w, nil)
			Step(cold, nil)
			if cold.Hash() != w.Hash() {
				t.Fatal("bound item cold continuation differs")
			}
		}
	}
}

func TestDropIntoExistingOccupiedTargetSackKeepsTheAim(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5}, {ID: 2, X: 6, Y: 5}}, []Stock{{ID: 1, Items: []uint16{0x101}}})
	w.pourSack(6, 5, 0, plainItems([]uint16{0x102}))
	at, ok := w.DropLanding(1, 6, 5)
	if !ok || at != (CellPoint{X: 6, Y: 5}) {
		t.Fatal("existing target sack changed aim", at, ok)
	}
	Step(w, []Command{DropCarried(1, 0, at)})
	if sacks := w.Sacks(); len(sacks) != 1 || sacks[0].X != 6 || sacks[0].Y != 5 || !equalCodes(sacks[0].Items, []uint16{0x102, 0x101}) {
		t.Fatal("existing target sack did not merge", sacks)
	}
}

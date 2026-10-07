package game

import (
	"fmt"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestCurrentCloudRecordsOwnFirstBlockProjection(t *testing.T) {
	for _, retained := range []bool{false, true} {
		for _, rawPlanes := range []bool{false, true} {
			t.Run(fmt.Sprintf("retained=%v/planes=%v", retained, rawPlanes), func(t *testing.T) {
				script, err := sim.NewScript(nil, []sim.ScriptInstant{{Op: sim.ScriptInstantCastAtCell, Args: [10]int32{10, 10, 16, 16, 3, 10}}},
					[]sim.ScriptTrigger{{Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}, Once: true, Latch: 999}})
				if err != nil {
					t.Fatal(err)
				}
				grid := make([]byte, 32*32)
				grid[16*32+16] = 3
				w, err := sim.NewSpelledWorld(7, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, grid, nil, script,
					[]sim.SpellRule{{ID: 3, Area: true, Distribution: 3, Radius: 1, AreaDuration: 15, Damaging: true, DamageMin: 1, DamageMax: 2}})
				if err != nil {
					t.Fatal(err)
				}
				for i := 0; !w.HasNativeAreaEffects() && i < 32; i++ {
					sim.Step(w, nil)
				}
				areas, err := w.NativeAreaSaveStates()
				if err != nil || len(areas) != 1 || areas[0].Mode != sim.AreaModeCloud || len(areas[0].Cells) < 2 {
					t.Fatal("script did not create a multi-cell cloud", areas, err)
				}
				if rawPlanes {
					planes := &sim.SavedCellPlanes{}
					planes.Costs[0] = 255
					for _, key := range areas[0].Cells {
						planes.Cost[key], planes.CostKnown[key] = 17, 1
						planes.Static[key], planes.Dynamic[key] = 0x12, 0x96
					}
					planes.Static[0x1818], planes.Dynamic[0x1818] = 0x40, 0x90
					if err := w.ImportOriginalCellPlanes(planes); err != nil {
						t.Fatal(err)
					}
				}
				before := w.Hash()
				b := generatedDocumentBuilder{doc: sav.DocumentData{World: &sav.DocumentWorldData{}}, state: &SnapshotSAVDocument{}}
				old := sav.DocumentCellData{Cell: areas[0].Cells[0], Cost: 77, Static: 5, Operation: 11, Power: 13}
				m := &alm.Map{Width: 32, Height: 32}
				if retained {
					b.doc.World.Cells = []sav.DocumentCellData{old}
					err = b.currentActorCells(w, m, nil)
				} else {
					err = b.currentSpatial(w, m, &mapload.Table{}, nil)
				}
				if err != nil {
					t.Fatal(err)
				}
				blocks := map[uint16]sav.BlockRecord{}
				for _, row := range b.doc.World.Blocks {
					blocks[row.Cell] = row
				}
				cells := map[uint16]sav.DocumentCellData{}
				for _, row := range b.doc.World.Cells {
					cells[row.Cell] = row
				}
				for _, key := range areas[0].Cells {
					stat, dyn := uint8(0x20), uint8(0x20)
					if key == 0x1010 {
						stat, dyn = 0x27, 0x27
					}
					if rawPlanes {
						stat, dyn = 0x32, 0xb6
					}
					if got := blocks[key]; got != (sav.BlockRecord{Cell: key, Static: stat, Dyn: dyn}) {
						t.Fatalf("cloud cell %04x block=%+v, want static/dynamic %02x/%02x", key, got, stat, dyn)
					}
					if _, ok := cells[key]; !ok {
						t.Fatalf("cloud cell %04x has no record", key)
					}
				}
				if retained && cells[old.Cell] != old {
					t.Fatal("cloud changed the existing cell baseline or tail", cells[old.Cell], old)
				}
				if rawPlanes && blocks[0x1818] != (sav.BlockRecord{Cell: 0x1818, Static: 0x40, Dyn: 0x90}) {
					t.Fatal("cloud changed an unrelated block", blocks[0x1818])
				}
				if w.Hash() != before {
					t.Fatal("cloud projection changed the live World")
				}
			})
		}
	}
}

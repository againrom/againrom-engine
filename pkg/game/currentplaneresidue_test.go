package game

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
)

func currentPlaneResidueSource(t *testing.T) *FrontEnd {
	t.Helper()
	f := cellStateFront(t)
	open, town, err := f.RestoreOriginal(cellStateLiteral(t, f, false))
	if err != nil || town {
		t.Fatal("cell source", town, err)
	}
	if err := f.App("current plane source").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	f.live.tick()
	return f
}

func loadCurrentPlaneDocument(t *testing.T, doc sav.DocumentData) *FrontEnd {
	t.Helper()
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	f := cellStateFront(t)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("current plane LOAD", town, err)
	}
	if err := f.App("current plane cold LOAD").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestCurrentPlaneResidueYieldsToOrdinaryCellAndBlockEdits(t *testing.T) {
	for _, edit := range []string{"none", "deleted-cell", "deleted-cell-block", "outside-block", "low-dynamic-block"} {
		t.Run(edit, func(t *testing.T) {
			f := currentPlaneResidueSource(t)
			doc, a := currentRootSAVDocument(t, f)
			if len(a.CellPlaneResidue) != 4 {
				t.Fatal("fixture must isolate four omitted plane cells", a.CellPlaneResidue)
			}
			leaf, _, _ := sav.NativeActions(doc.State)
			switch edit {
			case "deleted-cell":
				doc.World.Cells = append(doc.World.Cells, sav.DocumentCellData{Cell: 0x100f, Cost: 23})
			case "deleted-cell-block":
				for i := range doc.World.Blocks {
					if doc.World.Blocks[i].Cell == 0x100f {
						doc.World.Blocks[i].Static = 4
					}
				}
			case "outside-block":
				doc.World.Blocks = append(doc.World.Blocks, sav.BlockRecord{Cell: 0x0806, Static: 4, Dyn: 18})
			case "low-dynamic-block":
				doc.World.Blocks = append(doc.World.Blocks, sav.BlockRecord{Cell: 0x0909, Static: 5, Dyn: 9})
			}
			slices.SortFunc(doc.World.Blocks, func(a, b sav.BlockRecord) int { return int(a.Cell) - int(b.Cell) })
			unchanged, _, _ := sav.NativeActions(doc.State)
			if !bytes.Equal(leaf, unchanged) {
				t.Fatal("ordinary edit changed the residue policy")
			}
			cold := loadCurrentPlaneDocument(t, doc)
			planes, present := cold.live.world.SavedCellPlanes()
			if !present {
				t.Fatal("plane carrier lost")
			}
			cost := byte(17)
			if edit == "deleted-cell" || edit == "deleted-cell-block" {
				cost = 8
			}
			if planes.Cost[0x100f] != cost {
				t.Fatalf("deleted-cell cost=%d want%d", planes.Cost[0x100f], cost)
			}
			outsideStatic, outsideDynamic := byte(0), byte(16)
			if edit == "outside-block" {
				outsideStatic, outsideDynamic = 4, 18
			}
			if planes.Static[0x0806] != outsideStatic || planes.Dynamic[0x0806] != outsideDynamic {
				t.Fatal("outside Block edit lost", planes.Static[0x0806], planes.Dynamic[0x0806])
			}
			lowStatic, lowDynamic := byte(0), byte(15)
			if edit == "low-dynamic-block" {
				lowStatic, lowDynamic = 5, 9
			}
			if planes.Static[0x0909] != lowStatic || planes.Dynamic[0x0909] != lowDynamic {
				t.Fatal("low-Dyn Block edit lost", planes.Static[0x0909], planes.Dynamic[0x0909])
			}
			if edit == "none" && cold.live.world.Hash() != f.live.world.Hash() {
				t.Fatal("unchanged residue did not preserve the entire current World")
			}
			if edit == "deleted-cell" {
				_, cells, _, _ := cold.live.world.SavedActorMotions()
				found := false
				for _, cell := range cells {
					if cell.Cell == 0x100f {
						found = cell.Payload[0] == 23
					}
				}
				if !found {
					t.Fatal("new ordinary cell baseline was not adopted")
				}
			}
			for range 2 {
				next, _ := currentRootSAVDocument(t, cold)
				second := loadCurrentPlaneDocument(t, next)
				if cold.live.world.Hash() != second.live.world.Hash() {
					currentMenuWorldDiagnostics(t, cold.live.world, second.live.world)
					t.Fatal("ordinary-edited planes changed during the next cold cycle")
				}
				for range 20 {
					cold.live.tick()
					second.live.tick()
					if cold.live.world.Hash() != second.live.world.Hash() {
						t.Fatal("ordinary-edited plane continuation changed")
					}
				}
				cold = second
			}
		})
	}
}

func TestCurrentPlaneResidueMalformedPoliciesAreAtomic(t *testing.T) {
	for _, fault := range []string{"duplicate", "empty-fields", "field-domain", "unselected-value", "truncated", "ordinary-conflict"} {
		t.Run(fault, func(t *testing.T) {
			f := currentPlaneResidueSource(t)
			doc, a := currentRootSAVDocument(t, f)
			if len(a.CellPlaneResidue) == 0 {
				t.Fatal("missing controlled residue")
			}
			encoded, err := json.Marshal(a.CellPlaneResidue)
			if err != nil {
				t.Fatal(err)
			}
			var packed []byte
			if err := json.Unmarshal(encoded, &packed); err != nil {
				t.Fatal(err)
			}
			switch fault {
			case "duplicate":
				packed = append(append([]byte(nil), packed[:22]...), packed...)
			case "empty-fields":
				packed[2] = 0
			case "field-domain":
				packed[2] = 8
			case "unselected-value":
				packed[3] = 41
			case "truncated":
				packed = packed[:len(packed)-1]
			case "ordinary-conflict":
				cell := a.CellPlaneResidue[0].Cell
				doc.World.Blocks = append(doc.World.Blocks, sav.BlockRecord{Cell: cell, Dyn: 17})
				slices.SortFunc(doc.World.Blocks, func(a, b sav.BlockRecord) int { return int(a.Cell) - int(b.Cell) })
				anchor := currentPlaneOrdinaryFields(&doc).anchor(cell)
				copy(packed[6:22], anchor[:])
			}
			leaf, _, err := sav.NativeActions(doc.State)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(leaf, &fields); err != nil {
				t.Fatal(err)
			}
			fields["CellPlaneResidue"], err = json.Marshal(packed)
			if err != nil {
				t.Fatal(err)
			}
			leaf, err = json.Marshal(fields)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
				t.Fatal(err)
			}
			raw, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			before := f.live.world.Hash()
			open, town, err := f.RestoreOriginal(raw)
			if err == nil && !town {
				err = f.App("invalid plane policy").OpenMission(open)
			}
			if err == nil || f.live.world.Hash() != before {
				t.Fatal("malformed policy was accepted or partially adopted", err)
			}
		})
	}
}

func TestCurrentPlaneResidueCodecFitsTheFullCellKeyspace(t *testing.T) {
	rows := make(currentCellPlaneResidues, 65536)
	for i := range rows {
		rows[i] = currentCellPlaneResidue{Cell: uint16(i), Fields: currentPlaneCost, Cost: 17}
	}
	raw, err := json.Marshal(rows)
	if err != nil || len(raw) > 2<<20 {
		t.Fatal("full plane residue framing", len(raw), err)
	}
	var decoded currentCellPlaneResidues
	if err := json.Unmarshal(raw, &decoded); err != nil || len(decoded) != len(rows) || decoded[65535] != rows[65535] {
		t.Fatal("full u16 keyspace did not decode", err)
	}
}

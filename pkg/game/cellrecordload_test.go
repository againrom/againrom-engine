package game

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// cellRecordsAfterLoad names, by cell, the rows a LOAD drops as marked absent
// (DIV-2885) and the Ground and Air keys it releases as a terminal corpse's
// (DIV-2029), running both production decisions on copies.
func cellRecordsAfterLoad(doc *sav.DocumentData) (dropped map[uint16]bool, released map[uint16][2]bool, err error) {
	if doc == nil || doc.World == nil {
		return nil, nil, fmt.Errorf("cell records: the document has no world")
	}
	dropped, released = map[uint16]bool{}, map[uint16][2]bool{}
	last := map[uint16]int{}
	for i, c := range doc.World.Cells {
		last[c.Cell] = i
	}
	actions, err := readCurrentActions(doc)
	if err != nil {
		return nil, nil, err
	}
	if actions != nil {
		var w sim.World
		records := make([]sim.SavedCellRecord, 0, len(last))
		for cell := range last {
			records = append(records, sim.SavedCellRecord{Cell: cell})
		}
		if err := w.ImportOriginalCellRecords(records); err != nil {
			return nil, nil, err
		}
		if err := restoreCurrentCarrierAbsence(&w, doc, actions); err != nil {
			return nil, nil, err
		}
		kept := map[uint16]bool{}
		for _, row := range w.SavedCellRecords() {
			kept[row.Cell] = true
		}
		for cell := range last {
			if !kept[cell] {
				dropped[cell] = true
			}
		}
	}
	// The projection writes cells, Blocks and dead records' raw fields.
	projected, world := *doc, *doc.World
	world.Cells = append([]sav.DocumentCellData(nil), doc.World.Cells...)
	world.Blocks = append([]sav.BlockRecord(nil), doc.World.Blocks...)
	projected.World = &world
	projected.Objects = append([]sav.DocumentRecordData(nil), doc.Objects...)
	for _, index := range doc.DeadActors {
		if index == 0 || int(index) > len(projected.Objects) {
			continue
		}
		r := &projected.Objects[index-1]
		r.Raw = append([]sav.DocumentRawData(nil), r.Raw...)
		for i := range r.Raw {
			r.Raw[i].Bytes = append([]byte(nil), r.Raw[i].Bytes...)
		}
	}
	if err := projectTerminalActorRegistry(&projected, nil); err != nil {
		return nil, nil, err
	}
	for cell, i := range last {
		before, after := doc.World.Cells[i], projected.World.Cells[i]
		slots := [2]bool{before.GroundActor != 0 && after.GroundActor == 0, before.AirActor != 0 && after.AirActor == 0}
		if slots != [2]bool{} {
			released[cell] = slots
		}
	}
	return dropped, released, nil
}

func TestCellRecordsAfterLoadFollowsTheAbsenceAndTerminalPolicies(t *testing.T) {
	doc := sav.DocumentData{World: &sav.DocumentWorldData{}}
	for i, tuple := range []struct {
		hp    int16
		stage uint32
	}{{-76, 4}, {-1000, 1}, {-5, 4}} {
		r := mustNewRecord("Unit")
		mustSetValue(&r, "Identity", uint32(i+1)*0x100)
		mustSetValue(&r, "Stage", tuple.stage)
		mustSetValue(&r, "Health", uint32(uint16(tuple.hp)))
		for _, field := range []string{"U50", "U54"} {
			value, _ := savedMotionRaw(&r, field, 4)
			binary.LittleEndian.PutUint32(value, 7)
		}
		doc.Objects = append(doc.Objects, r)
		doc.DeadActors = append(doc.DeadActors, uint16(i+1))
	}
	doc.World.Cells = []sav.DocumentCellData{
		{Cell: 0x0807, GroundActor: 0x100},
		{Cell: 0x0808, GroundActor: 0x200, AirActor: 0x100},
		{Cell: 0x0809, GroundActor: 0x300, Sack: 0x51000010},
	}
	terminal := map[uint16][2]bool{0x0807: {true, false}, 0x0808: {false, true}}
	dropped, released, err := cellRecordsAfterLoad(&doc)
	if err != nil || len(dropped) != 0 || !reflect.DeepEqual(released, terminal) {
		t.Fatalf("a file without current actions: dropped %v released %v, %v; want only the terminal corpse's keys released", dropped, released, err)
	}
	if u50, _ := savedMotionRaw(&doc.Objects[0], "U50", 4); doc.World.Cells[0].GroundActor != 0x100 || doc.World.Cells[1].AirActor != 0x100 || binary.LittleEndian.Uint32(u50) != 7 {
		t.Fatal("the helper edited the document it reads")
	}
	setAbsent := func(cells ...sav.DocumentCellData) {
		a := currentActionData{Version: 1}
		for _, c := range cells {
			a.AbsentCellRecords = append(a.AbsentCellRecords, currentAbsentCellCarrier{Cell: c.Cell, Anchor: currentCellCarrierAnchor(c, false)})
		}
		raw, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		if err := sav.SetNativeActions(&doc.State, raw); err != nil {
			t.Fatal(err)
		}
	}
	setAbsent(doc.World.Cells[2])
	dropped, released, err = cellRecordsAfterLoad(&doc)
	if err != nil || !reflect.DeepEqual(dropped, map[uint16]bool{0x0809: true}) || !reflect.DeepEqual(released, terminal) {
		t.Fatalf("a row its writer marked absent: dropped %v released %v, %v; want 0809 dropped", dropped, released, err)
	}
	doc.World.Cells[2].Sack ^= 0x5a
	dropped, _, err = cellRecordsAfterLoad(&doc)
	if err != nil || len(dropped) != 0 {
		t.Fatalf("a row edited after the anchor: dropped %v, %v; want it kept", dropped, err)
	}
}

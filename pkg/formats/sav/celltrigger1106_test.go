package sav

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestCellTriggers1106LiteralProjectionOrderAndFreshOffsets(t *testing.T) {
	for _, extended := range []bool{false, true} {
		source, off := exactFixture1103(true, false, 0)
		f, err := Open(source.Marshal())
		if err != nil {
			t.Fatal(err)
		}
		// Literal key + 52-byte payload. Poisoned identity/baseline/residue
		// bytes distinguish the trigger from adjacent six-byte windows.
		var row [54]byte
		for i := range row {
			row[i] = 0xa7
		}
		row[0], row[1] = 0xfe, 0xff
		copy(row[46:52], []byte{26, 0, 31, 47, 99, 123})
		table := []byte{2, 0}
		if extended {
			table = []byte{255, 255, 2, 0, 0, 0}
		}
		table = append(table, row[:]...)
		row[46] = 0
		table = append(table, row[:]...)
		// Edit the body without refreshing File.World. The reader must walk
		// the actual wire counts, not the cached empty table or old starts.
		at := off["cells"]
		body := append([]byte(nil), f.Body[:at]...)
		body = append(body, table...)
		f.Body = append(body, f.Body[at+2:]...)
		want := []CellTrigger{{Cell: 0xfffe, Bytes: [6]byte{26, 0, 31, 47, 99, 123}},
			{Cell: 0xfffe, Bytes: [6]byte{0, 0, 31, 47, 99, 123}}}
		got, present, err := f.CellTriggers()
		if err != nil || !present || !reflect.DeepEqual(got, want) {
			t.Fatalf("extended=%t: %+v %t %v", extended, got, present, err)
		}
		got[0].Bytes[0] = 255
		again, _, err := f.CellTriggers()
		if err != nil || !reflect.DeepEqual(again, want) {
			t.Fatal("projection aliases archive bytes")
		}
	}
}

func TestCellTriggers1106RejectsLateTruncationAndOversizedCount(t *testing.T) {
	for _, kind := range []string{"short last cell", "short trailer", "huge count"} {
		source, off := exactFixture1103(true, true, 0)
		f, err := Open(source.Marshal())
		if err != nil {
			t.Fatal(err)
		}
		switch kind {
		case "short last cell":
			f.Body = f.Body[:off["cellsData"]+53]
		case "short trailer":
			f.Body = f.Body[:len(f.Body)-10]
		case "huge count":
			binary.LittleEndian.PutUint32(f.Body[off["cells"]+2:], 0xffffffff)
		}
		if rows, present, err := f.CellTriggers(); err == nil || !present || rows != nil {
			t.Fatalf("%s returned partial data %+v %t %v", kind, rows, present, err)
		}
	}
	for _, world := range []bool{false, true} {
		source, _ := exactFixture1103(world, false, 0)
		f, err := Open(source.Marshal())
		if err != nil {
			t.Fatal(err)
		}
		if rows, present, err := f.CellTriggers(); err != nil || present != world || len(rows) != 0 {
			t.Fatalf("empty/absent %+v %t %v", rows, present, err)
		}
	}
}

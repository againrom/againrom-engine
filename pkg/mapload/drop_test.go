package mapload_test

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

// The record sizes and the two offsets are written out here rather than imported,
// so a test asserting where a coordinate is read from is not reading that offset
// out of the code it is testing.
const (
	testNodeSize    = 796
	testTriggerSize = 184
	testOpcodeAt    = 0x40
	testValuesAt    = 0x4c
	testDropOpcode  = 0x10002
)

// node is one 796-byte action record carrying an opcode and its first two stored
// values. Every other byte is left zero, which is what an unused parameter slot
// holds.
func dropNode(opcode uint32, v0, v1 uint32) []byte {
	rec := make([]byte, testNodeSize)
	binary.LittleEndian.PutUint32(rec[testOpcodeAt:], opcode)
	binary.LittleEndian.PutUint32(rec[testValuesAt:], v0)
	binary.LittleEndian.PutUint32(rec[testValuesAt+4:], v1)
	return rec
}

// payload assembles a trigger body in the format tier's own shape: the leading
// action count is decoded away, so the body begins at the first action record and
// carries the condition and trigger arrays behind it.
func payload(nCond, nTrg int, actions ...[]byte) (uint32, []byte) {
	var body []byte
	for _, a := range actions {
		body = append(body, a...)
	}
	body = append(body, u32(uint32(nCond))...)
	body = append(body, make([]byte, nCond*testNodeSize)...)
	body = append(body, u32(uint32(nTrg))...)
	body = append(body, make([]byte, nTrg*testTriggerSize)...)
	return uint32(len(actions)), body
}

func u32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

func dropMap(nCond, nTrg int, actions ...[]byte) *alm.Map {
	n, body := payload(nCond, nTrg, actions...)
	return &alm.Map{Triggers: alm.Triggers{EntryCount: n, Body: body}}
}

// The cell is the low byte of each of the first two stored values, in that
// order, and the drop node need not be first.
func TestDropCellsReadsTheDropNodes(t *testing.T) {
	m := dropMap(2, 3,
		dropNode(2, 99, 99),                    // a message action: not a drop
		dropNode(testDropOpcode, 17, 66),       // 10.alm's own cell
		dropNode(0x10003, 5, 5),                // the builder's other special case
		dropNode(testDropOpcode, 0x1211, 0x13), // the high halves are not the coordinate
	)
	got := mapload.DropCells(m)
	want := []mapload.Cell{{X: 17, Y: 66}, {X: 0x11, Y: 0x13}}
	if len(got) != len(want) {
		t.Fatalf("DropCells = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("cell %d = %v, want %v", i, got[i], want[i])
		}
	}
}

// A map whose script authorises none, and a map with no trigger record at all,
// answer alike — the caller tells them apart by asking the map.
func TestDropCellsWithNoDropNode(t *testing.T) {
	if got := mapload.DropCells(dropMap(0, 0, dropNode(2, 1, 1))); got != nil {
		t.Errorf("a script with no drop node yielded %v", got)
	}
	if got := mapload.DropCells(&alm.Map{}); got != nil {
		t.Errorf("a map with no trigger record yielded %v", got)
	}
	if got := mapload.DropCells(nil); got != nil {
		t.Errorf("a nil map yielded %v", got)
	}
}

// The framing must tile the payload EXACTLY. Each case below carries a real drop
// node, so a walk that read it anyway would return a cell and pass every other
// assertion in this file.
func TestDropCellsRefusesAPayloadTheFramingDoesNotTile(t *testing.T) {
	drop := dropNode(testDropOpcode, 17, 66)

	for _, tc := range []struct {
		name string
		mut  func(n uint32, body []byte) (uint32, []byte)
	}{
		{"one trailing byte", func(n uint32, b []byte) (uint32, []byte) {
			return n, append(b, 0)
		}},
		{"one byte short", func(n uint32, b []byte) (uint32, []byte) {
			return n, b[:len(b)-1]
		}},
		{"the action count overruns", func(n uint32, b []byte) (uint32, []byte) {
			return n + 1, b
		}},
		{"the condition count overruns", func(n uint32, b []byte) (uint32, []byte) {
			out := append([]byte(nil), b...)
			binary.LittleEndian.PutUint32(out[testNodeSize:], 0xffffffff)
			return n, out
		}},
		{"the trigger count overruns", func(n uint32, b []byte) (uint32, []byte) {
			out := append([]byte(nil), b...)
			binary.LittleEndian.PutUint32(out[testNodeSize+4:], 0xffffffff)
			return n, out
		}},
		{"the payload ends inside a count word", func(n uint32, b []byte) (uint32, []byte) {
			return n, b[:testNodeSize+2]
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, body := payload(0, 0, drop)
			n, body = tc.mut(n, body)
			m := &alm.Map{Triggers: alm.Triggers{EntryCount: n, Body: body}}
			if got := mapload.DropCells(m); got != nil {
				t.Errorf("a payload the framing does not tile yielded %v", got)
			}
		})
	}
}

// The reference shape: the framing tiles, so the same bytes that fail above pass
// here. Without this the case set above would be satisfied by a function that
// always answered nothing.
func TestDropCellsAcceptsTheTilingPayload(t *testing.T) {
	n, body := payload(0, 0, dropNode(testDropOpcode, 17, 66))
	m := &alm.Map{Triggers: alm.Triggers{EntryCount: n, Body: body}}
	got := mapload.DropCells(m)
	if len(got) != 1 || got[0] != (mapload.Cell{X: 17, Y: 66}) {
		t.Fatalf("DropCells = %v, want one cell (17,66)", got)
	}
}

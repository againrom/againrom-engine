package reg_test

import (
	"bytes"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
)

// writeFixture builds a small well-formed registry with one heap-resident
// value (A/Str) BEFORE the array under test in heap order and one (B/Tail)
// AFTER it, so a boundedness test can tell "moved" from "shifted".
func writeFixture() []byte {
	return synth.Reg(0x11, []synth.RegNode{
		{Name: "A", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Str", Kind: 0x00, Str: "hello"},
			{Name: "Num", Kind: 0x02, Int: 5},
		}},
		{Name: "B", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Arr", Kind: 0x06, Ints: []int32{1, 2, 3}},
			{Name: "Tail", Kind: 0x06, Ints: []int32{9, 9}},
		}},
		{Name: "C", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Val", Kind: 0x02, Int: 7},
		}},
	})
}

func TestSetIntChangesOnlyItsOwnDataWord(t *testing.T) {
	data := writeFixture()
	r, err := reg.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	out, err := reg.SetInt(data, r, "C", "Val", 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(data) {
		t.Fatalf("length changed: %d -> %d", len(data), len(out))
	}
	// C/Val is the last node built (index 7 breadth-first), so its record's
	// data word is the last four bytes of the whole stream. Nothing OUTSIDE
	// that word may change; a byte inside it may legitimately equal its old
	// value (e.g. the high three zero bytes of a small int32).
	valOff := 0x18 + 0x20*7
	for i := range data {
		inWord := i >= valOff+4 && i < valOff+8
		if !inWord && data[i] != out[i] {
			t.Fatalf("byte %d changed outside C/Val's own data word [%d,%d)", i, valOff+4, valOff+8)
		}
	}
	if bytes.Equal(data[valOff+4:valOff+8], out[valOff+4:valOff+8]) {
		t.Fatal("C/Val's own data word did not change at all")
	}

	got, err := reg.Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := got.GetInt("C", "Val"); !ok || v != 42 {
		t.Fatalf("C/Val = %d, %v, want 42, true", v, ok)
	}
	// Every other leaf reads back exactly as it did before.
	if s, ok := got.GetString("A", "Str"); !ok || s != "hello" {
		t.Fatalf("A/Str = %q, %v", s, ok)
	}
	if n, ok := got.GetInt("A", "Num"); !ok || n != 5 {
		t.Fatalf("A/Num = %d, %v", n, ok)
	}
	if a, ok := got.GetIntArray("B", "Arr"); !ok || !equalInts(a, []int32{1, 2, 3}) {
		t.Fatalf("B/Arr = %v, %v", a, ok)
	}
	if tl, ok := got.GetIntArray("B", "Tail"); !ok || !equalInts(tl, []int32{9, 9}) {
		t.Fatalf("B/Tail = %v, %v", tl, ok)
	}
}

func TestSetIntRejectsWrongTypeAndMissingPath(t *testing.T) {
	data := writeFixture()
	r, err := reg.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reg.SetInt(data, r, "A", "Str", 1); err == nil {
		t.Fatal("SetInt on a TypeString node must be refused")
	}
	if _, err := reg.SetInt(data, r, "A", "Missing", 1); err == nil {
		t.Fatal("SetInt on a missing key must be refused")
	}
	if _, err := reg.SetInt(data, r, "Missing", "Val", 1); err == nil {
		t.Fatal("SetInt on a missing section must be refused")
	}
}

func TestSetIntArrayShrinkingLeavesEarlierHeapByteIdentical(t *testing.T) {
	data := writeFixture()
	r, err := reg.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	// Locate B/Arr's own heap span (12 bytes for 3 int32s) directly from the
	// input, by finding the recomputed heap start and A/Str's own span.
	// A/Str = "hello\x00", 6 bytes, at heap offset 0. B/Arr's array follows
	// it directly per synth.Reg's own packing order, at heap offset 6.
	nodeCount := 8 // A,B,C,Str,Num,Arr,Tail,Val, breadth-first
	heapOrigin := 0x18 + 0x20*nodeCount
	heapStart := heapOrigin + 4
	arrHeapOff := 6 // len("hello")+1

	out, err := reg.SetIntArray(data, r, "B", "Arr", nil) // shrink to empty
	if err != nil {
		t.Fatal(err)
	}
	// Every byte before the target's own heap span, header and node table
	// included, is untouched EXCEPT the two position words this setter is
	// allowed to move: B/Arr's own size word and B/Tail's data (offset)
	// word. Everything else in that prefix, byte for byte, must agree.
	arrRecOff := -1
	tailRecOff := -1
	for i := 0; i < nodeCount; i++ {
		off := 0x18 + 0x20*i
		name := string(bytes.TrimRight(data[off+0x10:off+0x20], "\x00"))
		if name == "Arr" {
			arrRecOff = off
		}
		if name == "Tail" {
			tailRecOff = off
		}
	}
	if arrRecOff < 0 || tailRecOff < 0 {
		t.Fatal("fixture layout assumption broke: Arr or Tail not found by hand-walking the table")
	}
	prefixEnd := heapStart + arrHeapOff
	diff := 0
	for i := 0; i < prefixEnd; i++ {
		if i >= arrRecOff+8 && i < arrRecOff+12 { // Arr's own size word
			continue
		}
		if i >= tailRecOff+4 && i < tailRecOff+8 { // Tail's own data(offset) word
			continue
		}
		if i >= heapOrigin && i < heapStart { // the heap's own length word
			continue
		}
		if data[i] != out[i] {
			diff++
		}
	}
	if diff != 0 {
		t.Fatalf("%d bytes changed before the target's own heap span, outside the three position/length words a resize may move", diff)
	}

	got, err := reg.Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	if a, ok := got.GetIntArray("B", "Arr"); !ok || len(a) != 0 {
		t.Fatalf("B/Arr = %v, %v, want an empty array", a, ok)
	}
	if s, ok := got.GetString("A", "Str"); !ok || s != "hello" {
		t.Fatalf("A/Str = %q, %v", s, ok)
	}
	if tl, ok := got.GetIntArray("B", "Tail"); !ok || !equalInts(tl, []int32{9, 9}) {
		t.Fatalf("B/Tail = %v, %v, want [9 9] unchanged at its DECODED value", tl, ok)
	}
	if v, ok := got.GetInt("C", "Val"); !ok || v != 7 {
		t.Fatalf("C/Val = %d, %v", v, ok)
	}
	if got.NodeCount != r.NodeCount {
		t.Fatalf("node count changed: %d -> %d", r.NodeCount, got.NodeCount)
	}
}

func TestSetIntArrayGrowingShiftsOnlyLaterHeapNodes(t *testing.T) {
	data := writeFixture()
	r, err := reg.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	out, err := reg.SetIntArray(data, r, "B", "Arr", []int32{1, 2, 3, 4, 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(data)+2*4 { // two more int32s
		t.Fatalf("stream grew by %d bytes, want 8", len(out)-len(data))
	}
	got, err := reg.Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	if a, ok := got.GetIntArray("B", "Arr"); !ok || !equalInts(a, []int32{1, 2, 3, 4, 5}) {
		t.Fatalf("B/Arr = %v, %v", a, ok)
	}
	if tl, ok := got.GetIntArray("B", "Tail"); !ok || !equalInts(tl, []int32{9, 9}) {
		t.Fatalf("B/Tail = %v, %v, its own decoded value must survive the shift", tl, ok)
	}
	if s, ok := got.GetString("A", "Str"); !ok || s != "hello" {
		t.Fatalf("A/Str = %q, %v", s, ok)
	}
}

func TestSetIntArrayRejectsWrongType(t *testing.T) {
	data := writeFixture()
	r, err := reg.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reg.SetIntArray(data, r, "C", "Val", []int32{1}); err == nil {
		t.Fatal("SetIntArray on a TypeInt node must be refused")
	}
}

func TestSetIntArraySkipsDirectoryNodesEntirely(t *testing.T) {
	data := synth.Reg(0x11, []synth.RegNode{
		{Name: "Data", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Arr", Kind: 0x06, Ints: []int32{}},
		}},
		{Name: "Other", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Val", Kind: 0x02, Int: 5},
		}},
	})
	r, err := reg.Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	nodeCount := 4 // Data, Other, Arr, Val, breadth-first
	dataRecOff, otherRecOff, valRecOff := -1, -1, -1
	for i := 0; i < nodeCount; i++ {
		off := synth.RegNodeOffset(i)
		switch string(bytes.TrimRight(data[off+0x10:off+0x20], "\x00")) {
		case "Data":
			dataRecOff = off
		case "Other":
			otherRecOff = off
		case "Val":
			valRecOff = off
		}
	}
	if dataRecOff < 0 || otherRecOff < 0 || valRecOff < 0 {
		t.Fatal("fixture layout assumption broke: Data, Other or Val not found by hand-walking the table")
	}
	dataChildFirst := data[dataRecOff+4 : dataRecOff+8]
	otherChildFirst := data[otherRecOff+4 : otherRecOff+8]
	valData := data[valRecOff+4 : valRecOff+8]

	out, err := reg.SetIntArray(data, r, "Data", "Arr", []int32{1, 2, 3}) // grow from empty
	if err != nil {
		t.Fatal(err)
	}

	// Both directories' own first-child index words must be byte-identical:
	// a directory carries no heap span of its own for SetIntArray to move.
	if !bytes.Equal(dataChildFirst, out[dataRecOff+4:dataRecOff+8]) {
		t.Fatalf("Data's own first-child index moved: %v -> %v", dataChildFirst, out[dataRecOff+4:dataRecOff+8])
	}
	if !bytes.Equal(otherChildFirst, out[otherRecOff+4:otherRecOff+8]) {
		t.Fatalf("Other's own first-child index moved: %v -> %v", otherChildFirst, out[otherRecOff+4:otherRecOff+8])
	}
	// Other/Val is a real TypeInt leaf and must be equally untouched, so
	// this is not merely proving the type filter alone would have saved it.
	if !bytes.Equal(valData, out[valRecOff+4:valRecOff+8]) {
		t.Fatalf("Other/Val's own data word moved: %v -> %v", valData, out[valRecOff+4:valRecOff+8])
	}

	got, err := reg.Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	if a, ok := got.GetIntArray("Data", "Arr"); !ok || !equalInts(a, []int32{1, 2, 3}) {
		t.Fatalf("Data/Arr = %v, %v, want [1 2 3]", a, ok)
	}
	if v, ok := got.GetInt("Other", "Val"); !ok || v != 5 {
		t.Fatalf("Other/Val = %d, %v, want 5 unchanged", v, ok)
	}
}

func equalInts(a, b []int32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

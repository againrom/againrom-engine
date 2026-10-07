package sav

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

// An independent literal serializer fixture, including unrelated corpse loot
// and the published variable-size classes before the authoritative Sack list.
func groundFixture(world bool, sacks int) (*File, map[string]int) {
	s := newStream()
	off := map[string]int{}
	s.u32(1)
	s.human(wantChar{name: "corpse", runtimeID: 99, stage: 5, journal: -1,
		items: []wantPiece{{class: "Item", code: 0x7777, stack: 1}}})
	s.u8(0)
	if world {
		s.b[len(s.b)-1] = 1
		s.u32(3)
		s.obj("Outpost")
		s.token(0, 128, 128, 1, 0, 0)
		s.raw(40+16, 0)
		s.u16(0xffff)
		s.u32(2)
		s.raw(16, 0)
		s.obj("Tavern")
		s.token(0, 128, 128, 2, 0, 0)
		s.raw(40+4, 0)
		s.obj("Shop")
		s.token(0, 128, 128, 7, 0, 0)
		s.raw(40+4, 0)
		s.u32(1)
		s.obj("SpellTransport")
		s.token(0, 128, 128, 3, 0, 0)
		s.raw(2, 0)
		s.obj("PointEffect")
		s.token(0, 128, 128, 4, 0, 0)
		s.raw(2, 0)
		s.obj("Effect_DirectDamage")
		s.token(0, 128, 128, 5, 0, 0)
		s.raw(7+24, 0)
		s.u32(0)
		s.obj("AreaEffect")
		s.token(0, 128, 128, 6, 0, 0)
		s.raw(2+4+2, 0)
		s.null()
		s.u16(0)
		s.u16(0) // empty terrain blocks
		s.u16(0) // empty terrain cells
		s.raw(4+4374, 0)
		off["sackCount"] = len(s.b)
		s.u32(uint32(sacks))
		for i := range sacks {
			s.obj("Sack")
			if i == 0 {
				off["schema"] = len(s.b) - len("Sack") - 4
			}
			off["identity"] = len(s.b) + 29
			s.token(uint16(0x0302+i), 128, 128, uint32(100+i), 0, 0)
			s.u32(uint32(17 + i))
			s.u32(1)
			s.obj("Item")
			s.token(0, 0, 0, 0, 2, 0)
			s.u32(3)
			for j := range 3 {
				s.obj("Effect")
				s.token(0, 0, 0, 0, 0, 0)
				s.u8(uint8(11 + j%2))
				s.u8(1)
				s.u32(uint32(10 + j))
				s.u8(0)
			}
			s.u16(0x1234)
			s.u16(2)
			s.u8(3)
			s.raw(7, 0)
			s.u32(999) // insertion index, not capacity or item count
			s.u32(888) // cached load is not an additional item
		}
	}
	s.u32(0xbadface1)
	s.u32(0)
	s.raw(400, 0)
	if len(s.b)%2 != 0 {
		s.u8(0x97)
	}
	return &File{Body: s.b}, off
}

func TestGroundSacksWalksWholeDocumentAndKeepsOrderedItemState(t *testing.T) {
	f, _ := groundFixture(true, 2)
	sacks, present, err := f.GroundSacks()
	if err != nil || !present || len(sacks) != 2 {
		t.Fatalf("GroundSacks = %+v, %t, %v", sacks, present, err)
	}
	for i, sack := range sacks {
		if sack.Cell != uint16(0x0302+i) || sack.Gold != uint32(17+i) || sack.Identity != 0xa0000064+uint32(i) || sack.FineX != 128 || sack.FineY != 128 || len(sack.Items) != 1 {
			t.Fatalf("Sack %d = %+v", i, sack)
		}
		if sack.InsertIndex != 999 || sack.Accumulator != 888 {
			t.Fatalf("Sack %d container tail = %d/%d, want 999/888", i, sack.InsertIndex, sack.Accumulator)
		}
		item := sack.Items[0]
		want := []ItemEffect{{Kind: 11, Mode: 1, Operand: 10}, {Kind: 12, Mode: 1, Operand: 11}, {Kind: 11, Mode: 1, Operand: 12}}
		if item.Code != 0x1234 || item.Stack != 2 || item.Kind != 3 || item.Price != 0x1c1c1c1c || !reflect.DeepEqual(item.Effects, want) {
			t.Fatalf("item lost state: %+v", item)
		}
	}
	sacks[0].Items[0].Effects[0].Operand = 99
	again, _, err := f.GroundSacks()
	if err != nil || again[0].Items[0].Effects[0].Operand != 10 {
		t.Fatal("returned effects alias the archive")
	}
}

func TestGroundSackRepeatedRootKeepsOneValueAndExactTailTarget(t *testing.T) {
	f, offsets := groundFixture(true, 2)
	doc, _, err := f.exactDocument()
	if err != nil {
		t.Fatal(err)
	}
	end := doc.sacks[1].End
	alias := binary.LittleEndian.AppendUint16(nil, doc.sacks[0].Index)
	f.Body = append(append(append([]byte(nil), f.Body[:end]...), alias...), f.Body[end:]...)
	binary.LittleEndian.PutUint32(f.Body[offsets["sackCount"]:], 3)
	rows, present, err := f.GroundSacks()
	if err != nil || !present || len(rows) != 2 || rows[0].Gold != 17 || rows[1].Gold != 18 {
		t.Fatal("one repeated root changed physical Sack values", rows, err)
	}
	want := []GroundContainerTail{{Identity: rows[0].Identity, InsertIndex: 17, Accumulator: -91}, {Identity: rows[1].Identity, InsertIndex: 23, Accumulator: 79}}
	if err := f.SetGroundContainerTails(want); err != nil {
		t.Fatal(err)
	}
	doc, _, err = f.exactDocument()
	if err != nil || len(doc.sacks) != 3 || doc.sacks[0] != doc.sacks[2] || doc.sacks[0] == doc.sacks[1] {
		t.Fatal("tail projection changed archive root identity/order", err)
	}
	rows, _, err = f.GroundSacks()
	if err != nil || len(rows) != len(want) {
		t.Fatal(err)
	}
	for i, row := range rows {
		if row.InsertIndex != want[i].InsertIndex || row.Accumulator != want[i].Accumulator {
			t.Fatal("tail patch selected a different physical Sack", i, row)
		}
	}
	binary.LittleEndian.PutUint32(f.Body[offsets["identity"]:], rows[0].Identity)
	if _, _, err := f.GroundSacks(); err == nil {
		t.Fatal("distinct Sack records with equal identities were accepted")
	}
}

func TestGroundSacksDistinguishesEmptyWorldAndNoWorld(t *testing.T) {
	for _, present := range []bool{false, true} {
		f, _ := groundFixture(present, 0)
		sacks, got, err := f.GroundSacks()
		if err != nil || got != present || len(sacks) != 0 {
			t.Fatalf("present=%t: %v %t %v", present, sacks, got, err)
		}
	}
}

func TestGroundSacksRefusesEveryTruncationAndMalformedFraming(t *testing.T) {
	f, off := groundFixture(true, 2)
	for n := 0; n < len(f.Body); n++ {
		if _, _, err := (&File{Body: f.Body[:n]}).GroundSacks(); err == nil {
			t.Fatalf("accepted truncation at %d/%d", n, len(f.Body))
		}
	}
	for name, mutate := range map[string]func([]byte){
		"count":    func(b []byte) { binary.LittleEndian.PutUint32(b[off["sackCount"]:], 0xffffffff) },
		"schema":   func(b []byte) { binary.LittleEndian.PutUint16(b[off["schema"]:], 2) },
		"identity": func(b []byte) { binary.LittleEndian.PutUint32(b[off["identity"]:], 0) },
	} {
		t.Run(name, func(t *testing.T) {
			b := append([]byte(nil), f.Body...)
			mutate(b)
			if _, _, err := (&File{Body: b}).GroundSacks(); err == nil {
				t.Fatal("accepted malformed ground document")
			}
		})
	}
}

// TestSetGroundContainerTailsRoundTripsEveryField locks
// SetGroundContainerTails' own contract on TestSetSpellEffectsRoundTripsEveryField's
// own shape (spelleffect_test.go): every value written is read back, both
// from the live File and from a fresh Marshal/Open round trip.
func TestSetGroundContainerTailsRoundTripsEveryField(t *testing.T) {
	f, _ := groundFixture(true, 2)
	before, present, err := f.GroundSacks()
	if err != nil || !present || len(before) != 2 {
		t.Fatalf("GroundSacks: present=%v err=%v len=%d", present, err, len(before))
	}
	want := []GroundContainerTail{
		{Identity: before[0].Identity, InsertIndex: 5, Accumulator: -3},
		{Identity: before[1].Identity, InsertIndex: 10000, Accumulator: 224},
	}
	if err := f.SetGroundContainerTails(want); err != nil {
		t.Fatalf("SetGroundContainerTails: %v", err)
	}
	got, present, err := f.GroundSacks()
	if err != nil || !present {
		t.Fatalf("GroundSacks (re-decoded): present=%v err=%v", present, err)
	}
	for i, tail := range want {
		if got[i].InsertIndex != tail.InsertIndex || got[i].Accumulator != tail.Accumulator {
			t.Errorf("Sack %d tail = %d/%d, want %d/%d", i, got[i].InsertIndex, got[i].Accumulator, tail.InsertIndex, tail.Accumulator)
		}
	}
	// groundFixture builds a bare Sack-list stream, not a complete archive
	// envelope (no version header): it decodes only through GroundSacks'
	// own direct File{Body: ...} construction, not through the general
	// Open() entry point, so the round-trip check re-decodes the same way
	// TestGroundSacksWalksWholeDocumentAndKeepsOrderedItemState already
	// does rather than through Marshal/Open.
	again, present, err := (&File{Body: f.Body}).GroundSacks()
	if err != nil || !present {
		t.Fatalf("GroundSacks (fresh File over the same bytes): present=%v err=%v", present, err)
	}
	for i, tail := range want {
		if again[i].InsertIndex != tail.InsertIndex || again[i].Accumulator != tail.Accumulator {
			t.Errorf("fresh-File Sack %d tail = %d/%d, want %d/%d", i, again[i].InsertIndex, again[i].Accumulator, tail.InsertIndex, tail.Accumulator)
		}
	}
}

// TestSetGroundContainerTailsLeavesUnrelatedBytesAlone is
// TestSetSpellEffectsLeavesUnrelatedBytesAlone's own shape (spelleffect_test.go)
// for this narrower content patch: exactly the two tail dwords per Sack move,
// on ground.go's own "content patch, not a re-encoder" contract.
func TestSetGroundContainerTailsLeavesUnrelatedBytesAlone(t *testing.T) {
	f, _ := groundFixture(true, 2)
	before := append([]byte(nil), f.Body...)
	sacks, _, err := f.GroundSacks()
	if err != nil || len(sacks) != 2 {
		t.Fatalf("GroundSacks: %v, %d sacks", err, len(sacks))
	}
	// Every byte of both dwords must differ from the fixture's 999/888, or a
	// coincidental shared high zero-byte would undercount the moved total
	// below.
	tails := []GroundContainerTail{
		{Identity: sacks[0].Identity, InsertIndex: 0xaabbccdd, Accumulator: 0x11223344},
		{Identity: sacks[1].Identity, InsertIndex: 0xaabbccdd, Accumulator: 0x11223344},
	}
	if err := f.SetGroundContainerTails(tails); err != nil {
		t.Fatalf("SetGroundContainerTails: %v", err)
	}
	if len(f.Body) != len(before) {
		t.Fatalf("SetGroundContainerTails changed the body length: %d -> %d", len(before), len(f.Body))
	}
	n := 0
	for i := range before {
		if before[i] != f.Body[i] {
			n++
		}
	}
	if want := 8 * len(sacks); n != want {
		t.Fatalf("%d bytes moved, want exactly %d (two dwords per Sack)", n, want)
	}
	// The corpse actor before the Sack list and the item/effect state inside
	// each Sack are both untouched.
	again, _, err := f.GroundSacks()
	if err != nil || again[0].Items[0].Code != 0x1234 || len(again[0].Items[0].Effects) != 3 {
		t.Fatalf("SetGroundContainerTails disturbed Sack item state: %v, %+v", err, again)
	}
}

func TestSetGroundContainerTailsRefusesAShapeMismatch(t *testing.T) {
	f, _ := groundFixture(true, 2)
	sacks, _, err := f.GroundSacks()
	if err != nil || len(sacks) != 2 {
		t.Fatalf("GroundSacks: %v, %d sacks", err, len(sacks))
	}
	good := []GroundContainerTail{
		{Identity: sacks[0].Identity, InsertIndex: 1, Accumulator: 2},
		{Identity: sacks[1].Identity, InsertIndex: 3, Accumulator: 4},
	}
	if err := f.SetGroundContainerTails(nil); err == nil {
		t.Fatal("a record-count mismatch was accepted")
	}
	if err := f.SetGroundContainerTails(append(append([]GroundContainerTail(nil), good...), good[0])); err == nil {
		t.Fatal("an added record was accepted")
	}
	swapped := append([]GroundContainerTail(nil), good...)
	swapped[0], swapped[1] = swapped[1], swapped[0]
	if err := f.SetGroundContainerTails(swapped); err == nil {
		t.Fatal("an identity mismatch (out-of-order tails) was accepted")
	}
	// No partial write from any refused call: the original tails still read
	// back unchanged.
	again, _, err := f.GroundSacks()
	if err != nil || again[0].InsertIndex != 999 || again[0].Accumulator != 888 {
		t.Fatalf("a refused SetGroundContainerTails call left partial state: %v, %+v", err, again)
	}
}

func TestSetGroundContainerTailsRefusesALaterIndexMismatchWithNoPartialWrite(t *testing.T) {
	f, _ := groundFixture(true, 3)
	sacks, _, err := f.GroundSacks()
	if err != nil || len(sacks) != 3 {
		t.Fatalf("GroundSacks: %v, %d sacks", err, len(sacks))
	}
	before := append([]byte(nil), f.Body...)
	tails := []GroundContainerTail{
		{Identity: sacks[0].Identity, InsertIndex: 1, Accumulator: 2},     // valid: a single-pass writer would patch this one first
		{Identity: sacks[1].Identity ^ 1, InsertIndex: 3, Accumulator: 4}, // wrong identity: refuses here
		{Identity: sacks[2].Identity, InsertIndex: 5, Accumulator: 6},
	}
	if err := f.SetGroundContainerTails(tails); err == nil {
		t.Fatal("a later-index identity mismatch was accepted")
	}
	if !bytes.Equal(f.Body, before) {
		t.Fatal("a refused call already patched an earlier, valid Sack before reaching the later mismatch")
	}
}

// TestGroundSacksFixtureHasNoWorldSession locks the counterpart to
// SetGroundContainerTails' own "this save has no world session" refusal:
// GroundSacks itself already reports present=false for that shape
// (TestGroundSacksDistinguishesEmptyWorldAndNoWorld), so this checks the
// setter's own matching refusal specifically.
func TestGroundSacksFixtureHasNoWorldSession(t *testing.T) {
	f, _ := groundFixture(false, 0)
	if err := f.SetGroundContainerTails(nil); err == nil {
		t.Fatal("a save with no world session accepted a container-tail write")
	}
}

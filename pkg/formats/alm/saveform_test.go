package alm_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/alm"
)

// A synthetic map for the save form: players type-5 records, the given type-6
// records, lootN type-8 records, and optional type-9 and type-7 payloads.
func saveFormMap(players int, units [][]byte, lootN uint32, loot, t9, t7 []byte) []byte {
	p := baseSections(2, 2, uint32(players), 0, uint32(len(units)))
	meta := append([]byte(nil), p[0]...)
	copy(meta[0x2c:], le32(lootN))
	p[0] = meta
	var t5 []byte
	for i := 0; i < players; i++ {
		t5 = concat(t5, type5Record(0, "g"))
	}
	p[5] = t5
	p[6] = concat(units...)
	p[8] = loot
	if t9 != nil {
		p[9] = t9
	}
	if t7 != nil {
		p[7] = t7
	}
	return buildMap(p)
}

// saveUnit is a 70-byte unit with the owner slot (+0x14), the loot link (+0x18)
// and the script unit id (+0x40) set.
func saveUnit(x, y, owner, link uint32, id uint16) []byte {
	r := type6Record(x, y, 1, 0, 0, 0)
	copy(r[0x14:], le32(owner))
	copy(r[0x18:], le32(link))
	copy(r[0x40:], le16(id))
	return r
}

func saveLoot(owner, x, y uint32) []byte {
	return concat(lootHeadWide(1, owner, x, y, 0), lootElement(0x0205, 0, 0))
}

func saveFormDoc(t *testing.T, data []byte) (*alm.Document, []byte) {
	t.Helper()
	out, err := alm.SaveForm(data)
	if err != nil {
		t.Fatalf("SaveForm: %v", err)
	}
	d, err := alm.OpenDocument(out)
	if err != nil {
		t.Fatalf("the save form does not open: %v", err)
	}
	return d, out
}

func payloadOf(d *alm.Document, typeID uint32) []byte {
	for i, id := range d.RecordTypeIDs() {
		if id == typeID {
			return d.RecordPayload(i)
		}
	}
	return nil
}

func word(b []byte, off int) uint32 { return binary.LittleEndian.Uint32(b[off:]) }

// ALM-SAVE-256: a unit whose owner matches no player is saved with owner 1.
// The control is the unchanged stream, which keeps owners 0 and 3.
func TestSaveFormBindsAnUnmatchedOwnerToPlayerOne(t *testing.T) {
	in := saveFormMap(2, [][]byte{
		saveUnit(1, 1, 0, 0, 10), saveUnit(2, 2, 3, 0, 11), saveUnit(3, 3, 2, 0, 12), saveUnit(4, 4, 1, 0, 13),
	}, 0, nil, nil, nil)
	want := []uint32{1, 1, 2, 1}
	owners := func(units []byte) []uint32 {
		var got []uint32
		for i := range want {
			got = append(got, word(units, i*70+0x14))
		}
		return got
	}
	d0, err := alm.OpenDocument(in)
	if err != nil {
		t.Fatal(err)
	}
	if got := owners(payloadOf(d0, 6)); got[0] == want[0] || got[1] == want[1] {
		t.Fatalf("control: the unchanged stream already holds %v", got)
	}
	d, _ := saveFormDoc(t, in)
	for i, o := range owners(payloadOf(d, 6)) {
		if o != want[i] {
			t.Errorf("unit %d owner %d, want %d", i, o, want[i])
		}
	}

	// A map with no type-5 record has no first player: owners stay.
	none := saveFormMap(0, [][]byte{saveUnit(1, 1, 5, 0, 10)}, 0, nil, nil, nil)
	if out, err := alm.SaveForm(none); err != nil || !bytes.Equal(out, none) {
		t.Errorf("a map with no player changed (%v)", err)
	}
}

// ALM-SAVE-255 and ALM-SAVE-239 item 2: a unit's loot link past the loot count
// is written as 0, a loot record a unit links takes that unit's id as owner,
// and a loot record with an owner unit takes that unit's position.
func TestSaveFormLootFollowsItsUnit(t *testing.T) {
	units := [][]byte{
		saveUnit(0x1100, 0x2100, 1, 1, 7),  // links loot 1
		saveUnit(0x3300, 0x4400, 1, 5, 8),  // link past the count of 2
		saveUnit(0x5500, 0x6600, 1, 0, 9),  // no link
		saveUnit(0x7700, 0x8800, 1, 0, 21), // owns loot 2 by id
	}
	loot := concat(saveLoot(99, 1, 1), saveLoot(21, 2, 2))
	in := saveFormMap(2, units, 2, loot, nil, nil)
	d, out := saveFormDoc(t, in)
	u, l := payloadOf(d, 6), payloadOf(d, 8)
	if got := word(u, 70+0x18); got != 0 {
		t.Errorf("a link past the loot count is written as %d, want 0", got)
	}
	if got := word(u, 0x18); got != 1 {
		t.Errorf("a link in range is written as %d, want 1", got)
	}
	if word(l, 4) != 7 || word(l, 8) != 0x1100 || word(l, 12) != 0x2100 {
		t.Errorf("loot 1 owner %d at %#x,%#x, want unit 7 at 0x1100,0x2100", word(l, 4), word(l, 8), word(l, 12))
	}
	const second = 30
	if word(l, second+4) != 21 || word(l, second+8) != 0x7700 || word(l, second+12) != 0x8800 {
		t.Errorf("loot 2 owner and position are not unit 21's")
	}
	// Control: the input holds the authored positions.
	d0, _ := alm.OpenDocument(in)
	if l0 := payloadOf(d0, 8); word(l0, 8) != 1 || word(l0, second+8) != 2 {
		t.Fatal("control: input positions are not the authored ones")
	}
	if again, err := alm.SaveForm(out); err != nil || !bytes.Equal(again, out) {
		t.Errorf("the save form changes again on a second save (%v)", err)
	}
}

// ALM-SAVE-255 item 3: a loot record whose owner is no unit moves to unit 1,
// which links to it. A ground record is untouched.
func TestSaveFormMovesOrphanLootToUnitOne(t *testing.T) {
	units := [][]byte{saveUnit(0x1100, 0x2100, 1, 0, 1), saveUnit(0x3300, 0x4400, 1, 0, 2)}
	loot := concat(saveLoot(0, 0x0a00, 0x0b00), saveLoot(55, 3, 4))
	d, _ := saveFormDoc(t, saveFormMap(1, units, 2, loot, nil, nil))
	u, l := payloadOf(d, 6), payloadOf(d, 8)
	ground, orphan := l[:30], l[30:]
	if word(ground, 4) != 0 || word(ground, 8) != 0x0a00 || word(ground, 12) != 0x0b00 {
		t.Error("a ground record changed")
	}
	if word(orphan, 4) != 1 || word(orphan, 8) != 0x1100 || word(orphan, 12) != 0x2100 {
		t.Errorf("the orphan record is %x, want owner 1 at unit 1's position", orphan[:16])
	}
	if got := word(u, 0x18); got != 2 {
		t.Errorf("unit 1 links %d, want the loot index plus one (2)", got)
	}
}

// A file below version 990 is returned as it is.
func TestSaveFormLeavesOlderFilesAlone(t *testing.T) {
	in := saveFormMap(1, [][]byte{saveUnit(1, 1, 9, 0, 1)}, 0, nil, nil, nil)
	binary.LittleEndian.PutUint32(in[0x10:], 989)
	if out, err := alm.SaveForm(in); err != nil || !bytes.Equal(out, in) {
		t.Errorf("a version 989 file changed (%v)", err)
	}
}

// A file of version 991 takes the save form as 990 does: the code applies it
// from 990 up (the original's version rule above 990 is not stated).
func TestSaveFormAppliesFromVersion990Up(t *testing.T) {
	for _, v := range []uint32{990, 991} {
		in := saveFormMap(1, [][]byte{saveUnit(1, 1, 9, 0, 1)}, 0, nil, nil, nil)
		binary.LittleEndian.PutUint32(in[0x10:], v)
		out, err := alm.SaveForm(in)
		if err != nil || bytes.Equal(out, in) {
			t.Errorf("a version %d file was left alone (%v)", v, err)
		}
	}
}

// ALM-SAVE-252: the action nodes (796 bytes), condition nodes and triggers
// (184 bytes) ride the save at the offsets they were read from.
func TestSaveFormKeepsScriptNodesAndTriggersWhole(t *testing.T) {
	node := func(seed byte) []byte {
		b := make([]byte, 796)
		for i := range b {
			b[i] = seed + byte(i*7)
		}
		return b
	}
	trig := make([]byte, 184)
	for i := range trig {
		trig[i] = byte(i*3 + 1)
	}
	t7 := concat(le32(2), node(1), node(2), le32(1), node(3), le32(1), trig)
	in := saveFormMap(2, [][]byte{saveUnit(1, 1, 7, 0, 3)}, 0, nil, nil, t7)
	d, out := saveFormDoc(t, in)
	if got := payloadOf(d, 7); !bytes.Equal(got, t7) {
		t.Fatalf("the type-7 payload changed: %d bytes in, %d out", len(t7), len(got))
	}
	diff := 0
	for i := range in {
		if in[i] != out[i] {
			diff++
		}
	}
	if diff != 1 {
		t.Errorf("%d bytes differ, want only the unit owner byte", diff)
	}
	// Control: the comparison sees a changed node byte.
	bad := append([]byte(nil), t7...)
	bad[10] ^= 1
	if bytes.Equal(bad, payloadOf(d, 7)) {
		t.Error("control: the comparison cannot see a changed node byte")
	}
}

// type9Entry is the wire entry of ALM-SAVE-253: +0x00, +0x08, +0x0c, six bytes
// at +0x10, +0x16, the element count and 6-byte elements.
func type9Entry(a, b, w10, w16 uint32, elems int) []byte {
	e := concat(le32(0), le32(a), le32(b), le16(uint16(w10)), make([]byte, 4), le32(w16), le32(uint32(elems)))
	return concat(e, make([]byte, 6*elems))
}

// ALM-SAVE-253 and ALM-SAVE-254: the original drops a type-9 entry on load
// when its +8 and +0xc are non-zero, the word at +0x10 is 4 and a helper
// returns 0, and removes one on save when +8 is non-zero, +4 is zero and the
// 16-bit +0x16 is zero, leaving the entry that moved up out of the record size.
// The save form keeps every entry and the record size equals the bytes.
func TestSaveFormKeepsEveryType9EntryAndItsSize(t *testing.T) {
	entries := [][]byte{
		type9Entry(5, 6, 4, 1, 1), // the load-drop precondition
		type9Entry(7, 0, 0, 0, 0), // the save-removal precondition
		type9Entry(8, 9, 0, 3, 2), // the entry that moves up into its slot
	}
	t9 := concat(le32(3), entries[0], entries[1], entries[2])
	in := saveFormMap(2, nil, 0, nil, t9, nil)
	d, out := saveFormDoc(t, in)
	if got := payloadOf(d, 9); !bytes.Equal(got, t9) {
		t.Fatalf("the type-9 payload is %d bytes, want the %d read", len(got), len(t9))
	}
	var size uint32
	off := 20
	for _, id := range d.RecordTypeIDs() {
		sz := word(out, off+8)
		if id == 9 {
			size = sz
		}
		off += 20 + int(sz)
	}
	if int(size) != len(t9) {
		t.Errorf("the record size is %d, the bytes written %d", size, len(t9))
	}
	// Control: the original removes the second entry and its size sum skips
	// the third, which it still writes, so the size is below the bytes written.
	size = uint32(4 + len(entries[0]))
	written := 4 + len(entries[0]) + len(entries[2])
	if int(size) == written {
		t.Error("control: the original's size sum equals its bytes")
	}
}

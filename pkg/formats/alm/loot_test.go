package alm_test

// These tests are derived exclusively from docs/0103-ground-sacks/spec.md and
// the public API of package alm. No test reads any file from disk; every
// fixture is a synthetic byte stream assembled in test code from the
// documented byte layout (SC-2).

import (
	"bytes"
	"testing"

	"againrom/pkg/formats/alm"
)

// ---------------------------------------------------------------------------
// Fixture primitives for the type-8 section.

// lootMeta returns a payload map like baseSections (2x2, no other content),
// with the type-0 metadata's #type8 word (+0x2c) set to n.
func lootMeta(n uint32) map[uint32][]byte {
	p := baseSections(2, 2, 0, 0, 0)
	meta := append([]byte(nil), p[0]...)
	copy(meta[0x2c:], le32(n))
	p[0] = meta
	return p
}

// lootHeadWide builds a 20-byte type-8 record head (FormatVersion >= 989).
func lootHeadWide(n, owner, x, y, gold uint32) []byte {
	return concat(le32(n), le32(owner), le32(x), le32(y), le32(gold))
}

// lootHeadShort builds a 16-byte type-8 record head (FormatVersion < 989):
// the same first four words, with no fifth (gold) word.
func lootHeadShort(n, owner, x, y uint32) []byte {
	return concat(le32(n), le32(owner), le32(x), le32(y))
}

// lootElement builds one 10-byte type-8 element.
func lootElement(code uint32, field04 uint16, tileMarkerIndex uint32) []byte {
	return concat(le32(code), le16(field04), le32(tileMarkerIndex))
}

func TestLootDecodesAMinimalRecord(t *testing.T) {
	e0 := lootElement(0x0205, 3, 9)
	body := concat(lootHeadWide(1, 0, 0x0100, 0x0200, 12345), e0)
	p := lootMeta(1)
	p[8] = body

	m := openOK(t, buildMap(p))

	loot, err := m.Loot()
	if err != nil {
		t.Fatalf("Loot: unexpected error: %v", err)
	}
	if len(loot.Records) != 1 {
		t.Fatalf("len(Records) = %d, want 1", len(loot.Records))
	}
	r := loot.Records[0]
	if !r.Ground() {
		t.Errorf("Ground() = false, want true (owner word is 0)")
	}
	if r.Gold != 12345 {
		t.Errorf("Gold = %d, want 12345", r.Gold)
	}
	if r.CellX() != 1 || r.CellY() != 2 {
		t.Errorf("cell = (%d,%d), want (1,2)", r.CellX(), r.CellY())
	}
	if len(r.Elements) != 1 {
		t.Fatalf("len(Elements) = %d, want 1", len(r.Elements))
	}
	el := r.Elements[0]
	if el.ItemCode() != 0x0205 {
		t.Errorf("ItemCode() = 0x%04x, want 0x0205", el.ItemCode())
	}
	if el.Field04 != 3 {
		t.Errorf("Field04 = %d, want 3", el.Field04)
	}
	if el.TileMarkerIndex != 9 {
		t.Errorf("TileMarkerIndex = %d, want 9", el.TileMarkerIndex)
	}

	if !bytes.Equal(m.LootSection.Body, body) {
		t.Errorf("LootSection.Body changed after Loot(); Loot() must store nothing")
	}
	loot2, err := m.Loot()
	if err != nil || len(loot2.Records) != 1 {
		t.Errorf("second Loot() call disagreed with the first: %+v, err=%v", loot2, err)
	}

	// A stock record: owner != 0.
	stockBody := lootHeadWide(0, 7, 0x0300, 0x0400, 0)
	sp := lootMeta(1)
	sp[8] = stockBody
	sm := openOK(t, buildMap(sp))
	sl, err := sm.Loot()
	if err != nil {
		t.Fatalf("Loot (stock): unexpected error: %v", err)
	}
	if len(sl.Records) != 1 {
		t.Fatalf("len(Records) (stock) = %d, want 1", len(sl.Records))
	}
	if sl.Records[0].Ground() {
		t.Errorf("Ground() = true, want false (owner word is 7)")
	}
}

func TestLootHeadWidths(t *testing.T) {
	t.Run("wide head (FormatVersion >= 989) carries gold", func(t *testing.T) {
		e0 := lootElement(0x0100, 0, 0)
		body := concat(lootHeadWide(1, 0, 0x0100, 0x0200, 500), e0)
		p := lootMeta(1)
		p[8] = body

		// stdFileHeader() (used by buildMap) writes FormatVersion 990.
		m := openOK(t, buildMap(p))
		loot, err := m.Loot()
		if err != nil {
			t.Fatalf("Loot: unexpected error: %v", err)
		}
		if len(loot.Records) != 1 || loot.Records[0].Gold != 500 {
			t.Fatalf("Records = %+v, want one record with Gold=500", loot.Records)
		}
	})

	t.Run("short head (FormatVersion < 989) reads zero gold", func(t *testing.T) {
		e0 := lootElement(0x0100, 0, 0)
		body := concat(lootHeadShort(1, 0, 0x0100, 0x0200), e0)
		p := lootMeta(1)
		p[8] = body

		data := buildFile(fileHeader(almMagic, 20, 999, uint32(len(physicalOrder)), 988), wrapAll(p))
		m := openOK(t, data)
		loot, err := m.Loot()
		if err != nil {
			t.Fatalf("Loot: unexpected error: %v", err)
		}
		if len(loot.Records) != 1 {
			t.Fatalf("len(Records) = %d, want 1", len(loot.Records))
		}
		r := loot.Records[0]
		if r.Gold != 0 {
			t.Errorf("Gold = %d, want 0 (short head)", r.Gold)
		}
		if r.CellX() != 1 || r.CellY() != 2 {
			t.Errorf("cell = (%d,%d), want (1,2)", r.CellX(), r.CellY())
		}
		if len(r.Elements) != 1 || r.Elements[0].ItemCode() != 0x0100 {
			t.Errorf("Elements = %+v, want one element coded 0x0100", r.Elements)
		}
	})
}

func TestLootAbsentSection(t *testing.T) {
	p := lootMeta(0)
	// Omit typeId 8 from the physical order entirely, so the file carries no
	// type-8 record at all (as opposed to one with a zero-length payload).
	order := []uint32{0, 1, 2, 3, 5, 4, 9, 6, 7}
	data := buildPartial(p, order...)

	m := openOK(t, data)
	if m.Present(8) {
		t.Fatalf("Present(8) = true, want false (no type-8 record in the stream)")
	}
	loot, err := m.Loot()
	if err != nil {
		t.Fatalf("Loot: unexpected error: %v", err)
	}
	if len(loot.Records) != 0 {
		t.Errorf("len(Records) = %d, want 0 (absent section)", len(loot.Records))
	}
}

func TestLootEmptySection(t *testing.T) {
	p := lootMeta(0)
	// p[8] is already nil from baseSections: an empty, present, type-8 record.
	m := openOK(t, buildMap(p))
	if !m.Present(8) {
		t.Fatalf("Present(8) = false, want true (record present with a 0-byte payload)")
	}
	loot, err := m.Loot()
	if err != nil {
		t.Fatalf("Loot: unexpected error: %v", err)
	}
	if len(loot.Records) != 0 {
		t.Errorf("len(Records) = %d, want 0 (empty section)", len(loot.Records))
	}
}

func TestLootRecordWithZeroElements(t *testing.T) {
	body := lootHeadWide(0, 0, 0x0100, 0x0200, 500)
	p := lootMeta(1)
	p[8] = body

	m := openOK(t, buildMap(p))
	loot, err := m.Loot()
	if err != nil {
		t.Fatalf("Loot: unexpected error: %v", err)
	}
	if len(loot.Records) != 1 {
		t.Fatalf("len(Records) = %d, want 1", len(loot.Records))
	}
	if len(loot.Records[0].Elements) != 0 {
		t.Errorf("len(Elements) = %d, want 0", len(loot.Records[0].Elements))
	}
}

func TestLootRecordOverrunsPayload(t *testing.T) {
	// Declares one element but supplies none: the element read runs past the
	// (short) payload.
	body := lootHeadWide(1, 0, 0, 0, 0)
	p := lootMeta(1)
	p[8] = body

	m := openOK(t, buildMap(p))
	if _, err := m.Loot(); err == nil {
		t.Fatalf("Loot: expected an error for a record overrunning the payload, got nil")
	}
}

func TestLootHeadOverrunsPayload(t *testing.T) {
	// Declares one record but the payload is too short to hold even its head.
	p := lootMeta(1)
	p[8] = lootHeadWide(0, 0, 0, 0, 0)[:10] // 10 of 20 head bytes

	m := openOK(t, buildMap(p))
	if _, err := m.Loot(); err == nil {
		t.Fatalf("Loot: expected an error for a head overrunning the payload, got nil")
	}
}

func TestLootPayloadHasATrailingByte(t *testing.T) {
	body := concat(lootHeadWide(0, 0, 0, 0, 0), []byte{0xAB})
	p := lootMeta(1)
	p[8] = body

	m := openOK(t, buildMap(p))
	if _, err := m.Loot(); err == nil {
		t.Fatalf("Loot: expected an error for a trailing byte, got nil")
	}
}

func TestLootItemClassAndIndex(t *testing.T) {
	// Two elements in one record: a class-2 code beside a class-14 code, so
	// the index widening (bits 0..4 vs. bits 0..7) has somewhere to fail.
	e0 := lootElement(0x0205, 0, 0) // class=2 (bits 8..11), index=5 (bits 0..4)
	e1 := lootElement(0x0ef3, 0, 0) // class=14, index widens to bits 0..7 = 0xf3
	body := concat(lootHeadWide(2, 0, 0, 0, 0), e0, e1)
	p := lootMeta(1)
	p[8] = body

	m := openOK(t, buildMap(p))
	loot, err := m.Loot()
	if err != nil {
		t.Fatalf("Loot: unexpected error: %v", err)
	}
	if len(loot.Records) != 1 || len(loot.Records[0].Elements) != 2 {
		t.Fatalf("Records = %+v, want one record with two elements", loot.Records)
	}
	els := loot.Records[0].Elements

	if got := alm.ItemClass(els[0].ItemCode()); got != 2 {
		t.Errorf("ItemClass(0x%04x) = %d, want 2", els[0].ItemCode(), got)
	}
	if got := alm.ItemIndex(els[0].ItemCode()); got != 5 {
		t.Errorf("ItemIndex(0x%04x) = %d, want 5", els[0].ItemCode(), got)
	}
	if got := alm.ItemClass(els[1].ItemCode()); got != 14 {
		t.Errorf("ItemClass(0x%04x) = %d, want 14", els[1].ItemCode(), got)
	}
	if got := alm.ItemIndex(els[1].ItemCode()); got != 0xf3 {
		t.Errorf("ItemIndex(0x%04x) = 0x%02x, want 0xf3", els[1].ItemCode(), got)
	}

	// A class of 0 or 15 names no item: the split still returns the bits as
	// written, rather than a sentinel, for either.
	if got := alm.ItemClass(0x0000); got != 0 {
		t.Errorf("ItemClass(0x0000) = %d, want 0", got)
	}
	if got := alm.ItemClass(0x0f00); got != 15 {
		t.Errorf("ItemClass(0x0f00) = %d, want 15", got)
	}
}

// ---------------------------------------------------------------------------
// AC-3 — a map opened, Loot()-decoded and written back is byte-identical.
// ---------------------------------------------------------------------------

func TestLootRoundTripThroughDocument(t *testing.T) {
	e0 := lootElement(0x0205, 3, 9)
	e1 := lootElement(0x0ef3, 0, 0)
	body := concat(lootHeadWide(2, 0, 0x0100, 0x0200, 42), e0, e1)
	p := lootMeta(1)
	p[8] = body
	data := buildMap(p)

	doc, err := alm.OpenDocument(data)
	if err != nil {
		t.Fatalf("OpenDocument: unexpected error: %v", err)
	}
	mp, err := doc.Map()
	if err != nil {
		t.Fatalf("Map: unexpected error: %v", err)
	}
	if _, err := mp.Loot(); err != nil {
		t.Fatalf("Loot: unexpected error: %v", err)
	}

	out := doc.Write()
	if !bytes.Equal(out, data) {
		t.Errorf("Write() after Loot() differs from the input (%d vs %d bytes)", len(out), len(data))
	}
}

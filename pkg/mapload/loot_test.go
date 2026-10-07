// Package mapload_test exercises sacksFrom and FromALMWith's placement pass
// over the exported surface alone: a map fixture built as raw type-8 bytes,
// and the world's own Sacks().
//
// Nothing here reads a game install (SC-2): the type-8 payload is assembled
// by hand from the documented byte layout, the same grammar
// pkg/formats/alm/loot_test.go's fixtures use.
package mapload_test

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

func llLE32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

func llLE16(v uint16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, v)
	return b
}

func llJoin(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// llHead builds a 20-byte (wide) type-8 record head: element count, owner,
// X, Y and gold, each a raw fixed-point word at the file's own width.
func llHead(n, owner, x, y, gold uint32) []byte {
	return llJoin(llLE32(n), llLE32(owner), llLE32(x), llLE32(y), llLE32(gold))
}

// llElement builds one 10-byte type-8 element.
func llElement(code uint32, field04 uint16, tileMarkerIndex uint32) []byte {
	return llJoin(llLE32(code), llLE16(field04), llLE32(tileMarkerIndex))
}

// llCell is a record's axis word for cell (x, y): shifted left by 8, the
// inverse of LootRecord.CellX/CellY (spec.md "The cell").
func llCell(v int32) uint32 { return uint32(v) << 8 }

func TestFromALMPlacesExactlyTheGroundRecordsOfAMapWithBothKinds(t *testing.T) {
	body := llJoin(
		// record 0: ground, cell (2,3), gold 100, two elements in order.
		llHead(2, 0, llCell(2), llCell(3), 100),
		llElement(0x0205, 3, 0),
		llElement(0x0102, 0, 0),
		// record 1: stock (owner != 0) — places nothing, whatever its cell.
		llHead(0, 7, llCell(4), llCell(4), 999),
		// record 2: ground, cell (7,1), no gold, no elements.
		llHead(0, 0, llCell(7), llCell(1), 0),
	)
	m := &alm.Map{
		Width: 10, Height: 10, FormatVersion: 990,
		Meta:        alm.Meta{Word2C: 3},
		LootSection: alm.LootSection{Body: body},
	}

	w := mapload.FromALM(m)
	if w == nil {
		t.Fatal("FromALM returned a nil world")
	}
	got := w.Sacks()
	// Ascending by (Y, X): (7,1) before (2,3).
	if len(got) != 2 {
		t.Fatalf("Sacks() = %+v, want exactly the two ground records' cells", got)
	}
	if got[0].X != 7 || got[0].Y != 1 || got[0].Gold != 0 || len(got[0].Items) != 0 {
		t.Errorf("sack 0 is %+v, want (7,1) with no gold and no items", got[0])
	}
	if got[1].X != 2 || got[1].Y != 3 || got[1].Gold != 100 {
		t.Errorf("sack 1 is %+v, want (2,3) with 100 gold", got[1])
	}
	if len(got[1].Items) != 2 || got[1].Items[0] != 0x0205 || got[1].Items[1] != 0x0102 {
		t.Errorf("sack 1's items are %v, want [0x0205 0x0102] — element order", got[1].Items)
	}
	for _, s := range got {
		if s.X == 4 && s.Y == 4 {
			t.Errorf("the stock record placed a sack at its cell: %+v", got)
		}
	}
}

func TestAStockRecordReachesItsUnitsContainerInElementOrder(t *testing.T) {
	body := llJoin(
		// record 0: ground, cell (2,3), gold 50, one element.
		llHead(1, 0, llCell(2), llCell(3), 50),
		llElement(0x0104, 0, 0),
		// record 1: stock, owner 7 — matches unit 1's UnitID below — cell
		// (4,4), two elements in order. Cell and gold are never read on this
		// arm.
		llHead(2, 7, llCell(4), llCell(4), 999),
		llElement(0x0205, 0, 0),
		llElement(0x0309, 0, 0),
	)
	m := &alm.Map{
		Width: 10, Height: 10, FormatVersion: 990,
		Meta:        alm.Meta{Word2C: 2},
		LootSection: alm.LootSection{Body: body},
		Units: []alm.Unit{
			{X: 0x0100, Y: 0x0100, UnitID: 1},
			{X: 0x0180, Y: 0x0180, UnitID: 7},
		},
	}

	w := mapload.FromALM(m)
	if w == nil {
		t.Fatal("FromALM returned a nil world")
	}

	got, ok := w.Carried(1)
	if !ok {
		t.Fatal("entity 1 (the named unit) is not held by the world")
	}
	if len(got) != 2 || got[0] != 0x0205 || got[1] != 0x0309 {
		t.Errorf("Carried(1) = %v, want [0x0205 0x0309] — element order", got)
	}

	if got, ok := w.Carried(0); !ok || len(got) != 0 {
		t.Errorf("Carried(0) = %v, ok=%v, want an empty container — the stock record named entity 1, not 0", got, ok)
	}

	sacks := w.Sacks()
	if len(sacks) != 1 || sacks[0].X != 2 || sacks[0].Y != 3 || sacks[0].Gold != 50 {
		t.Errorf("Sacks() = %+v, want exactly the ground record's sack at (2,3) with 50 gold", sacks)
	}
	for _, s := range sacks {
		if s.X == 4 && s.Y == 4 {
			t.Errorf("the stock record placed a sack at its own cell: %+v", sacks)
		}
	}
}

func TestAStockRecordMatchingNoUnitContributesNothing(t *testing.T) {
	body := llHead(1, 99, llCell(4), llCell(4), 0)
	body = llJoin(body, llElement(0x0104, 0, 0))
	m := &alm.Map{
		Width: 10, Height: 10, FormatVersion: 990,
		Meta:        alm.Meta{Word2C: 1},
		LootSection: alm.LootSection{Body: body},
		Units:       []alm.Unit{{X: 0x0100, Y: 0x0100, UnitID: 5}},
	}

	w := mapload.FromALM(m)
	if w == nil {
		t.Fatal("a map whose stock record names no unit failed to load a world")
	}
	if got, ok := w.Carried(0); !ok || len(got) != 0 {
		t.Errorf("Carried(0) = %v, ok=%v, want an empty container — the record's owner (99) names no unit", got, ok)
	}
	if got := w.Sacks(); len(got) != 0 {
		t.Errorf("Sacks() = %+v, want none", got)
	}
}

func TestAStockRecordMatchingTwoUnitsContributesNothing(t *testing.T) {
	body := llHead(1, 3, llCell(4), llCell(4), 0)
	body = llJoin(body, llElement(0x0104, 0, 0))
	m := &alm.Map{
		Width: 10, Height: 10, FormatVersion: 990,
		Meta:        alm.Meta{Word2C: 1},
		LootSection: alm.LootSection{Body: body},
		Units: []alm.Unit{
			{X: 0x0100, Y: 0x0100, UnitID: 3},
			{X: 0x0180, Y: 0x0180, UnitID: 3},
		},
	}

	w := mapload.FromALM(m)
	if w == nil {
		t.Fatal("a map whose stock record names two units failed to load a world")
	}
	if got, ok := w.Carried(0); !ok || len(got) != 0 {
		t.Errorf("Carried(0) = %v, ok=%v, want an empty container — the owner (3) is ambiguous", got, ok)
	}
	if got, ok := w.Carried(1); !ok || len(got) != 0 {
		t.Errorf("Carried(1) = %v, ok=%v, want an empty container — the owner (3) is ambiguous", got, ok)
	}
	if got := w.Sacks(); len(got) != 0 {
		t.Errorf("Sacks() = %+v, want none", got)
	}
}

func TestARecordOutsideTheMapIsDroppedAndTheLoadSucceeds(t *testing.T) {
	body := llJoin(
		// record 0: ground, cell (5,5) — inside the 10x10 map.
		llHead(0, 0, llCell(5), llCell(5), 1),
		// record 1: ground, cell (50,50) — outside it.
		llHead(0, 0, llCell(50), llCell(50), 2),
	)
	m := &alm.Map{
		Width: 10, Height: 10, FormatVersion: 990,
		Meta:        alm.Meta{Word2C: 2},
		LootSection: alm.LootSection{Body: body},
		Units:       []alm.Unit{{X: 0x0180, Y: 0x0180}},
	}

	w := mapload.FromALM(m)
	if w == nil {
		t.Fatal("a map with one out-of-bounds loot record failed to load a world")
	}
	if got := w.Entities(); len(got) != 1 {
		t.Errorf("the map's own unit did not load: %d entities, want 1", len(got))
	}
	got := w.Sacks()
	if len(got) != 1 || got[0].X != 5 || got[0].Y != 5 {
		t.Errorf("Sacks() = %+v, want exactly one sack at (5,5) — the out-of-bounds record dropped", got)
	}
}

func TestALootSectionThisBuildCannotDecodeYieldsNoSacksAndNoLoadFailure(t *testing.T) {
	body := llHead(5, 0, llCell(1), llCell(1), 0)
	m := &alm.Map{
		Width: 10, Height: 10, FormatVersion: 990,
		Meta:        alm.Meta{Word2C: 1},
		LootSection: alm.LootSection{Body: body},
		Units:       []alm.Unit{{X: 0x0180, Y: 0x0180}},
	}
	if _, err := m.Loot(); err == nil {
		t.Fatal("the fixture's Loot() unexpectedly succeeded; it no longer exercises FR-23")
	}

	w := mapload.FromALM(m)
	if w == nil {
		t.Fatal("a map whose loot section cannot be decoded failed to load a world at all")
	}
	if got := w.Entities(); len(got) != 1 {
		t.Errorf("the map's own unit did not load: %d entities, want 1", len(got))
	}
	if got := w.Sacks(); len(got) != 0 {
		t.Errorf("Sacks() = %+v, want none — an undecodable section places nothing", got)
	}
}

func TestAMapWithNoLootSectionLoadsWithNoSacks(t *testing.T) {
	m := &alm.Map{Width: 10, Height: 10, FormatVersion: 990}
	w := mapload.FromALM(m)
	if w == nil {
		t.Fatal("a map with no type-8 record failed to load a world")
	}
	if got := w.Sacks(); len(got) != 0 {
		t.Errorf("Sacks() = %+v, want none", got)
	}
}

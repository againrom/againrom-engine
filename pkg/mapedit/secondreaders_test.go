package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/words"
)

// Every field a second-game unit, structure and sack lists is named by a word
// both languages hold, none by its offset alone, and a field no game reader
// was found for says so; the sack item word states its two places and the
// English gate.
func TestSecondFieldsNameTheirReaders(t *testing.T) {
	en, ru := words.For(""), words.For("russian")
	e, _ := mapedit.New(secondPlacedMap())
	label := func(k mapedit.Kind, i int, key string) string {
		t.Helper()
		fs, err := e.Fields(k, i)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range fs {
			if f.Key == key {
				return f.Label
			}
		}
		t.Fatalf("%v %d lists no %s", k, i, key)
		return ""
	}
	for _, c := range []struct {
		kind mapedit.Kind
		i    int
	}{{mapedit.KindUnit, 0}, {mapedit.KindStructure, 0}, {mapedit.KindLoot, 0}, {mapedit.KindLoot, 1}} {
		fs, err := e.Fields(c.kind, c.i)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range fs {
			for _, book := range []words.Book{en, ru} {
				if w := book.Text(f.Label); w == f.Label || strings.HasPrefix(w, "Field 0x") || strings.HasPrefix(w, "Поле 0x") || strings.Contains(w, "field 0x") || strings.Contains(w, "поле 0x") {
					t.Errorf("%v %d field %s is worded %q", c.kind, c.i, f.Key, w)
				}
			}
		}
	}
	unread := func(what, key string) {
		t.Helper()
		if !strings.Contains(en.Text(key), "no game reader found") || !strings.Contains(ru.Text(key), "чтение игрой") {
			t.Errorf("%s: %q / %q lacks the no-reader mark", what, en.Text(key), ru.Text(key))
		}
	}
	unread("unit 0x0a", label(mapedit.KindUnit, 0, "field0a"))
	unread("unit 0x20", label(mapedit.KindUnit, 0, "field20"))
	unread("shop word 0x0c", label(mapedit.KindStructure, 0, "field0c"))
	unread("unit sack 0x10", label(mapedit.KindLoot, 0, "field10"))
	unread("ground sack item word", label(mapedit.KindLoot, 1, "field04_1"))
	if w := en.Text(label(mapedit.KindLoot, 1, "field10")); strings.Contains(w, "no game reader") || !strings.Contains(w, "placed") {
		t.Errorf("ground sack 0x10 = %q", w)
	}
	hand := en.Text(label(mapedit.KindLoot, 0, "field04_1"))
	for _, want := range []string{"nonzero", "hand", "Medium", "bag", "English", "+0x74 = 0"} {
		if !strings.Contains(hand, want) {
			t.Errorf("unit sack item word %q lacks %q", hand, want)
		}
	}
	if err := e.SetField(mapedit.KindStructure, 0, "kind", 5); err != nil {
		t.Fatal(err)
	}
	if w := en.Text(label(mapedit.KindStructure, 0, "field0c")); !strings.Contains(w, "0 clears") || strings.Contains(w, "no game reader found") {
		t.Errorf("building word 0x0c = %q", w)
	}
}

// A passability stroke on a second-game map writes the type-1 flag and clears
// the type-3 byte of each cell it covers, so the cell closes move kind 1
// alone or none, and one Undo restores both planes.
func TestSecondPassabilityStrokeSetsTheFlagAndClearsTheObjectByte(t *testing.T) {
	raw := secondPlacedMap()
	tiles, objects := secondRecordOffset(t, raw, alm.TypeTiles), secondRecordOffset(t, raw, alm.TypeOverlay)
	at := func(x, y int) int { return y*32 + x }
	raw[objects+at(12, 12)] = 7
	raw[objects+at(16, 12)] = 8
	binary.LittleEndian.PutUint16(raw[tiles+2*at(16, 12):], alm.TileImpassable)
	raw[objects+at(14, 12)] = 9 // no stroke covers this cell
	e, _ := secondOpen(t, raw)
	for _, c := range []struct {
		value, x int
		flag     uint16
	}{{1, 12, alm.TileImpassable}, {0, 16, 0}} {
		before := e.Bytes()
		if changed, err := e.Paint(mapedit.Brush{Plane: mapedit.PlaneImpassable, Value: c.value}, []mapedit.Cell{{X: c.x, Y: 12}}, nil); err != nil || !changed {
			t.Fatalf("value %d stroke = %v, %v", c.value, changed, err)
		}
		got := e.Bytes()
		if w := binary.LittleEndian.Uint16(got[tiles+2*at(c.x, 12):]); w&alm.TileImpassable != c.flag {
			t.Fatalf("value %d: tile word %#x", c.value, w)
		}
		if got[objects+at(c.x, 12)] != 0 || got[objects+at(14, 12)] != 9 {
			t.Fatalf("value %d: type-3 bytes %d, %d", c.value, got[objects+at(c.x, 12)], got[objects+at(14, 12)])
		}
		if !e.Undo() || !bytes.Equal(e.Bytes(), before) {
			t.Fatalf("value %d: Undo does not restore both planes", c.value)
		}
		e.Redo()
	}
}

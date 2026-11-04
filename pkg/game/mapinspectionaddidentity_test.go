package game

import (
	"bytes"
	"encoding/hex"
	"image"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
)

// addPairEditor is a new 32x32 map over a table holding the class keys the
// editor's release tests place a new unit by: creature class 69 (one Units
// row, secondary key 1), creature class 64 (four Units rows, secondary keys
// 1..4) and person type 10 (a Humans row with no server id).
func addPairEditor(t *testing.T) *mapInspectionEdits {
	t.Helper()
	table := &mapload.Table{
		Units: dbCollection{{}, unitsRow("Ghost", map[int]int32{0x1d: 69, 0x1e: 1}),
			unitsRow("Goblin_Pike.1", map[int]int32{0x1d: 64, 0x1e: 1}), unitsRow("Goblin_Pike.2", map[int]int32{0x1d: 64, 0x1e: 2}),
			unitsRow("Goblin_Pike.3", map[int]int32{0x1d: 64, 0x1e: 3}), unitsRow("Goblin_Pike.4", map[int]int32{0x1d: 64, 0x1e: 4})},
		Humans: dbCollection{{}, {name: "Peasant", params: figureRow(10, 0, 0)}},
	}
	x := &MapInspector{tiles: &terrain.Tileset{}, table: table}
	e := x.Editor()
	if err := e.Open(newMapSource(32, 32)); err != nil {
		t.Fatal(err)
	}
	return e.Document().Edits.(*mapInspectionEdits)
}

// The release tests that placed a new unit by class key and tier place it
// by identity. For each (class key, tier) pair they used on the shipped
// installs, AddUnitIdentity with that pair's identity writes the unit
// record the class-and-tier placement wrote, pinned here.
func TestEditorAddUnitIdentityWritesTheClassAndTierRecord(t *testing.T) {
	// The 70-byte record at cell 22,22: its position words, the class and
	// secondary keys, owner slot 2, the unset override words and unit id 1.
	record := func(keys string) []byte {
		b, err := hex.DecodeString("8016000080160000" + keys + strings.Repeat("00", 8) + "02000000" + strings.Repeat("00", 8) +
			strings.Repeat("ff", 8) + strings.Repeat("00", 24) + "010000000000")
		if err != nil || len(b) != 70 {
			t.Fatalf("pinned record %q: %v, %d bytes", keys, err, len(b))
		}
		return b
	}
	for _, c := range []struct {
		name     string
		identity mapload.Identity
		keys     string
	}{
		{"creature class 69, tier 0", mapload.Identity{Kind: mapload.UnitCreature, Row: 1}, "45000100"},
		{"person type 10, tier 0", mapload.Identity{Kind: mapload.UnitPerson, Row: 1}, "0a000000"},
		{"creature class 64, tier 3", mapload.Identity{Kind: mapload.UnitCreature, Row: 4}, "40000300"},
	} {
		s := addPairEditor(t)
		if _, err := s.AddUnitIdentity(image.Pt(22, 22), identityChoiceValue(c.identity)); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		m, _ := s.model.Map()
		got, _ := s.model.UnitRecord(len(m.AuthoredUnits) - 1)
		if want := record(c.keys); !bytes.Equal(got, want) {
			t.Errorf("%s: AddUnitIdentity wrote\n% x\nwant\n% x", c.name, got, want)
		}
	}
}

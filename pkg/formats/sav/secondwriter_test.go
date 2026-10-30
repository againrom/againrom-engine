package sav

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

// secondCanonicalStore is a complete second-game state store in the form the
// codec writes one: the first game's directories with an empty shortcut
// array (DIV-4154), Fog and Projectiles only in a mission's store.
func secondCanonicalStore(t *testing.T, world bool) []byte {
	t.Helper()
	var dirs []literalStateDirectory
	for _, d := range literalWorldDirectories(266, 7) {
		if !world && (d.name == "Fog" || d.name == "Projectiles" || strings.HasPrefix(d.name, "Prj")) {
			continue
		}
		for i := range d.leaves {
			if d.name == "SpellBook" && d.leaves[i].name == "Shortcuts" {
				d.leaves[i].data = nil
			}
		}
		dirs = append(dirs, d)
	}
	raw := literalStateStore(dirs)
	var out []byte
	if world {
		state, err := parseWorldStateFor(raw, SecondGame)
		if err != nil {
			t.Fatalf("world store: %v", err)
		}
		out, err = serializeWorldStateFor(state, SecondGame)
		if err != nil {
			t.Fatalf("world store: %v", err)
		}
	} else {
		state, err := parseCityStateFor(raw, SecondGame)
		if err != nil {
			t.Fatalf("town store: %v", err)
		}
		out, err = serializeCityStateFor(state, SecondGame)
		if err != nil {
			t.Fatalf("town store: %v", err)
		}
	}
	return out
}

// secondWriterFixtures are the walk's second-game town and mission files
// with complete stores, mission cells whose bytes past +0x2b are not the
// first game's, and session values outside the first game's widths.
func secondWriterFixtures(t *testing.T) map[string][]byte {
	t.Helper()
	town := secondTownFixture()
	town.tail = append(secondCanonicalStore(t, false), secondTownTail().bytes()...)
	mission := secondMissionFixture()
	mission.tail = append(secondCanonicalStore(t, true), secondTownTail().bytes()...)
	cell := make([]byte, SecondGame.cellLen)
	binary.LittleEndian.PutUint16(cell, 0x0203)
	cell[2], cell[3], cell[4] = 5, 6, 1
	binary.LittleEndian.PutUint32(cell[2+12:], 0xa0000042)
	for i := 2 + 0x2c; i < len(cell); i++ {
		cell[i] = byte(i)
	}
	other := append([]byte(nil), cell...)
	binary.LittleEndian.PutUint16(other, 0x0204)
	mission.cells = [][]byte{cell, other}
	mission.session = func(b []byte) {
		le := binary.LittleEndian
		le.PutUint32(b[4*7:], 0xfffffffb)
		le.PutUint32(b[4*901:], 3)
		le.PutUint32(b[4*999:], 141)
		b[4000+17] = 1
		b[5000], b[5048], b[5448] = 0xaa, 0xbb, 0xcc
		b[5456+3*70+4] = 1
		b[5456+3*70+60] = 2
		b[5456+65*70+1] = 3
		b[10356], b[10357] = 1, 1
		le.PutUint32(b[10358:], 9)
		le.PutUint32(b[10362:], 2)
		le.PutUint32(b[10366:], 4)
		le.PutUint32(b[10370:], 5)
	}
	return map[string][]byte{"town": town.file(), "mission": mission.file()}
}

// A second-game town and mission document decode and re-encode byte for
// byte: the walk's inverse writes the second game's programmes, cells,
// session, extra list and tail in the Bsg& envelope.
func TestSecondGameDocumentsReencodeByteForByte(t *testing.T) {
	for name, file := range secondWriterFixtures(t) {
		data, err := DecodeDocumentData(file)
		if err != nil {
			t.Fatalf("%s: decode: %v", name, err)
		}
		if data.Game != "rom2" || data.SecondTail == nil {
			t.Fatalf("%s: game %q tail %v", name, data.Game, data.SecondTail != nil)
		}
		if name == "mission" {
			w := data.World
			if w == nil || len(w.CellRest) != 2*16 || len(w.SessionResults) != 900 || len(w.SessionDiplomacy) != 2400 {
				t.Fatalf("mission world widths: %+v", w != nil)
			}
			if w.Cells[0].Operation != 0 || w.CellRest[0] != 2+0x2c || w.SessionResults[801] != 3 || w.SessionResults[899] != 141 {
				t.Fatalf("mission cell tail %d rest %d results %d %d", w.Cells[0].Operation, w.CellRest[0], w.SessionResults[801], w.SessionResults[899])
			}
			if w.Session.Results[7] != -5 || w.Session.Diplomacy[3][4] != 1 || w.SessionDiplomacy[3*20+10] != 2 || w.Session.Lost != 5 {
				t.Fatalf("mission session fields moved")
			}
		}
		out, err := EncodeDocumentData(data)
		if err != nil {
			t.Fatalf("%s: encode: %v", name, err)
		}
		if !bytes.Equal(out, file) {
			t.Fatalf("%s: re-encoded %d bytes differ from the %d-byte file", name, len(out), len(file))
		}
		clone, err := CloneDocumentData(data)
		if err != nil {
			t.Fatalf("%s: clone: %v", name, err)
		}
		if again, err := EncodeDocumentData(clone); err != nil || !bytes.Equal(again, file) {
			t.Fatalf("%s: the clone does not re-encode the file: %v", name, err)
		}
	}
}

// The second game's document refuses what its programmes do not hold, and
// the first game's refuses the second game's parts.
func TestSecondGameDocumentRefusesForeignParts(t *testing.T) {
	file := secondWriterFixtures(t)["mission"]
	cases := map[string]func(*DocumentData){
		"first-game trigger tail": func(d *DocumentData) { d.World.Cells[0].Operation = 1 },
		"short session results":   func(d *DocumentData) { d.World.SessionResults = d.World.SessionResults[1:] },
		"first-game campaign":     func(d *DocumentData) { d.Campaign.Scalars[0] = 1 },
		"missing tail":            func(d *DocumentData) { d.SecondTail = nil },
		"unknown game":            func(d *DocumentData) { d.Game = "rom3" },
		"first-game profile":      func(d *DocumentData) { d.Game = "" },
	}
	for name, edit := range cases {
		data, err := DecodeDocumentData(file)
		if err != nil {
			t.Fatal(err)
		}
		edit(&data)
		if _, err := EncodeDocumentData(data); err == nil {
			t.Fatalf("%s: encoded", name)
		}
	}
}

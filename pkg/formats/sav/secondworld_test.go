package sav

import (
	"encoding/binary"
	"strings"
	"testing"
)

// secondMissionFixture is a mission-like second-game file: the town fixture
// with a world half and an in-battle state store.
func secondMissionFixture() walkFixture {
	fx := secondTownFixture()
	fx.town = false
	fx.tail = append(secondStore(1), secondTownTail().bytes()...)
	return fx
}

// The second game's session block is read by position (R2-SESSION-019):
// 1,000 result slots, the latches at 4000, the raw regions at 5000 and
// 5048, a 70-wide matrix at 5456 and the counters at 10362 and 10370. The
// hundred registers take the first slots; a nonzero slot past them is
// counted, and a stance outside the 50x50 relations is refused (DIV-4170).
func TestTheSecondGameSessionReadsByPosition(t *testing.T) {
	le := binary.LittleEndian
	fx := secondMissionFixture()
	fx.session = func(b []byte) {
		le.PutUint32(b[4*7:], uint32(0xfffffffb))
		le.PutUint32(b[4*901:], 3)
		b[4000+17] = 1
		b[5000] = 0xaa
		b[5048] = 0xbb
		b[5456+3*70+4] = 1
		le.PutUint32(b[10362:], 2)
		le.PutUint32(b[10370:], 5)
	}
	s, err := walkOpen(t, fx).SessionState()
	if err != nil {
		t.Fatalf("SessionState: %v", err)
	}
	if s.Registers[7] != -5 || s.Unheld != 1 || s.TriggerLatches[17] != 1 || s.RawHead[0] != 0xaa || s.RawMid[0] != 0xbb {
		t.Fatalf("register 7 %d unheld %d latch 17 %d raw %#x %#x", s.Registers[7], s.Unheld, s.TriggerLatches[17], s.RawHead[0], s.RawMid[0])
	}
	if s.Diplomacy[3*diplomacySide+4] != 1 || s.Diplomacy[4*diplomacySide+3] != 0 || s.Won != 2 || s.Lost != 5 {
		t.Fatalf("stance 3/4 %d won %d lost %d", s.Diplomacy[3*diplomacySide+4], s.Won, s.Lost)
	}
	fx.session = func(b []byte) { b[5456+60*70+3] = 1 }
	if _, err := walkOpen(t, fx).SessionState(); err == nil || !strings.Contains(err.Error(), "outside the 50x50") {
		t.Fatalf("a stance of side 60: %v", err)
	}
}

// A second-game cell entry is a key and 60 bytes read through the first
// game's fields to +0x2b: no trigger tail, and no cell entry is written
// (DIV-4171).
func TestTheSecondGameCellsReadToTheirBuildingKey(t *testing.T) {
	cell := make([]byte, SecondGame.cellLen)
	binary.LittleEndian.PutUint16(cell, 0x0203)
	cell[2], cell[3], cell[4] = 5, 6, 1
	binary.LittleEndian.PutUint32(cell[2+12:], 0xa0000042)
	for i := 2 + 0x2c; i < len(cell); i++ {
		cell[i] = 0x77
	}
	fx := secondMissionFixture()
	fx.cells = [][]byte{cell, cell}
	f := walkOpen(t, fx)
	cells, present, err := f.Cells()
	if err != nil || !present || len(cells) != 2 {
		t.Fatalf("Cells %d %v %v", len(cells), present, err)
	}
	c := cells[1]
	if c.Key != 0x0203 || c.BaselineCost != 5 || c.BaselineStatic != 6 || c.LayerCount != 1 || c.Building != 0xa0000042 || c.Trigger != [6]byte{} || c.Residue1 != [2]byte{} {
		t.Fatalf("cell %+v", c)
	}
	structures, _, err := f.StructureCells()
	if err != nil || len(structures) != 2 || structures[1].BuildingKey != 0xa0000042 || structures[1].Cell != 0x0203 {
		t.Fatalf("StructureCells %+v %v", structures, err)
	}
	if triggers, present, err := f.CellTriggers(); triggers != nil || present || err != nil {
		t.Fatalf("CellTriggers %v %v %v", triggers, present, err)
	}
	if err := f.SetCell(0, c); err == nil {
		t.Fatal("a second-game cell entry is written")
	}
}

// A mission file's state store is read by its grammar: the second game's
// empty shortcut array passes, and the first game's exact shape refuses it.
func TestTheSecondGameStateDataTakesAnEmptyShortcutArray(t *testing.T) {
	state, present, err := walkOpen(t, secondMissionFixture()).StateData()
	if err != nil || !present {
		t.Fatalf("StateData %v %v", present, err)
	}
	if v, ok, err := state.ReadLeaf("/CurrentState/InBattle"); err != nil || !ok || v.Int32 != 1 {
		t.Fatalf("InBattle %+v %v %v", v, ok, err)
	}
	if v, ok, err := state.ReadLeaf("/SpellBook/Shortcuts"); err != nil || !ok || v.Kind != 6 || len(v.Bytes) != 0 {
		t.Fatalf("Shortcuts %+v %v %v", v, ok, err)
	}
	f := walkOpen(t, secondMissionFixture())
	f.Profile = FirstGame
	if _, _, err := f.StateData(); err == nil {
		t.Fatal("the first game's shape takes an empty shortcut array")
	}
}

// secondBuildingDocument is a second-game world whose Building list holds a
// Building with the Token +0x0c word 0xa700 and a Pointer, a Building with
// one more DWORD (R2-ENGINE-209, R2-ENGINE-603).
func secondBuildingDocument() *File {
	s := newStream()
	s.second = true
	s.u32(0) // dead actors
	s.u8(1)
	s.u32(2)
	for i, class := range []string{"Building", "Pointer"} {
		s.obj(class)
		at := len(s.b)
		s.token(uint16(0x0605+i), 128, 128, uint32(90-i), 0, uint16(52-i))
		binary.LittleEndian.PutUint16(s.b[at+16:], 0xa700+uint16(i))
		s.raw(22, 0)
		s.u8(1)
		s.u16(7)
		s.u16(37)
		s.u16(0)
		s.u8(2)
		s.u8(3)
		s.u8(2)
		s.u32(5)
		s.u32(63)
		if class == "Pointer" {
			s.u32(0x6c6c6c6c)
		}
	}
	s.u32(0) // SpellEffects
	s.u16(0)
	s.u16(0)
	s.raw(4+secondSessionLen, 0)
	s.u32(0) // Sacks
	s.u32(0) // the extra Token list
	s.u32(0xbadface1)
	s.u32(0)
	s.raw(400, 0)
	if len(s.b)&1 != 0 {
		s.u8(0)
	}
	return &File{Body: s.b, Profile: SecondGame}
}

func TestTheSecondGameBuildingKeepsItsTokenWord(t *testing.T) {
	got, present, err := secondBuildingDocument().Buildings()
	if err != nil || !present || len(got) != 2 {
		t.Fatalf("Buildings %+v %v %v", got, present, err)
	}
	if got[0].Class != "Building" || got[0].Token0C != 0xa700 || got[1].Class != "Pointer" || got[1].Token0C != 0xa701 || got[1].Blocking != 5 || got[1].Attach != 63 {
		t.Fatalf("Buildings %+v", got)
	}
}

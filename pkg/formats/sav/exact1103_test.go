package sav

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

// Literal full envelope. The opaque Player Diary contains believable Player,
// world and actor-head decoys. None is an archive call at this cursor.
func exactFixture1103(world, extended bool, actorCount int) (*File, map[string]int) {
	s := newStream()
	off := map[string]int{}
	s.u32(80)
	s.u32(5)
	s.cstr("10.alm")
	s.raw(44, 0)
	s.u32(10)
	s.u32(2)
	s.u32(6)
	s.u32(4) // null, Player, same Player, null
	s.null()
	s.obj("Player") // class 1, object 2; null took no index
	s.b = append(s.b, (fixPlayer{name: "owner", slot: 1, human: true, money: 123}).encode()...)
	s.u32(1)
	s.u16(0)
	s.raw(80, 0)
	s.u16(0)
	s.u32(uint32(actorCount))
	for i := 0; i < actorCount; i++ {
		c := wantChar{name: "saved", cell: uint16(0x0907 + i), fineX: 23, fineY: 87,
			runtimeID: uint32(i + 1), mapUnitID: uint16(91 + i), journal: -1}
		c.stats[8], c.stats[9] = 7, 31
		s.human(c)
	}
	s.raw(12+32, 0)
	s.u16(1500) // Diary dword payload is raw, not archive calls
	raw := make([]byte, 6000)
	decoy := []byte{1, 128}
	decoy = append(decoy, (fixPlayer{name: "fake", slot: 2, money: 999}).encode()...)
	copy(raw, decoy)
	for i := 0; i < 8; i++ {
		copy(raw[100+32*i:], (fixActor{cell: uint16(0x1010 + i), fineX: 128, fineY: 128,
			state: 0x1234a020, runtimeID: uint32(50 + i), mapUnitID: uint16(91 + i)}).encode())
	}
	// Old world-shape discriminator: one block, matching key, 4374 session bytes.
	binary.LittleEndian.PutUint16(raw[500:], 1)
	binary.LittleEndian.PutUint32(raw[502:], 0x08100020)
	binary.LittleEndian.PutUint16(raw[506:], 1)
	binary.LittleEndian.PutUint16(raw[508:], 0x0810)
	copy(raw[510:], []byte{255, 255, 1, 0, 6, 0, 'P', 'l', 'a', 'y', 'e', 'r'})
	s.b = append(s.b, raw...)
	s.u16(0)
	s.u32(0)
	s.backref(2)
	s.null()
	s.u32(0) // dead list
	s.u8(0)
	if world {
		s.b[len(s.b)-1] = 2 // load arm: any nonzero selector
		s.u32(1)
		s.obj("Building")
		s.token(0x1212, 128, 128, 99, 0, 91) // matching ID, wrong class for a Unit join
		s.raw(40, 0)
		s.u32(0)
		for _, name := range []string{"blocks", "cells"} {
			off[name] = len(s.b)
			if extended {
				s.u16(0xffff)
				s.u32(1)
			} else {
				s.u16(0)
			}
			off[name+"Data"] = len(s.b)
			if extended {
				if name == "blocks" {
					s.u32(0x08102345)
				} else {
					s.u16(0x0810)
					s.raw(52, 0x71)
				}
			}
		}
		s.u32(0x2468abcd)
		off["session"] = len(s.b)
		s.raw(4374, 0)
		binary.LittleEndian.PutUint32(s.b[off["session"]+4*3:], 0xfedcba98)
		s.b[off["session"]+400+17] = 1
		s.b[off["session"]+1856+3*50+4] = 0xa5
		s.u32(0) // Sacks
	}
	s.u32(0xbadface1)
	s.u32(17)
	s.raw(400, 0x93)
	if len(s.b)%2 != 0 {
		s.u8(0x57)
	}
	return &File{Version: MinVersion, Body: s.b}, off
}

func TestExactDocument1103SparseAndEmptyIgnoreRawDecoys(t *testing.T) {
	for _, world := range []bool{false, true} {
		for n := 0; n <= 2; n++ {
			source, off := exactFixture1103(world, false, n)
			f, err := Open(source.Marshal())
			if err != nil {
				t.Fatal(err)
			}
			if f.Head.PlayerCount != 4 || len(f.Players) != 1 || f.Players[0].Money != 123 {
				t.Fatalf("reference slots became Players: %+v", f.Players)
			}
			party, rec, err := f.PartyWalk()
			if err != nil || len(party) != n || rec.Off != f.Players[0].Off {
				t.Fatalf("first non-null owner: %d %v", len(party), err)
			}
			if len(f.Actors) != n {
				t.Fatalf("raw heads entered actor index: %d, want %d", len(f.Actors), n)
			}
			for i, a := range f.Actors {
				if a.Class != "Human" || a.Cell != uint16(0x0907+i) || a.FineX != 23 || a.FineY != 87 || a.MapUnitID != uint16(91+i) {
					t.Fatalf("actor %d = %+v", i, a)
				}
			}
			if (f.World != nil) != world {
				t.Fatalf("raw world or empty terrain changed selector: world=%t", world)
			}
			if world {
				if len(f.World.Blocks) != 0 || f.World.CellRecCount != 0 || f.World.SessionOff != off["session"] {
					t.Fatalf("empty world lost its exact session: %+v", f.World)
				}
				result, _ := f.TriggerResult(3)
				latch, _ := f.TriggerLatch(17)
				relation, _ := f.Diplomacy(3, 4)
				if uint32(result) != 0xfedcba98 || latch != 1 || relation != 0xa5 {
					t.Fatalf("session reset: %x/%d/%x", result, latch, relation)
				}
			}
			if !bytes.Equal(f.Body, source.Body) {
				t.Fatal("index changed source bytes")
			}
		}
	}
}

func TestExactDocument1103ExtendedCountEditOffsets(t *testing.T) {
	source, off := exactFixture1103(true, true, 1)
	f, err := Open(source.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	w := f.World
	if w.BlocksDataOff != off["blocksData"] || w.CellRecDataOff != off["cellsData"] || w.SessionOff != off["session"] {
		t.Fatalf("extended count payload offsets: %+v", w)
	}
	rec, err := f.CellRecord(0)
	if err != nil || len(rec) != 54 || rec[2] != 0x71 || binary.LittleEndian.Uint16(rec) != 0x0810 {
		t.Fatalf("cell record: %x %v", rec, err)
	}
	before := append([]byte(nil), f.Body...)
	if err := f.SetBlockRecord(0, 0x67, 0x89); err != nil {
		t.Fatal(err)
	}
	before[off["blocksData"]], before[off["blocksData"]+1] = 0x89, 0x67
	if !bytes.Equal(before, f.Body) {
		t.Fatal("block edit changed count or unrelated bytes")
	}
	if err := f.SetActorPosition(0, 11, 12, 13, 14); err != nil {
		t.Fatal(err)
	}
	again, err := Open(f.Marshal())
	if err != nil || again.Actors[0].Cell != 0x0b0c || again.Actors[0].FineX != 13 || again.World.Blocks[0].Dyn != 0x67 {
		t.Fatalf("edited reopen: %v", err)
	}
}

func TestExactDocument1103CityNullAndRepeatedPlayerProvenance(t *testing.T) {
	f, err := Open(cityTestSource(t))
	if err != nil {
		t.Fatal(err)
	}
	want, err := f.Party()
	if err != nil {
		t.Fatal(err)
	}
	_, rec, err := f.PartyWalk()
	if err != nil || f.Head.PlayerCount != 1 || rec.Index != 2 {
		t.Fatal("city fixture topology changed")
	}
	b := append([]byte(nil), f.Body[:f.Head.End]...)
	b = append(b, 0, 0)
	b = append(b, f.Body[f.Head.End:rec.End]...)
	b = append(b, 2, 0)
	b = append(b, f.Body[rec.End:]...)
	binary.LittleEndian.PutUint32(b[f.Head.End-4:], 3)
	f.Body = b
	f, err = Open(f.Marshal())
	if err != nil || len(f.Players) != 1 {
		t.Fatalf("sparse city Open: %v", err)
	}
	p, err := f.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	d := p.Data()
	if len(d.Players) != 3 || d.Players[0] != 0 || d.Players[1] != d.Players[2] {
		t.Fatalf("city DTO lost reference topology: %+v", d.Players)
	}
	p, err = CityFromData(d)
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.SourceParty()
	if err != nil || len(got) != len(want) {
		t.Fatalf("native city source: %d %v", len(got), err)
	}
	for i := range got {
		got[i].Off, want[i].Off = 0, 0 // source offsets move when null slots are inserted
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("city source party changed after reference-slot round trip")
	}
	encoded, err := p.Marshal(CityUpdate{Money: 123, Characters: cityTestUpdates(p.Roster())})
	if err != nil {
		t.Fatal(err)
	}
	again, err := Open(encoded)
	if err != nil || len(again.Players) != 1 || again.Head.PlayerCount != 3 {
		t.Fatalf("sparse city re-export: %v", err)
	}
	if _, err := again.CityProvenance(); err != nil {
		t.Fatal(err)
	}
}

func TestExactDocument1103RejectsMalformedEnvelopeAndUnsupportedStrings(t *testing.T) {
	for name, mutate := range map[string]func(*File, map[string]int){
		"Player count":           func(f *File, _ map[string]int) { put32(f.Body, 71, 65537) },
		"unallocated Player":     func(f *File, _ map[string]int) { put16(f.Body, 75, 0x7777) },
		"terrain count":          func(f *File, off map[string]int) { put32(f.Body, off["blocks"]+2, 65537) },
		"truncated trailer":      func(f *File, _ map[string]int) { f.Body = f.Body[:len(f.Body)-2] },
		"extra suffix":           func(f *File, _ map[string]int) { f.Body = append(f.Body, 0, 0) },
		"extended map string":    func(f *File, _ map[string]int) { f.Body[8] = 0xff },
		"extended Player string": func(f *File, _ map[string]int) { f.Body[75+2+6+6] = 0xff },
	} {
		t.Run(name, func(t *testing.T) {
			f, off := exactFixture1103(true, true, 1)
			mutate(f, off)
			if opened, err := Open(f.Marshal()); err == nil || opened != nil {
				t.Fatal("malformed envelope published a partial index")
			}
		})
	}
}

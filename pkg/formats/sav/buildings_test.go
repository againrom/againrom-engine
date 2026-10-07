package sav

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func buildingDocumentFixture() (*File, []int) {
	s := newStream()
	s.u32(0) // dead actors
	s.u8(1)
	s.u32(3)
	var offsets []int
	for i, hp := range []uint16{7, 0, 0xffff} {
		s.obj("Building")
		offsets = append(offsets, len(s.b))
		// Runtime ordering differs from authored identity and record ordering.
		s.token(uint16(0x0605+i), 128, 128, uint32(90-i), 1, uint16(52-i))
		s.raw(22, 0)
		s.u8(1)
		s.u16(hp)
		s.u16(uint16(37 + i))
		s.u16(0)
		s.u8(2)
		s.u8(3)
		s.u8(2)
		s.u32(5)
		s.u32(63)
	}
	s.u32(0) // SpellEffects
	s.u16(0)
	s.u16(0)
	s.raw(4+4374, 0)
	s.u32(0) // Sacks
	s.u32(0xbadface1)
	s.u32(0)
	s.raw(400, 0)
	if len(s.b)&1 != 0 {
		s.u8(0)
	}
	return &File{Body: s.b}, offsets
}

func TestBuildingsExactCountedHealthProjection(t *testing.T) {
	f, offsets := buildingDocumentFixture()
	got, present, err := f.Buildings()
	if err != nil || !present || len(got) != 3 {
		t.Fatalf("Buildings = %+v %t %v", got, present, err)
	}
	for i, hp := range []uint16{7, 0, 0xffff} {
		want := Building{Off: offsets[i], Class: "Building", Identity: 0xa0000000 | uint32(90-i),
			AuthoredID: uint32(52 - i), Col: uint8(5 + i), Row: 6, Kind: 1,
			Health: hp, MaxHealth: uint16(37 + i), Width: 3, Height: 2, Blocking: 5, Attach: 63,
			ArchiveIndex: uint16(2 + i), RuntimeID: uint32(90 - i), Token0C: 1, Token0E: 0x21,
			Token18: 0x1818, Token1C: 0x1c1c1c1c, Field48: 2}
		want.Position = [12]byte{uint8(5 + i), 6, uint8(5 + i), 6, 128, 128, 0x77, 0x07, 0x20, 0xa0, 0x34, 0x12}
		want.Base52[14], want.Base52[15], want.Base52[18] = 3, 2, 5
		if !reflect.DeepEqual(got[i], want) {
			t.Fatalf("row %d = %+v, want %+v", i, got[i], want)
		}
	}
	got[0].Health = 999
	again, _, _ := f.Buildings()
	if again[0].Health != 7 {
		t.Fatal("health projection aliases the decoded document")
	}
	sacks, _, err := f.GroundSacks()
	if err != nil || len(sacks) != 0 {
		t.Fatalf("shared walk changed sacks: %+v %v", sacks, err)
	}
	for _, world := range []bool{true, false} {
		f, _ := groundFixture(world, 0)
		rows, present, err := f.Buildings()
		if err != nil || present != world {
			t.Fatalf("world %t: %+v %t %v", world, rows, present, err)
		}
		if world {
			var classes []string
			for _, row := range rows {
				classes = append(classes, row.Class)
			}
			if !reflect.DeepEqual(classes, []string{"Outpost", "Tavern", "Shop"}) {
				t.Fatalf("subclasses lost: %v", classes)
			}
		} else if len(rows) != 0 {
			t.Fatal("no-world save invented buildings")
		}
	}
}

func TestBuildingsRejectMalformedDocumentWithoutPartialRows(t *testing.T) {
	f, offsets := buildingDocumentFixture()
	for end := 0; end < len(f.Body); end++ {
		rows, _, err := (&File{Body: f.Body[:end]}).Buildings()
		if err == nil || len(rows) != 0 {
			t.Fatalf("accepted truncation at %d/%d: %+v %v", end, len(f.Body), rows, err)
		}
	}
	for name, mutate := range map[string]func([]byte){
		"count":             func(b []byte) { binary.LittleEndian.PutUint32(b[5:], 0xffffffff) },
		"zero identity":     func(b []byte) { binary.LittleEndian.PutUint32(b[offsets[1]+29:], 0) },
		"repeated identity": func(b []byte) { copy(b[offsets[1]+29:offsets[1]+33], b[offsets[0]+29:offsets[0]+33]) },
		"schema":            func(b []byte) { binary.LittleEndian.PutUint16(b[11:], 2) },
	} {
		t.Run(name, func(t *testing.T) {
			body := append([]byte(nil), f.Body...)
			mutate(body)
			if rows, _, err := (&File{Body: body}).Buildings(); err == nil || len(rows) != 0 {
				t.Fatalf("accepted malformed document: %+v %v", rows, err)
			}
		})
	}
	for _, head := range []Head{{End: -1}, {End: len(f.Body) + 1}, {PlayerCount: 65537}} {
		if _, _, err := (&File{Body: f.Body, Head: head}).Buildings(); err == nil {
			t.Fatalf("accepted invalid head %+v", head)
		}
	}
	var nilFile *File
	if _, _, err := nilFile.Buildings(); err == nil {
		t.Fatal("accepted nil save")
	}
}

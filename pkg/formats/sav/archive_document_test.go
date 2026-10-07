package sav

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

// A literal whole document. No production encoder builds the expected bytes.
// It uses every root list under one archive index namespace, including a
// Player null/repeat, Group/dead alias, and nested/top-level effect alias.
func archiveDocument1115Fixture() ([]byte, bool) {
	s := newStream()
	s.u32(0x87654321)
	s.u32(0x76543210)
	s.cstr("23.alm")
	for i := range 11 {
		s.u32(uint32(0x100 + i))
	}
	s.u32(23)
	s.u32(3)
	s.u32(9)
	s.u32(3)
	s.null()
	s.obj("Player")
	player := s.next - 1
	s.cstr("literal owner")
	at := len(s.b)
	s.raw(51, 0)
	binary.LittleEndian.PutUint16(s.b[at:], 1)
	binary.LittleEndian.PutUint32(s.b[at+2:], 1)
	binary.LittleEndian.PutUint32(s.b[at+43:], 0xa0000009)
	binary.LittleEndian.PutUint32(s.b[at+47:], 0x11000011)
	s.u32(1) // one inline Group
	s.u16(0)
	s.raw(80, 0)
	s.u16(0)
	s.u32(1)
	s.human(wantChar{name: "living alias", runtimeID: 9, journal: -1})
	actor := s.classes["Human"] + 1
	s.u32(17)
	s.u32(18)
	s.u32(0x11000011)
	s.raw(32, 0x45)
	s.u16(0) // inline Diary
	s.u16(0)
	s.u32(0x11000011)
	s.backref(player)
	s.u32(2) // two dead references to the already introduced actor
	s.backref(actor)
	s.backref(actor)
	s.u8(1)
	s.u32(2)
	s.obj("Outpost")
	outpost := s.next - 1
	s.token(0x2010, 45, 64, 11, 2, 301)
	s.raw(40, 0x31)
	s.u32(81)
	s.u32(82)
	s.u32(83)
	s.u32(84)
	s.u16(2)
	s.raw(16, 0x67)
	s.backref(outpost)
	s.u32(2)
	s.obj("SpellTransport")
	s.token(0x2122, 17, 45, 12, 1, 0)
	s.u8(2)
	s.u8(3)
	s.obj("PointEffect")
	s.token(0x2324, 34, 78, 13, 2, 0)
	s.raw(2, 0x43)
	s.obj("Effect_DirectDamage")
	s.token(0x2526, 44, 82, 14, 3, 0)
	s.raw(31, 0x52)
	s.u32(0xa0000009)
	s.obj("AreaEffect")
	area := s.next - 1
	s.token(0x2728, 37, 56, 15, 4, 0)
	s.raw(8, 0x64)
	s.null()
	s.u16(67)
	s.backref(area)
	s.u16(2) // block records
	s.u32(0x10013412)
	s.u32(0xfffe7856)
	s.u16(3) // duplicate cell records remain ordered, including zero clears
	for i, key := range []uint16{0x1001, 0x1001, 0xffff} {
		s.u16(key)
		for j := range 52 {
			s.u8(byte(i*61 + j))
		}
	}
	s.u32(0x11223344) // terrain identity, not an archive tag
	for i := range 4374 {
		s.u8(byte(i*37 + 11))
	}
	s.u32(2)
	s.obj("Sack")
	sack := s.next - 1
	s.token(0x3031, 31, 63, 16, 0, 0)
	s.u32(156)
	s.u32(4)
	s.null()
	s.item(wantPiece{class: "Item", code: 0x5432, row: 3, stack: 4})
	item := s.next - 1
	s.null()
	s.backref(item)
	s.u32(57)
	s.u32(58)
	s.backref(sack)
	s.u32(0xbadface1)
	s.u32(0x8899aabb)
	for i := range 400 {
		s.u8(byte(i*19 + 5))
	}
	pad := len(s.b)&1 != 0
	if pad {
		s.u8(0x97)
	}
	return s.b, pad
}

func TestArchiveDocument1115CompleteDetachedBody(t *testing.T) {
	if binary.Size(archiveCell{}) != 54 || binary.Size(archiveSession{}) != 4374 {
		t.Fatal("typed wire layout changed")
	}
	source, pad := archiveDocument1115Fixture()
	expected := append([]byte(nil), source...)
	if pad {
		expected[len(expected)-1] = 0 // independent new transport pad
	}
	d, err := parseArchiveDocument(source)
	if err != nil {
		t.Fatal(err)
	}
	clear(source)
	if len(d.players) != 3 || d.players[0] != nil || d.players[1] != d.players[2] || d.players[1].Index != 0 || d.head.End != 0 {
		t.Fatal("Player slots/alias or detached coordinates are wrong")
	}
	actor := d.players[1].Groups[0].RefSlots["Actors"][0]
	if len(d.dead) != 2 || d.dead[0] != actor || d.dead[1] != actor {
		t.Fatal("cross-root actor alias lost")
	}
	world := d.world
	if world == nil || len(world.effects) != 2 || world.effects[0].RefSlots["ST48"][0] != world.effects[1] || world.buildings[0] != world.buildings[1] || world.sacks[0] != world.sacks[1] {
		t.Fatal("world roots/aliases lost")
	}
	if len(world.cells) != 3 || world.cells[0].Cell != world.cells[1].Cell || world.cells[1].Cost != 61 || world.cells[0].GroundActor != 0x07060504 || world.cells[0].SourceX != 46 || world.cells[0].Residue32 != 0x3332 {
		t.Fatal("typed cell layout or duplicate order differs")
	}
	if world.terrainIdentity != 0x11223344 || world.session.Latches[0] != byte((400*37+11)%256) || world.session.Diplomacy[0][0] != byte((1856*37+11)%256) {
		t.Fatal("terrain/session field lost or shifted")
	}
	got, err := serializeArchiveDocument(d)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, expected) {
		at := 0
		for at < len(got) && at < len(expected) && got[at] == expected[at] {
			at++
		}
		t.Fatalf("complete body differs at %d: lengths %d/%d", at, len(got), len(expected))
	}
	// Independent existing File index must also accept the rebuilt envelope.
	if err := (&File{Body: got}).index(); err != nil {
		t.Fatalf("existing reader: %v", err)
	}
	world.session.Won = 17
	world.cells[1].Operation = 26
	world.cells[1].TargetX = 112
	world.buildings[0].Value["B64"] = 98
	actor.Value["Health"] = 27
	d.head.CounterA++
	d.trailer.Residual[97] = 0x01020304
	changed, err := serializeArchiveDocument(d)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := parseArchiveDocument(changed)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.world.session.Won != 17 || reloaded.world.cells[1].Operation != 26 || reloaded.world.cells[1].TargetX != 112 || reloaded.world.buildings[1].Value["B64"] != 98 || reloaded.dead[1].Value["Health"] != 27 || reloaded.head.CounterA != 0x87654322 || reloaded.trailer.Residual[97] != 0x01020304 {
		t.Fatal("writer did not consume changed fields")
	}
}

func TestArchiveDocument1115NoWorldAndAbsentMarkerArm(t *testing.T) {
	body, _ := archiveDocument1115Fixture()
	d, err := parseArchiveDocument(body)
	if err != nil {
		t.Fatal(err)
	}
	d.world = nil
	d.head.Mission = 0
	d.marker, d.global = 0xbadface0, 0
	b, err := serializeArchiveDocument(d)
	if err != nil {
		t.Fatal(err)
	}
	again, err := parseArchiveDocument(b)
	if err != nil || again.world != nil || again.marker != 0xbadface0 || again.trailer != d.trailer || len(again.dead) != 2 {
		t.Fatalf("no-world/marker branch: %v", err)
	}
}

func TestArchiveDocument1115RejectsCorruptOrUnboundedShape(t *testing.T) {
	body, _ := archiveDocument1115Fixture()
	for n := range len(body) {
		if d, err := parseArchiveDocument(body[:n]); err == nil || d != nil {
			t.Fatalf("accepted truncated body at %d of %d", n, len(body))
		}
	}
	for name, mutate := range map[string]func(*archiveDocument){
		"null building":      func(d *archiveDocument) { d.world.buildings[0] = nil },
		"wrong effect class": func(d *archiveDocument) { d.world.effects[0] = d.dead[0] },
		"wrong Player class": func(d *archiveDocument) { d.players[0] = d.dead[0] },
		"block duplicate":    func(d *archiveDocument) { d.world.blocks[1].Cell = d.world.blocks[0].Cell },
		"cell count":         func(d *archiveDocument) { d.world.cells = make([]archiveCell, maxListElements+1) },
		"map CString":        func(d *archiveDocument) { d.head.MapName = strings.Repeat("m", 255) },
		"unframed global":    func(d *archiveDocument) { d.marker = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			d, err := parseArchiveDocument(body)
			if err != nil {
				t.Fatal(err)
			}
			mutate(d)
			if b, err := serializeArchiveDocument(d); err == nil || b != nil {
				t.Fatal("invalid document produced output")
			}
		})
	}
}

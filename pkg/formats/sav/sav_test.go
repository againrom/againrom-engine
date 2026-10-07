package sav

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// ---------------------------------------------------------------------------
// The fixture. Every byte below is written from spec.md's own contract; nothing
// here is copied out of a game file, and no test in this package reads one.
// ---------------------------------------------------------------------------

// fixture is a whole synthetic save, described by what a test wants to vary.
type fixture struct {
	mapName   string
	mission   uint32
	players   []fixPlayer
	actors    []fixActor
	blocks    []BlockRecord
	cellKeys  []uint16 // the keys the cell-record table carries
	noWorld   bool     // omit the world half entirely
	buildings int      // how many Building instances to chain after the roster
	label     []byte
	labelJunk []byte // what the label region held before, left past the NUL

	// tail is the whole uncompressed tail. Nil is the signature dword alone,
	// which is what every fixture predating 0150 carried and which does not
	// frame as a state store — so those fixtures exercise the unframed arm
	// and a fixture that sets this one exercises the framed one.
	tail []byte

	// trailerTurnTracing and trailerScriptTracing are world+0x118's dwords 0
	// and 1 (TrailerBody.TurnTracing/ScriptTracing).
	trailerTurnTracing, trailerScriptTracing uint32
}

type fixPlayer struct {
	name    string
	slot    uint16
	human   bool
	money   uint32
	outcome uint8
}

type fixActor struct {
	cell      uint16
	fineX     uint8
	fineY     uint8
	state     uint32
	runtimeID uint32
	mapUnitID uint16
}

// fixState is an arbitrary raw terrain identity; no whitelist governs indexing.
const fixState = 0x1234a020

// fixBuilding is one 77-byte Building record: the 37-byte Token head then the
// class's own 40 bytes.
func fixBuilding(i int) []byte {
	var b []byte
	// The head's first twelve bytes are the block the token points at, and
	// carry the cell, the fine position and the state.
	le16(&b, uint16(0x0500+i*0x101))
	le16(&b, uint16(0x0500+i*0x101))
	b = append(b, restFine, restFine)
	le16(&b, 0x0777)
	le32(&b, fixState)
	le32(&b, uint32(i+2))               // RuntimeID: creation order, hero is 1
	b = append(b, 0x11)                 // T0C
	le16(&b, 0x0022)                    // T0E
	le32(&b, uint32(0x00330000+i+1))    // T08, low u16 is the map unit id
	le16(&b, 0x0044)                    // T18
	le32(&b, 0x55555555)                // T1C
	le32(&b, uint32(0x02c90000+i*0x80)) // Identity: an allocator's stride
	le32(&b, 0x02c8f4c0)                // Reference: one longer-lived object
	for j := 0; j < 22; j++ {           // B52
		b = append(b, byte(0x60+j))
	}
	b = append(b, 0x01) // B40
	le16(&b, 1000)      // B42
	le16(&b, 1000)      // B44
	le16(&b, 7)         // B46
	b = append(b, 0x02) // B48
	b = append(b, 0x03) // B60
	b = append(b, 0x04) // B61
	le32(&b, 506)       // B64
	le32(&b, 511)       // B68
	return b
}

func le16(b *[]byte, v uint16) { *b = binary.LittleEndian.AppendUint16(*b, v) }
func le32(b *[]byte, v uint32) { *b = binary.LittleEndian.AppendUint32(*b, v) }

// body assembles the decoded stream.
func (f fixture) body() []byte {
	var b []byte
	le32(&b, 0x50)
	le32(&b, 0x05)
	b = append(b, byte(len(f.mapName)))
	b = append(b, f.mapName...)
	for i := 0; i < reservedDwords; i++ {
		le32(&b, 0)
	}
	le32(&b, f.mission)
	le32(&b, 2)
	le32(&b, 6)
	le32(&b, uint32(len(f.players)))
	s := newStream()
	s.b = b
	for i, p := range f.players {
		s.obj("Player")
		player := p.encode()
		// Distinct objects need distinct Player keys. The old fixture used
		// the same filler dword for every This field.
		put32(player, len(player)-4, uint32(0x100000+i))
		s.b = append(s.b, player...)
		s.u32(1)
		s.u16(0)
		s.raw(80, 0)
		s.u16(0)
		n := 0
		if i == 0 {
			n = len(f.actors)
		}
		s.u32(uint32(n))
		for _, a := range f.actors[:n] {
			start := len(s.b)
			c := wantChar{cell: a.cell, fineX: a.fineX, fineY: a.fineY, runtimeID: a.runtimeID, mapUnitID: a.mapUnitID, journal: -1}
			c.stats[StatHealth] = 1
			c.stats[StatHealthMax] = 1
			s.human(c)
			tagLen := 2
			if u16(s.b, start) == 0xffff {
				tagLen = 6 + len("Human")
			}
			put32(s.b, start+tagLen+8, a.state)
		}
		s.raw(12+32+2+2+4, 0)
	}
	s.u32(0) // dead list
	s.u8(0)
	if !f.noWorld {
		s.b[len(s.b)-1] = 1
		s.u32(uint32(f.buildings))
		for i := 0; i < f.buildings; i++ {
			s.obj("Building")
			s.b = append(s.b, fixBuilding(i)...)
		}
		s.u32(0) // SpellEffects
	}
	b = s.b
	if !f.noWorld {
		le16(&b, uint16(len(f.blocks)))
		for _, r := range f.blocks {
			le32(&b, uint32(r.Cell)<<16|uint32(r.Dyn)<<8|uint32(r.Static))
		}
		le16(&b, uint16(len(f.cellKeys)))
		for _, k := range f.cellKeys {
			rec := make([]byte, cellRecLen)
			binary.LittleEndian.PutUint16(rec, k)
			rec[7] = 0x99 // undecoded, and must survive a round trip
			b = append(b, rec...)
		}
		b = append(b, 0xde, 0xad, 0xbe, 0xef) // the four unattributed bytes
		b = append(b, make([]byte, sessionLen)...)
		le32(&b, 0) // Sacks
	}
	le32(&b, 0xbadface1)
	le32(&b, 0)
	le32(&b, f.trailerTurnTracing)
	le32(&b, f.trailerScriptTracing)
	b = append(b, make([]byte, 392)...)
	if len(b)%2 != 0 {
		b = append(b, 0)
	}
	return b
}

func (p fixPlayer) encode() []byte {
	var b []byte
	b = append(b, byte(len(p.name)))
	b = append(b, p.name...)
	le16(&b, p.slot)
	le32(&b, uint32(p.slot))
	b = append(b, 1, 2, 3, 4, 5, 6, 7, 8)
	b = append(b, 0x02)
	if p.human {
		le32(&b, 0)
	} else {
		le32(&b, 1)
	}
	le16(&b, 0x0002)
	le32(&b, p.money^obfuscator)
	b = append(b, p.outcome, 0x01)
	le32(&b, 0x11223344^obfuscator)
	le32(&b, 0x55667788)
	le16(&b, 0x0123)
	le16(&b, 0x0456)
	le32(&b, 0x99aabbcc)
	le32(&b, 0xddeeff00)
	le32(&b, 0x0f0f0f0f)
	return b
}

func (a fixActor) encode() []byte {
	b := make([]byte, 0, 0x20)
	le16(&b, a.cell)
	le16(&b, a.cell)
	b = append(b, a.fineX, a.fineY)
	le16(&b, 0x0777)
	le32(&b, a.state)
	le32(&b, a.runtimeID)
	b = append(b, 0xaa, 0xbb, 0xcc) // +0x10..+0x12, undecoded members
	le16(&b, a.mapUnitID)
	b = append(b, bytes.Repeat([]byte{0x3c}, 11)...) // more undecoded members
	return b
}

// file assembles the whole container around the body.
func (f fixture) file() []byte {
	blob := Compress(f.body())
	out := make([]byte, headerLen)
	copy(out, Magic)
	binary.LittleEndian.PutUint32(out[4:], uint32(headerLen+len(blob)))
	binary.LittleEndian.PutUint32(out[8:], MinVersion)
	binary.LittleEndian.PutUint32(out[12:], uint32(len(blob)))
	out = append(out, blob...)
	region := make([]byte, labelLen)
	copy(region, f.labelJunk)
	copy(region, f.label)
	if len(f.label) < labelLen {
		region[len(f.label)] = 0
	}
	out = append(out, region...)
	if f.tail != nil {
		return append(out, f.tail...)
	}
	return append(out, "&YA1"...)
}

// standard is a complete envelope with two Players and ten owner-graph actors.
func standard() fixture {
	f := fixture{
		mapName: "t.alm",
		mission: 7,
		players: []fixPlayer{
			{name: "Hero", slot: 1, human: true, money: 100, outcome: 1},
			{name: "Foe", slot: 2, money: 4000000000, outcome: 0},
		},
		label:     []byte("one"),
		labelJunk: []byte("Restart last mission"),
		blocks: []BlockRecord{
			{Cell: 0x0810, Dyn: 0x41, Static: 0x21},
			{Cell: 0x0a20, Dyn: 0xc5, Static: 0x05},
			{Cell: 0x1234, Dyn: 0x30, Static: 0x20},
		},
		cellKeys: []uint16{0x0810, 0x1234},
	}
	for i := 0; i < 10; i++ {
		f.actors = append(f.actors, fixActor{
			cell:      uint16(0x0500 + i*0x101),
			fineX:     restFine,
			fineY:     restFine,
			state:     fixState,
			runtimeID: uint32(i + 1),
			mapUnitID: uint16(20 + i),
		})
	}
	return f
}

func open(t *testing.T, f fixture) *File {
	t.Helper()
	got, err := Open(f.file())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return got
}

// ---------------------------------------------------------------------------

func TestOpenReadsTheCampaignHead(t *testing.T) {
	f := open(t, standard())
	h := f.Head
	if h.MapName != "t.alm" || h.Mission != 7 || h.Difficulty != 2 || h.PlayerCount != 2 {
		t.Fatalf("head reads %+v", h)
	}
	if h.CounterA != 0x50 || h.CounterB != 0x05 {
		t.Fatalf("counters %d/%d", h.CounterA, h.CounterB)
	}
	if string(f.Label) != "one" {
		t.Fatalf("label %q", f.Label)
	}
	// The label region keeps the debris past the NUL, because the original
	// never clears it and a save that cleared it would be a save no game
	// wrote.
	if !bytes.Contains(f.LabelRegion[:24], []byte("art last mission")) {
		t.Fatalf("label region lost what was under it: %q", f.LabelRegion[:24])
	}
	// The fixture's tail is the signature dword alone, which is too short to
	// frame as a store, so it is carried whole as the trailing region and no
	// store is reported. That is the shape a file this package could open
	// before the tail was split has to keep.
	if f.Store != nil {
		t.Fatalf("a 4-byte tail framed as a store: %q", f.Store)
	}
	if string(f.TailRest) != "&YA1" {
		t.Fatalf("tail rest %q", f.TailRest)
	}
}

// TestMapNameShiftsEveryOffsetAfterIt is the head's one variable: a longer name
// moves the mission number, so a reader with a tabled offset is wrong on every
// map whose name is not the one the table was written for.
func TestMapNameShiftsEveryOffsetAfterIt(t *testing.T) {
	for _, name := range []string{"a", "10.alm", "a-very-long-map-name.alm"} {
		s := standard()
		s.mapName = name
		f := open(t, s)
		if f.Head.MapName != name || f.Head.Mission != 7 {
			t.Fatalf("%q: name %q mission %d", name, f.Head.MapName, f.Head.Mission)
		}
	}
}

func TestPlayersAndTheInvolution(t *testing.T) {
	f := open(t, standard())
	if len(f.Players) != 2 {
		t.Fatalf("%d players", len(f.Players))
	}
	if f.Players[0].Name != "Hero" || f.Players[0].Money != 100 || f.Players[0].Participant != 0 {
		t.Fatalf("player 0 reads %+v", f.Players[0])
	}
	// A reader that skipped the involution would see the raw dword here, and
	// on a shipped save that is about 1.54 billion gold.
	if f.Players[1].Money != 4000000000 {
		t.Fatalf("player 1 money %d", f.Players[1].Money)
	}
	if f.Players[0].Outcome != 1 || f.Players[1].Outcome != 0 {
		t.Fatalf("outcomes %d/%d", f.Players[0].Outcome, f.Players[1].Outcome)
	}
	// A field the layout names and this package has no better name for is
	// reachable by that name and by no offset arithmetic at the call site.
	if v, ok := f.Players[0].Field("F48"); !ok || v != 0x11223344 {
		t.Fatalf("the second obfuscated dword reads %#x %v", v, ok)
	}
	if _, ok := f.Players[0].Field("NoSuchMember"); ok {
		t.Fatal("a member the layout does not carry was answered")
	}
}

func TestOpenRefusesAMalformedContainer(t *testing.T) {
	good := standard().file()
	bad := func(mut func([]byte)) []byte {
		b := append([]byte(nil), good...)
		mut(b)
		return b
	}
	for _, c := range []struct {
		name string
		in   []byte
	}{
		{"nothing at all", nil},
		{"a header and no blob", good[:headerLen]},
		{"the wrong magic", bad(func(b []byte) { b[0] = 'X' })},
		{"an older version", bad(func(b []byte) {
			binary.LittleEndian.PutUint32(b[8:], MinVersion-1)
		})},
		{"a blob ending past the file", bad(func(b []byte) {
			binary.LittleEndian.PutUint32(b[4:], uint32(len(b))+1)
		})},
		{"a blob length disagreeing with its end", bad(func(b []byte) {
			binary.LittleEndian.PutUint32(b[12:], 7)
		})},
		{"a file with no room for a label region", good[:len(good)-8]},
		{"a body too short to hold a head", func() []byte {
			return (&File{Version: MinVersion, Body: []byte{0, 0}}).Marshal()
		}()},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := Open(c.in); err == nil {
				t.Fatal("want an error, got none")
			}
		})
	}
}

// TestOpenRefusesARosterThatDoesNotAddUp: the head's count and the records found
// must agree, because the count is the only independent statement of how many
// there are.
func TestOpenRefusesARosterThatDoesNotAddUp(t *testing.T) {
	s := standard()
	b := s.body()
	// Break the second record's two slot ids so it cannot be accepted.
	i := bytes.Index(b, []byte("Foe"))
	if i < 0 {
		t.Fatal("fixture changed: no second player")
	}
	b[i+3+2] = 0x7e
	fix := s
	blob := Compress(b)
	whole := fix.file()
	head := whole[:headerLen]
	binary.LittleEndian.PutUint32(head[4:], uint32(headerLen+len(blob)))
	binary.LittleEndian.PutUint32(head[12:], uint32(len(blob)))
	out := append(append(append([]byte(nil), head...), blob...), make([]byte, labelLen)...)
	if _, err := Open(out); err == nil {
		t.Fatal("want an error, got none")
	}
}

// ---------------------------------------------------------------------------
// The world half
// ---------------------------------------------------------------------------

func TestWorldHalfIsLocated(t *testing.T) {
	f := open(t, standard())
	if f.World == nil {
		t.Fatal("no world half located")
	}
	if len(f.World.Blocks) != 3 || f.World.CellRecCount != 2 {
		t.Fatalf("blocks %d cell records %d", len(f.World.Blocks), f.World.CellRecCount)
	}
	if got := f.World.Blocks[1]; got.Cell != 0x0a20 || got.Dyn != 0xc5 || got.Static != 0x05 {
		t.Fatalf("block 1 reads %+v", got)
	}
	if r := f.World.Blocks[0]; r.Row() != 8 || r.Col() != 0x10 {
		t.Fatalf("cell 0x0810 unpacks to (%d,%d)", r.Col(), r.Row())
	}
	rec, err := f.CellRecord(1)
	if err != nil || len(rec) != cellRecLen || rec[7] != 0x99 {
		t.Fatalf("cell record 1: %v %d", err, len(rec))
	}
}

// Terrain identity comes from the envelope, not key-set plausibility.
func TestWorldHalfDoesNotDependOnKeySetAgreement(t *testing.T) {
	s := standard()
	s.cellKeys = []uint16{0x0810, 0x0a20}
	if f := open(t, s); f.World == nil || f.World.CellRecCount != 2 {
		t.Fatal("exact world lost")
	}
	s.cellKeys = nil
	if f := open(t, s); f.World == nil || f.World.CellRecCount != 0 {
		t.Fatal("empty cells lost")
	}
}

func TestASaveWithNoWorldHalfOpens(t *testing.T) {
	s := standard()
	s.noWorld = true
	s.mission = 0
	f := open(t, s)
	if f.World != nil {
		t.Fatal("a world half was invented")
	}
	if f.Head.Mission != 0 || f.Head.MapName != "t.alm" {
		t.Fatalf("head reads mission %d map %q", f.Head.Mission, f.Head.MapName)
	}
	// The roster is in the campaign half and is therefore still here — which
	// is the only reason a between-mission save is worth reading at all.
	if len(f.Players) != 2 || f.Players[0].Outcome != 1 {
		t.Fatalf("roster lost: %d players", len(f.Players))
	}
	if _, err := f.CellRecord(0); err == nil {
		t.Fatal("cell records answered from a save that has none")
	}
	if _, _, err := f.Counters(); err == nil {
		t.Fatal("counters answered from a save that has no session block")
	}
}

func TestSessionAccessors(t *testing.T) {
	f := open(t, standard())
	if err := f.SetTriggerLatch(999, 1); err != nil {
		t.Fatalf("SetTriggerLatch: %v", err)
	}
	if v, err := f.TriggerLatch(999); err != nil || v != 1 {
		t.Fatalf("latch 999 reads %d %v", v, err)
	}
	if err := f.SetTriggerResult(99, -5); err != nil {
		t.Fatalf("SetTriggerResult: %v", err)
	}
	if v, err := f.TriggerResult(99); err != nil || v != -5 {
		t.Fatalf("result 99 reads %d %v", v, err)
	}
	if err := f.SetDiplomacy(49, 49, 3); err != nil {
		t.Fatalf("SetDiplomacy: %v", err)
	}
	if v, err := f.Diplomacy(49, 49); err != nil || v != 3 {
		t.Fatalf("diplomacy (49,49) reads %d %v", v, err)
	}
	if err := f.SetCounters(1, 2); err != nil {
		t.Fatalf("SetCounters: %v", err)
	}
	won, lost, err := f.Counters()
	if err != nil || won != 1 || lost != 2 {
		t.Fatalf("counters read %d/%d %v", won, lost, err)
	}
	for _, err := range []error{
		f.SetTriggerLatch(1000, 1), f.SetTriggerResult(100, 1),
		f.SetDiplomacy(50, 0, 1), f.SetDiplomacy(0, -1, 1),
	} {
		if err == nil {
			t.Fatal("an out-of-range session index was accepted")
		}
	}
}

func TestSessionStateHandsOffTheCompleteSupportedPopulations(t *testing.T) {
	f := open(t, standard())
	if err := f.SetTriggerLatch(17, 1); err != nil {
		t.Fatalf("SetTriggerLatch: %v", err)
	}
	if err := f.SetDiplomacy(3, 4, 0xa5); err != nil {
		t.Fatalf("SetDiplomacy: %v", err)
	}
	if err := f.SetDiplomacy(4, 3, 0x5a); err != nil {
		t.Fatalf("SetDiplomacy mirror: %v", err)
	}

	state, err := f.SessionState()
	if err != nil {
		t.Fatalf("SessionState: %v", err)
	}
	if len(state.TriggerLatches) != 1000 || len(state.Diplomacy) != 50*50 {
		t.Fatalf("SessionState sizes = %d/%d, want 1000/2500",
			len(state.TriggerLatches), len(state.Diplomacy))
	}
	if state.TriggerLatches[17] != 1 || state.Diplomacy[3*50+4] != 0xa5 ||
		state.Diplomacy[4*50+3] != 0x5a {
		t.Fatalf("SessionState values = latch %#x, relation %#x/%#x",
			state.TriggerLatches[17], state.Diplomacy[3*50+4], state.Diplomacy[4*50+3])
	}

	// The handoff is detached: sim owns a copy after import, and a caller that
	// edits this projection must not edit File.Body behind the decoder.
	state.TriggerLatches[17] = 0
	state.Diplomacy[3*50+4] = 0
	if v, _ := f.TriggerLatch(17); v != 1 {
		t.Fatalf("mutating SessionState changed the file latch to %d", v)
	}
	if v, _ := f.Diplomacy(3, 4); v != 0xa5 {
		t.Fatalf("mutating SessionState changed the file relation to %#x", v)
	}
}

func TestSessionStateRefusesASaveWithNoWorldHalf(t *testing.T) {
	s := standard()
	s.noWorld = true
	s.mission = 0
	if _, err := open(t, s).SessionState(); err == nil {
		t.Fatal("SessionState invented a session for a between-mission save")
	}
}

// TestSessionAccessorsLandAtTheOffsetsTheFormatSpecifies is the independent
// half of TestSessionAccessors above. That test writes through a setter and
// reads back through the matching getter, and both resolve their position
// through the SAME offset constant -- so an off-by-one in sessDiplomacy, or
// in any of the others, moves the write and the read together and the round
// trip still agrees with itself.
//
// This one writes through the setters and then reads f.Body directly, at
// offsets spelt out as literals. THE LITERALS ARE DELIBERATE and must not be
// "tidied" into the sess* constants: those are the constants the accessors
// themselves read, and using them here would restore exactly the defect this
// test exists to catch.
//
// The offsets are the session block's own, from its first byte: trigger
// results at 0 as signed dwords, trigger latches at 400 as bytes, the 50x50
// diplomacy matrix at 1856 as bytes, and the two counters at 4362 and 4370 as
// dwords.
func TestSessionAccessorsLandAtTheOffsetsTheFormatSpecifies(t *testing.T) {
	f := open(t, standard())
	if f.World == nil {
		t.Fatal("the standard fixture has no world half, so there is no session block")
	}
	base := f.World.SessionOff

	// Distinct values throughout, so a wrong offset that happens to land on
	// another written field is still a failure rather than a coincidence.
	if err := f.SetTriggerResult(99, -5); err != nil {
		t.Fatalf("SetTriggerResult: %v", err)
	}
	if err := f.SetTriggerLatch(999, 7); err != nil {
		t.Fatalf("SetTriggerLatch: %v", err)
	}
	var head [48]byte
	head[0], head[47] = 0xaa, 0xbb
	if err := f.SetRawHead(head); err != nil {
		t.Fatalf("SetRawHead: %v", err)
	}
	var mid [400]byte
	mid[0], mid[399] = 0xcc, 0xdd
	if err := f.SetRawMid(mid); err != nil {
		t.Fatalf("SetRawMid: %v", err)
	}
	if err := f.SetDiplomacy(49, 49, 3); err != nil {
		t.Fatalf("SetDiplomacy: %v", err)
	}
	if err := f.SetCounters(11, 22); err != nil {
		t.Fatalf("SetCounters: %v", err)
	}

	if got := int32(u32(f.Body, base+0+4*99)); got != -5 {
		t.Errorf("trigger result 99 sits at session+%d reading %d, want -5", 0+4*99, got)
	}
	if got := f.Body[base+400+999]; got != 7 {
		t.Errorf("trigger latch 999 sits at session+%d reading %d, want 7", 400+999, got)
	}
	if got0, got47 := f.Body[base+1400], f.Body[base+1400+47]; got0 != 0xaa || got47 != 0xbb {
		t.Errorf("raw head sits at session+1400..1448 reading %#x/%#x, want 0xaa/0xbb", got0, got47)
	}
	if got0, got399 := f.Body[base+1448], f.Body[base+1448+399]; got0 != 0xcc || got399 != 0xdd {
		t.Errorf("raw mid sits at session+1448..1848 reading %#x/%#x, want 0xcc/0xdd", got0, got399)
	}
	if got := f.Body[base+1856+49*50+49]; got != 3 {
		t.Errorf("diplomacy (49,49) sits at session+%d reading %d, want 3",
			1856+49*50+49, got)
	}
	if got := u32(f.Body, base+4362); got != 11 {
		t.Errorf("won counter sits at session+4362 reading %d, want 11", got)
	}
	if got := u32(f.Body, base+4370); got != 22 {
		t.Errorf("lost counter sits at session+4370 reading %d, want 22", got)
	}

	// The constants themselves, so a change to one names itself rather than
	// only moving a byte somewhere the assertions above happen to look.
	for _, c := range []struct {
		name string
		got  int
		want int
	}{
		{"sessTriggerResults", sessTriggerResults, 0},
		{"sessTriggerLatches", sessTriggerLatches, 400},
		{"sessRawHead", sessRawHead, 1400},
		{"sessRawMid", sessRawMid, 1448},
		{"sessDiplomacy", sessDiplomacy, 1856},
		{"sessWon", sessWon, 4362},
		{"sessLose", sessLose, 4370},
		{"diplomacySide", diplomacySide, 50},
		{"triggerResultCount", triggerResultCount, 100},
		{"triggerLatchCount", triggerLatchCount, 1000},
		{"rawHeadLen", rawHeadLen, 48},
		{"rawMidLen", rawMidLen, 400},
	} {
		if c.got != c.want {
			t.Errorf("%s is %d, want %d", c.name, c.got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Actor heads
// ---------------------------------------------------------------------------

func TestActorHeadsAreLocated(t *testing.T) {
	f := open(t, standard())
	if len(f.Actors) != 10 {
		t.Fatalf("%d heads, want %d", len(f.Actors), 10)
	}
	a := f.Actors[0]
	if a.Cell != 0x0500 || a.Row() != 5 || a.Col() != 0 || !a.AtRest() || a.RuntimeID != 1 {
		t.Fatalf("head 0 reads %+v", a)
	}
	if a.MapUnitID != 20 {
		t.Fatalf("head 0 map unit id %d", a.MapUnitID)
	}
	if i, ok := f.ActorByMapUnitID(23); !ok || f.Actors[i].RuntimeID != 4 {
		t.Fatalf("join on unit 23 answered %d %v", i, ok)
	}
	if _, ok := f.ActorByMapUnitID(9999); ok {
		t.Fatal("an id no head carries was joined")
	}
}

// TestADeadHeadIsNotJoined: a dead object keeps its head and its unit id, so a
// join that did not skip it would resume a corpse into a living unit's place.
func TestADeadHeadIsNotJoined(t *testing.T) {
	s := standard()
	s.actors[3].runtimeID = 0
	f := open(t, s)
	if !f.Actors[3].Dead() {
		t.Fatal("runtime id 0 is not being read as dead")
	}
	if _, ok := f.ActorByMapUnitID(23); ok {
		t.Fatal("a dead head was joined")
	}
}

// Exact actor framing does not depend on the raw terrain identity value.
func TestActorTerrainIdentityDoesNotControlIndexing(t *testing.T) {
	for _, state := range []uint32{fixState, 0x06735020, 0x0612a020, 0x7fffffff} {
		s := standard()
		for i := range s.actors {
			s.actors[i].state = state
		}
		if got := len(open(t, s).Actors); got != 10 {
			t.Fatalf("state %#x: %d heads", state, got)
		}
	}
}

// Sparse records and arbitrary terrain identities need no statistical calibration.
func TestExactActorIndexHasNoCalibrationThreshold(t *testing.T) {
	for _, n := range []int{0, 1, 2} {
		s := standard()
		s.actors = s.actors[:n]
		for i := range s.actors {
			s.actors[i].state = 0
		}
		if got := len(open(t, s).Actors); got != n {
			t.Fatalf("got %d actors, want %d", got, n)
		}
	}
}

// ---------------------------------------------------------------------------
// Round trip and edits
// ---------------------------------------------------------------------------

func TestMarshalIsByteIdentical(t *testing.T) {
	for _, c := range []struct {
		name string
		f    fixture
	}{
		{"a whole save", standard()},
		{"a save with no world half", func() fixture { s := standard(); s.noWorld = true; return s }()},
		{"a long map name", func() fixture { s := standard(); s.mapName = "a-long-name.alm"; return s }()},
		{"a save with named trailer dwords", func() fixture {
			s := standard()
			s.trailerTurnTracing, s.trailerScriptTracing = 0x11111111, 0x22222222
			return s
		}()},
	} {
		t.Run(c.name, func(t *testing.T) {
			in := c.f.file()
			got := open(t, c.f).Marshal()
			if !bytes.Equal(got, in) {
				t.Fatalf("round trip differs at %d (%d in, %d out)", firstDiff(in, got), len(in), len(got))
			}
		})
	}
}

func firstDiff(a, b []byte) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return -1
}

// TestAnEditChangesTheFieldAndNOTHINGELSE is what makes a written save
// trustworthy: everything this package did not decode is carried because it was
// never moved.
func TestAnEditChangesTheFieldAndNothingElse(t *testing.T) {
	for _, c := range []struct {
		name  string
		edit  func(*testing.T, *File)
		check func(*testing.T, *File)
	}{
		{"money", func(t *testing.T, f *File) {
			if err := f.SetMoney(0, 777); err != nil {
				t.Fatal(err)
			}
		}, func(t *testing.T, f *File) {
			if f.Players[0].Money != 777 {
				t.Fatalf("money reads %d", f.Players[0].Money)
			}
		}},
		{"the outcome latch", func(t *testing.T, f *File) {
			if err := f.SetOutcome(1, 2); err != nil {
				t.Fatal(err)
			}
		}, func(t *testing.T, f *File) {
			if f.Players[1].Outcome != 2 {
				t.Fatalf("outcome reads %d", f.Players[1].Outcome)
			}
		}},
		{"an actor's cell and fine position", func(t *testing.T, f *File) {
			if err := f.SetActorPosition(2, 40, 41, 0x12, 0x34); err != nil {
				t.Fatal(err)
			}
		}, func(t *testing.T, f *File) {
			a := f.Actors[2]
			if a.Row() != 40 || a.Col() != 41 || a.FineX != 0x12 || a.FineY != 0x34 {
				t.Fatalf("head 2 reads %+v", a)
			}
		}},
		{"a block record", func(t *testing.T, f *File) {
			if err := f.SetBlockRecord(1, 0x11, 0x02); err != nil {
				t.Fatal(err)
			}
		}, func(t *testing.T, f *File) {
			// The static byte still carries no bit 5, so the key set is
			// unchanged and the world half must still locate.
			if f.World == nil || f.World.Blocks[1].Dyn != 0x11 {
				t.Fatalf("block 1 reads %+v", f.World)
			}
		}},
		{"the mission number", func(t *testing.T, f *File) { f.SetMission(11) },
			func(t *testing.T, f *File) {
				if f.Head.Mission != 11 {
					t.Fatalf("mission reads %d", f.Head.Mission)
				}
			}},
		{"the head's PlayerListField", func(t *testing.T, f *File) {
			if err := f.SetPlayerListField(9); err != nil {
				t.Fatal(err)
			}
		}, func(t *testing.T, f *File) {
			if f.Head.PlayerListField != 9 {
				t.Fatalf("PlayerListField reads %d", f.Head.PlayerListField)
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			in := standard().file()
			f := open(t, standard())
			before := append([]byte(nil), f.Body...)
			c.edit(t, f)
			out := f.Marshal()
			// Exactly the bytes of the named field moved.
			n := 0
			for i := range before {
				if before[i] != f.Body[i] {
					n++
				}
			}
			if n == 0 {
				t.Fatal("the edit changed no byte at all")
			}
			if len(f.Body) != len(before) {
				t.Fatalf("the body changed length: %d -> %d", len(before), len(f.Body))
			}
			back, err := Open(out)
			if err != nil {
				t.Fatalf("re-Open: %v", err)
			}
			c.check(t, back)
			// And the label region and tail are untouched.
			if !bytes.Equal(back.LabelRegion, f.LabelRegion) ||
				!bytes.Equal(back.Store, f.Store) || !bytes.Equal(back.TailRest, f.TailRest) {
				t.Fatal("an edit to the body disturbed the label region or the tail")
			}
			_ = in
		})
	}
}

func TestSetLabelKeepsTheDebrisUnderIt(t *testing.T) {
	f := open(t, standard())
	if err := f.SetLabel([]byte("2")); err != nil {
		t.Fatal(err)
	}
	back, err := Open(f.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	if string(back.Label) != "2" {
		t.Fatalf("label reads %q", back.Label)
	}
	if !bytes.HasPrefix(back.LabelRegion, []byte("2\x00e")) {
		t.Fatalf("the region past the NUL was cleared: %q", back.LabelRegion[:8])
	}
	if err := f.SetLabel(make([]byte, labelLen)); err == nil {
		t.Fatal("a label with no room for its NUL was accepted")
	}
}

func TestEditsRefuseAnIndexOutOfRange(t *testing.T) {
	f := open(t, standard())
	for _, err := range []error{
		f.SetMoney(9, 1), f.SetOutcome(9, 1), f.SetOutcome(0, 3),
		f.SetActorPosition(99, 1, 1, 0, 0), f.SetActorPosition(0, 256, 1, 0, 0),
		f.SetBlockRecord(99, 0, 0), f.SetField54(9, 1), f.SetField4C(9, 1),
	} {
		if err == nil {
			t.Fatal("an out-of-range edit was accepted")
		}
	}
}

// TestSaturationIsREPRODUCED, not corrected: the original clamps these two
// fields on the way out, so a value above the ceiling must not survive a write
// here either.
func TestSaturationIsReproduced(t *testing.T) {
	f := open(t, standard())
	if err := f.SetField54(0, 0x10000); err != nil {
		t.Fatal(err)
	}
	if err := f.SetField4C(0, 0x8000); err != nil {
		t.Fatal(err)
	}
	back, err := Open(f.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	f54, _ := back.Players[0].Field("F54")
	f4c, _ := back.Players[0].Field("F4C")
	if f54 != saturate || f4c != saturate {
		t.Fatalf("saturated fields read %d/%d", f54, f4c)
	}
}

// ---------------------------------------------------------------------------
// Trailer
// ---------------------------------------------------------------------------

// TestTrailerDwordsAreNamedAndResidualCarried decodes world+0x118's leading
// two dwords as File.Trailer.TurnTracing/ScriptTracing (SAV-647, SAV-694,
// SAV-791) and confirms the remaining 98 dwords land in Residual UNCHANGED —
// this package names no meaning for them (SAV-790) and must not reinterpret
// what it cannot explain.
func TestTrailerDwordsAreNamedAndResidualCarried(t *testing.T) {
	s := standard()
	s.trailerTurnTracing, s.trailerScriptTracing = 0x11111111, 0x22222222
	f := open(t, s)
	if f.Trailer.TurnTracing != 0x11111111 {
		t.Fatalf("TurnTracing = %#x, want %#x", f.Trailer.TurnTracing, 0x11111111)
	}
	if f.Trailer.ScriptTracing != 0x22222222 {
		t.Fatalf("ScriptTracing = %#x, want %#x", f.Trailer.ScriptTracing, 0x22222222)
	}
	var wantResidual [98]uint32 // the fixture writes this block as zero
	if f.Trailer.Residual != wantResidual {
		t.Fatalf("Residual = %v, want all zero", f.Trailer.Residual)
	}
}

// TestTrailerResidualIsByteIdenticalOnReExport confirms a Marshal round trip
// changes not one of the 392 residual trailer bytes even when the named
// dwords ahead of them are non-zero, distinguishing "this package names
// dwords 0/1" from "this package rewrites the block it decoded."
func TestTrailerResidualIsByteIdenticalOnReExport(t *testing.T) {
	s := standard()
	s.trailerTurnTracing, s.trailerScriptTracing = 0x11111111, 0x22222222
	in := s.file()
	out := open(t, s).Marshal()
	if !bytes.Equal(in, out) {
		t.Fatalf("round trip differs at %d (%d in, %d out)", firstDiff(in, out), len(in), len(out))
	}
}

package sav

import "fmt"

// obfuscator is the constant the original XORs two Player dwords with. It is an
// INVOLUTION, so the same operation applies in both directions — and a reader
// that skips it sees about 1.54 billion gold rather than the hundred that is
// there.
const obfuscator = 0x5c073f4d

// saturate is the ceiling the original's writer silently clamps two Player
// fields to on the way out. This package clamps them the same way (spec L-5): a
// writer that preserved a larger value would write a file saying something the
// original cannot mean.
const saturate = 0x7fff

// reservedDwords is the run of eleven dwords between the map name and the
// mission number. Their STORE ORDER IS NOT ASCENDING in the original's own
// fields, which is why they are carried as a block and not named one by one.
const reservedDwords = 11

// Head is the campaign half's fixed head, decoded in one straight run from body
// offset 0. Offsets after the map name shift with its length, so they are
// computed and never tabled.
type Head struct {
	// CounterA is the world's subtick and CounterB its independent fulltick
	// (SAV-HEAD-025). The original four-save quotient agreement is not a LOAD
	// normalization rule; later source pairs distinguish the two clocks.
	CounterA, CounterB uint32

	// MapName is the map this save names, WITHOUT its directory. It is not
	// what says whether a mission is in progress — see Mission.
	MapName string

	// MapNameOff is where the length byte of MapName sits in the body.
	MapNameOff int

	// Reserved is the eleven dwords, in the order the file stores them.
	Reserved [reservedDwords]uint32

	// Mission is the campaign mission number, and IT is what says whether a
	// mission is in progress: a save taken between missions carries 0 here
	// while MapName still holds the previous mission's name.
	Mission uint32

	// Difficulty is the stored difficulty; the original keeps it only if it
	// is 1..3.
	Difficulty uint32

	// PlayerListField is the player list's own dword. Its meaning is not
	// decoded.
	PlayerListField uint32

	// PlayerCount is the number of archive reference slots, including nulls and repeats.
	PlayerCount uint32

	// MissionOff, DifficultyOff and End are body offsets: the first two for
	// editing, End for where the object stream begins.
	MissionOff, DifficultyOff, End int
}

// headFixed is everything after the map name: eleven dwords and four more.
const headFixed = reservedDwords*4 + 16

func (f *File) readHead() error {
	b := f.Body
	if len(b) < 9 {
		return fmt.Errorf("sav: decoded stream is %d bytes, too short for a head", len(b))
	}
	h := Head{CounterA: u32(b, 0), CounterB: u32(b, 4), MapNameOff: 8}
	n := int(b[8])
	if n == 0xff {
		return fmt.Errorf("sav: extended or Unicode map CString is unsupported")
	}
	if 9+n+headFixed > len(b) {
		return fmt.Errorf("sav: map name of %d bytes at 8 overruns a %d-byte stream", n, len(b))
	}
	h.MapName = string(b[9 : 9+n])
	p := 9 + n
	for i := 0; i < reservedDwords; i++ {
		h.Reserved[i] = u32(b, p+4*i)
	}
	p += reservedDwords * 4
	h.MissionOff, h.DifficultyOff = p, p+4
	h.Mission, h.Difficulty = u32(b, p), u32(b, p+4)
	h.PlayerListField, h.PlayerCount = u32(b, p+8), u32(b, p+12)
	h.End = p + 16
	f.Head = h
	return nil
}

// Player is one distinct roster object, as the class layout in class.go reads it.
//
// The fields below are a VIEW of that decode and not a second copy of it:
// Fields carries every member the layout names, with the offset it was read at,
// and the named fields here are the ones a caller has a reason to ask for
// without knowing a member name. Nothing here restates where a field sits.
type Player struct {
	// Off is the body offset of the record's first byte.
	Off int

	// Fields is the whole record as its layout read it.
	Fields Fields

	// Name is the slot's name, 8-bit text.
	Name string

	// Slot is the 1-based type-5 slot id. The file stores it TWICE, as a u16
	// and then as a u32, and this package accepts a record only where the two
	// agree — that disagreement is the cheapest misalignment detector the
	// record has.
	Slot uint16

	// Participant retains the complete Player control word (SESS-074).
	// Consumers that select zero test the whole word; it is not a Boolean.
	Participant uint32

	// Money is the purse, AFTER the involution the layout applies.
	Money uint32

	// Outcome is the mission-outcome latch: 0 in progress, 1 complete, 2
	// failed. It lives HERE, in the campaign half, so it is in every save
	// including one taken between missions — which is the whole reason it is
	// the flag and the session block's win counter is not.
	Outcome uint8
}

// Field answers one member of the record by the layout's own name, for the
// fields this package has no better name for than the offset they occupy.
func (p Player) Field(name string) (uint32, bool) {
	v, ok := p.Fields.Value[name]
	return v, ok
}

// playerTag is the u16 that introduces every Player instance after the first.
// Player is the first class the stream introduces in every corpus file, so its
// class index is 1 and its tag is 0x8000|1.
const playerTag = 0x8001

// classIntro is the two bytes that introduce a NEW class: FF FF, then a u16
// schema, a u16 name length and that many ASCII bytes.
const classIntro = 0xffff

// maxPlayers bounds the supported type-5 slot IDs, not archive reference slots.
const maxPlayers = 16

// playerAt parses one record at off and reports whether it is one.
//
// It runs the class layout and then checks equal slot ids, a supported slot,
// and an outcome latch in its three-value range. Those are
// acceptance and not decoding, which is why they are here and the field
// positions are not.
func (f *File) playerAt(off int) (Player, bool) {
	c := Lookup("Player")
	fields, err := c.decode(f.Body, off)
	if err != nil {
		return Player{}, false
	}
	slot := fields.Value["Slot"]
	if slot != fields.Value["SlotAgain"] || slot == 0 || slot > maxPlayers {
		return Player{}, false
	}
	if fields.Value["Outcome"] > 2 {
		return Player{}, false
	}
	return Player{
		Off:         off,
		Fields:      fields,
		Name:        fields.Text["Name"],
		Slot:        uint16(slot),
		Participant: fields.Value["Participant"],
		Money:       fields.Value["Money"],
		Outcome:     uint8(fields.Value["Outcome"]),
	}, true
}

// setPlayer writes one member of one player's record through the layout.
func (f *File) setPlayer(i int, name string, v uint32) error {
	if i < 0 || i >= len(f.Players) {
		return fmt.Errorf("sav: player %d of %d", i, len(f.Players))
	}
	p := &f.Players[i]
	if err := Lookup("Player").set(f.Body, p.Fields, name, v); err != nil {
		return err
	}
	// Re-read the record so the view and the bytes cannot disagree — which
	// is also how a saturated write reports what it actually stored.
	if re, ok := f.playerAt(p.Off); ok {
		*p = re
	}
	return nil
}

// SetMoney writes a player's purse. The involution is the LAYOUT's, not this
// function's.
func (f *File) SetMoney(i int, v uint32) error { return f.setPlayer(i, "Money", v) }

// SetOutcome writes a player's mission-outcome latch.
func (f *File) SetOutcome(i int, v uint8) error {
	if v > 2 {
		return fmt.Errorf("sav: outcome %d is not 0 (in progress), 1 (complete) or 2 (failed)", v)
	}
	return f.setPlayer(i, "Outcome", uint32(v))
}

// SetField54 and SetField4C write the two fields the original SATURATES at
// 32767 on the way out. The clamp is the layout's, so it cannot be forgotten by
// a second caller.
func (f *File) SetField54(i int, v uint32) error { return f.setPlayer(i, "F54", v) }
func (f *File) SetField4C(i int, v uint32) error { return f.setPlayer(i, "F4C", v) }

// SetMission writes the campaign mission number, and SetDifficulty the
// difficulty. They are the two head fields a caller has a reason to change: the
// first is what says a mission is in progress, the second is the only head field
// the original validates on load.
func (f *File) SetMission(v uint32) {
	put32(f.Body, f.Head.MissionOff, v)
	f.Head.Mission = v
}

func (f *File) SetDifficulty(v uint32) {
	put32(f.Body, f.Head.DifficultyOff, v)
	f.Head.Difficulty = v
}

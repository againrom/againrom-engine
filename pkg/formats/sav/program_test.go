package sav

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// The walk fixture. Every byte below is written from the serializer programmes
// in knowledge/formats/sav/format.md; nothing here is copied out of a game file,
// and no test in this package reads one (golden rule 2).
//
// This fixture authors the complete owner graph and document envelope.
// ---------------------------------------------------------------------------

// stream builds an object stream, keeping the archive's shared index counter the
// way the writer does: a class record takes the next index and its instance
// takes the one after.
//
// IT EMITS IN FILE ORDER AND NEVER COMPOSES A NESTED RECORD SEPARATELY, because
// an object nested inside another takes an index between its parent's and its
// parent's next sibling's. A fixture that built a child's bytes first and
// appended them later would mint the indices in the wrong order and then agree
// with a walker that made the same mistake.
type stream struct {
	b       []byte
	next    uint16
	classes map[string]uint16
}

func newStream() *stream {
	return &stream{next: 1, classes: map[string]uint16{}}
}

// obj writes the tag that introduces one object of class name. The caller
// appends the record's own bytes after it.
func (s *stream) obj(name string) {
	if idx, ok := s.classes[name]; ok {
		le16(&s.b, instanceTagBit|idx)
	} else {
		le16(&s.b, classIntro)
		le16(&s.b, 1)
		le16(&s.b, uint16(len(name)))
		s.b = append(s.b, name...)
		s.classes[name] = s.next
		s.next++
	}
	s.next++
}

// null writes the null arm of an object reference.
func (s *stream) null() { le16(&s.b, 0) }

// backref writes a reference to an object already read, by its index.
func (s *stream) backref(i uint16) { le16(&s.b, i) }

func (s *stream) u8(v uint8)   { s.b = append(s.b, v) }
func (s *stream) u16(v uint16) { le16(&s.b, v) }
func (s *stream) u32(v uint32) { le32(&s.b, v) }
func (s *stream) raw(n int, fill byte) {
	s.b = append(s.b, bytes.Repeat([]byte{fill}, n)...)
}
func (s *stream) bytes(b []byte) { s.b = append(s.b, b...) }

// u16list writes one stpU16List site: its count, then each element.
func (s *stream) u16list(v []uint16) {
	s.u16(uint16(len(v)))
	for _, e := range v {
		s.u16(e)
	}
}
func (s *stream) cstr(v string) {
	s.b = append(s.b, byte(len(v)))
	s.b = append(s.b, v...)
}

// token writes the 37-byte placeable-object head.
func (s *stream) token(cell uint16, fineX, fineY uint8, runtimeID uint32, defRow uint8,
	mapUnitID uint16) {

	s.u16(cell)
	s.u16(cell)
	s.u8(fineX)
	s.u8(fineY)
	s.u16(0x0777)
	s.u32(fixState)
	s.u32(runtimeID)
	s.u8(defRow)
	s.u16(0x0021)
	// The map unit id is the LOW u16 of this dword.
	s.u32(uint32(mapUnitID))
	s.u16(0x1818)
	s.u32(0x1c1c1c1c)
	s.u32(0xa0000000 | uint32(runtimeID)) // the identity key
	s.u32(0)                              // the reference: null
}

// wantPiece is one item a fixture character wears or carries.
type wantPiece struct {
	class string
	code  uint16
	row   uint8
	stack uint16
}

// item writes one Item, Weapon, Armor or Shield record.
func (s *stream) item(p wantPiece) {
	s.obj(p.class)
	s.token(0x0000, 0, 0, 0, p.row, 0)
	s.u32(0) // Item's effect list: empty
	s.u16(p.code)
	s.u16(p.stack)
	s.u8(0x44)
	s.u8(0x45)
	s.u8(0x46)
	s.u16(0x4848)
	s.u16(0x4a4a)
	s.u8(0x47)
	switch p.class {
	case "Shield":
		s.raw(22, 0x50)
	case "Armor":
		s.raw(22, 0x52)
		s.u8(p.row)
	case "Weapon":
		s.raw(24, 0x52)
		s.raw(22, 0x6a)
		s.u8(0x50)
		s.null() // the weapon's spell reference
	}
}

// wantChar is one fixture character.
type wantChar struct {
	name         string
	cell         uint16
	fineX, fineY uint8
	runtimeID    uint32
	defRow       uint8
	mapUnitID    uint16
	stats        [UnitStatWords]uint16
	skillLevels  [CharacterSkillSlots]uint16
	skillXP      [CharacterSkillSlots]uint32
	experience   uint32
	stage        uint8
	// sight is +0xa4 (Fields.Sight), a u16 in 1/256 cell whose high byte is the
	// whole-cell scan range (SAV-795).
	sight   uint16
	worn    []wantPiece   // the twelve armour references
	weapon  *wantPiece    // +0x74
	shield  *wantPiece    // +0x78
	items   []wantPiece   // the container at +0x7c
	spells  int           // the Spellbook's declared element count, 0 for absent
	book    func(*stream) // optional explicit presence flag and sparse slots
	journal int           // the CDWordArray length, -1 for no Diary at all
	// staticRoute/dynamicRoute are the two embedded u16 lists at +0x15c/+0x178;
	// nil keeps the empty list every fixture predating that story wrote.
	staticRoute, dynamicRoute []uint16
	// mover, when non-nil, replaces the raw 180-byte block's default 0x54
	// filler. It must be exactly 180 bytes.
	mover []byte
	// reference, when non-zero, replaces the Token head's null owner reference.
	reference uint32
}

// human writes one Human record: Unit's programme, then Humanoid's 24 raw bytes,
// its twelve worn references and the thirteenth.
func (s *stream) human(c wantChar) {
	s.obj("Human")
	s.token(c.cell, c.fineX, c.fineY, c.runtimeID, c.defRow, c.mapUnitID)
	if c.reference != 0 {
		binary.LittleEndian.PutUint32(s.b[len(s.b)-4:], c.reference)
	}
	s.u32(0)                  // the effect list at +0x20
	s.u16list(c.staticRoute)  // u16list +0x15c
	s.u16list(c.dynamicRoute) // u16list +0x178
	s.u16(0xa6a6)
	for _, level := range c.skillLevels {
		s.u16(level)
	}
	s.raw(10, 0xa6)
	s.raw(22, 0xbe)
	s.raw(24, 0x14)
	s.raw(64, 0xd4)
	if c.mover != nil {
		s.bytes(c.mover)
	} else {
		s.raw(180, 0x54)
	}
	s.raw(148, 0x58)
	s.u16(0) // u16list *(*(+0x158)+0x90)
	// The store arm's first nineteen bytes.
	s.u8(0x49)
	s.u8(0x4a)
	s.u8(0x4b)
	s.u8(0x4c)
	s.raw(4, 0x50)
	s.raw(4, 0x54)
	s.raw(4, 0x58)
	s.u8(0x60)
	s.u8(0x61)
	s.u8(0x6c)
	if c.weapon != nil {
		s.item(*c.weapon)
	} else {
		s.null()
	}
	if c.shield != nil {
		s.item(*c.shield)
	} else {
		s.null()
	}
	s.cstr(c.name)
	for _, v := range c.stats {
		s.u16(v)
	}
	s.u8(0xa2)
	s.u8(0xa3)
	s.u16(0xa0a0)
	s.u16(c.sight)
	s.u8(0x2c)
	s.u32(c.experience)
	s.u8(0x34)
	s.u8(0x35)
	s.u8(0x36)
	s.u32(0x38383838)
	s.u8(c.stage)
	s.u32(0x48484848)
	s.u32(0x44444444)
	s.null() // the reference at +0x68
	// The inventory presence flag, and the container when it is set.
	if len(c.items) == 0 {
		s.u8(0)
	} else {
		s.u8(1)
		s.u32(uint32(len(c.items)))
		for _, p := range c.items {
			s.item(p)
		}
		s.u32(0x1c1c1c1c)
		s.u32(0x20202020)
	}
	// The spellbook presence flag, and the book when it is set. A book carries
	// n and n-1 references: index 0 is skipped in both directions.
	if c.book != nil {
		c.book(s)
	} else if c.spells == 0 {
		s.u8(0)
	} else {
		s.u8(1)
		s.u32(0x18181818)
		s.u32(uint32(c.spells))
		for i := 1; i < c.spells; i++ {
			s.obj("Spell")
			s.u8(uint8(i))
			s.u8(0x09)
			s.u8(0x0a)
			s.u16(0x0c0c)
			s.u32(0xb0000000 | uint32(i))
		}
	}
	s.u32(0x5c5c5c5c)
	s.u32(0x64646464)
	s.u32(0x44444444)
	s.u32(0x40404040)
	s.u8(0x48)
	// Humanoid's six per-skill XP fields.
	for _, xp := range c.skillXP {
		s.u32(xp)
	}
	for i := 0; i < 12; i++ {
		if i < len(c.worn) {
			s.item(c.worn[i])
			continue
		}
		s.null()
	}
	if c.journal < 0 {
		s.null()
		return
	}
	s.obj("Diary")
	s.u16(uint16(c.journal))
	s.raw(4*c.journal, 0xd4)
	s.u16(uint16(c.journal))
	s.raw(2*c.journal, 0xd2)
	s.u32(0)
}

// The Player's own inline Diary in a fixture: two array lengths that are
// different from each other, so a programme that read one count for both would
// end the record at the wrong offset instead of merely reading the wrong
// number. The corpus writes 119 for both, which cannot tell those apart.
const (
	playerJournal      = 5
	playerJournalWords = 3
)

// heroFixtureID is the runtime id of the character a fixture's Player names as
// the participant's own starting character by default.
const heroFixtureID = 1

// afterPlayerList is the marker a fixture writes where SAV-DOC-053's next item
// begins, which is one byte past the Player record.
const afterPlayerList = 0

// walkFixture is a whole synthetic save whose object graph a walk can read.
type walkFixture struct {
	mapName string
	mission uint32
	chars   []wantChar
	// heroKey is the identity key the Player names as the participant's own
	// starting character. Zero takes the default, which is the key token()
	// gives the character carrying heroFixtureID.
	heroKey uint32
	// groups splits chars into that many group records, so a walk over more
	// than one group is exercised: the actor lists are separate lists and a
	// walker that read only the first would still pass on one group.
	groups       int
	spellEffects func(*stream)
	// morePlayers writes further top-level Player records after the first.
	morePlayers []func(*stream)
}

func (f walkFixture) body() []byte {
	s := newStream()
	le32(&s.b, 0x50)
	le32(&s.b, 0x05)
	s.b = append(s.b, byte(len(f.mapName)))
	s.b = append(s.b, f.mapName...)
	for i := 0; i < reservedDwords; i++ {
		le32(&s.b, 0)
	}
	le32(&s.b, f.mission)
	le32(&s.b, 2)
	le32(&s.b, 6)
	le32(&s.b, uint32(1+len(f.morePlayers)))

	s.obj("Player")
	// The seventeen fields, in playerMembers' own order.
	s.cstr("Danath")
	s.u16(1)
	s.u32(1)
	s.raw(8, 0x10)
	s.u8(0x02)
	s.u32(0) // Participant: the human side
	s.u16(0x0002)
	s.u32(700 ^ obfuscator)
	s.u8(0) // Outcome: in progress
	s.u8(0x3d)
	s.u32(0x11223344 ^ obfuscator)
	s.u32(0x50505050)
	s.u16(0x0123)
	s.u16(0x0456)
	s.u32(0x58585858)
	// The participant's own starting character, by identity key. token()
	// writes an actor's key as 0xa0000000|runtimeID, so the default names the
	// character carrying runtime id 1 — which is what every corpus save does.
	// A fixture that sets heroKey names somebody else, or nobody.
	heroKey := f.heroKey
	if heroKey == 0 {
		heroKey = 0xa0000000 | heroFixtureID
	}
	s.u32(heroKey)
	s.u32(0xa1a1a1a1) // the Player's own identity key

	groups := f.groups
	if groups < 1 {
		groups = 1
	}
	s.u32(uint32(groups))
	for g := 0; g < groups; g++ {
		s.u16(0) // u16list +0x20
		s.raw(80, 0x3c)
		s.u16(0) // u16list *(+0x4c)
		var mine []wantChar
		for i, c := range f.chars {
			if i%groups == g {
				mine = append(mine, c)
			}
		}
		s.u32(uint32(len(mine)))
		for _, c := range mine {
			s.human(c)
		}
		s.u32(0x1c1c1c1c)
		s.u32(0x40404040)
		s.u32(0x44444444)
	}
	s.raw(32, 0x99)

	// THE PLAYER'S OWN Diary, INLINE (SAV-PLDIARY-054): no tag introduces it
	// and it takes no index. Its trailing dword is a reference resolved through
	// the identity map on load, and on every corpus save it is the enclosing
	// Player's own key — which is the fixture's value here too, so a programme
	// that landed those four bytes anywhere else would read something else.
	s.u16(playerJournal)
	s.raw(4*playerJournal, 0xd4)
	s.u16(playerJournalWords)
	s.raw(2*playerJournalWords, 0xd2)
	s.u32(0xa1a1a1a1)
	for _, write := range f.morePlayers {
		write(s)
	}

	// Complete world envelope after the Player list.
	s.u32(afterPlayerList) // empty dead list
	s.u8(1)
	s.u32(0) // Buildings
	if f.spellEffects != nil {
		f.spellEffects(s)
	} else {
		s.u32(0) // SpellEffects
	}
	s.u16(0) // terrain blocks
	s.u16(0) // terrain cells
	s.raw(4+4374, 0)
	s.u32(0) // Sacks
	s.u32(0xbadface1)
	s.u32(0)
	s.raw(400, 0)
	if len(s.b)%2 != 0 {
		s.u8(0)
	}
	return s.b
}

func (f walkFixture) file() []byte {
	blob := Compress(f.body())
	out := make([]byte, headerLen)
	copy(out, Magic)
	binary.LittleEndian.PutUint32(out[4:], uint32(headerLen+len(blob)))
	binary.LittleEndian.PutUint32(out[8:], MinVersion)
	binary.LittleEndian.PutUint32(out[12:], uint32(len(blob)))
	out = append(out, blob...)
	region := make([]byte, labelLen)
	copy(region, "1234")
	region[4] = 0
	out = append(out, region...)
	return append(out, "&YA1"...)
}

// hero is one fully furnished character: four statistics, both pool pairs at
// values that differ from their maxima, a weapon, a shield, three armour pieces,
// two pack items, a spellbook and a journal.
func hero() wantChar {
	var c wantChar
	c = wantChar{
		name: "Danath", cell: 0x4211, fineX: 0x80, fineY: 0x80,
		runtimeID: 1, defRow: 26, mapUnitID: 0,
		weapon:  &wantPiece{class: "Weapon", code: 0x1106, row: 6, stack: 1},
		shield:  &wantPiece{class: "Shield", code: 0x0205, row: 5, stack: 1},
		worn:    []wantPiece{{class: "Armor", code: 0xb70f, row: 15, stack: 1}},
		items:   []wantPiece{{class: "Item", code: 0x0e1c, row: 28, stack: 3}},
		spells:  4,
		journal: 6,
		sight:   0xa4a4,
	}
	c.stats = [UnitStatWords]uint16{
		StatBody: 41, StatReaction: 35, StatMind: 20, StatSpirit: 15,
		StatSpeed: 19, StatOwnWeight: 39, StatLoad: 56, StatCapacity: 411,
		StatHealth: 107, StatHealthMax: 131, StatHealthRegen: 100,
		StatMana: 30, StatManaMax: 90, StatManaRegen: 50,
	}
	c.skillLevels = [CharacterSkillSlots]uint16{1, 2, 3, 4, 5, 6}
	c.skillXP = [CharacterSkillSlots]uint32{11, 22, 33, 44, 55, 66}
	c.experience = 231
	return c
}

// merc is a character the map placed and the player then took into his group: he
// carries a map unit id, which is what makes the withdrawal question exist.
func merc() wantChar {
	c := wantChar{
		name: "Sarindar", cell: 0x1e21, fineX: 0x43, fineY: 0x43,
		runtimeID: 111, defRow: 201, mapUnitID: 136,
		worn:    []wantPiece{{class: "Armor", code: 0x1502, row: 2, stack: 1}},
		journal: -1,
		sight:   0xa4a4,
	}
	c.stats = [UnitStatWords]uint16{
		StatBody: 20, StatReaction: 25, StatMind: 30, StatSpirit: 35,
		StatHealth: 42, StatHealthMax: 42, StatHealthRegen: 100, StatManaRegen: 50,
	}
	c.skillLevels = [CharacterSkillSlots]uint16{6, 5, 4, 3, 2, 1}
	c.skillXP = [CharacterSkillSlots]uint32{66, 55, 44, 33, 22, 11}
	c.experience = 231
	return c
}

func walkOpen(t *testing.T, f walkFixture) *File {
	t.Helper()
	sf, err := Open(f.file())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return sf
}

// ---------------------------------------------------------------------------

// TestTheWalkReachesTheParticipantsOwnCharacters is the whole point of the
// complete object programme, including the nested character fields.
func TestTheWalkReachesTheParticipantsOwnCharacters(t *testing.T) {
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10,
		chars: []wantChar{hero(), merc()}})
	chars, err := f.Party()
	if err != nil {
		t.Fatalf("Party: %v", err)
	}
	if len(chars) != 2 {
		t.Fatalf("the walk reached %d characters, want 2", len(chars))
	}
	h, want := chars[0], hero()
	if h.Name != want.name || h.Class != "Human" {
		t.Errorf("character 0 is %q of class %s", h.Name, h.Class)
	}
	if h.Cell != want.cell || h.FineX != want.fineX || h.FineY != want.fineY {
		t.Errorf("position reads cell %#04x fine %#02x,%#02x", h.Cell, h.FineX, h.FineY)
	}
	if h.Row() != 0x42 || h.Col() != 0x11 {
		t.Errorf("the cell unpacks to (col %d, row %d)", h.Col(), h.Row())
	}
	if h.RuntimeID != want.runtimeID || h.MapUnitID != 0 || h.DefRow != want.defRow {
		t.Errorf("id %d, map unit %d, definition row %d", h.RuntimeID, h.MapUnitID, h.DefRow)
	}
	if h.Stats != want.stats {
		t.Errorf("statistics read %v, want %v", h.Stats, want.stats)
	}
	if h.SkillLevels != want.skillLevels || h.SkillXP != want.skillXP || h.Experience != want.experience {
		t.Errorf("progress read levels=%v xp=%v total=%d, want levels=%v xp=%v total=%d",
			h.SkillLevels, h.SkillXP, h.Experience, want.skillLevels, want.skillXP, want.experience)
	}
	// THE PAIRS ARE UNEQUAL IN THE FIXTURE ON PURPOSE. A reader that pointed
	// both members of a pair at one word would pass a fixture whose health and
	// maximum agreed, which is what every corpse-free save looks like from the
	// outside.
	if h.Stat(StatHealth) == h.Stat(StatHealthMax) || h.Stat(StatMana) == h.Stat(StatManaMax) {
		t.Fatal("the fixture's pools are equal, so this test cannot tell the pair apart")
	}
	if h.Stat(StatHealth) != 107 || h.Stat(StatHealthMax) != 131 {
		t.Errorf("health reads %d/%d, want 107/131", h.Stat(StatHealth), h.Stat(StatHealthMax))
	}
	if h.Stat(StatMana) != 30 || h.Stat(StatManaMax) != 90 {
		t.Errorf("mana reads %d/%d, want 30/90", h.Stat(StatMana), h.Stat(StatManaMax))
	}
	// The second character is in the SECOND position of the same actor list, so
	// reading him at all is what says the walk stepped the first record exactly.
	if m := chars[1]; m.Name != "Sarindar" || m.MapUnitID != 136 || m.Stat(StatHealthMax) != 42 {
		t.Errorf("character 1 reads %q map unit %d health max %d",
			m.Name, m.MapUnitID, m.Stat(StatHealthMax))
	}
}

// TestTheWalkReadsWhatACharacterWearsAndCarries. The weapon, the shield and the
// twelve armour references are one equipment set in the file and the piece's own
// code says which slot it belongs to, so this test checks the CODES rather than
// any site-to-slot mapping.
func TestTheWalkReadsWhatACharacterWearsAndCarries(t *testing.T) {
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{hero()}})
	chars, err := f.Party()
	if err != nil {
		t.Fatalf("Party: %v", err)
	}
	h := chars[0]
	if len(h.Worn) != 3 {
		t.Fatalf("%d worn pieces, want the weapon, the shield and one armour: %+v", len(h.Worn), h.Worn)
	}
	for i, want := range []wantPiece{
		{class: "Weapon", code: 0x1106, row: 6, stack: 1},
		{class: "Shield", code: 0x0205, row: 5, stack: 1},
		{class: "Armor", code: 0xb70f, row: 15, stack: 1},
	} {
		got := h.Worn[i]
		if got.Class != want.class || got.Code != want.code || got.Row != want.row {
			t.Errorf("worn %d reads %+v, want %+v", i, got, want)
		}
	}
	if len(h.Items) != 1 || h.ItemCount != 1 {
		t.Fatalf("%d of %d declared pack items", len(h.Items), h.ItemCount)
	}
	if p := h.Items[0]; p.Class != "Item" || p.Code != 0x0e1c || p.Stack != 3 {
		t.Errorf("the pack item reads %+v", p)
	}
	// The two axes that are READ AND NOT APPLIED: their counts have to survive
	// the walk, because a report that could not say "1 journal, 12 entries"
	// would be back to saying nothing.
	if !h.HasSpellbook || h.SpellCount != 4 {
		t.Errorf("spellbook %v holding %d", h.HasSpellbook, h.SpellCount)
	}
	if !h.HasDiary || h.JournalLen != 6 || h.JournalWords != 6 {
		t.Errorf("diary %v holding %d/%d", h.HasDiary, h.JournalLen, h.JournalWords)
	}
}

// TestTheWalkEndsWhereTheNextItemBegins is the one available proof that the
// programme tiled the record EXACTLY rather than merely not crashing.
// SAV-DOC-053 enumerates the top-level document, so what follows the Player
// list is a known item and the record's end is a known offset. A programme with
// one field at the wrong width ends somewhere else and every field after that
// width is read from the wrong offset.
//
// THIS TEST FAILED TO CATCH A SHORT PROGRAMME ONCE AND ITS OLD FORM IS WHY.
// Before 0149 the Player programme was short by the record's own Diary
// (SAV-PLDIARY-054), and this test asserted that the two words after the record
// were a back-reference and a null. They were: the Diary's CDWordArray count
// read as a tag, and the first two zero bytes of its array read as a
// terminator. An extent test whose expected value can be produced by the bytes
// of the record it is checking proves nothing, so the marker below is a value
// no field of a Player record can hold.
func TestTheWalkEndsWhereTheNextItemBegins(t *testing.T) {
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10,
		chars: []wantChar{hero(), merc()}})
	_, rec, err := f.PartyWalk()
	if err != nil {
		t.Fatalf("PartyWalk: %v", err)
	}
	if rec.End+4 > len(f.Body) {
		t.Fatalf("the record ends at %d of a %d-byte stream", rec.End, len(f.Body))
	}
	if got := u32(f.Body, rec.End); got != afterPlayerList {
		t.Errorf("the dword after the record is %#08x, want the next item's marker %#08x",
			got, uint32(afterPlayerList))
	}
}

// TestThePlayerCarriesItsOwnDiary. SAV-PLDIARY-054 states Player::Serialize's
// third tail call, and without it the record is short by that whole object.
//
// THE TRAILING DWORD IS THE CHECK. Diary::Serialize's last field is a reference
// resolved through the identity map on load, and on every corpus save it is the
// enclosing Player's own identity key. A programme that landed those four bytes
// anywhere else would have to find that same value there by accident.
func TestThePlayerCarriesItsOwnDiary(t *testing.T) {
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10,
		chars: []wantChar{hero()}})
	_, rec, err := f.PartyWalk()
	if err != nil {
		t.Fatalf("PartyWalk: %v", err)
	}
	d := rec.Refs["Diary"]
	if len(d) != 1 {
		t.Fatalf("the Player carries %d Diary records, want 1", len(d))
	}
	if n := d[0].Counts["Journal"]; n != playerJournal {
		t.Errorf("the journal holds %d dwords, want %d", n, playerJournal)
	}
	if n := d[0].Counts["JournalWords"]; n != playerJournalWords {
		t.Errorf("the journal holds %d words, want %d", n, playerJournalWords)
	}
	if got, want := d[0].value("D2C"), rec.value("This"); got != want {
		t.Errorf("the Diary's trailing reference is %#08x, want the Player's own key %#08x",
			got, want)
	}
	// THE ARCHIVE DID NOT NUMBER IT. A Diary written through its owner's
	// vtable takes no index, so filing one would shift every back-reference
	// after it by one.
	if d[0].Index != 0 {
		t.Errorf("the inline Diary took archive index %d, want none", d[0].Index)
	}
	if d[0].End != rec.End {
		t.Errorf("the Diary ends at %d and the Player at %d; the Diary is the record's last "+
			"construct and the two must agree", d[0].End, rec.End)
	}
}

func TestThePlayerNamesTheParticipantsOwnCharacter(t *testing.T) {
	for _, tc := range []struct {
		name    string
		heroKey uint32
		want    int // the index that should carry Hero, or -1 for none
	}{
		{"the file names the character written second", 0, 1},
		{"the file names nobody this walk reached", 0xdeadbeef, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// merc() is written first and hero() second, so a rule that took
			// the head of the list would answer 0 on the first case.
			f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10,
				chars: []wantChar{merc(), hero()}, heroKey: tc.heroKey})
			chars, err := f.Party()
			if err != nil {
				t.Fatalf("Party: %v", err)
			}
			at := -1
			for i, c := range chars {
				if c.Key == 0 {
					t.Errorf("character %d carries no identity key", i)
				}
				if !c.Hero {
					continue
				}
				if at >= 0 {
					t.Fatalf("characters %d and %d both claim to be the participant's own", at, i)
				}
				at = i
			}
			if at != tc.want {
				t.Errorf("the participant's own character is at %d, want %d", at, tc.want)
			}
		})
	}
}

// TestTheWalkReadsEveryGroupsActorList. A player's actors are split across group
// records, each with its own counted list, so a walk that read the first group
// and stopped would pass every single-group fixture.
func TestTheWalkReadsEveryGroupsActorList(t *testing.T) {
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10,
		chars: []wantChar{hero(), merc(), merc()}, groups: 2})
	chars, err := f.Party()
	if err != nil {
		t.Fatalf("Party: %v", err)
	}
	if len(chars) != 3 {
		t.Fatalf("the walk reached %d characters over two groups, want 3", len(chars))
	}
}

// TestTheUnitProgrammeReproducesThePublishedFixedPart. SAV-UNITLEN-045 states a
// Unit record's fixed part term by term and sums it to 603, with three null
// references and a name taking it to 609 + L. That sum is an independent
// statement of the same programme: a field at the wrong width, or a construct in
// the wrong place, changes it.
//
// It is measured off the fixture rather than off a table, so the number comes
// from the bytes the walk actually stepped.
func TestTheUnitProgrammeReproducesThePublishedFixedPart(t *testing.T) {
	// A character wearing and carrying nothing, with no book and no journal:
	// every one of the three Unit references null, and Humanoid's own thirteen
	// null too.
	bare := wantChar{name: "abcdefghijkl", cell: 0x0101, runtimeID: 7, journal: -1}
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{bare}})
	_, rec, err := f.PartyWalk()
	if err != nil {
		t.Fatalf("PartyWalk: %v", err)
	}
	actors := rec.Refs["Actors"]
	if len(actors) != 1 {
		t.Fatalf("%d actors", len(actors))
	}
	a := actors[0]
	// A Human is a Unit plus Humanoid's 24 raw bytes and its thirteen
	// references, each two bytes when null.
	const humanoid = 24 + 13*2
	got := a.End - a.Off - humanoid
	want := 609 + len(bare.name)
	if got != want {
		t.Errorf("the Unit half of the record is %d bytes, want SAV-UNITLEN-045's 609 + %d = %d",
			got, len(bare.name), want)
	}
}

// TestTheWalkRefusesWhatItCannotStep. A class with no programme is not an error
// the walk can work around: the record's length depends on its content, so there
// is no way to step over one.
func TestTheWalkRefusesWhatItCannotStep(t *testing.T) {
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{hero()}})
	// Rewrite the Human class introduction's name to one this package has no
	// programme for. The name length is unchanged, so nothing else moves.
	i := bytes.Index(f.Body, []byte("Human"))
	if i < 0 {
		t.Fatal("the fixture introduces no Human class")
	}
	copy(f.Body[i:], "Tavrn")
	chars, err := f.Party()
	if err == nil {
		t.Fatal("a class with no programme was walked")
	}
	if !strings.Contains(err.Error(), "no serialization programme") {
		t.Errorf("the refusal does not say why: %v", err)
	}
	if len(chars) != 0 {
		t.Errorf("%d characters came back from a walk that failed", len(chars))
	}
}

// TestTheWalkRefusesABackReferenceItHasNotRead. A plain tag names an object
// already in the archive's own map; one that names an index the walk never
// minted is a stream this package is reading at the wrong offset, and answering
// a zero value for it would carry that error forward silently.
func TestTheWalkRefusesABackReferenceItHasNotRead(t *testing.T) {
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{hero()}})
	// The inventory list's single element is an object reference; a tag of
	// 0x00ff there is a back-reference to an index no fixture object holds.
	i := bytes.Index(f.Body, []byte("Item"))
	if i < 4 {
		t.Fatal("the fixture introduces no Item class")
	}
	binary.LittleEndian.PutUint16(f.Body[i-6:], 0x00ff)
	if _, err := f.Party(); err == nil || !strings.Contains(err.Error(), "back-reference") {
		t.Errorf("a dangling back-reference was accepted: %v", err)
	}
}

package sav

import (
	"encoding/binary"
	"errors"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
)

// secondTailFixture is a second-game campaign tail: the pair list, the
// Scenario bank and grid, the availability records, the current ordinal and
// nine client buffers, in file order.
type secondTailFixture struct {
	pairs     [][2]uint32
	bank      map[int]int32
	locations []SecondLocation
	current   uint32
	buffers   [9][]byte
}

func (t secondTailFixture) bytes() []byte {
	var b []byte
	u32 := func(v uint32) { b = binary.LittleEndian.AppendUint32(b, v) }
	u32(uint32(len(t.pairs)))
	for _, p := range t.pairs {
		u32(p[0])
		u32(p[1])
	}
	u32(0x7a7a7a7a)
	u32(0x7b7b7b7b)
	bank := make([]byte, secondBankLen)
	for slot, v := range t.bank {
		binary.LittleEndian.PutUint32(bank[4*slot:], uint32(v))
	}
	b = append(b, bank...)
	// Cell (location L, category C) field k holds 0x100*L + 0x10*C + k.
	for i := range secondGridLen / 4 {
		cell, k := i/5, i%5
		u32(uint32(0x100*(cell/4) + 0x10*(cell%4) + k))
	}
	u32(uint32(len(t.locations)))
	for _, l := range t.locations {
		u32(l.ID)
		u32(l.Kind)
		b = append(b, l.Raw[:]...)
	}
	u32(t.current)
	for i, d := range t.buffers {
		b = binary.LittleEndian.AppendUint16(b, uint16(i))
		b = binary.LittleEndian.AppendUint16(b, 0xb0b0)
		u32(uint32(len(d)))
		b = append(b, d...)
	}
	return b
}

// secondTownTail is a town-like tail: two places, the current one a town.
func secondTownTail() secondTailFixture {
	t := secondTailFixture{
		pairs: [][2]uint32{{3, 4}},
		bank:  map[int]int32{0: 7, 513: -1, 1023: 0x01020304},
		locations: []SecondLocation{
			{ID: 10, Kind: 1, Raw: [16]byte{1, 2, 3}},
			{ID: 1, Kind: 2, Raw: [16]byte{15: 9}},
		},
		current: 1,
	}
	t.buffers[4] = []byte{0xc1, 0xc2, 0xc3}
	return t
}

// secondStore is the state store a second-game file writes before its tail.
func secondStore(inBattle int32) []byte {
	return synth.Reg(0x11, []synth.RegNode{
		{Name: "CurrentState", Kind: 0x01, Children: []synth.RegNode{
			{Name: "InBattle", Kind: 0x02, Int: inBattle},
		}},
		{Name: "SpellBook", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Shortcuts", Kind: 0x06, Ints: []int32{}},
		}},
	})
}

// secondTownFixture is a town-like second-game file: one Player whose group
// holds the hero, no world half, and the town tail.
func secondTownFixture() walkFixture {
	return walkFixture{mapName: "town.alm", mission: 2, chars: []wantChar{hero()},
		second: true, town: true,
		tail: append(secondStore(0), secondTownTail().bytes()...)}
}

func TestSecondTailReadsEachPartToTheEnd(t *testing.T) {
	f := walkOpen(t, secondTownFixture())
	if !f.SecondGame() || f.EnvelopeGame() != "rom2" {
		t.Fatal("a Bsg& file does not open as the second game")
	}
	tail, err := f.SecondTail()
	if err != nil {
		t.Fatalf("SecondTail: %v", err)
	}
	want := secondTownTail()
	if !reflect.DeepEqual(tail.Pairs, want.pairs) || tail.PairTail != [2]uint32{0x7a7a7a7a, 0x7b7b7b7b} {
		t.Fatalf("pairs %v tail %v", tail.Pairs, tail.PairTail)
	}
	if tail.Bank[0] != 7 || tail.Bank[513] != -1 || tail.Bank[1023] != 0x01020304 || tail.Bank[1] != 0 {
		t.Fatalf("bank slots 0, 1, 513, 1023: %d %d %d %d", tail.Bank[0], tail.Bank[1], tail.Bank[513], tail.Bank[1023])
	}
	for l := range 4 {
		for c := range 4 {
			v := int32(0x100*l + 0x10*c)
			want := SecondShopCell{Min: v, Max: v + 1, Draws: v + 2, Bound: v + 3, Mask: uint32(v + 4)}
			if tail.Shops[l][c] != want {
				t.Fatalf("shop cell %d/%d: %+v, want %+v", l, c, tail.Shops[l][c], want)
			}
		}
	}
	if !reflect.DeepEqual(tail.Locations, want.locations) {
		t.Fatalf("locations %+v", tail.Locations)
	}
	current, ok := tail.CurrentLocation()
	if !ok || current != want.locations[1] {
		t.Fatalf("current %+v %v, want the town record", current, ok)
	}
	for i, buf := range tail.Buffers {
		if buf.A != uint16(i) || buf.B != 0xb0b0 || string(buf.Data) != string(want.buffers[i]) {
			t.Fatalf("buffer %d: %+v", i, buf)
		}
	}
}

func TestSecondTailRefusesATailThatDoesNotTile(t *testing.T) {
	good := secondTownTail().bytes()
	if _, err := parseSecondTail(good); err != nil {
		t.Fatalf("control: %v", err)
	}
	long := append(append([]byte(nil), good...), 0)
	if _, err := parseSecondTail(long); err == nil || !strings.Contains(err.Error(), "ends at") {
		t.Fatalf("one byte past the buffers: %v", err)
	}
	if _, err := parseSecondTail(good[:len(good)-1]); err == nil || !strings.Contains(err.Error(), "overruns") {
		t.Fatalf("one byte short: %v", err)
	}
	countOff := 4 + 8 + 8 + secondBankLen + secondGridLen
	huge := append([]byte(nil), good...)
	binary.LittleEndian.PutUint32(huge[countOff:], 1<<20)
	if _, err := parseSecondTail(huge); err == nil || !strings.Contains(err.Error(), "location count") {
		t.Fatalf("a location count past the tail: %v", err)
	}
	pairs := append([]byte(nil), good...)
	binary.LittleEndian.PutUint32(pairs, 1<<20)
	if _, err := parseSecondTail(pairs); err == nil || !strings.Contains(err.Error(), "pair count") {
		t.Fatalf("a pair count past the tail: %v", err)
	}
	bufOff := countOff + 4 + 2*secondNodeLen + 4
	size := append([]byte(nil), good...)
	binary.LittleEndian.PutUint32(size[bufOff+4:], 1<<30)
	if _, err := parseSecondTail(size); err == nil || !strings.Contains(err.Error(), "buffer 0") {
		t.Fatalf("a buffer size past the tail: %v", err)
	}
}

func TestSecondTailNamesNoCurrentPastItsList(t *testing.T) {
	tail := SecondTail{Locations: make([]SecondLocation, 2), Current: 2}
	if _, ok := tail.CurrentLocation(); ok {
		t.Fatal("an ordinal past the list names a record")
	}
}

// The magic decides the profile: the first game's envelope over the same
// store and tail has no second-game tail.
func TestTheFirstGameHasNoSecondTail(t *testing.T) {
	fx := walkFixture{mapName: "t.alm", mission: 2, chars: []wantChar{hero()}, town: true,
		tail: append(secondStore(0), secondTownTail().bytes()...)}
	f := walkOpen(t, fx)
	if f.SecondGame() || f.EnvelopeGame() != "" {
		t.Fatal("an Asg& file opens as the second game")
	}
	if _, err := f.SecondTail(); err == nil {
		t.Fatal("a first-game file answers a second-game tail")
	}
}

// The second game's Player and Human shapes walk to the hero: its Token
// head's u16 word, Unit's 184-byte block, no Humanoid Diary, the Player's
// 2,560 and 36 raw bytes. The experience block opens with the main skill
// (R2-ENGINE-320).
func TestTheSecondGameWalkReadsTheHero(t *testing.T) {
	f := walkOpen(t, secondTownFixture())
	chars, err := f.Party()
	if err != nil {
		t.Fatalf("Party: %v", err)
	}
	if len(chars) != 1 || !chars[0].Hero {
		t.Fatalf("party %d characters, want the hero", len(chars))
	}
	c, want := chars[0], hero()
	if c.Name != want.name || c.Stats != want.stats || c.SkillLevels != want.skillLevels || c.Experience != want.experience {
		t.Fatalf("hero %q stats %v skills %v experience %d", c.Name, c.Stats, c.SkillLevels, c.Experience)
	}
	if c.MainSkill != 11 || c.SkillXP != [CharacterSkillSlots]uint32{0, 22, 33, 44, 55, 66} {
		t.Fatalf("main skill %d experiences %v, want 11 and the five after it", c.MainSkill, c.SkillXP)
	}
	if len(c.Items) != 1 || c.Items[0].Code != 0x0e1c || c.Items[0].Stack != 3 {
		t.Fatalf("bag %+v", c.Items)
	}
	if len(c.Spells) != 3 || c.Spells[2].ID != 3 {
		t.Fatalf("book %+v", c.Spells)
	}
	rec, err := f.Walk()
	if err != nil || rec.value("Money") != 700 {
		t.Fatalf("Player money %d (%v), want 700", rec.value("Money"), err)
	}
}

// The first game's shapes behind a Bsg& magic do not open or walk: the profile, not
// the bytes, picks the programmes.
func TestTheSecondGameRefusesTheFirstGamesShapes(t *testing.T) {
	b := walkFixture{mapName: "t.alm", mission: 2, chars: []wantChar{hero()}, town: true}.file()
	copy(b, SecondGame.Magic)
	f, err := Open(b)
	if err == nil {
		_, err = f.Party()
	}
	if err == nil {
		t.Fatal("first-game records walk under the second game's programmes")
	}
}

// A mission-like second-game file's world half has the second game's session
// extent and the extra Token list after the Sacks (R2-ENGINE-603).
func TestTheSecondGameWorldHalfReadsItsWidths(t *testing.T) {
	fx := secondTownFixture()
	fx.town = false
	fx.tail = append(secondStore(1), secondTownTail().bytes()...)
	f := walkOpen(t, fx)
	_, present, err := f.GroundSacks()
	if err != nil || !present {
		t.Fatalf("GroundSacks present %v: %v", present, err)
	}
	first := fx
	first.second = false
	b := first.file()
	copy(b, SecondGame.Magic)
	g, err := Open(b)
	if err == nil {
		_, _, err = g.GroundSacks()
	}
	if err == nil {
		t.Fatal("the first game's world widths walk under the second game's profile")
	}
}

// The second game's book holds spell 29; the first game's refuses it.
func TestTheSecondGameBookHoldsSpellTwentyNine(t *testing.T) {
	fx := secondTownFixture()
	fx.chars[0].spells = 30
	chars, err := walkOpen(t, fx).Party()
	if err != nil {
		t.Fatalf("Party: %v", err)
	}
	if n := len(chars[0].Spells); n != 29 || chars[0].Spells[28].ID != 29 {
		t.Fatalf("book of %d spells", n)
	}
	first := walkFixture{mapName: "t.alm", mission: 2, chars: []wantChar{hero()}, town: true}
	first.chars[0].spells = 30
	_, err = walkOpen(t, first).Party()
	if !errors.Is(err, ErrSpellbook) {
		t.Fatalf("first-game book with spell 29: %v, want ErrSpellbook", err)
	}
}

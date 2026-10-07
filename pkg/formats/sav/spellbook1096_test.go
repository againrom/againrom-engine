package sav

import (
	"errors"
	"reflect"
	"testing"
)

func spellRecord1096(s *stream, id byte) uint16 {
	s.obj("Spell")
	index := s.next - 1
	s.u8(id)
	s.u8(19)
	s.u8(1)
	s.u16(347)
	s.u32(0x11223300 | uint32(id))
	return index
}

func TestSavedSpellbook1096SparseAliasesAndDetachedValues(t *testing.T) {
	var alias uint16
	a, b := hero(), hero()
	a.name, b.name = "learned", "different"
	a.book = func(s *stream) {
		s.u8(1)
		s.u32(28)
		s.u32(29)
		for slot := 1; slot <= 28; slot++ {
			switch slot {
			case 4:
				alias = spellRecord1096(s, 4)
			case 28:
				spellRecord1096(s, 28)
			default:
				s.null()
			}
		}
	}
	b.book = func(s *stream) {
		s.u8(1)
		s.u32(4)
		s.u32(5)
		s.null()
		s.null()
		s.null()
		s.backref(alias)
	}
	f := walkOpen(t, walkFixture{mission: 10, mapName: "10.alm", chars: []wantChar{a, b}})
	chars, rec, err := f.PartyWalk()
	if err != nil || len(chars) != 2 {
		t.Fatalf("PartyWalk = %d characters, %v", len(chars), err)
	}
	if chars[0].KnownSpells() != 1<<4|1<<28 || chars[1].KnownSpells() != 1<<4 {
		t.Fatalf("character-owned membership = %#x, %#x", chars[0].KnownSpells(), chars[1].KnownSpells())
	}
	first := SavedSpell{Slot: 4, ID: 4, Range: 19, Defensive: 1, ManaCost: 347, ArchiveIndex: alias, Key: 0x11223304}
	if !reflect.DeepEqual(chars[0].Spells[0], first) || !reflect.DeepEqual(chars[1].Spells[0], first) {
		t.Fatalf("alias lost typed fields: %+v %+v", chars[0].Spells, chars[1].Spells)
	}
	if chars[0].SpellCount != 29 || len(chars[0].Spells) != 2 || len(rec.Refs["Actors"][0].SpellSlots) != 28 {
		t.Fatal("array capacity, sparse slots and learned count were conflated")
	}
	chars[0].Spells[0].ID = 8
	if chars[1].Spells[0].ID != 4 {
		t.Fatal("one character owns the other character's mutable projection")
	}
	if rec.Refs["Actors"][0].SpellSlots[3] != rec.Refs["Actors"][1].SpellSlots[3] {
		t.Fatal("record-level archive alias was not retained")
	}
}

func TestSavedSpellbook1096AbsentAndEmptyAreDistinct(t *testing.T) {
	for _, count := range []int{-1, 0, 1, 29} {
		c := hero()
		c.book = func(s *stream) {
			if count < 0 {
				s.u8(0)
				return
			}
			s.u8(1)
			s.u32(0)
			s.u32(uint32(count))
			for i := 1; i < count; i++ {
				s.null()
			}
		}
		chars, err := walkOpen(t, walkFixture{mission: 10, chars: []wantChar{c}}).Party()
		if err != nil || len(chars) != 1 {
			t.Fatalf("count %d: %v", count, err)
		}
		got := chars[0]
		if got.HasSpellbook != (count >= 0) || got.KnownSpells() != 0 || len(got.Spells) != 0 {
			t.Fatalf("count %d: %+v", count, got)
		}
	}
}

func TestSavedSpellbook1096RefusesMalformedMembership(t *testing.T) {
	for _, tc := range []struct {
		name string
		book func(*stream)
	}{
		{"zero ID", func(s *stream) { spellRecord1096(s, 0) }},
		{"unknown ID", func(s *stream) { spellRecord1096(s, 29) }},
		{"slot mismatch", func(s *stream) { spellRecord1096(s, 4) }},
		{"wrong class", func(s *stream) { s.backref(s.next - 1) }},
		{"unknown reference", func(s *stream) { s.backref(0x7fff) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := hero()
			c.book = func(s *stream) { s.u8(1); s.u32(0); s.u32(2); tc.book(s) }
			f, err := Open((walkFixture{mission: 10, chars: []wantChar{c}}).file())
			if err == nil {
				_, err = f.Party()
			}
			if !errors.Is(err, ErrSpellbook) {
				t.Fatalf("malformed spellbook error = %v", err)
			}
		})
	}
	for _, payload := range [][]byte{{2}, {1}, {1, 0, 0, 0, 0, 0xff, 0xff, 0xff, 0xff}} {
		w := &walker{b: payload}
		err := w.step(step{op: stpFlagged, name: "HasSpellbook", sub: programmes["Spellbook"]}, newRecord("Human", 0, 1), 0)
		if !errors.Is(err, ErrSpellbook) {
			t.Fatalf("malformed bytes %x: %v", payload, err)
		}
	}
}

func TestCityDescriptor1096ReadsTheSourceGraphBook(t *testing.T) {
	actor := cityTestHuman(4, 100, "source", 1, 20)
	actor.unit.spellbookFlag, actor.unit.spellbookCount = 1, 7
	actor.unit.spells = make([]*cityObject, 6)
	fields := []byte{6, 9, 1, 17, 0, 1, 2, 3, 4}
	actor.unit.spells[5] = &cityObject{sourceIndex: 7, class: "Spell", spell: &citySpell{fields: fields}}
	desc, err := cityCharacterDescriptor(actor, 100, true)
	if err != nil || !desc.HasSpellbook || desc.KnownSpells != 1<<6 {
		t.Fatalf("source descriptor: %+v %v", desc, err)
	}
	fields[0] = 5
	if _, err := cityCharacterDescriptor(actor, 100, true); !errors.Is(err, ErrSpellbook) {
		t.Fatalf("misindexed graph accepted: %v", err)
	}
}

package sav

import (
	"strings"
	"testing"
)

// deadMerc is merc() torn down, stage 3 and no health, whose Token names owner
// as its Reference. The owner graph carries it without a derive basis.
func deadMerc(owner uint32) wantChar {
	c := merc()
	c.stage, c.stats[StatHealth], c.reference = 3, 0, owner
	return c
}

// otherPlayer writes a second Player, slot 2 with identity key, whose one Group
// holds the object at index member.
func otherPlayer(key uint32, member uint16) func(*stream) {
	return func(s *stream) {
		s.obj("Player")
		s.cstr("Other")
		s.u16(2)
		s.u32(2)
		s.raw(8, 0x10)
		s.u8(0x02)
		s.u32(1)
		s.u16(0x0002)
		s.u32(0 ^ obfuscator)
		s.u8(0)
		s.u8(0x3d)
		s.u32(0 ^ obfuscator)
		s.u32(0x50505050)
		s.u16(0x0123)
		s.u16(0x0456)
		s.u32(0x58585858)
		s.u32(0) // no hero
		s.u32(key)
		s.u32(1)
		s.u16(0)
		s.raw(80, 0x3c)
		s.u16(0)
		s.u32(1)
		s.backref(member)
		s.u32(0x1c1c1c1c)
		s.u32(0)
		s.u32(key)
		s.raw(32, 0x99)
		s.u16(playerJournal)
		s.raw(4*playerJournal, 0xd4)
		s.u16(playerJournalWords)
		s.raw(2*playerJournalWords, 0xd2)
		s.u32(key)
	}
}

func partyOwnerFixture(owner uint32, more ...func(*stream)) walkFixture {
	return walkFixture{mapName: "141.alm", mission: 141, groups: 2,
		chars: []wantChar{hero(), deadMerc(owner)}, morePlayers: more}
}

// A dead Group member of the walked Player whose saved owner Reference names
// no Player is that Player's: the Player's LOAD stamps every member of its own
// Groups with itself after the Token resolved the reference. The walk reads
// the dead character with this Player as its owner.
func TestPartyWalkOwnsADeadMemberWhoseReferenceNamesNoPlayer(t *testing.T) {
	chars, _, err := walkOpen(t, partyOwnerFixture(0xdeadbeef)).PartyWalk()
	if err != nil {
		t.Fatalf("PartyWalk: %v", err)
	}
	var found bool
	for _, c := range chars {
		if c.RuntimeID != 111 {
			continue
		}
		found = true
		if c.Basis == nil || !c.Basis.Human.HasOwner {
			t.Fatalf("dead merc basis %+v, want the walked Player as owner", c.Basis)
		}
	}
	if !found {
		t.Fatal("the walk did not reach the dead merc")
	}
}

// A dead character whose effective owner is another Player is not this
// Player's: a later Player's Group holds it too, and that Player's LOAD
// stamps it last. The walk stops rather than read it as this Player's.
func TestPartyWalkRefusesADeadMemberAnotherPlayerOwns(t *testing.T) {
	const other = 0xa2a2a2a2
	chars, _, err := walkOpen(t, partyOwnerFixture(other)).PartyWalk()
	if err != nil {
		t.Fatalf("one-Player PartyWalk: %v", err)
	}
	var merc uint16
	for _, c := range chars {
		if c.RuntimeID == 111 {
			merc = c.ArchiveIndex
		}
	}
	if merc == 0 {
		t.Fatal("the walk did not reach the dead merc")
	}
	_, _, err = walkOpen(t, partyOwnerFixture(other, otherPlayer(other, merc))).PartyWalk()
	if err == nil || !strings.Contains(err.Error(), "another Player") {
		t.Fatalf("PartyWalk err %v, want a refusal naming another Player", err)
	}
}

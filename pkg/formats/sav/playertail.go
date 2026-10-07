package sav

import "fmt"

// PlayerTailLen is the width of the raw block Player::Serialize's tail
// writes as one literal CArchive::Write/Read call (SAV-662): the 32 bytes at
// runtime offset Player+0x30, past the seventeen-field straight run
// (SAV-PLAYER-028, playerMembers) and the group list (stpGroups), and before
// the inline Diary (SAV-PLDIARY-054). program.go's "PRaw32" member is this
// exact span.
const PlayerTailLen = 32

// PlayerTailFormationByte is PRaw32's own index 0x1f. AI-FORM-037 (active,
// consulted by this story and not re-derived here) already gives its whole
// surface: the player constructor's own default (2), the one setter, and the
// two group-move readers.
const PlayerTailFormationByte = 0x1f

// PlayerTail is Player+0x30's own 32-byte block, decoded as far as a
// promoted claim names it. Raw is a byte-for-byte copy; Formation() names the
// one byte a claim gives meaning to. The other 31 bytes are carried in Raw
// and not further named: SAV-662 finds two of them (+0x8, +0x9) have a known
// writer and no known reader (the constructor's own redundant zero-fill,
// after the same routine's REP STOSD already zeroed the block) and one
// (+0x1e) has three known readers and no known writer (a stride-50 matrix
// subscript on a second, unidentified object) — neither is a decoded field,
// since neither claim gives either byte a value this project could apply.
// docs/DIVERGENCES.md records the open span.
type PlayerTail struct {
	Raw [PlayerTailLen]byte
}

// Formation reads the original Player receiver's byte (SAV-PLAYERIDENT-830).
// The game importer binds that receiver to an opaque native Player identity;
// its command identifier, trigger identifier and explicit Group owner remain
// separate. This decoder reports the source byte and enacts no player policy.
func (t PlayerTail) Formation() uint8 { return t.Raw[PlayerTailFormationByte] }

// PlayerTailFromRecord decodes rec's own "PRaw32" member, where rec is
// PartyWalk's own second return value (the Player record). KindRaw members
// are retained on every walk regardless of retainGraph (program.go's run()),
// so this needs no walker changes, unlike Diary's two counted arrays.
func PlayerTailFromRecord(rec *Record) (PlayerTail, error) {
	if rec == nil {
		return PlayerTail{}, fmt.Errorf("sav: no player record")
	}
	raw, ok := rec.Raw["PRaw32"]
	if !ok || len(raw) != PlayerTailLen {
		return PlayerTail{}, fmt.Errorf("sav: player at %d has no %d-byte tail", rec.Off, PlayerTailLen)
	}
	var t PlayerTail
	copy(t.Raw[:], raw)
	return t, nil
}

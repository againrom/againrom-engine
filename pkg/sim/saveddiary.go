package sim

import (
	"cmp"
	"fmt"
	"slices"
)

// SavedDiaryOwner names which record a carried Diary was read from. Player
// reports the local Player's own embedded Diary (sav.PlayerDiary, SAV-662,
// SAV-669: one per save). A nonzero PlayerID reports another Player's
// embedded Diary, named by its current Player identity (CurrentPlayers).
// Otherwise Actor is the owning living entity's ID, resolved from the
// Humanoid's own +0x1e4 reference (SAV-EMBED-039) by the importer, not by
// this package.
type SavedDiaryOwner struct {
	Player   bool
	Actor    EntityID
	PlayerID uint32 `json:",omitempty"`
}

// IsActor reports an actor-owned Diary.
func (o SavedDiaryOwner) IsActor() bool { return !o.Player && o.PlayerID == 0 }

// CompareSavedDiaryOwners orders owners: the local Player, the other Players
// by identity, then actors by entity ID.
func CompareSavedDiaryOwners(a, b SavedDiaryOwner) int {
	rank := func(o SavedDiaryOwner) (int, uint32) {
		switch {
		case o.Player:
			return 0, 0
		case o.PlayerID != 0:
			return 1, o.PlayerID
		}
		return 2, uint32(o.Actor)
	}
	ra, ia := rank(a)
	rb, ib := rank(b)
	if ra != rb {
		return ra - rb
	}
	return cmp.Compare(ia, ib)
}

// savedDiaryOwnerFault admits at most one local Player owner and no repeated
// owner; a Player owner names no actor, and another Player names no actor.
func savedDiaryOwnerFault(diaries []SavedDiary) error {
	seen := make(map[SavedDiaryOwner]bool, len(diaries))
	for _, d := range diaries {
		o := d.Owner
		switch {
		case o.Player && (o.Actor != 0 || o.PlayerID != 0):
			return fmt.Errorf("sim: carried Player diary carries a nonzero Actor id")
		case o.PlayerID != 0 && o.Actor != 0:
			return fmt.Errorf("sim: carried diary of Player %d carries a nonzero Actor id", o.PlayerID)
		case seen[o] && o.Player:
			return fmt.Errorf("sim: carried diaries have two Player owners")
		case seen[o] && o.PlayerID != 0:
			return fmt.Errorf("sim: carried diaries have duplicate owner Player %d", o.PlayerID)
		case seen[o]:
			return fmt.Errorf("sim: carried diaries have duplicate owner entity %d", o.Actor)
		}
		seen[o] = true
	}
	return nil
}

// SavedDiaryEntry is the sim-layer mirror of sav.DiaryEntry: one element pair
// departing from the array's own construction default. What either number
// counts is Unknown (SAV-668); this project carries them, it does not
// interpret them (docs/DIVERGENCES.md).
type SavedDiaryEntry struct {
	Index     int
	Count     uint32
	Remaining uint16
}

// SavedDiary is one Diary record's carried content, the sim-layer mirror of
// sav.Diary. Length is the two arrays' own shared declared element count,
// carried separately from len(Entries) so a byte-for-byte re-export can
// reproduce an all-default Diary's own array length too.
type SavedDiary struct {
	Owner   SavedDiaryOwner
	Length  int
	Entries []SavedDiaryEntry
}

func cloneSavedDiaries(diaries []SavedDiary) []SavedDiary {
	out := make([]SavedDiary, len(diaries))
	for i, d := range diaries {
		out[i] = SavedDiary{Owner: d.Owner, Length: d.Length, Entries: append([]SavedDiaryEntry(nil), d.Entries...)}
	}
	return out
}

// SavedDiaries returns this world's carried Diary list. It is a fresh copy:
// mutating the result cannot reach the world it came from.
func (w *World) SavedDiaries() []SavedDiary {
	return cloneSavedDiaries(w.savedDiaries)
}

// SetSavedDiaries replaces the carried list outright, with no validation.
// resumeWorld (pkg/game/resume.go) never needed a call here even then: it
// decodes onto the SAME receiver rather than a fresh one, so the field
// already survived. Form85 gives the field a wire position
// (carriedresumebinary.go), so both staging round trips now carry it through
// UnmarshalBinary like every other field and neither calls this any more; it
// remains for a caller that wants to set the field directly without a
// byte-form round trip.
func (w *World) SetSavedDiaries(diaries []SavedDiary) {
	w.savedDiaries = cloneSavedDiaries(diaries)
}

// ImportOriginalDiaries overlays the saved projection after map
// construction and actor admission, on ImportOriginalProjectiles' own
// standing: it validates rather than trusts its argument. No two entries may
// name the same owner — sav.PlayerDiary and the actor importer each produce
// at most one SavedDiary per owner, so a caller passing anything else built
// the argument wrong, not this save.
//
// Another Player's Diary at its construction default (no entry, one element
// per Units row) is not carried: SAVE writes that default for a Player whose
// Diary the World does not hold.
func (w *World) ImportOriginalDiaries(diaries []SavedDiary) error {
	if err := savedDiaryOwnerFault(diaries); err != nil {
		return err
	}
	diaries = slices.DeleteFunc(slices.Clone(diaries), func(d SavedDiary) bool {
		return d.Owner.PlayerID != 0 && len(d.Entries) == 0 && d.Length == w.DiaryRows()
	})
	w.savedDiaries = cloneSavedDiaries(diaries)
	return nil
}

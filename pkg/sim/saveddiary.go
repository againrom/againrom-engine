package sim

import "fmt"

// SavedDiaryOwner names which record a carried Diary was read from. Player
// reports the Player's own embedded Diary (sav.PlayerDiary, SAV-662,
// SAV-669: one per save); when Player is false, Actor is the owning living
// entity's ID, resolved from the Humanoid's own +0x1e4 reference
// (SAV-EMBED-039) by the importer, not by this package.
type SavedDiaryOwner struct {
	Player bool
	Actor  EntityID
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
func (w *World) ImportOriginalDiaries(diaries []SavedDiary) error {
	seenPlayer := false
	seenActor := make(map[EntityID]bool, len(diaries))
	for _, d := range diaries {
		if d.Owner.Player {
			if seenPlayer {
				return fmt.Errorf("sim: saved diaries have two Player owners")
			}
			seenPlayer = true
			continue
		}
		if seenActor[d.Owner.Actor] {
			return fmt.Errorf("sim: saved diaries have duplicate owner entity %d", d.Owner.Actor)
		}
		seenActor[d.Owner.Actor] = true
	}
	w.savedDiaries = cloneSavedDiaries(diaries)
	return nil
}

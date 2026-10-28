package sav

import (
	"slices"
	"sort"
)

// DocumentDiaryLocation exposes structural starts for an independent reader.
// OwnerArchiveIndex and ArchiveIndex are original-stream indices, not native
// identities. Player Diaries are inline (ArchiveIndex zero, RefOff -1).
// Off is -1 for a null Humanoid Diary; RefOff still locates its raw null tag.
type DocumentDiaryLocation struct {
	OwnerArchiveIndex, ArchiveIndex uint16
	OwnerClass                      string
	OwnerOff, RefOff, Off           int
}

func (f *File) DocumentDiaryLocations() ([]DocumentDiaryLocation, error) {
	owners, err := f.DiaryOwners()
	if err != nil {
		return nil, err
	}
	var out []DocumentDiaryLocation
	for _, o := range owners {
		loc := DocumentDiaryLocation{OwnerArchiveIndex: o.ArchiveIndex, OwnerClass: o.Owner.Class,
			OwnerOff: o.Owner.Off, RefOff: o.Owner.DiaryRefOff, Off: -1}
		if !o.Actor {
			loc.RefOff = -1
		}
		if o.Diary != nil {
			loc.ArchiveIndex, loc.Off = o.Diary.Index, o.Diary.Off
		}
		out = append(out, loc)
	}
	return out, nil
}

// DiaryOwner is a Player, whose Diary is inline, or a Human or Humanoid
// (Actor), whose Diary object is nil when its reference is null.
type DiaryOwner struct {
	ArchiveIndex uint16
	Actor        bool
	Owner, Diary *Record
}

// DiaryOwners returns every Diary owner by archive index, dead list included.
func (f *File) DiaryOwners() ([]DiaryOwner, error) {
	doc, _, err := f.exactDocument()
	if err != nil {
		return nil, err
	}
	var out []DiaryOwner
	for index, owner := range doc.objects {
		actor := owner.Class == "Human" || owner.Class == "Humanoid"
		if !actor && owner.Class != "Player" {
			continue
		}
		o := DiaryOwner{ArchiveIndex: index, Actor: actor, Owner: owner}
		if diaries := owner.Refs["Diary"]; len(diaries) != 0 {
			o.Diary = diaries[0]
		}
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ArchiveIndex < out[j].ArchiveIndex })
	return out, nil
}

// DocumentPlayerGroupLocation locates a Group and its actor-reference tags.
// Counts, reference values and resolved membership are deliberately absent.
type DocumentPlayerGroupLocation struct {
	PlayerOff, Off int
	ActorRefOffs   []int
}

func (f *File) DocumentPlayerGroupLocations() ([]DocumentPlayerGroupLocation, error) {
	doc, _, err := f.exactDocument()
	if err != nil {
		return nil, err
	}
	var out []DocumentPlayerGroupLocation
	seen := map[*Record]bool{}
	for _, player := range doc.players {
		if player == nil || seen[player] {
			continue
		}
		seen[player] = true
		for _, group := range player.Groups {
			out = append(out, DocumentPlayerGroupLocation{player.Off, group.Off, slices.Clone(group.GroupActorRefOffs)})
		}
	}
	return out, nil
}

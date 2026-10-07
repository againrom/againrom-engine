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
	doc, _, err := f.exactDocument()
	if err != nil {
		return nil, err
	}
	var out []DocumentDiaryLocation
	for index, owner := range doc.objects {
		if owner.Class != "Player" && owner.Class != "Human" && owner.Class != "Humanoid" {
			continue
		}
		loc := DocumentDiaryLocation{OwnerArchiveIndex: index, OwnerClass: owner.Class,
			OwnerOff: owner.Off, RefOff: owner.DiaryRefOff, Off: -1}
		if owner.Class == "Player" {
			loc.RefOff = -1
		}
		if diaries := owner.Refs["Diary"]; len(diaries) != 0 {
			loc.ArchiveIndex, loc.Off = diaries[0].Index, diaries[0].Off
		}
		out = append(out, loc)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OwnerArchiveIndex < out[j].OwnerArchiveIndex })
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

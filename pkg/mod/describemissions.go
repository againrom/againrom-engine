package mod

import (
	"errors"
	"io/fs"
)

// DescribeMissions is what the mod id at dir, a folder or a packed mod's RES
// file, declares in its missions file, worded in lang through the mod's own
// strings, and its number-named maps. It opens no map and no text file.
// source is the missions file's bytes, nil when the mod has none.
func DescribeMissions(dir, id, lang string) (d MissionData, source []byte, err error) {
	tree, err := OpenTree(dir)
	if err != nil {
		return MissionData{}, nil, err
	}
	if _, err := tree.Lstat(MissionsFile); err == nil {
		if source, _, err = tree.ReadInside(MissionsFile); err != nil {
			return MissionData{}, nil, err
		}
		look, err := TextLookup(dir, lang)
		if err != nil {
			return MissionData{}, nil, err
		}
		if d, err = ParseMissions(id, MissionsFile, source, look, lang); err != nil {
			return MissionData{}, nil, err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return MissionData{}, nil, err
	}
	found, err := missionMapFiles(tree, id, lang, d.Missions)
	if err != nil {
		return MissionData{}, nil, err
	}
	for _, m := range found {
		d.maps = append(d.maps, missionMap{Mission: m, at: len(d.Missions)})
	}
	return d, source, nil
}

// Playable are the missions of a listed set a player can name to start:
// every mission that is not switched off and has a number, in load order.
func (d MissionData) Playable() []Mission {
	var out []Mission
	for _, m := range d.Missions {
		if !m.Off && m.Number > 0 {
			out = append(out, m)
		}
	}
	return out
}

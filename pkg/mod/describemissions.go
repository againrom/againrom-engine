package mod

import (
	"errors"
	"io/fs"
)

// DescribeMissions is what the mod id at dir, a folder or a packed mod's RES
// file, declares in its missions file, worded in lang through the mod's own
// strings. It opens no map and no text file. source is the bytes of the
// missions file, nil when the mod has none. The map editor and the starter
// both read a mod's missions through it.
func DescribeMissions(dir, id, lang string) (d MissionData, source []byte, err error) {
	tree, err := OpenTree(dir)
	if err != nil {
		return MissionData{}, nil, err
	}
	if _, err := tree.Lstat(MissionsFile); errors.Is(err, fs.ErrNotExist) {
		return MissionData{}, nil, nil
	}
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
	return d, source, nil
}

// Listed are the missions of d a player can name to start: every declared
// row that is not switched off and has a number, in the file's order.
func (d MissionData) Listed() []Mission {
	var out []Mission
	for _, m := range d.Missions {
		if !m.Off && m.Number > 0 {
			out = append(out, m)
		}
	}
	return out
}

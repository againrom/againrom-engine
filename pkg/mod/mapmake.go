package mod

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/namefold"
)

// MakeMission is the row that makes unused map e a mission of mod id: a
// guest mission when replace is 0, else a replacing mission of mission
// number replace. A guest is the default guest row the editor's Add builds,
// passed as guest (its number, the mission it plays after and its party),
// naming e. held is the install's answer for a number (nil for a mission it
// plays); with no install it is nil, and a replacement is then refused. The
// caller appends the row and writes it with WriteMissions.
func MakeMission(id string, d MissionData, e MapEntry, guest Mission, replace int, held func(n int) error) (Mission, error) {
	if e.Role() != MapUnused {
		return Mission{}, errors.New("map is already named by a mission")
	}
	m := Mission{Mod: id, File: MissionsFile, Key: mapKey(d, e.Stem), Map: e.Path}
	if replace != 0 {
		if held == nil || held(replace) != nil {
			return Mission{}, fmt.Errorf("number %d is not a mission of the install", replace)
		}
		if _, ok := d.Replacing(replace); ok {
			return Mission{}, fmt.Errorf("number %d already replaces a mission", replace)
		}
		m.Kind, m.Number = MissionReplace, replace
		return m, nil
	}
	guest.Mod, guest.File, guest.Key, guest.Map = m.Mod, m.File, m.Key, m.Map
	return guest, nil
}

// MakeRow is the replacing row of a map that replaces its mission by file
// name: it names the same map under the number the file name gives. The
// install must hold that number, as for MakeMission.
func MakeRow(id string, d MissionData, e MapEntry, held func(n int) error) (Mission, error) {
	if e.Role() != MapByName {
		return Mission{}, errors.New("map does not replace a mission by its file name")
	}
	if held == nil || held(e.Number) != nil {
		return Mission{}, fmt.Errorf("number %d is not a mission of the install", e.Number)
	}
	return Mission{Mod: id, File: MissionsFile, Key: mapKey(d, e.Stem), Kind: MissionReplace,
		Number: e.Number, Map: e.Path}, nil
}

// mapKey is the stem as a row key, with _2, _3 and so on where d holds the key.
func mapKey(d MissionData, stem string) string {
	key := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			return r
		case r >= 'A' && r <= 'Z':
			return r + 'a' - 'A'
		}
		return '_'
	}, stem)
	if key == "" || key[0] < 'a' || key[0] > 'z' {
		key = "m" + key
	}
	key = key[:min(len(key), 28)]
	stem = key
	for suffix := 2; slices.ContainsFunc(d.Missions, func(m Mission) bool { return m.Key == key }); suffix++ {
		key = stem + "_" + strconv.Itoa(suffix)
	}
	return key
}

// FolderMap is one map of a folder being added to a mod: its file name, its
// stem and extension, and the folder's files that belong to it in folder
// paths: the map itself, its text pages and its variable labels.
type FolderMap struct {
	Name, Stem, Ext string
	Files           []string
}

// FolderMaps are the maps of a folder's file names, in name order: every
// file at the folder's top with a map extension. A stem's text pages and
// variable labels go with its first map; a map whose stem an earlier map
// of the folder holds takes none. Other names are not maps and are left out.
func FolderMaps(names []string) []FolderMap {
	sorted := slices.Clone(names)
	slices.Sort(sorted)
	var out []FolderMap
	seen := map[string]bool{}
	for _, name := range sorted {
		if strings.Contains(name, "/") || !isMapFile(name) {
			continue
		}
		ext := path.Ext(name)
		stem := strings.TrimSuffix(name, ext)
		f := FolderMap{Name: name, Stem: stem, Ext: ext, Files: []string{name}}
		if !seen[namefold.ModTree(stem)] {
			seen[namefold.ModTree(stem)] = true
			for _, other := range sorted {
				if belongsTo(other, stem) && other != name {
					f.Files = append(f.Files, other)
				}
			}
		}
		out = append(out, f)
	}
	return out
}

// belongsTo reports a folder file that holds the pages or the labels of a
// map of stem.
func belongsTo(name, stem string) bool {
	if len(name) <= len(stem) || namefold.ModTree(name[:len(stem)]) != namefold.ModTree(stem) {
		return false
	}
	rest := name[len(stem):]
	return rest == ".variables" || strings.HasPrefix(rest, ".text/")
}

// PlaceFolderMaps is where each file of folder goes in the mod, keyed by its
// folder path: the map and its files under maps/ at the place the mod's one
// placement rule (MissionPlace.Free) gives its stem. The maps are placed in
// folder order against the mod's files and each other, so a second copy of
// a stem takes the next suffix.
func PlaceFolderMaps(files []string, folder []FolderMap) map[string]string {
	held := slices.Clone(files)
	out := map[string]string{}
	for _, f := range folder {
		place := PlaceMission(MissionStem(f.Stem), f.Ext == ExtendedMapExt).Free(held)
		held = append(held, place.Map)
		stem := strings.TrimSuffix(place.Map, path.Ext(place.Map))
		for _, name := range f.Files {
			out[name] = stem + name[len(f.Stem):]
		}
	}
	return out
}

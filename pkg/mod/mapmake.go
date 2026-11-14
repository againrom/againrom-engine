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
// number replace. A guest plays after the last guest row of d and takes the
// first number from the mission numbers the install does not hold. held is
// the install's answer for a number (nil for a mission it plays); with no
// install it is nil, and a replacement is then refused. The caller appends
// the row and writes it with WriteMissions.
func MakeMission(id string, d MissionData, e MapEntry, replace int, held func(n int) error) (Mission, error) {
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
	m.Kind = MissionGuest
	for i := len(d.Missions) - 1; i >= 0; i-- {
		if g := d.Missions[i]; g.Kind == MissionGuest && !g.Off {
			m.After = g.Number
			break
		}
	}
	m.Number = freeNumber(d, held)
	return m, nil
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
	key := stem
	for suffix := 2; slices.ContainsFunc(d.Missions, func(m Mission) bool { return m.Key == key }); suffix++ {
		key = stem + "_" + strconv.Itoa(suffix)
	}
	return key
}

// freeNumber is the lowest number above every row of d that the install does
// not hold as a mission, so a guest row never takes a campaign number.
func freeNumber(d MissionData, held func(n int) error) int {
	top := 0
	for _, m := range d.Missions {
		top = max(top, m.Number)
	}
	for n := top + 1; ; n++ {
		if held != nil && held(n) == nil {
			continue
		}
		return n
	}
}

// PlaceMap is the path of a copy of a map called stem with extension ext in
// mod files: maps/<stem><ext>, or with -2, -3 and so on when a map of that
// stem is already among the maps folder's files, whatever its extension.
func PlaceMap(files []string, stem, ext string) string {
	taken := mapStems(files)
	return MapsFolder + "/" + placeStem(taken, stem) + ext
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
// folder path: the map and its files under maps/ with the stem the mod
// takes for it. The stems are placed in folder order against the mod's maps
// and each other, so a second copy of a stem takes the next suffix.
func PlaceFolderMaps(files []string, folder []FolderMap) map[string]string {
	taken := mapStems(files)
	out := map[string]string{}
	for _, f := range folder {
		stem := placeStem(taken, f.Stem)
		for _, name := range f.Files {
			out[name] = MapsFolder + "/" + stem + name[len(f.Stem):]
		}
	}
	return out
}

// mapStems is the folded stems of the map files directly in the maps folder.
func mapStems(files []string) map[string]bool {
	taken := map[string]bool{}
	for _, f := range files {
		if path.Dir(f) == MapsFolder && isMapFile(f) {
			taken[namefold.ModTree(strings.TrimSuffix(path.Base(f), path.Ext(f)))] = true
		}
	}
	return taken
}

// placeStem is stem, or stem-2, stem-3 and so on, the first not taken; it
// marks the stem taken.
func placeStem(taken map[string]bool, stem string) string {
	name := stem
	for n := 2; taken[namefold.ModTree(name)]; n++ {
		name = stem + "-" + strconv.Itoa(n)
	}
	taken[namefold.ModTree(name)] = true
	return name
}

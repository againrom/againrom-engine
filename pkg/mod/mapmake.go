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

var (
	ErrMapNamed    = errors.New("map is already named by a mission")
	ErrNotByName   = errors.New("map does not replace a mission by its file name")
	ErrNotCampaign = errors.New("number is not a mission of the install")
	ErrReplaced    = errors.New("number already replaces a mission")
)

func MakeMission(id string, d MissionData, e MapEntry, guest Mission, replace int, held func(n int) error) (Mission, error) {
	if e.Role() != MapUnused {
		return Mission{}, ErrMapNamed
	}
	m := Mission{Mod: id, File: MissionsFile, Key: mapKey(d, e.Stem), Map: e.Path}
	if replace != 0 {
		if held == nil || held(replace) != nil {
			return Mission{}, fmt.Errorf("%w: %d", ErrNotCampaign, replace)
		}
		if _, ok := d.Replacing(replace); ok {
			return Mission{}, fmt.Errorf("%w: %d", ErrReplaced, replace)
		}
		m.Kind, m.Number = MissionReplace, replace
		return m, nil
	}
	guest.Mod, guest.File, guest.Key, guest.Map = m.Mod, m.File, m.Key, m.Map
	return guest, nil
}

func MakeRow(id string, d MissionData, e MapEntry, held func(n int) error) (Mission, error) {
	if e.Role() != MapByName {
		return Mission{}, ErrNotByName
	}
	if held == nil || held(e.Number) != nil {
		return Mission{}, fmt.Errorf("%w: %d", ErrNotCampaign, e.Number)
	}
	return Mission{Mod: id, File: MissionsFile, Key: mapKey(d, e.Stem), Kind: MissionReplace,
		Number: e.Number, Map: e.Path}, nil
}

func (d MissionData) ReplaceNumbers(candidates []int, held func(n int) error) []int {
	var out []int
	for _, n := range candidates {
		if _, taken := d.Replacing(n); !taken && held != nil && held(n) == nil && !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	slices.Sort(out)
	return out
}

func mapKey(d MissionData, stem string) string {
	var b strings.Builder
	for _, r := range namefold.ModTree(stem) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	stem = b.String()
	if stem == "" || stem[0] < 'a' || stem[0] > 'z' {
		stem = "m" + stem
	}
	stem = stem[:min(len(stem), 28)]
	key := stem
	for suffix := 2; slices.ContainsFunc(d.Missions, func(m Mission) bool { return m.Key == key }); suffix++ {
		key = stem + "_" + strconv.Itoa(suffix)
	}
	return key
}

type FolderMap struct {
	Name, Stem, Ext string
	Files           []string
}

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

func belongsTo(name, stem string) bool {
	if len(name) <= len(stem) || namefold.ModTree(name[:len(stem)]) != namefold.ModTree(stem) {
		return false
	}
	rest := name[len(stem):]
	return namefold.ModTree(name) == namefold.ModTree(MapLabelFile(stem+MapExt)) || strings.HasPrefix(rest, mapTextSuffix+"/")
}

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

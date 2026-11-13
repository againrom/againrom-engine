package mod

import (
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/namefold"
)

// The extensions of a mission map: a plain map, and a map that holds the
// extended record.
const (
	MapExt         = ".alm"
	ExtendedMapExt = ".almx"
	// defaultMissionStem is the stem of a map whose source names none.
	defaultMissionStem = "map"
)

// MissionPlace is where one mission's files lie in a mod: the map, the pages
// beside it in the Save As form, and the editor's variable labels. It is the
// one path rule of a mission's files: every writer of a mission into a mod
// takes its paths here, and the tree rows, the page reader and the tree
// description read the same names.
type MissionPlace struct{ Map string }

// PlaceMission is the place of a map that serves every language:
// maps/<stem>.alm, or .almx for an extended map.
func PlaceMission(stem string, extended bool) MissionPlace {
	return MissionPlace{Map: MapsFolder + "/" + stem + mapExt(extended)}
}

// PlaceLanguageMission is the place of a map of one language:
// maps/<lang>/<stem>.alm, or .almx for an extended map. A mission row names
// it with langToken in place of the language.
func PlaceLanguageMission(lang, stem string, extended bool) MissionPlace {
	return MissionPlace{Map: MapsFolder + "/" + lang + "/" + stem + mapExt(extended)}
}

func mapExt(extended bool) string {
	if extended {
		return ExtendedMapExt
	}
	return MapExt
}

// MissionStem is the stem a mission takes from the name of its source map:
// the file name without its extension. A stem of digits takes the prefix
// "m", so a written map is never the numbered file that replaces a campaign
// mission by its name; a name a packed mod or a mission row cannot hold gives
// "map".
func MissionStem(name string) string {
	base := path.Base(filepath.ToSlash(name))
	base = strings.TrimSuffix(base, path.Ext(base))
	if base != "" && strings.Trim(base, "0123456789") == "" {
		base = "m" + base
	}
	if checkPackedComponent(base+mapVariablesSuffix) != nil || base == "" || strings.ContainsAny(base, `<>:"|?*{}`) ||
		strings.HasPrefix(base, ".") || strings.HasPrefix(base, " ") || strings.HasSuffix(base, ".") || strings.HasSuffix(base, " ") {
		return defaultMissionStem
	}
	return base
}

// Pages is the folder of the pages beside the map, <stem>.text, or the
// folder of one language's pages, <stem>.text/<lang>, when lang is set. The
// map may be a slash path in the mod or a host path.
func (p MissionPlace) Pages(lang string) string {
	folder := strings.TrimSuffix(p.Map, filepath.Ext(p.Map)) + mapTextSuffix
	if lang != "" {
		folder += "/" + lang
	}
	return folder
}

// Labels is the editor's variable labels file beside the map.
func (p MissionPlace) Labels() string { return MapLabelFile(p.Map) }

// InTree reports that the map lies in the tree's maps folder; a row map
// elsewhere is an old path, read with a warning.
func (p MissionPlace) InTree() bool {
	return strings.HasPrefix(namefold.ModTree(path.Clean(p.Map)), MapsFolder+"/")
}

// holdsPages reports that folder is the pages folder beside the map or a
// folder inside it.
func (p MissionPlace) holdsPages(folder string) bool {
	beside := p.Pages("")
	return folder == beside || strings.HasPrefix(folder, beside+"/")
}

// Free is p, or p with -2, -3 and on added to its stem, whose map in either
// extension, pages and labels name no file of held.
func (p MissionPlace) Free(held []string) MissionPlace {
	return p.free(func(stem string) bool { return stemTaken(held, stem) })
}

func (p MissionPlace) free(taken func(stem string) bool) MissionPlace {
	ext := path.Ext(p.Map)
	stem := strings.TrimSuffix(p.Map, ext)
	candidate := stem
	for n := 2; taken(candidate); n++ {
		candidate = stem + "-" + strconv.Itoa(n)
	}
	return MissionPlace{Map: candidate + ext}
}

// stemTaken reports that a file of held is the map, a page or the labels of
// the place whose path without extension is stem.
func stemTaken(held []string, stem string) bool {
	stem = namefold.ModTree(stem)
	return slices.ContainsFunc(held, func(f string) bool {
		f = namefold.ModTree(f)
		return f == stem+MapExt || f == stem+ExtendedMapExt || f == stem+mapVariablesSuffix || strings.HasPrefix(f, stem+mapTextSuffix+"/")
	})
}

// IsMissionFile reports that rel is a mission map, a mission page or a map's
// variable labels in the tree's rows.
func IsMissionFile(rel string) bool {
	for _, k := range dataKinds {
		if (k.Name == missionMapsKind || k.Name == missionPagesKind || k.Name == mapLabelsKind) && k.takes(rel) {
			return true
		}
	}
	return false
}

// missionTreeRows are the tree rows of the mission maps, their pages and
// their labels, written from the place rule.
func missionTreeRows() (maps, pages, labels []string) {
	for _, p := range []MissionPlace{PlaceMission("*", false), PlaceMission("*", true), PlaceLanguageMission(langToken, "*", false), PlaceLanguageMission(langToken, "*", true)} {
		maps = append(maps, p.Map)
		if path.Ext(p.Map) == MapExt {
			pages = append(pages, p.Pages("")+"/*.txt", p.Pages(langToken)+"/*.txt")
			labels = append(labels, p.Labels())
		}
	}
	return maps, pages, labels
}

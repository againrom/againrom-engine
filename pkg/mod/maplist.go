package mod

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"againrom/pkg/locale"
	"againrom/pkg/namefold"
)

// MapRole is what a mod does with one of its maps.
type MapRole uint8

const (
	// MapUnused is a map file no declaration names: the game does not play it.
	MapUnused MapRole = iota
	// MapRows is a map one or more mission rows name.
	MapRows
	// MapByName is a map whose file name declares it: it replaces the
	// campaign mission of that number.
	MapByName
	// MapMissing is a map a row names and the mod holds no file of.
	MapMissing
)

// MapEntry is one map of a mod: its file, or its files when the path holds
// the language, joined to the declarations that name it.
type MapEntry struct {
	// Path is the map's path in the mod, with {lang} for a map kept per
	// language.
	Path string
	// Stem and Ext are the file name's parts.
	Stem, Ext string
	// Files are the map's files the mod holds, and Languages the language
	// of each for a map kept per language. Lacking are the files a row names
	// and the mod does not hold.
	Files, Languages, Lacking []string
	// PageLanguages are the languages the pages folder beside the map holds
	// pages of; "" stands for pages of every language.
	PageLanguages []string
	// Labels reports a variable labels file beside the map.
	Labels bool
	// Rows are the mission rows that name the map, in their order.
	Rows []Mission
	// ByName and Number are the file form's declaration: the file's name is
	// a mission number and no row names the file.
	ByName bool
	Number int
}

// Role is what the mod does with the map.
func (e MapEntry) Role() MapRole {
	switch {
	case len(e.Files) == 0:
		return MapMissing
	case len(e.Rows) > 0:
		return MapRows
	case e.ByName:
		return MapByName
	}
	return MapUnused
}

// File is the map's file for lang: the path with the language in it.
func (e MapEntry) File(lang string) string { return missionPath(e.Path, lang) }

// isMapFile reports a name with a map's extension.
func isMapFile(rel string) bool {
	ext := path.Ext(namefold.ModTree(rel))
	return ext == MapExt || ext == ExtendedMapExt
}

// languageMap is the path of rel with its language folder as {lang}, when
// rel lies in a language folder of the maps folder.
func languageMap(rel string) (string, string, bool) {
	parts := strings.Split(rel, "/")
	if len(parts) != 3 || namefold.ModTree(parts[0]) != MapsFolder || !locale.Known(parts[1]) {
		return "", "", false
	}
	return parts[0] + "/" + langToken + "/" + parts[2], parts[1], true
}

// MapList is the maps of mod id: every map among files, the mod's files as
// its tree lists them, joined to the declarations of d that name it. The
// maps rows name come first in row order, a map two rows name once; then
// the other map files in path order, each a replacement by its file name
// where d holds the file form's mission of it, or unused. A row whose map
// the files lack is an entry too. It is the one
// list of a mod's maps: the editor's mod window, its Open window and the
// check's warnings read it.
func MapList(files []string, id string, d MissionData) []MapEntry {
	held := newHeldFiles(files)
	var out []MapEntry
	at := map[string]int{}
	claimed := map[string]bool{}
	entry := func(template string) *MapEntry {
		key := namefold.ModTree(template)
		i, ok := at[key]
		if !ok {
			i = len(out)
			at[key] = i
			ext := path.Ext(template)
			out = append(out, MapEntry{Path: template, Stem: strings.TrimSuffix(path.Base(template), ext), Ext: ext})
		}
		return &out[i]
	}
	hold := func(e *MapEntry, rel, lang string, wanted bool) {
		actual, ok := held.actual(rel)
		if !ok {
			if wanted && !slices.Contains(e.Lacking, rel) {
				e.Lacking = append(e.Lacking, rel)
			}
			return
		}
		if claimed[namefold.ModTree(actual)] {
			return
		}
		claimed[namefold.ModTree(actual)] = true
		e.Files = append(e.Files, actual)
		if lang != "" {
			e.Languages = append(e.Languages, lang)
		}
	}
	numbers := map[string]int{}
	for _, m := range d.Missions {
		if id != "" && m.Mod != id || m.Off || m.Map == "" {
			continue
		}
		if m.byName {
			numbers[namefold.ModTree(m.Map)] = m.Number
			continue
		}
		template := m.Map
		if m.mapTemplate != "" {
			template = m.mapTemplate
		}
		template = path.Clean(template)
		e := entry(template)
		e.Rows = append(e.Rows, m)
		if !strings.Contains(template, langToken) {
			hold(e, template, "", true)
			continue
		}
		for _, l := range locale.All() {
			hold(e, missionPath(template, l.Code), l.Code, true)
		}
	}
	// The file form's missions that still wait for the install's answer.
	for _, m := range d.maps {
		if id == "" || m.Mod == id {
			numbers[namefold.ModTree(m.Map)] = m.Number
		}
	}
	rest := slices.Clone(files)
	slices.Sort(rest)
	for _, rel := range rest {
		if !isMapFile(rel) || claimed[namefold.ModTree(rel)] {
			continue
		}
		if template, lang, ok := languageMap(rel); ok {
			hold(entry(template), rel, lang, false)
			continue
		}
		e := entry(rel)
		hold(e, rel, "", false)
		e.Number, e.ByName = numbers[namefold.ModTree(rel)]
	}
	for i := range out {
		e := &out[i]
		for _, rel := range e.Files {
			place := MissionPlace{Map: rel}
			e.Labels = e.Labels || held.has(place.Labels())
			prefix := namefold.ModTree(place.Pages("")) + "/"
			for _, f := range files {
				tail, ok := strings.CutPrefix(namefold.ModTree(f), prefix)
				if !ok {
					continue
				}
				lang := ""
				if first, _, deep := strings.Cut(tail, "/"); deep && locale.Known(first) {
					lang = first
				}
				if !slices.Contains(e.PageLanguages, lang) {
					e.PageLanguages = append(e.PageLanguages, lang)
				}
			}
		}
		slices.Sort(e.PageLanguages)
	}
	return out
}

// MapWarnings are the check's lines about the maps of list: a map no
// declaration names, and a map a row names that the mod holds no file of.
func MapWarnings(list []MapEntry) []string {
	var out []string
	for _, e := range list {
		if e.Role() == MapUnused {
			for _, rel := range e.Files {
				out = append(out, (&ItemFileError{File: rel, Msg: "no mission names this map; the game does not play it"}).Error())
			}
		}
		for _, rel := range e.Lacking {
			if e.Role() != MapMissing {
				break
			}
			row := e.Rows[0]
			out = append(out, (&ItemFileError{File: row.File, Line: row.Line, Msg: fmt.Sprintf("map %s is missing", rel)}).Error())
		}
	}
	return out
}

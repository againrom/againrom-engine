package mod

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"againrom/pkg/locale"
	"againrom/pkg/namefold"
	"againrom/pkg/town"
)

type referenceRole uint8

const (
	referenceFile referenceRole = iota
	referenceMap
	referencePages
	referenceTown
)

type treeReference struct {
	table, field string
	role         referenceRole
}

func (r treeReference) paths(value string) []string {
	if r.role != referenceMap && r.role != referencePages {
		return []string{value}
	}
	var out []string
	for _, l := range locale.All() {
		out = append(out, missionPath(value, l.Code))
	}
	return out
}

type TreeFile struct {
	Path   string
	Kind   string
	Digest bool
}

// TreeDescription is the table's admission of held files, including old row paths.
type TreeDescription struct {
	Files    []TreeFile
	Warnings []string
}

func (d TreeDescription) Admits(rel string) bool {
	return slices.ContainsFunc(d.Files, func(f TreeFile) bool { return namefold.ModTree(f.Path) == namefold.ModTree(rel) })
}

func treeMatch(pattern, rel string) bool {
	a, b := strings.Split(namefold.ModTree(pattern), "/"), strings.Split(namefold.ModTree(rel), "/")
	var match func([]string, []string) bool
	match = func(a, b []string) bool {
		if len(a) == 0 {
			return len(b) == 0
		}
		if a[0] == "**" {
			return match(a[1:], b) || len(b) > 0 && match(a, b[1:])
		}
		if len(b) == 0 {
			return false
		}
		switch a[0] {
		case "{lang}":
			if !locale.Known(b[0]) {
				return false
			}
		case "{layer}":
			if b[0] != CommonFiles && !locale.Known(b[0]) {
				return false
			}
		default:
			if strings.HasPrefix(a[0], "<n>") {
				ext := path.Ext(a[0])
				if _, ok := missionMapNumber(strings.TrimSuffix(b[0], ext) + ".alm"); !ok || ext != path.Ext(b[0]) {
					return false
				}
			} else if ok, _ := path.Match(a[0], b[0]); !ok {
				return false
			}
		}
		return match(a[1:], b[1:])
	}
	return match(a, b)
}

func (k DataKind) takes(rel string) bool {
	for _, p := range append(slices.Clone(k.Files), k.Paths...) {
		if treeMatch(p, rel) {
			return true
		}
	}
	return false
}

// Describe is the file description shared by packing, hashing and the check report.
func (t Tree) Describe() (TreeDescription, error) {
	var out TreeDescription
	references := map[string]string{}
	patterns := map[string]string{}
	add := func(rel, kind string) {
		if resolved, err := modFilePath(rel); err == nil {
			references[namefold.ModTree(resolved)] = kind
		}
	}
	warn := func(file string, line int, format string, args ...any) {
		out.Warnings = append(out.Warnings, (&ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}).Error())
	}
	manifest, err := t.ReadFile(ManifestName)
	if err == nil {
		m, err := ParseManifest(manifest)
		if err == nil && m.Entry != "" {
			add(m.Entry, "entry script")
		}
	} else if !fsNotExist(err) {
		return out, err
	}
	for _, k := range dataKinds {
		if !k.Validates() {
			continue
		}
		for index, file := range k.Files {
			if st, err := t.Stat(file); fsNotExist(err) || err == nil && st.IsDir() {
				continue
			} else if err != nil {
				return out, err
			}
			src, _, err := t.ReadInside(file)
			if err != nil {
				return out, err
			}
			if index > 0 {
				warn(file, 0, "use %s for %s", k.Files[0], k.Name)
			}
			if len(k.references) == 0 {
				continue
			}
			tables, err := parseTOMLWith(src, tomlOptions{arrays: true, quotedKeys: true})
			if err != nil {
				return out, fmt.Errorf("%s: %w", file, err)
			}
			for _, row := range tables {
				for _, ref := range k.references {
					if row.name != ref.table {
						continue
					}
					for _, pair := range row.pairs {
						if pair.key != ref.field || pair.val.kind != tomlString {
							continue
						}
						for _, name := range ref.paths(pair.val.str) {
							rel, err := modFilePath(name)
							if err != nil {
								continue
							}
							switch ref.role {
							case referenceMap:
								place := MissionPlace{Map: rel}
								add(rel, missionMapsKind)
								patterns[namefold.ModTree(place.Pages("")+"/"+pagePattern)] = missionPagesKind
								add(place.Labels(), mapLabelsKind)
							case referencePages:
								patterns[namefold.ModTree(path.Join(rel, pagePattern))] = missionPagesKind
								if locale.Known(path.Base(rel)) {
									patterns[namefold.ModTree(path.Join(path.Dir(rel), path.Base(pagePattern)))] = missionPagesKind
								}
							case referenceTown:
								if !IsTownDescriptionFile(rel) {
									continue
								}
								add(rel, k.Name)
								if err := t.describeTownArt(rel, add); err != nil {
									return out, err
								}
							default:
								add(rel, k.Name)
							}
						}
						if ref.role == referenceMap && !(MissionPlace{Map: pair.val.str}).InTree() {
							folder := MapsFolder
							if strings.Contains(pair.val.str, langToken) {
								folder = path.Join(folder, langToken)
							}
							warn(file, pair.line, "map %s uses an old path; use %s/<stem>%s", pair.val.str, folder, path.Ext(pair.val.str))
						}
					}
				}
				if row.name == "mission" && k.Name == "missions" {
					t.describeMissionPages(file, row, warn)
				}
			}
		}
	}
	err = t.WalkDir(func(rel string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if dottedPath(rel) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		kind, digest := references[namefold.ModTree(rel)], true
		for _, k := range dataKinds {
			if k.takes(rel) {
				kind, digest = k.Name, !k.NoDigest
				break
			}
		}
		if kind == "" {
			var matching []string
			for p := range patterns {
				if treeMatch(p, rel) {
					matching = append(matching, p)
				}
			}
			slices.Sort(matching)
			if len(matching) > 0 {
				kind = patterns[matching[0]]
			}
		}
		if kind == "" {
			warn(rel, 0, "no fixed-tree kind reads this file; it is excluded from packing and the digest")
			return nil
		}
		out.Files = append(out.Files, TreeFile{Path: rel, Kind: kind, Digest: digest})
		return nil
	})
	slices.Sort(out.Warnings)
	return out, err
}

func fsNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

func (t Tree) describeTownArt(rel string, add func(string, string)) error {
	if _, err := t.Stat(rel); fsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	src, _, err := t.ReadInside(rel)
	if fsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	d, err := town.Parse(src)
	if err != nil {
		return nil
	}
	files, err := TownArtFiles(d)
	if err != nil {
		return err
	}
	for _, f := range files {
		add(f.Path, "towns")
	}

	return nil
}

func (t Tree) describeMissionPages(file string, row tomlTable, warn func(string, int, string, ...any)) {
	values := map[string]tomlPair{}
	for _, p := range row.pairs {
		values[p.key] = p
	}
	mapPath, texts := path.Clean(values["map"].val.str), values["texts"].val.str
	if texts != "" {
		texts = path.Clean(texts)
	}
	place := MissionPlace{Map: mapPath}
	if texts != "" && !place.holdsPages(texts) {
		p := values["texts"]
		preferred := place
		if !place.InTree() {
			preferred = PlaceMission(MissionStem(mapPath), false)
			if strings.Contains(mapPath, langToken) {
				preferred = PlaceLanguageMission(langToken, MissionStem(mapPath), false)
			}
		}
		warn(file, p.line, "texts %s uses an old path; use %s or its language folder", texts, preferred.Pages(""))
	}
	for _, key := range []string{"title", "briefing"} {
		p := values[key]
		if p.val.str == "" {
			continue
		}
		for _, l := range locale.All() {
			lang := l.Code
			m := Mission{Map: missionPath(mapPath, lang), Texts: missionPath(texts, lang), textsTemplate: texts}
			layer, err := m.ReadPageLayer(PageMod, t.Path(), lang)
			if _, held := ResolvePages(nil, layer)[key+".txt"]; err == nil && held {
				warn(file, p.line, "%s key overrides %s.txt for %s", key, key, lang)
			}
		}
	}
}

package mod

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// MissionsFile is the data file that declares the missions a mod adds. A mod
// loads it from its entry script with game.data.add.
const MissionsFile = "data/missions.toml"

// The kinds of a mission declaration. A guest mission is a new mission a
// guest party plays between a win and the town it opens; a replacing mission
// is a campaign mission of the install played on the mod's map. A Mission
// whose Kind is empty is a guest mission.
const (
	MissionGuest   = "guest"
	MissionReplace = "replace"
)

// CarryJoin is the one carry rule: the guest member joins the party through
// the next town's own grant of that companion.
const CarryJoin = "join"

// Bounds of a mission declaration.
const (
	maxMissionNumber = 65535
	maxMissionGold   = 1_000_000
	// MaxMissionTexts bounds the files a mission's text directory may hold.
	MaxMissionTexts = 64
	// langToken in a path is replaced by the language the game runs in.
	langToken = "{lang}"
)

// missionTextName is the name of a file a mission's text directory serves.
var missionTextName = regexp.MustCompile(`^[a-z0-9_]{1,32}\.txt$`)

// Mission is one mission a mod declares: it runs as mission Number with the
// map Map and the texts of Texts, both inside the mod folder. A guest mission
// (Kind MissionGuest) plays after the player wins main mission After.
// Companion Guest, as the town of chapter Chapter would grant her over the
// party, plays it alone with Gold in her purse; the party waits. When she
// wins, the party goes on to that town, where she joins as she ended the
// mission (Carry). A replacing mission (Kind MissionReplace) is the install's
// campaign mission Number, reached and continued as the campaign does; only
// its map and texts are the mod's.
type Mission struct {
	Mod  string
	File string
	Line int

	Kind    string
	Key     string
	Number  int
	Map     string
	Texts   string
	Title   string
	Brief   string
	After   int
	Guest   int
	Chapter int
	Gold    int
	Carry   string

	language string

	// MapBytes and TextFiles are the map and the text files as read from the
	// mod folder when the mod loaded. TextFiles is keyed by file name.
	MapBytes  []byte
	TextFiles map[string][]byte
}

// MissionData is the missions the mods of a set declare, in load order.
type MissionData struct {
	Missions []Mission
}

// Empty reports that no mod declares a mission.
func (d MissionData) Empty() bool { return len(d.Missions) == 0 }

// AfterWin returns the guest mission that plays after main mission n is won.
func (d MissionData) AfterWin(n int) (Mission, bool) {
	for _, m := range d.Missions {
		if m.Kind != MissionReplace && m.After == n {
			return m, true
		}
	}
	return Mission{}, false
}

// ByNumber returns the guest mission that runs as mission n.
func (d MissionData) ByNumber(n int) (Mission, bool) {
	for _, m := range d.Missions {
		if m.Kind != MissionReplace && m.Number == n {
			return m, true
		}
	}
	return Mission{}, false
}

// Replacing returns the mission that replaces the map of campaign mission n.
func (d MissionData) Replacing(n int) (Mission, bool) {
	for _, m := range d.Missions {
		if m.Kind == MissionReplace && m.Number == n {
			return m, true
		}
	}
	return Mission{}, false
}

// Find returns the mission key of mod id.
func (d MissionData) Find(id, key string) (Mission, bool) {
	for _, m := range d.Missions {
		if m.Mod == id && m.Key == key {
			return m, true
		}
	}
	return Mission{}, false
}

// ParseMissions reads data/missions.toml of the mod id: [[mission]] tables
// with the keys kind, key, number, map, texts, title, briefing, after, guest,
// chapter, gold and carry. A replacing mission takes no key of the guest. title and briefing are text keys of the mod read
// through look; "{lang}" in map or texts becomes lang. A refusal names the file
// and the line.
func ParseMissions(id, file string, data []byte, look func(string) (string, bool), lang string) (MissionData, error) {
	tables, err := parseTOMLWith(data, tomlOptions{arrays: true})
	if err != nil {
		return MissionData{}, fileError(file, err)
	}
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	if len(tables[0].pairs) != 0 {
		p := tables[0].pairs[0]
		return MissionData{}, bad(p.line, "%s is outside a [[mission]] table", p.key)
	}
	var out MissionData
	keys, numbers, afters := map[string]int{}, map[int]int{}, map[int]int{}
	for _, t := range tables[1:] {
		if t.name != "mission" || !t.array {
			return MissionData{}, bad(t.line, "unknown table %q (this file holds [[mission]])", t.name)
		}
		m := Mission{Mod: id, File: file, Line: t.line, Kind: MissionGuest, Carry: CarryJoin, language: lang}
		have := map[string]bool{}
		for _, p := range t.pairs {
			have[p.key] = true
			number := func(dst *int, lo, hi int64) error {
				if p.val.kind != tomlInt {
					return bad(p.line, "%s must be an integer, not %s", p.key, p.val.kind)
				}
				if p.val.num < lo || p.val.num > hi {
					return bad(p.line, "%s is %d; use %d to %d", p.key, p.val.num, lo, hi)
				}
				*dst = int(p.val.num)
				return nil
			}
			str := func() (string, error) {
				if p.val.kind != tomlString {
					return "", bad(p.line, "%s must be a string, not %s", p.key, p.val.kind)
				}
				return p.val.str, nil
			}
			pathIn := func(dst *string) error {
				s, err := str()
				if err != nil {
					return err
				}
				s = strings.ReplaceAll(s, langToken, lang)
				if !insidePath(s) {
					return bad(p.line, "%s %q is not a path inside the mod folder", p.key, s)
				}
				*dst = path.Clean(s)
				return nil
			}
			text := func(dst *string) error {
				k, err := str()
				if err != nil {
					return err
				}
				v, ok := look(k)
				if !ok {
					return bad(p.line, "%s names text key %q, which text/%s/strings.toml and text/en/strings.toml do not hold", p.key, k, lang)
				}
				*dst = v
				return nil
			}
			var err error
			switch p.key {
			case "kind":
				if m.Kind, err = str(); err == nil && m.Kind != MissionGuest && m.Kind != MissionReplace {
					err = bad(p.line, "kind is %q; use %q or %q", m.Kind, MissionGuest, MissionReplace)
				}
			case "key":
				if m.Key, err = str(); err == nil && !itemKeyPattern.MatchString(m.Key) {
					err = bad(p.line, "key %q must be 1 to 32 of a-z, 0-9 and '_', starting with a letter", m.Key)
				}
			case "number":
				err = number(&m.Number, 1, maxMissionNumber)
			case "after":
				err = number(&m.After, 1, maxMissionNumber)
			case "guest":
				err = number(&m.Guest, 1, maxMissionNumber)
			case "chapter":
				err = number(&m.Chapter, 1, maxMissionNumber)
			case "gold":
				err = number(&m.Gold, 0, maxMissionGold)
			case "map":
				err = pathIn(&m.Map)
			case "texts":
				err = pathIn(&m.Texts)
			case "title":
				err = text(&m.Title)
			case "briefing":
				err = text(&m.Brief)
			case "carry":
				if m.Carry, err = str(); err == nil && m.Carry != CarryJoin {
					err = bad(p.line, "carry is %q; only %q is supported (the guest joins through the next town's grant)", m.Carry, CarryJoin)
				}
			default:
				err = bad(p.line, "unknown key %q in [[mission]]", p.key)
			}
			if err != nil {
				return MissionData{}, err
			}
		}
		need := []string{"key", "number", "map", "after", "guest", "chapter"}
		if m.Kind == MissionReplace {
			need = need[:3]
			for _, p := range t.pairs {
				if slices.Contains(guestOnly, p.key) {
					return MissionData{}, bad(p.line, "%s belongs to a guest mission; a replacing mission takes the campaign's", p.key)
				}
			}
			m.Carry = ""
		}
		for _, k := range need {
			if !have[k] {
				return MissionData{}, bad(t.line, "[[mission]] has no %s", k)
			}
		}
		if first, dup := keys[m.Key]; dup {
			return MissionData{}, bad(t.line, "mission key %q is already used at line %d", m.Key, first)
		}
		keys[m.Key] = t.line
		if first, dup := numbers[m.Number]; dup {
			return MissionData{}, bad(t.line, "mission number %d is already used at line %d", m.Number, first)
		}
		numbers[m.Number] = t.line
		if m.Kind == MissionReplace {
			out.Missions = append(out.Missions, m)
			continue
		}
		if first, dup := afters[m.After]; dup {
			return MissionData{}, bad(t.line, "a mission already plays after mission %d at line %d", m.After, first)
		}
		afters[m.After] = t.line
		if m.Number == m.After {
			return MissionData{}, bad(t.line, "mission %d cannot play after itself", m.Number)
		}
		out.Missions = append(out.Missions, m)
	}
	return out, nil
}

// guestOnly are the keys only a guest mission takes.
var guestOnly = []string{"after", "guest", "chapter", "gold", "carry"}

// insidePath reports that p is a relative slash path that stays inside the
// folder it is read from.
func insidePath(p string) bool {
	if p == "" || strings.ContainsAny(p, "\\:\x00") || strings.HasPrefix(p, "/") {
		return false
	}
	c := path.Clean(p)
	return c != "." && c != ".." && !strings.HasPrefix(c, "../")
}

// ReadFiles reads the map and the text files of m from the mod folder dir.
// Every file stays inside the folder; a refusal names the file and the line
// of the declaration.
func (m *Mission) ReadFiles(dir string) error {
	bad := func(format string, args ...any) error {
		return &ItemFileError{File: m.File, Line: m.Line, Msg: fmt.Sprintf(format, args...)}
	}
	b, _, err := ReadInside(dir, m.Map)
	if err != nil {
		return bad("map: %v", err)
	}
	m.MapBytes = b
	m.TextFiles, err = m.ReadTexts(dir, m.language)
	if err != nil {
		return bad("texts: %v", err)
	}
	return nil
}

// ReadTexts reads a map's pages before its declared mission text folder.
func (m Mission) ReadTexts(dir, lang string) (map[string][]byte, error) {
	if lang == "" {
		lang = path.Base(m.Texts)
		if lang != "en" && lang != "ru" {
			lang = "en"
		}
	}
	if lang != "en" && lang != "ru" {
		return nil, fmt.Errorf("unsupported page language %q", lang)
	}
	folder := strings.TrimSuffix(m.Map, path.Ext(m.Map)) + ".text"
	folders := []string{folder + "/" + lang, folder}
	fallback := ""
	if m.Texts != "" {
		declared := strings.ReplaceAll(m.Texts, "{lang}", lang)
		folders = append(folders, declared)
		if tail := path.Base(declared); tail == "en" || tail == "ru" {
			fallback = path.Dir(declared)
		}
	}
	files, err := readMissionTexts(dir, fallback, folders...)
	if errors.Is(err, os.ErrNotExist) && m.Texts == "" {
		return map[string][]byte{}, nil
	}
	return files, err
}

func readMissionTexts(dir, fallback string, folders ...string) (map[string][]byte, error) {
	t, err := OpenTree(dir)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	found := false
	if fallback != "" {
		folders = append(folders, fallback)
	}
	for _, folder := range folders {
		if !insidePath(folder) && !(folder == "." && folder == fallback) {
			return nil, fmt.Errorf("texts %q is not a path inside its folder", folder)
		}
		entries, err := t.ReadDir(folder)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		found = true
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if !missionTextName.MatchString(e.Name()) {
				if folder == fallback {
					continue
				}
				return nil, fmt.Errorf("%s is not a mission text file name (a-z, 0-9 and '_', then .txt)", filepath.Join(dir, filepath.FromSlash(folder), e.Name()))
			}
			if _, exists := files[e.Name()]; exists {
				continue
			}
			if len(files) >= MaxMissionTexts {
				return nil, fmt.Errorf("at most %d files", MaxMissionTexts)
			}
			b, _, err := t.ReadInside(folder + "/" + e.Name())
			if err != nil {
				return nil, err
			}
			files[e.Name()] = b
		}
	}
	if !found {
		return nil, fmt.Errorf("no such folder %s: %w", filepath.Join(dir, filepath.FromSlash(folders[len(folders)-1])), os.ErrNotExist)
	}
	return files, nil
}

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
	// MaxGuestParty bounds the members of a guest party.
	MaxGuestParty = 8
	// MaxGuestLevel is the highest level a guest party member may take: the
	// four forty-mission bands of the campaign's tier rows (HERO-JOIN-120).
	MaxGuestLevel = 4
	// maxGuestArmour is the armour and jewellery cells of a definition row.
	maxGuestArmour = 8
	// langToken in a path is replaced by the language the game runs in.
	langToken = "{lang}"
)

// missionTextName is the name of a file a mission's text directory serves.
var missionTextName = regexp.MustCompile(`^[a-z0-9_]{1,32}\.txt$`)

// Mission is one mission a mod declares: it runs as mission Number with the
// map Map and the texts of Texts, both inside the mod folder. A guest mission
// (Kind MissionGuest) plays after the player wins mission After: a main
// mission whose win opens a town, or another guest mission, which makes the
// two one chain that ends in that town. Its party is either companion Guest,
// as the town of chapter Chapter would grant her over the party, who joins in
// that town as she ended the mission (Carry), or the members of Party; Gold
// is the party's purse, and the waiting party comes back when the chain is
// won. A replacing mission (Kind MissionReplace) is the install's campaign
// mission Number, reached and continued as the campaign does; only its map
// and texts are the mod's.
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

	// Party are the members of a guest party declared as data; empty for
	// the guest form above. PartyLine is the line that declares them.
	Party     []GuestMember
	PartyLine int

	// Off switches the mission of Number off, with the line that does.
	Off     bool
	OffLine int

	language string

	// MapBytes and TextFiles are the map and the text files as read from the
	// mod folder when the mod loaded. TextFiles is keyed by file name.
	MapBytes  []byte
	TextFiles map[string][]byte
}

// GuestMember is one member of a guest party: a character of the character
// catalog at a level, wearing what the row sets. A nil Weapon or Shield, and a
// nil Armour, keep the cells of the character's own row; an empty string
// empties the cell.
type GuestMember struct {
	// Character is the catalog key: npc<n> for a record of the install's
	// npc.reg, or the key of a mod's [[character]].
	Character string
	// Level is the tier, 1 to MaxGuestLevel; 0 keeps the character's own.
	Level  int
	Weapon *string
	Shield *string
	Armour []string
}

// MissionData is the missions the mods of a set declare, in load order, with
// the load order the mods state.
type MissionData struct {
	Missions []Mission
	Order    Precedence
}

// Empty reports that no mod declares a mission.
func (d MissionData) Empty() bool { return len(d.Missions) == 0 }

// AfterWin returns the guest mission that plays after mission n is won: a
// main mission's, or the next link of a chain after a guest mission.
func (d MissionData) AfterWin(n int) (Mission, bool) {
	for _, m := range d.Missions {
		if m.Kind != MissionReplace && !m.Off && m.After == n {
			return m, true
		}
	}
	return Mission{}, false
}

// ByNumber returns the guest mission that runs as mission n.
func (d MissionData) ByNumber(n int) (Mission, bool) {
	for _, m := range d.Missions {
		if m.Kind != MissionReplace && !m.Off && m.Number == n {
			return m, true
		}
	}
	return Mission{}, false
}

// Chain is the chain of guest missions m ends: the first link, which plays
// after a main mission's win, through m. A catalog MergeMissions answered
// holds no cycle.
func (d MissionData) Chain(m Mission) []Mission {
	chain := []Mission{m}
	for len(chain) <= len(d.Missions) {
		prev, ok := d.ByNumber(chain[0].After)
		if !ok {
			break
		}
		chain = append([]Mission{prev}, chain...)
	}
	return chain
}

// Replacing returns the mission that replaces the map of campaign mission n.
func (d MissionData) Replacing(n int) (Mission, bool) {
	for _, m := range d.Missions {
		if m.Kind == MissionReplace && !m.Off && m.Number == n {
			return m, true
		}
	}
	return Mission{}, false
}

// Find returns the mission key of mod id.
func (d MissionData) Find(id, key string) (Mission, bool) {
	for _, m := range d.Missions {
		if m.Mod == id && m.Key == key && !m.Off {
			return m, true
		}
	}
	return Mission{}, false
}

// missionField is the one field a mission record sets: a later mod's mission
// of one number replaces the whole mission, since its map, texts and party
// are read together.
const missionField = "mission"

// MergeMissions merges the mods' mission records, in load order, over layer
// 0, which holds no mod mission: a record adds the mission its number names,
// a later mod's record of that number replaces the whole mission when
// load-after orders it after the earlier one, and off = true switches the
// mission off. The merged catalog is refused when two guest missions play
// after one mission, or when guest missions follow each other in a cycle.
// It answers the missions in the order they were first supplied and the
// missions a later mod replaced.
func MergeMissions(d MissionData) (MissionData, []Override, error) {
	patches := make([]Patch[int, Mission], 0, len(d.Missions))
	for _, m := range d.Missions {
		p := Patch[int, Mission]{Source: Source{Mod: m.Mod, File: m.File, Line: m.Line}, Key: m.Number, Off: m.Off}
		if !m.Off {
			p.Fields = append(p.Fields, PatchField{Name: missionField, Line: m.Line})
			mission := m
			p.Apply = func(r *Mission) { *r = mission }
		}
		if m.OffLine != 0 {
			p.Fields = append(p.Fields, PatchField{Name: OffField, Line: m.OffLine})
		}
		patches = append(patches, p)
	}
	l, err := Merge("mission", nil, func(m Mission) int { return m.Number }, patches, d.Order, nil)
	if err != nil {
		return MissionData{}, nil, err
	}
	out := MissionData{Missions: slices.Clone(l.Records), Order: d.Order}
	if err := out.checkChains(); err != nil {
		return MissionData{}, nil, err
	}
	return out, l.Overrides, nil
}

// checkChains refuses two guest missions after one mission and a cycle of
// guest missions, at the line of the later mission.
func (d MissionData) checkChains() error {
	at := func(m Mission) Source { return Source{Mod: m.Mod, File: m.File, Line: m.Line} }
	for i, m := range d.Missions {
		if m.Kind == MissionReplace {
			continue
		}
		for _, first := range d.Missions[:i] {
			if first.Kind != MissionReplace && first.After == m.After {
				return at(m).Errorf("mod %q already plays a mission after mission %d (mission %d, %s:%d); one mission follows one win",
					first.Mod, m.After, first.Number, first.File, first.Line)
			}
		}
	}
	for _, m := range d.Missions {
		if m.Kind == MissionReplace {
			continue
		}
		seen := []int{m.Number}
		for cur := m; ; {
			prev, ok := d.ByNumber(cur.After)
			if !ok {
				break
			}
			if slices.Contains(seen, prev.Number) {
				return at(m).Errorf("mission %d is in a cycle of guest missions (%s); a chain starts after a campaign mission",
					m.Number, cycleText(append(seen, prev.Number)))
			}
			seen = append(seen, prev.Number)
			cur = prev
		}
	}
	return nil
}

func cycleText(numbers []int) string {
	parts := make([]string, len(numbers))
	for i, n := range numbers {
		parts[i] = fmt.Sprint(n)
	}
	return strings.Join(parts, " after ")
}

// ParseMissions reads data/missions.toml of the mod id: [[mission]] tables
// with the keys kind, key, number, map, texts, title, briefing, after, guest,
// chapter, gold, carry, party and off. A replacing mission takes no key of
// the guest. title and briefing are text keys of the mod read through look;
// "{lang}" in map or texts becomes lang. A refusal names the file and the
// line.
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
			case "party":
				m.PartyLine = p.line
				m.Party, err = parseGuestParty(file, p)
			case OffField:
				if p.val.kind != tomlBool {
					err = bad(p.line, "off must be true or false, not %s", p.val.kind)
				} else if !p.val.flag {
					err = bad(p.line, "off = false switches nothing; leave the key out")
				}
				m.Off, m.OffLine = true, p.line
			default:
				err = bad(p.line, "unknown key %q in [[mission]]", p.key)
			}
			if err != nil {
				return MissionData{}, err
			}
		}
		if m.Off {
			for _, p := range t.pairs {
				if p.key != OffField && p.key != "number" && p.key != "kind" {
					return MissionData{}, bad(p.line, "[[mission]] switches mission %d off and sets %s; an off table holds number and off", m.Number, p.key)
				}
			}
			if !have["number"] {
				return MissionData{}, bad(t.line, "[[mission]] has no number")
			}
			if first, dup := numbers[m.Number]; dup {
				return MissionData{}, bad(t.line, "mission number %d is already used at line %d", m.Number, first)
			}
			numbers[m.Number] = t.line
			m.Carry = ""
			out.Missions = append(out.Missions, m)
			continue
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
		} else if have["party"] {
			need = need[:4]
			for _, p := range t.pairs {
				if slices.Contains(guestFormOnly, p.key) {
					return MissionData{}, bad(p.line, "%s belongs to the guest form; a mission with a party names its members in party", p.key)
				}
			}
			m.Carry = ""
		}
		for _, k := range need {
			if !have[k] {
				if k == "guest" {
					return MissionData{}, bad(t.line, "[[mission]] has no guest (or party)")
				}
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
var guestOnly = []string{"after", "guest", "chapter", "gold", "carry", "party"}

// guestFormOnly are the keys of the guest form a mission with a party does
// not take.
var guestFormOnly = []string{"guest", "chapter", "carry"}

// parseGuestParty reads party, an array of inline tables with the keys
// character, level, weapon, shield and armour.
func parseGuestParty(file string, p tomlPair) ([]GuestMember, error) {
	bad := func(format string, args ...any) error {
		return &ItemFileError{File: file, Line: p.line, Msg: fmt.Sprintf(format, args...)}
	}
	if p.val.kind != tomlArray {
		return nil, bad("party must be an array of { character = ... } tables, not %s", p.val.kind)
	}
	if n := len(p.val.list); n == 0 || n > MaxGuestParty {
		return nil, bad("party has %d members; a guest party has 1 to %d", n, MaxGuestParty)
	}
	var out []GuestMember
	for i, v := range p.val.list {
		if v.kind != tomlInline {
			return nil, bad("party member %d is %s; write { character = \"...\" }", i+1, v.kind)
		}
		var g GuestMember
		has := map[string]bool{}
		for _, f := range v.tbl {
			has[f.key] = true
			str := func() (string, error) {
				if f.val.kind != tomlString {
					return "", bad("party member %d: %s must be a string, not %s", i+1, f.key, f.val.kind)
				}
				return f.val.str, nil
			}
			var err error
			switch f.key {
			case "character":
				if g.Character, err = str(); err == nil && g.Character == "" {
					err = bad("party member %d: character is empty", i+1)
				}
			case "level":
				switch {
				case f.val.kind != tomlInt:
					err = bad("party member %d: level must be an integer, not %s", i+1, f.val.kind)
				case f.val.num < 1 || f.val.num > MaxGuestLevel:
					err = bad("party member %d: level is %d; use 1 to %d", i+1, f.val.num, MaxGuestLevel)
				default:
					g.Level = int(f.val.num)
				}
			case "weapon", "shield":
				var s string
				if s, err = str(); err == nil {
					if f.key == "weapon" {
						g.Weapon = &s
					} else {
						g.Shield = &s
					}
				}
			case "armour":
				if f.val.kind != tomlArray {
					err = bad("party member %d: armour must be an array of item names, not %s", i+1, f.val.kind)
					break
				}
				if len(f.val.list) > maxGuestArmour {
					err = bad("party member %d: armour names %d items; a row holds %d", i+1, len(f.val.list), maxGuestArmour)
					break
				}
				g.Armour = []string{}
				for _, a := range f.val.list {
					if a.kind != tomlString {
						err = bad("party member %d: armour holds %s; use item names as strings", i+1, a.kind)
						break
					}
					g.Armour = append(g.Armour, a.str)
				}
			default:
				err = bad("party member %d: unknown key %q (character, level, weapon, shield, armour)", i+1, f.key)
			}
			if err != nil {
				return nil, err
			}
		}
		if !has["character"] {
			return nil, bad("party member %d has no character", i+1)
		}
		if slices.ContainsFunc(out, func(o GuestMember) bool { return o.Character == g.Character }) {
			return nil, bad("party names character %q twice", g.Character)
		}
		out = append(out, g)
	}
	return out, nil
}

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

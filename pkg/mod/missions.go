package mod

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/herokind"
	"againrom/pkg/locale"
)

// The kinds of a mission declaration. A guest mission is a new mission a
// guest party plays between a win and the town it opens; a replacing mission
// is a campaign mission of the install played on the mod's map. A Mission
// whose Kind is empty is a guest mission.
const (
	MissionGuest    = "guest"
	MissionReplace  = "replace"
	MissionCampaign = "campaign"
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
	// Campaign rows may replace routing and payment without replacing a map.
	AutoGetMission *int `json:",omitempty"`
	AutoLine       int  `json:",omitempty"`
	Payment        *int `json:",omitempty"`
	MainParty      bool `json:",omitempty"`

	// Class and Sex name the starting heroes a guest mission plays for, as
	// herokind.Parse reads them; both empty is every hero. Missions after one
	// win whose heroes do not overlap are its alternatives (Heroes).
	Class, Sex string

	// Party are the members of a guest party declared as data; empty for
	// the guest form above. PartyLine is the line that declares them.
	Party     []GuestMember
	PartyLine int

	// Off switches the mission of Number off, with the line that does.
	Off     bool
	OffLine int

	language string
	// mapTemplate is the declared map path that holds langToken, and
	// textsTemplate the declared texts path, before langToken was replaced.
	mapTemplate, textsTemplate string
	// byName marks a replacing mission the file form declares: a map whose
	// file name is the mission number (readMissionMaps).
	byName bool

	// Carried are other files of the mod a test play writes beside the mission,
	// keyed by their path in the mod.
	Carried map[string][]byte `json:",omitempty"`

	// MapBytes and TextFiles are the map and the text files as read from the
	// mod folder when the mod loaded. TextFiles is keyed by file name.
	MapBytes  []byte
	TextFiles map[string][]byte
	// OtherLanguageTexts names the TextFiles read from the other language.
	OtherLanguageTexts map[string]bool
}

// GuestMember is one member of a guest party: a character of the character
// catalog at a level, wearing what the row sets. A nil Weapon or Shield, and a
// nil Armour, keep the cells of the character's own row; an empty string
// empties the cell.
type GuestMember struct {
	// Character is the catalog key: npc<n> for a record of the install's
	// npc.reg, or the key of a mod's [[character]].
	Character string
	// Hero is the member that is the waiting party's own starting hero as
	// he stands; it names no character and takes no level or cell.
	Hero bool
	// Level is the tier, 1 to MaxGuestLevel; 0 keeps the character's own.
	Level  int
	Weapon *string
	Shield *string
	Armour []string
	// Skills, when GuestSkills, opens the mission with every skill of the
	// member at the rules' cap; 0 keeps the member's own.
	Skills int
	// Spells, when GuestSpellsAll, opens the mission with every spell the
	// member can learn in his book; empty keeps the member's own.
	Spells string
}

const (
	// GuestSkills is the one value the skills key of a party member takes.
	GuestSkills = 100
	// GuestSpellsAll is the one value the spells key of a party member takes.
	GuestSpellsAll = "all"
)

// ID is the member's name in a party: the character's key, or the hero's
// word.
func (g GuestMember) ID() string {
	if g.Hero {
		return "hero"
	}
	return g.Character
}

// MissionData is the missions the mods of a set declare, in load order, with
// the load order the mods state.
type MissionData struct {
	Missions []Mission
	Order    Precedence

	// maps are the file form's replacing missions, in load order, until
	// AdmitMaps keeps those whose campaign map the install holds.
	maps []missionMap
}

// Empty reports that no mod declares a mission.
func (d MissionData) Empty() bool { return len(d.Missions) == 0 && len(d.maps) == 0 }

// Heroes is the set of starting heroes the mission plays for.
func (m Mission) Heroes() herokind.Set { return herokind.Named(m.Class, m.Sex) }

func (m Mission) Campaign() bool { return m.Kind == MissionReplace || m.Kind == MissionCampaign }

func (m Mission) GuestParty() bool { return m.Guest != 0 || len(m.Party) != 0 }

// after is the guest missions that play after mission n is won, for any
// starting hero.
func (d MissionData) after(n int) []Mission {
	var out []Mission
	for _, m := range d.Missions {
		if !m.Campaign() && !m.Off && m.After == n {
			out = append(out, m)
		}
	}
	return out
}

// AfterWin returns the guest mission that plays after mission n is won for a
// starting hero of kind k: a main mission's, or the next link of a chain
// after a guest mission. The hero chooses among the missions after n by
// their class and sex.
func (d MissionData) AfterWin(n int, k herokind.Kind) (Mission, bool) {
	missions := d.after(n)
	sets := make([]herokind.Set, len(missions))
	for i, m := range missions {
		sets[i] = m.Heroes()
	}
	if i := herokind.Choose(k, sets, -1); i >= 0 {
		return missions[i], true
	}
	return Mission{}, false
}

// Follows reports that a guest mission plays after mission n for some
// starting hero.
func (d MissionData) Follows(n int) bool { return len(d.after(n)) > 0 }

// ByNumber returns the guest mission that runs as mission n.
func (d MissionData) ByNumber(n int) (Mission, bool) {
	for _, m := range d.Missions {
		if !m.Campaign() && !m.Off && m.Number == n {
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
		if m.Campaign() && m.Map != "" && !m.Off && m.Number == n {
			return m, true
		}
	}
	return Mission{}, false
}

func (d MissionData) CampaignMission(n int) (Mission, bool) {
	for _, m := range d.Missions {
		if m.Campaign() && !m.Off && m.Number == n {
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

// checkChains refuses two guest missions after one mission for one starting
// hero and a cycle of guest missions, at the line of the later mission.
func (d MissionData) checkChains() error {
	at := func(m Mission) Source { return Source{Mod: m.Mod, File: m.File, Line: m.Line} }
	for i, m := range d.Missions {
		if m.Campaign() {
			continue
		}
		for _, first := range d.Missions[:i] {
			if !first.Campaign() && first.After == m.After && first.Heroes().Overlaps(m.Heroes()) {
				return at(m).Errorf("mod %q already plays a mission after mission %d%s (mission %d, %s:%d); one mission follows one win",
					first.Mod, m.After, forHeroes(first.Heroes()&m.Heroes()), first.Number, first.File, first.Line)
			}
		}
	}
	for _, m := range d.Missions {
		if m.Campaign() {
			continue
		}
		hero := func(m Mission) bool { return slices.ContainsFunc(m.Party, func(g GuestMember) bool { return g.Hero }) }
		if prev, ok := d.ByNumber(m.After); ok && hero(m) && !hero(prev) {
			return Source{Mod: m.Mod, File: m.File, Line: m.PartyLine}.Errorf("mission %d names the hero, and mission %d before it in the chain does not; the hero enters a chain at its first link",
				m.Number, prev.Number)
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

// forHeroes names the starting heroes two missions after one win share, ""
// for every hero.
func forHeroes(shared herokind.Set) string {
	if shared == herokind.Every {
		return ""
	}
	return " for " + shared.String()
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
// chapter, gold, carry, party, class, sex and off. A replacing mission takes
// no key of the guest. title and briefing are text keys of the mod read through look;
// "{lang}" in map or texts becomes lang. A refusal names the file and the
// line. The missions kind adds the file form's declarations of the same mod
// after its rows (readMissionMaps).
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
	type afterAt struct {
		line   int
		heroes herokind.Set
	}
	keys, numbers, afters := map[string]int{}, map[int]int{}, map[int][]afterAt{}
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
			pathIn := func(ref Reference, dst *string) error {
				s, err := str()
				if err != nil {
					return err
				}
				*dst, err = ref.Check(file, p.line, p.key, missionPath(s, lang))
				return err
			}
			text := func(dst *string) error {
				k, err := str()
				if err != nil {
					return err
				}
				v, ok := look(k)
				if !ok {
					return bad(p.line, "%s names text key %q, which %s and %s do not hold", p.key, k, StringsFile(lang), StringsFile(locale.Fallback))
				}
				*dst = v
				return nil
			}
			var err error
			switch p.key {
			case "kind":
				if m.Kind, err = str(); err == nil && m.Kind != MissionGuest && !m.Campaign() {
					err = bad(p.line, "kind is %q; use %q or %q or %q", m.Kind, MissionGuest, MissionReplace, MissionCampaign)
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
			case "AutoGetMission", "Payment":
				v, lo, hi := 0, int64(-1), int64(maxMissionNumber)
				if p.key == "Payment" {
					lo, hi = 0, maxMissionGold
				}
				if err = number(&v, lo, hi); err == nil {
					if p.key == "AutoGetMission" {
						m.AutoGetMission, m.AutoLine = &v, p.line
					} else {
						m.Payment = &v
					}
				}
			case "map":
				if err = pathIn(MissionMap, &m.Map); err == nil && strings.Contains(p.val.str, langToken) {
					m.mapTemplate = p.val.str
				}
			case "texts":
				if err = pathIn(MissionTexts, &m.Texts); err == nil && p.val.kind == tomlString {
					m.textsTemplate = p.val.str
				}
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
				if p.val.kind == tomlString && p.val.str == "main" {
					m.MainParty = true
				} else {
					m.Party, err = parseGuestParty(file, p)
				}
			case "class", "sex":
				var word string
				if word, err = str(); err == nil {
					class, sex := m.Class, m.Sex
					if p.key == "class" {
						class = word
					} else {
						sex = word
					}
					if _, perr := herokind.Parse(class, sex); perr != nil {
						err = bad(p.line, "%v", perr)
					}
					m.Class, m.Sex = class, sex
				}
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
		if m.Campaign() {
			need = need[:3]
			if m.Kind == MissionCampaign {
				need = need[:2]
			}
			for _, p := range t.pairs {
				if p.key == "class" || p.key == "sex" {
					return MissionData{}, bad(p.line, "%s belongs to a guest mission; a replacing mission plays for every starting hero", p.key)
				}
				if p.key == "after" {
					return MissionData{}, bad(p.line, "%s belongs to a guest mission; a replacing mission takes the campaign's", p.key)
				}
			}
			if m.Guest == 0 {
				if have["chapter"] || have["carry"] {
					return MissionData{}, bad(t.line, "chapter and carry require guest")
				}
				m.Carry = ""
			} else if !have["chapter"] {
				return MissionData{}, bad(t.line, "guest requires chapter")
			}
			if m.MainParty && m.GuestParty() || m.Guest != 0 && len(m.Party) != 0 {
				return MissionData{}, bad(t.line, "choose party = main, guest with chapter, or a party array")
			}
			if !m.GuestParty() && have["gold"] {
				return MissionData{}, bad(t.line, "gold belongs to a guest mission party")
			}
		} else if have["party"] {
			if m.MainParty {
				return MissionData{}, bad(m.PartyLine, "party = main requires kind = campaign or replace")
			}
			need = need[:4]
			for _, p := range t.pairs {
				if slices.Contains(guestFormOnly, p.key) {
					return MissionData{}, bad(p.line, "%s belongs to the guest form; a mission with a party names its members in party", p.key)
				}
			}
			m.Carry = ""
		}
		if !m.Campaign() && (m.AutoGetMission != nil || m.Payment != nil) {
			return MissionData{}, bad(t.line, "AutoGetMission and Payment require kind = campaign or replace")
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
		if m.Campaign() {
			out.Missions = append(out.Missions, m)
			continue
		}
		for _, first := range afters[m.After] {
			if shared := first.heroes & m.Heroes(); shared != 0 {
				return MissionData{}, bad(t.line, "a mission already plays after mission %d%s at line %d", m.After, forHeroes(shared), first.line)
			}
		}
		afters[m.After] = append(afters[m.After], afterAt{t.line, m.Heroes()})
		if m.Number == m.After {
			return MissionData{}, bad(t.line, "mission %d cannot play after itself", m.Number)
		}
		out.Missions = append(out.Missions, m)
	}
	return out, nil
}

// guestFormOnly are the keys of the guest form a mission with a party does
// not take.
var guestFormOnly = []string{"guest", "chapter", "carry"}

// parseGuestParty reads party, an array of inline tables with the keys
// character or hero, level, weapon, shield, armour, skills and spells.
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
			case "hero":
				if f.val.kind != tomlBool || !f.val.flag {
					err = bad("party member %d: hero must be true; a member that is not the hero names a character", i+1)
				}
				g.Hero = true
			case "skills":
				if f.val.kind != tomlInt || f.val.num != GuestSkills {
					err = bad("party member %d: skills must be %d (every skill at the rules' cap); leave the key out to keep the member's own", i+1, GuestSkills)
				}
				g.Skills = GuestSkills
			case "spells":
				if f.val.kind != tomlString || f.val.str != GuestSpellsAll {
					err = bad("party member %d: spells must be %q (every spell the member can learn); leave the key out to keep the member's own", i+1, GuestSpellsAll)
				}
				g.Spells = GuestSpellsAll
			default:
				err = bad("party member %d: unknown key %q (character, hero, level, weapon, shield, armour, skills, spells)", i+1, f.key)
			}
			if err != nil {
				return nil, err
			}
		}
		if g.Hero {
			for _, key := range []string{"character", "level", "weapon", "shield", "armour"} {
				if has[key] {
					return nil, bad("party member %d is the hero and takes no %s; the hero comes as he stands", i+1, key)
				}
			}
		} else if !has["character"] {
			return nil, bad("party member %d has no character", i+1)
		}
		if slices.ContainsFunc(out, func(o GuestMember) bool { return o.ID() == g.ID() && o.Hero == g.Hero }) {
			if g.Hero {
				return nil, bad("party names the hero twice")
			}
			return nil, bad("party names character %q twice", g.Character)
		}
		out = append(out, g)
	}
	return out, nil
}

func missionPath(p, lang string) string { return strings.ReplaceAll(p, langToken, lang) }

// ReadFiles reads the map and the text files of m from the mod folder dir.
// Every file stays inside the folder; a refusal names the file and the line
// of the declaration.
func (m *Mission) ReadFiles(dir string) error {
	bad := func(format string, args ...any) error {
		return &ItemFileError{File: m.File, Line: m.Line, Msg: fmt.Sprintf(format, args...)}
	}
	if m.Map != "" {
		b, _, err := MissionMap.ReadIn(dir, m.Map)
		if err != nil {
			return bad("map: %v", err)
		}
		m.MapBytes = b
	}
	if m.Map == "" && m.Texts == "" {
		return nil
	}
	layer, err := m.ReadPageLayer(PageMod, dir, m.language)
	m.TextFiles, m.OtherLanguageTexts = layer.Files, layer.Other
	if err != nil {
		return bad("texts: %v", err)
	}
	return nil
}

// ReadTexts reads the pages the map carries in its texts record first, then
// the map's text folders, then its declared mission text folder. A name none
// of lang supplies is read from the other language's pages, the record's
// before the folders'; other names those.
func (m Mission) ReadTexts(dir, lang string) (files map[string][]byte, other map[string]bool, err error) {
	lang = m.pageLang(lang)
	if !locale.Known(lang) {
		return nil, nil, fmt.Errorf("unsupported page language %q", lang)
	}
	otherLang := locale.Other(lang)
	inside, held, err := m.mapTexts(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", m.Map, err)
	}
	record := insidePages{held: held, own: [2]map[string][]byte{{}, {}}, other: map[string][]byte{}}
	for name, b := range inside {
		folder, page := path.Split(name)
		switch folder {
		case lang + "/":
			record.own[0][page] = b
		case "":
			record.own[1][page] = b
		case otherLang + "/":
			record.other[page] = b
		}
	}
	place := MissionPlace{Map: m.Map}
	folders := []string{place.Pages(lang), place.Pages("")}
	foldersOther := []string{place.Pages(otherLang)}
	fallback, declared := "", ""
	if m.Texts != "" {
		declared = missionPath(m.Texts, lang)
		folders = append(folders, declared)
		if tail := path.Base(declared); locale.Known(tail) {
			fallback = path.Dir(declared)
		}
		template := m.textsTemplate
		if template == "" {
			template = m.Texts
		}
		if declaredOther := path.Clean(missionPath(template, otherLang)); declaredOther != declared {
			foldersOther = append(foldersOther, declaredOther)
		}
	}
	files, other, err = readMissionTexts(dir, fallback, declared, folders, foldersOther, record)
	if errors.Is(err, os.ErrNotExist) && m.Texts == "" {
		return map[string][]byte{}, nil, nil
	}
	return files, other, err
}

// mapTexts is the page tree the map's texts record holds, keyed as in its
// text folder, and whether the map carries the record: from MapBytes, else
// from the map file in dir. A map that cannot be read or framed carries none.
func (m Mission) mapTexts(dir string) (map[string][]byte, bool, error) {
	b := m.MapBytes
	if b == nil {
		if m.Map == "" {
			return nil, false, nil
		}
		var err error
		if b, _, err = MissionMap.ReadIn(dir, m.Map); err != nil {
			return nil, false, nil
		}
	}
	return alm.TextsOf(b)
}

// insidePages is a map's texts record split for one language: own holds the
// language's pages, then the shared ones; other the other language's.
type insidePages struct {
	held  bool
	own   [2]map[string][]byte
	other map[string][]byte
}

// readMissionTexts reads the record's own pages, folders in order, then the
// parent fallback, then the record's other-language pages and foldersOther
// for the names left. In foldersOther and the record's other-language pages
// a bad path or file name is passed over.
func readMissionTexts(dir, fallback, declared string, folders, foldersOther []string, record insidePages) (map[string][]byte, map[string]bool, error) {
	files := map[string][]byte{}
	other := map[string]bool{}
	found := record.held
	take := func(pages map[string][]byte, otherLanguage bool) error {
		for _, name := range slices.Sorted(maps.Keys(pages)) {
			if !missionTextName.MatchString(name) {
				if otherLanguage {
					continue
				}
				return fmt.Errorf("the map's texts record holds %s, which is not a mission text file name (a-z, 0-9 and '_', then .txt)", name)
			}
			if _, exists := files[name]; exists {
				continue
			}
			if len(files) >= MaxMissionTexts {
				return fmt.Errorf("at most %d files", MaxMissionTexts)
			}
			files[name] = pages[name]
			if otherLanguage {
				other[name] = true
			}
		}
		return nil
	}
	for _, pages := range record.own {
		if err := take(pages, false); err != nil {
			return nil, nil, err
		}
	}
	t, err := OpenTree(dir)
	if err != nil {
		if record.held {
			err = take(record.other, true)
			return files, other, err
		}
		return nil, nil, err
	}
	if fallback != "" {
		folders = append(folders, fallback)
	}
	read := func(folder string, otherLanguage bool) error {
		if _, err := MissionTexts.Path(folder); err != nil && !(folder == "." && folder == fallback) {
			// A folder derived from the map's name is one byte longer than
			// the name, so the rule can refuse it for a name it admits. No
			// mod holds such a folder: only the declared one keeps the rule.
			if otherLanguage || folder != declared && folder != fallback {
				return nil
			}
			return fmt.Errorf("texts %q is not a path inside its folder", folder)
		}
		entries, err := t.ReadDir(folder)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if !missionTextName.MatchString(e.Name()) {
				if folder == fallback || otherLanguage {
					continue
				}
				return fmt.Errorf("%s is not a mission text file name (a-z, 0-9 and '_', then .txt)", filepath.Join(dir, filepath.FromSlash(folder), e.Name()))
			}
			if _, exists := files[e.Name()]; exists {
				continue
			}
			if len(files) >= MaxMissionTexts {
				return fmt.Errorf("at most %d files", MaxMissionTexts)
			}
			b, _, err := t.ReadInside(folder + "/" + e.Name())
			if err != nil {
				return err
			}
			files[e.Name()] = b
			if otherLanguage {
				other[e.Name()] = true
			}
		}
		return nil
	}
	for _, folder := range folders {
		if err := read(folder, false); err != nil {
			return nil, nil, err
		}
	}
	if err := take(record.other, true); err != nil {
		return nil, nil, err
	}
	for _, folder := range foldersOther {
		if err := read(folder, true); err != nil {
			return nil, nil, err
		}
	}
	if !found {
		return nil, nil, fmt.Errorf("no such folder %s: %w", filepath.Join(dir, filepath.FromSlash(folders[len(folders)-1])), os.ErrNotExist)
	}
	return files, other, nil
}

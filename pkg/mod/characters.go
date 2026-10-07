package mod

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// CharactersFile is the data file that edits named characters. A mod loads it
// from its entry script with game.data.add.
const CharactersFile = "data/characters.toml"

// Equipment groups the strip key names. A person's definition row carries ten
// equipment cells: the weapon, the shield, and eight armour and jewellery
// cells.
const (
	StripWeapon = "weapon"
	StripShield = "shield"
	StripArmour = "armour"
)

// StripGroups lists the groups the strip key accepts.
func StripGroups() []string { return []string{StripWeapon, StripShield, StripArmour} }

const (
	maxCharacterNameRunes = 40
	// MaxFace is the highest face sheet number a character can name; the
	// installed sheets stop well below it.
	MaxFace = 127
)

// CharacterRow is one edit of a named character. Every field but Target is
// optional, and an absent field leaves the character as the game has it.
type CharacterRow struct {
	Mod  string
	File string
	Line int

	// Target names the definition rows the edit applies to: one row by its
	// full name, or every tier of a family by the name without its tier
	// ("NPC06" names NPC06_1 to NPC06_4). Nothing else is touched.
	Target string

	// NameKey is the text key of the display name; Name is its text in the
	// language the game runs in. Empty when the edit leaves the name alone.
	NameKey  string
	NameLine int
	Name     string

	// Kind names a definition row, or a family of tier rows by the name without
	// its tier digit, whose class and figure columns (type, face sheet, sex)
	// the character takes. KindLine is the line that gives it.
	Kind     string
	KindLine int

	// Face is the face sheet to draw with, 0 for the sheet the row (or its
	// kind) states. It is applied after Kind.
	Face int32

	// Strip lists equipment groups the character no longer wears or carries.
	Strip     []string
	StripLine int
}

// CharacterData is what the mods of a set edit, in load order.
type CharacterData struct {
	Rows []CharacterRow
}

// Empty reports that no mod edits a character.
func (d CharacterData) Empty() bool { return len(d.Rows) == 0 }

// ParseCharacters reads data/characters.toml of the mod id. text resolves a text
// key to its string in the game's language; lang names that language in a
// refusal. A refusal names the file and the line.
func ParseCharacters(id, file string, data []byte, text func(key string) (string, bool), lang string) (CharacterData, error) {
	tables, err := parseTOMLWith(data, tomlOptions{arrays: true})
	if err != nil {
		return CharacterData{}, fileError(file, err)
	}
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	if len(tables[0].pairs) != 0 {
		p := tables[0].pairs[0]
		return CharacterData{}, bad(p.line, "%s is outside a [[character]] table", p.key)
	}
	var out CharacterData
	for _, t := range tables[1:] {
		if t.name != "character" || !t.array {
			return CharacterData{}, bad(t.line, "unknown table %q (this file holds [[character]])", t.name)
		}
		row, err := characterRowFrom(id, file, t, text, lang)
		if err != nil {
			return CharacterData{}, err
		}
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

func characterRowFrom(id, file string, t tomlTable, text func(string) (string, bool), lang string) (CharacterRow, error) {
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	r := CharacterRow{Mod: id, File: file, Line: t.line}
	have := map[string]int{}
	nameLine := t.line
	for _, p := range t.pairs {
		have[p.key] = p.line
		str := func() (string, error) {
			if p.val.kind != tomlString {
				return "", bad(p.line, "%s must be a string, not %s", p.key, p.val.kind)
			}
			if strings.TrimSpace(p.val.str) == "" {
				return "", bad(p.line, "%s is empty", p.key)
			}
			return p.val.str, nil
		}
		var err error
		switch p.key {
		case "target":
			r.Target, err = str()
		case "name":
			nameLine = p.line
			r.NameKey, err = str()
		case "kind":
			r.KindLine = p.line
			r.Kind, err = str()
		case "face":
			switch {
			case p.val.kind != tomlInt:
				err = bad(p.line, "face must be an integer, not %s", p.val.kind)
			case p.val.num < 1 || p.val.num > MaxFace:
				err = bad(p.line, "face is %d, outside 1..%d", p.val.num, MaxFace)
			default:
				r.Face = int32(p.val.num)
			}
		case "strip":
			r.StripLine = p.line
			r.Strip, err = stripGroups(file, p)
		default:
			err = bad(p.line, "unknown key %q in [[character]]", p.key)
		}
		if err != nil {
			return CharacterRow{}, err
		}
	}
	if _, ok := have["target"]; !ok {
		return CharacterRow{}, bad(t.line, "[[character]] has no target")
	}
	if r.NameKey == "" && r.Kind == "" && r.Face == 0 && len(r.Strip) == 0 {
		return CharacterRow{}, bad(t.line, "[[character]] of %q changes nothing", r.Target)
	}
	if r.NameKey != "" {
		name, ok := text(r.NameKey)
		if !ok {
			return CharacterRow{}, bad(nameLine, "text key %q is not in %s (nor in %s)", r.NameKey, StringsFile(lang), StringsFile("en"))
		}
		if n := utf8.RuneCountInString(name); n == 0 || n > maxCharacterNameRunes {
			return CharacterRow{}, bad(nameLine, "the text of %q has %d characters; a character name has 1 to %d", r.NameKey, n, maxCharacterNameRunes)
		}
		r.Name, r.NameLine = name, nameLine
	}
	return r, nil
}

func stripGroups(file string, p tomlPair) ([]string, error) {
	bad := func(format string, args ...any) error {
		return &ItemFileError{File: file, Line: p.line, Msg: fmt.Sprintf(format, args...)}
	}
	if p.val.kind != tomlArray {
		return nil, bad("strip must be an array of groups, not %s", p.val.kind)
	}
	var out []string
	seen := map[string]bool{}
	for _, v := range p.val.list {
		if v.kind != tomlString {
			return nil, bad("strip holds %s; use group names as strings", v.kind)
		}
		ok := false
		for _, g := range StripGroups() {
			ok = ok || g == v.str
		}
		if !ok {
			return nil, bad("unknown strip group %q (groups: %s)", v.str, strings.Join(StripGroups(), ", "))
		}
		if seen[v.str] {
			return nil, bad("strip names %q twice", v.str)
		}
		seen[v.str] = true
		out = append(out, v.str)
	}
	if len(out) == 0 {
		return nil, bad("strip is empty")
	}
	return out, nil
}

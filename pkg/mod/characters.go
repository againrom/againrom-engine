package mod

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// CharactersFile is the data file that edits named characters and adds
// characters to the character catalog. A mod loads it from its entry script
// with game.data.add.
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

// installCharacterKey is the key shape of the install's npc.reg records in
// the character catalog; a mod's own character never takes it.
var installCharacterKey = regexp.MustCompile(`^npc[0-9]+$`)

// InstallCharacterKey is the catalog key of npc.reg record n.
func InstallCharacterKey(n int32) string { return fmt.Sprintf("npc%d", n) }

// CharacterRow is one [[character]] table: an edit of named definition rows
// (Target), or a character of the catalog (Key). Every field but the one that
// names the record is optional, and an absent field leaves the record as the
// layers below have it.
type CharacterRow struct {
	Mod  string
	File string
	Line int

	// Target names the definition rows the edit applies to: one row by its
	// full name, or every tier of a family by the name without its tier
	// ("NPC06" names NPC06_1 to NPC06_4). Nothing else is touched.
	Target string

	// Key names a character of the catalog this mod adds; Row is the
	// definition row, or a family of tier rows, it is built from.
	Key     string
	Row     string
	RowLine int

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
	Face     int32
	FaceLine int

	// Strip lists equipment groups the character no longer wears or carries.
	Strip     []string
	StripLine int

	// Off switches the character Key off, with the line that does.
	Off     bool
	OffLine int
}

// CharacterData is what the mods of a set edit and add, in load order, with
// the load order the mods state.
type CharacterData struct {
	Rows  []CharacterRow
	Order Precedence
}

// Empty reports that no mod edits or adds a character.
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
		case "key":
			r.Key, err = str()
			switch {
			case err != nil:
			case installCharacterKey.MatchString(r.Key):
				err = bad(p.line, "key %q is an install character's (npc.reg); a mod's character takes a key of its own", r.Key)
			case !itemKeyPattern.MatchString(r.Key):
				err = bad(p.line, "key %q must be 1 to 32 of a-z, 0-9 and '_', starting with a letter", r.Key)
			}
		case "row":
			r.RowLine = p.line
			r.Row, err = str()
		case "name":
			nameLine = p.line
			r.NameKey, err = str()
		case "kind":
			r.KindLine = p.line
			r.Kind, err = str()
		case "face":
			r.FaceLine = p.line
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
		case OffField:
			r.OffLine = p.line
			switch {
			case p.val.kind != tomlBool:
				err = bad(p.line, "off must be true or false, not %s", p.val.kind)
			case !p.val.flag:
				err = bad(p.line, "off = false switches nothing; leave the key out")
			default:
				r.Off = true
			}
		default:
			err = bad(p.line, "unknown key %q in [[character]]", p.key)
		}
		if err != nil {
			return CharacterRow{}, err
		}
	}
	_, hasTarget := have["target"]
	_, hasKey := have["key"]
	switch {
	case hasTarget && hasKey:
		return CharacterRow{}, bad(t.line, "[[character]] has both target and key; an edit names a target, a character a key")
	case !hasTarget && !hasKey:
		return CharacterRow{}, bad(t.line, "[[character]] has no target (an edit) or key (a character)")
	case hasTarget:
		for _, k := range []string{"row", OffField} {
			if line, ok := have[k]; ok {
				return CharacterRow{}, bad(line, "%s belongs to a character (key); an edit of %q takes name, kind, face and strip", k, r.Target)
			}
		}
		if r.NameKey == "" && r.Kind == "" && r.Face == 0 && len(r.Strip) == 0 {
			return CharacterRow{}, bad(t.line, "[[character]] of %q changes nothing", r.Target)
		}
	default:
		for _, k := range []string{"kind", "strip"} {
			if line, ok := have[k]; ok {
				return CharacterRow{}, bad(line, "%s edits a definition row (target); character %q takes row, name, face and off", k, r.Key)
			}
		}
		if r.Off && len(t.pairs) != 2 {
			return CharacterRow{}, bad(t.line, "[[character]] %q switches the character off and sets other keys", r.Key)
		}
		if !r.Off && r.Row == "" && r.NameKey == "" && r.Face == 0 {
			return CharacterRow{}, bad(t.line, "[[character]] %q sets nothing", r.Key)
		}
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

// Character is one record of the character catalog: a character a guest
// party may name. Layer 0 is the install's npc.reg records, keyed npc<n>
// (NPC); a mod's [[character]] with a key adds a character built from a
// definition row (Row). Both are one record shape, so a party names either
// the same way.
type Character struct {
	Source
	Key string
	// NPC is the npc.reg record of an install character; zero for a mod's.
	NPC int32
	// Row is the definition row, or a family of tier rows, a mod's character
	// is built from; RowLine the line that names it.
	Row     string
	RowLine int
	// NameKey and Name are the display name a mod gives the character; empty
	// keeps the name its row or record has.
	NameKey  string
	Name     string
	NameLine int
	// Face, when positive, is the face sheet the character is drawn with.
	Face int32
}

// InstallCharacters is layer 0 of the character catalog: one record per
// npc.reg record n, in the registry's order.
func InstallCharacters(npcs []int32) []Character {
	out := make([]Character, 0, len(npcs))
	for _, n := range npcs {
		out = append(out, Character{Key: InstallCharacterKey(n), NPC: n})
	}
	return out
}

// MergeCharacters preserves definition-row operations in load order and
// merges catalog characters by key over base. Target operations may overlap
// without load-after; each kind, face and strip applies at its own position.
// New catalog keys retain Merge's addition and field-conflict rules.
func MergeCharacters(d CharacterData, base []Character) ([]CharacterRow, []Character, []Override, error) {
	var edits []CharacterRow
	var chars []Patch[string, Character]
	for _, c := range d.Rows {
		src := Source{Mod: c.Mod, File: c.File, Line: c.Line}
		if c.Target != "" {
			edits = append(edits, c)
			continue
		}
		row := c
		p := Patch[string, Character]{Source: src, Key: c.Key, Off: c.Off}
		if row.Row != "" {
			p.Fields = append(p.Fields, PatchField{Name: "row", Line: row.RowLine})
		}
		if row.Name != "" || row.NameKey != "" {
			p.Fields = append(p.Fields, PatchField{Name: "name", Line: row.NameLine})
		}
		if row.Face != 0 {
			p.Fields = append(p.Fields, PatchField{Name: "face", Line: row.FaceLine})
		}
		if row.OffLine != 0 {
			p.Fields = append(p.Fields, PatchField{Name: OffField, Line: row.OffLine})
		}
		p.Apply = func(r *Character) {
			if r.Key == "" {
				r.Source, r.Key = src, row.Key
			}
			if row.Row != "" {
				r.Row, r.RowLine = row.Row, row.RowLine
			}
			if row.Name != "" || row.NameKey != "" {
				r.NameKey, r.Name, r.NameLine = row.NameKey, row.Name, row.NameLine
			}
			if row.Face != 0 {
				r.Face = row.Face
			}
		}
		chars = append(chars, p)
	}
	c, err := Merge("character", base, func(r Character) string { return r.Key }, chars, d.Order, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, r := range c.Records {
		if r.NPC == 0 && r.Row == "" {
			return nil, nil, nil, c.Named[r.Key].Errorf("character %q names no row; a character a mod adds is built from a definition row", r.Key)
		}
	}
	return edits, c.Records, c.Overrides, nil
}

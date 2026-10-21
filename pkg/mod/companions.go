package mod

import (
	"fmt"
	"slices"
)

// CompanionsFile is the data file that holds companion join conditions. A mod
// loads it from its entry script with game.data.add.
const CompanionsFile = "data/companions.toml"

// Buildings a join condition can name. A condition fires when the conversation
// of that building's list opens; only the tavern is wired.
const (
	BuildingTavern = "tavern"
)

// Bounds of the numbers a join condition names: the campaign registry stores
// its record numbers as 16-bit values.
const (
	maxJoinNumber = 65535
)

// CompanionJoin is one join condition: the campaign companion Companion, whom
// the town of chapter Chapter grants on arrival, joins the party only when the
// player opens the conversation Talk of Building in that town. As a layer
// record it names its condition by Chapter and Companion; Fields are the
// fields it sets, and Off switches the condition off.
type CompanionJoin struct {
	Mod  string
	File string
	Line int

	Key       string
	Companion int
	Chapter   int
	Building  string
	Talk      int

	Fields []PatchField
	Off    bool
}

// CompanionKey names one join condition: the companion and the chapter whose
// town grants it.
type CompanionKey struct {
	Companion, Chapter int
}

func (k CompanionKey) String() string {
	return fmt.Sprintf("%d of chapter %d", k.Companion, k.Chapter)
}

// companionFields are the fields a new join condition sets.
var companionFields = []string{"key", "building", "talk"}

// RecordKey is the condition the record names.
func (j CompanionJoin) RecordKey() CompanionKey {
	return CompanionKey{Companion: j.Companion, Chapter: j.Chapter}
}

// PatchFields are the fields the record sets. A record built without Fields
// sets every field.
func (j CompanionJoin) PatchFields() []PatchField {
	if j.Fields != nil {
		return j.Fields
	}
	out := make([]PatchField, 0, len(companionFields))
	for _, f := range companionFields {
		out = append(out, PatchField{Name: f, Line: j.Line})
	}
	return out
}

// Missing is the first field a new condition needs that the record does not
// set; "" when it sets them all.
func (j CompanionJoin) Missing() string {
	for _, need := range companionFields {
		if !slices.ContainsFunc(j.PatchFields(), func(f PatchField) bool { return f.Name == need }) {
			return need
		}
	}
	return ""
}

// Apply writes the fields the record sets into c, with the record's source.
func (j CompanionJoin) Apply(c *CompanionJoin) {
	c.Mod, c.File, c.Line = j.Mod, j.File, j.Line
	c.Companion, c.Chapter = j.Companion, j.Chapter
	for _, f := range j.PatchFields() {
		switch f.Name {
		case "key":
			c.Key = j.Key
		case "building":
			c.Building = j.Building
		case "talk":
			c.Talk = j.Talk
		}
	}
	c.Fields, c.Off = nil, false
}

// CompanionData is the join conditions the mods of a set declare, in load
// order, with the load order the mods state.
type CompanionData struct {
	Joins []CompanionJoin
	Order Precedence
}

// Empty reports that no mod declares a join condition.
func (d CompanionData) Empty() bool { return len(d.Joins) == 0 }

// Find returns the condition for the companion granted by a chapter.
func (d CompanionData) Find(chapter, companion int) (CompanionJoin, bool) {
	for _, j := range d.Joins {
		if j.Chapter == chapter && j.Companion == companion {
			return j, true
		}
	}
	return CompanionJoin{}, false
}

// ForTalk returns the conditions the conversation talk of building in the town
// of chapter starts, in load order.
func (d CompanionData) ForTalk(chapter int, building string, talk int) []CompanionJoin {
	var out []CompanionJoin
	for _, j := range d.Joins {
		if j.Chapter == chapter && j.Building == building && j.Talk == talk {
			out = append(out, j)
		}
	}
	return out
}

// ParseCompanions reads data/companions.toml of the mod id: [[join]] tables
// with the keys key, companion, chapter, building, talk and off. A table names
// its condition by chapter and companion; one that adds a condition sets every
// key, which the layer step judges. A refusal names the file and the line.
func ParseCompanions(id, file string, data []byte) (CompanionData, error) {
	tables, err := parseTOMLWith(data, tomlOptions{arrays: true})
	if err != nil {
		return CompanionData{}, fileError(file, err)
	}
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	if len(tables[0].pairs) != 0 {
		p := tables[0].pairs[0]
		return CompanionData{}, bad(p.line, "%s is outside a [[join]] table", p.key)
	}
	var out CompanionData
	keys := map[string]int{}
	granted := map[[2]int]int{}
	for _, t := range tables[1:] {
		if t.name != "join" || !t.array {
			return CompanionData{}, bad(t.line, "unknown table %q (this file holds [[join]])", t.name)
		}
		j := CompanionJoin{Mod: id, File: file, Line: t.line, Fields: []PatchField{}}
		have := map[string]bool{}
		for _, p := range t.pairs {
			have[p.key] = true
			if p.key != "companion" && p.key != "chapter" {
				j.Fields = append(j.Fields, PatchField{Name: p.key, Line: p.line})
			}
			number := func(dst *int) error {
				if p.val.kind != tomlInt {
					return bad(p.line, "%s must be an integer, not %s", p.key, p.val.kind)
				}
				if p.val.num < 1 || p.val.num > maxJoinNumber {
					return bad(p.line, "%s is %d; use 1 to %d", p.key, p.val.num, maxJoinNumber)
				}
				*dst = int(p.val.num)
				return nil
			}
			var err error
			switch p.key {
			case "key":
				if p.val.kind != tomlString {
					err = bad(p.line, "key must be a string, not %s", p.val.kind)
				} else if j.Key = p.val.str; !itemKeyPattern.MatchString(j.Key) {
					err = bad(p.line, "key %q must be 1 to 32 of a-z, 0-9 and '_', starting with a letter", j.Key)
				}
			case "companion":
				err = number(&j.Companion)
			case "chapter":
				err = number(&j.Chapter)
			case "talk":
				err = number(&j.Talk)
			case "building":
				if p.val.kind != tomlString {
					err = bad(p.line, "building must be a string, not %s", p.val.kind)
				} else if j.Building = p.val.str; j.Building != BuildingTavern {
					err = bad(p.line, "building is %q; only %q conversations can start a join", j.Building, BuildingTavern)
				}
			case OffField:
				if p.val.kind != tomlBool {
					err = bad(p.line, "off must be true or false, not %s", p.val.kind)
				}
				j.Off = p.val.flag
			default:
				err = bad(p.line, "unknown key %q in [[join]]", p.key)
			}
			if err != nil {
				return CompanionData{}, err
			}
		}
		for _, need := range []string{"companion", "chapter"} {
			if !have[need] {
				return CompanionData{}, bad(t.line, "[[join]] has no %s", need)
			}
		}
		switch {
		case len(j.Fields) == 0:
			return CompanionData{}, bad(t.line, "[[join]] for companion %d of chapter %d changes nothing", j.Companion, j.Chapter)
		case j.Off && len(j.Fields) > 1:
			return CompanionData{}, bad(t.line, "[[join]] for companion %d of chapter %d switches the condition off and sets other keys; off = true stands alone", j.Companion, j.Chapter)
		}
		if have["key"] {
			if first, dup := keys[j.Key]; dup {
				return CompanionData{}, bad(t.line, "join key %q is already used at line %d", j.Key, first)
			}
			keys[j.Key] = t.line
		}
		pair := [2]int{j.Chapter, j.Companion}
		if first, dup := granted[pair]; dup {
			return CompanionData{}, bad(t.line, "companion %d of chapter %d already has a join condition at line %d", j.Companion, j.Chapter, first)
		}
		granted[pair] = t.line
		out.Joins = append(out.Joins, j)
	}
	return out, nil
}

// MergeCompanions merges the mods' join records, in load order, over layer
// 0, which holds no condition: the base grants every town companion on
// arrival. A record that adds a condition sets every field; a later mod's
// record of the same companion and chapter changes the fields it names when
// load-after orders it after the earlier one, and off = true switches the
// condition off. It answers the conditions in the order they were added and
// the fields a later mod set over an earlier one.
func MergeCompanions(d CompanionData) (CompanionData, []Override, error) {
	patches := make([]Patch[CompanionKey, CompanionJoin], 0, len(d.Joins))
	held := map[CompanionKey]bool{}
	for _, j := range d.Joins {
		k := j.RecordKey()
		if need := j.Missing(); !held[k] && !j.Off && need != "" {
			return CompanionData{}, nil, Source{Mod: j.Mod, File: j.File, Line: j.Line}.Errorf("[[join]] has no %s", need)
		}
		held[k] = true
		patches = append(patches, Patch[CompanionKey, CompanionJoin]{Source: Source{Mod: j.Mod, File: j.File, Line: j.Line},
			Key: k, Fields: j.PatchFields(), Off: j.Off, Apply: j.Apply})
	}
	l, err := Merge("companion", nil, CompanionJoin.RecordKey, patches, d.Order, nil)
	if err != nil {
		return CompanionData{}, nil, err
	}
	return CompanionData{Joins: l.Records, Order: d.Order}, l.Overrides, nil
}

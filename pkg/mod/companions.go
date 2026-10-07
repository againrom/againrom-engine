package mod

import (
	"fmt"
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
// player opens the conversation Talk of Building in that town.
type CompanionJoin struct {
	Mod  string
	File string
	Line int

	Key       string
	Companion int
	Chapter   int
	Building  string
	Talk      int
}

// CompanionData is the join conditions the mods of a set declare, in load
// order.
type CompanionData struct {
	Joins []CompanionJoin
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
// with the keys key, companion, chapter, building and talk. A refusal names the
// file and the line.
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
		j := CompanionJoin{Mod: id, File: file, Line: t.line}
		have := map[string]bool{}
		for _, p := range t.pairs {
			have[p.key] = true
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
			default:
				err = bad(p.line, "unknown key %q in [[join]]", p.key)
			}
			if err != nil {
				return CompanionData{}, err
			}
		}
		for _, need := range []string{"key", "companion", "chapter", "building", "talk"} {
			if !have[need] {
				return CompanionData{}, bad(t.line, "[[join]] has no %s", need)
			}
		}
		if first, dup := keys[j.Key]; dup {
			return CompanionData{}, bad(t.line, "join key %q is already used at line %d", j.Key, first)
		}
		keys[j.Key] = t.line
		pair := [2]int{j.Chapter, j.Companion}
		if first, dup := granted[pair]; dup {
			return CompanionData{}, bad(t.line, "companion %d of chapter %d already has a join condition at line %d", j.Companion, j.Chapter, first)
		}
		granted[pair] = t.line
		out.Joins = append(out.Joins, j)
	}
	return out, nil
}

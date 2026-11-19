package mod

import (
	"fmt"
	"slices"
	"strconv"
)

// writeCompanions is the companions kind's writer: source made to hold
// d.Companions.Joins, each [[join]] table kept where it stands with its lines
// but for the keys it changes. A condition the source does not hold, its Line
// zero, is appended; one d lacks leaves with its table.
func writeCompanions(rel string, source []byte, d Data) ([]byte, error) {
	before, err := ParseCompanions("", rel, source)
	if err != nil {
		return nil, err
	}
	after := companionSourceRows(d.Companions.Joins)
	out, err := writeRows(source, companionSourceRows(before.Joins), after, func(key string) string { return key })
	if err != nil {
		return nil, err
	}
	back, err := ParseCompanions("", rel, out)
	if err != nil {
		return nil, err
	}
	if !sameRows(companionSourceRows(back.Joins), after) {
		return nil, fmt.Errorf("%s: the written rows do not read back as the rows given", rel)
	}
	return out, nil
}

// companionSourceRows is the conditions as the row writer sees them: a kept
// condition is matched by its line, and writes the keys its record sets.
func companionSourceRows(joins []CompanionJoin) []sourceRow {
	rows, lines := make([]sourceRow, len(joins)), make([]int, len(joins))
	for i, j := range joins {
		pairs := []missionPair{{"companion", strconv.Itoa(j.Companion)}, {"chapter", strconv.Itoa(j.Chapter)}}
		if j.Off {
			pairs = append(pairs, missionPair{OffField, "true"})
		} else {
			set := func(name string) bool {
				return slices.ContainsFunc(j.PatchFields(), func(f PatchField) bool { return f.Name == name })
			}
			if set("key") {
				pairs = append(pairs, missionPair{"key", strconv.Quote(j.Key)})
			}
			if set("building") {
				pairs = append(pairs, missionPair{"building", strconv.Quote(j.Building)})
			}
			if set("talk") {
				pairs = append(pairs, missionPair{"talk", strconv.Itoa(j.Talk)})
			}
		}
		rows[i], lines[i] = sourceRow{table: "join", pairs: pairs}, j.Line
	}
	numberRows(rows, lines)
	return rows
}

package mod

import (
	"fmt"
	"strconv"
	"strings"
)

func writeCharacters(_ string, source []byte, d Data) ([]byte, error) {
	look := func(string) (string, bool) { return "-", true }
	before, err := ParseCharacters("", CharactersFile, source, look, "")
	if err != nil {
		return nil, err
	}
	after := characterSourceRows(d.Characters)
	out, err := writeRows(source, characterSourceRows(before), after, func(key string) string { return key })
	if err != nil {
		return nil, err
	}
	back, err := ParseCharacters("", CharactersFile, out, look, "")
	if err != nil {
		return nil, err
	}
	if !sameRows(characterSourceRows(back), after) {
		return nil, fmt.Errorf("%s: written characters do not read back as given", CharactersFile)
	}
	return out, nil
}

func characterSourceRows(d CharacterData) []sourceRow {
	rows := make([]sourceRow, 0, len(d.Rows))
	lines := make([]int, 0, len(d.Rows))
	for _, c := range d.Rows {
		r := sourceRow{table: "character"}
		add := func(key, value string) {
			if value != "" {
				r.pairs = append(r.pairs, missionPair{key, strconv.Quote(value)})
			}
		}
		add("target", c.Target)
		add("key", c.Key)
		add("row", c.Row)
		add("name", c.NameKey)
		add("kind", c.Kind)
		if c.Face != 0 {
			r.pairs = append(r.pairs, missionPair{"face", strconv.Itoa(int(c.Face))})
		}
		if len(c.Strip) != 0 {
			parts := make([]string, len(c.Strip))
			for i, v := range c.Strip {
				parts[i] = strconv.Quote(v)
			}
			r.pairs = append(r.pairs, missionPair{"strip", "[" + strings.Join(parts, ", ") + "]"})
		}
		if c.Off {
			r.pairs = append(r.pairs, missionPair{OffField, "true"})
		}
		r.identity = c.Target + "\x00" + c.Key
		rows = append(rows, r)
		lines = append(lines, c.Line)
	}
	numberRows(rows, lines)
	return rows
}

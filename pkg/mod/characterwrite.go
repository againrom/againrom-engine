package mod

import (
	"fmt"
	"strconv"
)

func writeCharacters(_ string, source []byte, d Data) ([]byte, error) {
	look := func(string) (string, bool) { return "-", true }
	before, err := ParseCharacters("", CharactersFile, source, look, "")
	if err != nil {
		return nil, err
	}
	was, err := characterSourceRows(before)
	if err != nil {
		return nil, err
	}
	after, err := characterSourceRows(d.Characters)
	if err != nil {
		return nil, err
	}
	out, err := writeRows(source, was, after, func(key string) string { return key })
	if err != nil {
		return nil, err
	}
	back, err := ParseCharacters("", CharactersFile, out, look, "")
	if err != nil {
		return nil, err
	}
	again, err := characterSourceRows(back)
	if err != nil {
		return nil, err
	}
	if !sameRows(again, after) {
		return nil, fmt.Errorf("%s: written characters do not read back as given", CharactersFile)
	}
	return out, nil
}

func characterSourceRows(d CharacterData) ([]sourceRow, error) {
	rows := make([]sourceRow, 0, len(d.Rows))
	lines := make([]int, 0, len(d.Rows))
	for _, c := range d.Rows {
		r := sourceRow{table: "character"}
		add := func(key, value string) error {
			if value == "" {
				return nil
			}
			q, err := writeString(value)
			if err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
			r.pairs = append(r.pairs, missionPair{key, q})
			return nil
		}
		for _, f := range []missionPair{{"target", c.Target}, {"key", c.Key}, {"row", c.Row}, {"name", c.NameKey}, {"kind", c.Kind}} {
			if err := add(f.key, f.value); err != nil {
				return nil, err
			}
		}
		if c.Face != 0 {
			r.pairs = append(r.pairs, missionPair{"face", strconv.Itoa(int(c.Face))})
		}
		if len(c.Strip) != 0 {
			q, err := writeStringList(c.Strip)
			if err != nil {
				return nil, fmt.Errorf("strip: %w", err)
			}
			r.pairs = append(r.pairs, missionPair{"strip", q})
		}
		if c.Off {
			r.pairs = append(r.pairs, missionPair{OffField, "true"})
		}
		r.identity = c.Target + "\x00" + c.Key
		rows = append(rows, r)
		lines = append(lines, c.Line)
	}
	numberRows(rows, lines)
	return rows, nil
}

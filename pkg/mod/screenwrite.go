package mod

import (
	"fmt"
	"strings"
)

// writeScreens is the screens kind's writer: source made to hold
// d.Screens.Screens, each [[screen]] and [[action]] table kept where it
// stands with its lines but for the keys it changes. A screen the source does
// not hold, its Line zero, is appended; one d lacks leaves with its table.
// The rows read back with any text key taken as present: whether a key has a
// text is the strings files' business.
func writeScreens(rel string, source []byte, d Data) ([]byte, error) {
	anyText := func(string) (string, bool) { return "x", true }
	before, err := ParseScreens("", rel, source, anyText, "")
	if err != nil {
		return nil, err
	}
	after, err := screenSourceRows(d.Screens.Screens)
	if err != nil {
		return nil, err
	}
	was, err := screenSourceRows(before.Screens)
	if err != nil {
		return nil, err
	}
	out, err := writeRows(source, was, after, func(key string) string { return key })
	if err != nil {
		return nil, err
	}
	back, err := ParseScreens("", rel, out, anyText, "")
	if err != nil {
		return nil, err
	}
	again, err := screenSourceRows(back.Screens)
	if err != nil {
		return nil, err
	}
	if !sameRows(again, after) {
		return nil, fmt.Errorf("%s: the written rows do not read back as the rows given", rel)
	}
	return out, nil
}

// screenSourceRows is the screens as the row writer sees them: a kept screen
// is matched by its line.
func screenSourceRows(list []Screen) ([]sourceRow, error) {
	rows, lines := make([]sourceRow, len(list)), make([]int, len(list))
	for i, s := range list {
		row := sourceRow{table: "screen", identity: "screen"}
		add := func(key, value string) error {
			q, err := writeString(value)
			if err != nil {
				return fmt.Errorf("screen %q: %s: %w", s.Key, key, err)
			}
			row.pairs = append(row.pairs, missionPair{key, q})
			return nil
		}
		if IsAction(s.Kind) {
			row.table = "action"
			for _, f := range []missionPair{{"key", s.Key}, {"action", s.Kind}, {"menu", s.MenuKey}} {
				if err := add(f.key, f.value); err != nil {
					return nil, err
				}
			}
			if s.ConfirmKey != "" {
				if err := add("confirm", s.ConfirmKey); err != nil {
					return nil, err
				}
			}
		} else {
			place := ""
			switch {
			case s.Main && s.Game:
				place = PlaceBoth
			case s.Main:
				place = PlaceMain
			case s.Game:
				place = PlaceGame
			default:
				return nil, fmt.Errorf("screen %q has no menu to open from", s.Key)
			}
			for _, f := range []missionPair{{"key", s.Key}, {"kind", s.Kind}, {"title", s.TitleKey}, {"menu", s.MenuKey}, {"place", place}} {
				if err := add(f.key, f.value); err != nil {
					return nil, err
				}
			}
			switch s.Kind {
			case ScreenInfo:
				q, err := writeStringList(s.TextKeys)
				if err != nil {
					return nil, fmt.Errorf("screen %q: text: %w", s.Key, err)
				}
				row.pairs = append(row.pairs, missionPair{"text", q})
			case ScreenList:
				q, err := writeStringList(s.ItemKeys)
				if err != nil {
					return nil, fmt.Errorf("screen %q: items: %w", s.Key, err)
				}
				row.pairs = append(row.pairs, missionPair{"items", q})
			case ScreenTable:
				pairs := make([]string, len(s.RowKeys))
				for j, r := range s.RowKeys {
					q, err := writeStringList(r[:])
					if err != nil {
						return nil, fmt.Errorf("screen %q: rows: %w", s.Key, err)
					}
					pairs[j] = q
				}
				row.pairs = append(row.pairs, missionPair{"rows", "[" + strings.Join(pairs, ", ") + "]"})
			}
		}
		rows[i], lines[i] = row, s.Line
	}
	numberRows(rows, lines)
	return rows, nil
}

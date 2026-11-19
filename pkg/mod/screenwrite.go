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
		strs := func(keys []string) string {
			quoted := make([]string, len(keys))
			for j, k := range keys {
				quoted[j] = writeString(k)
			}
			return "[" + strings.Join(quoted, ", ") + "]"
		}
		row := sourceRow{table: "screen", identity: "screen"}
		if IsAction(s.Kind) {
			row.table = "action"
			row.pairs = []missionPair{{"key", writeString(s.Key)}, {"action", writeString(s.Kind)}, {"menu", writeString(s.MenuKey)}}
			if s.ConfirmKey != "" {
				row.pairs = append(row.pairs, missionPair{"confirm", writeString(s.ConfirmKey)})
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
			row.pairs = []missionPair{{"key", writeString(s.Key)}, {"kind", writeString(s.Kind)}, {"title", writeString(s.TitleKey)},
				{"menu", writeString(s.MenuKey)}, {"place", writeString(place)}}
			switch s.Kind {
			case ScreenInfo:
				row.pairs = append(row.pairs, missionPair{"text", strs(s.TextKeys)})
			case ScreenList:
				row.pairs = append(row.pairs, missionPair{"items", strs(s.ItemKeys)})
			case ScreenTable:
				pairs := make([]string, len(s.RowKeys))
				for j, r := range s.RowKeys {
					pairs[j] = "[" + writeString(r[0]) + ", " + writeString(r[1]) + "]"
				}
				row.pairs = append(row.pairs, missionPair{"rows", "[" + strings.Join(pairs, ", ") + "]"})
			}
		}
		rows[i], lines[i] = row, s.Line
	}
	numberRows(rows, lines)
	return rows, nil
}

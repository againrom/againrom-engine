package mod

import (
	"fmt"
	"strconv"
	"strings"

	"againrom/pkg/locale"
)

// SettingData is the settings a mod declares, in file order.
type SettingData struct {
	Settings []Setting
}

// writeSettings is the settings kind's writer: source made to hold
// d.Settings.Settings, each [table] kept where it stands with its lines but
// for the keys it changes. A setting the source does not hold is appended;
// one d lacks leaves with its table.
func writeSettings(rel string, source []byte, d Data) ([]byte, error) {
	before, err := ParseSettings(source)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	after := settingSourceRows(d.Settings.Settings)
	out, err := writeRows(source, settingSourceRows(before), after, func(key string) string { return key })
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	back, err := ParseSettings(out)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	if !sameRows(settingSourceRows(back), after) {
		return nil, fmt.Errorf("%s: the written rows do not read back as the rows given", rel)
	}
	return out, nil
}

// settingSourceRows is the settings as the row writer sees them: a setting is
// a plain table named by its key.
func settingSourceRows(list []Setting) []sourceRow {
	rows := make([]sourceRow, len(list))
	for i, s := range list {
		var pairs []missionPair
		if v := settingLabelValue(s.Labels); v != "" {
			pairs = append(pairs, missionPair{"label", v})
		}
		pairs = append(pairs, missionPair{"type", writeString(s.Kind.String())})
		switch s.Kind {
		case KindInt:
			pairs = append(pairs, missionPair{"default", strconv.FormatInt(s.Default.Int, 10)},
				missionPair{"min", strconv.FormatInt(s.Min, 10)}, missionPair{"max", strconv.FormatInt(s.Max, 10)})
		case KindBool:
			pairs = append(pairs, missionPair{"default", strconv.FormatBool(s.Default.Bool)})
		case KindChoice:
			quoted := make([]string, len(s.Choices))
			for j, c := range s.Choices {
				quoted[j] = writeString(c)
			}
			pairs = append(pairs, missionPair{"default", writeString(s.Default.Str)}, missionPair{"choices", "[" + strings.Join(quoted, ", ") + "]"})
		}
		rows[i] = sourceRow{table: s.Key, identity: s.Key, plain: true, pairs: pairs}
	}
	return rows
}

// settingLabelValue is the label pair's value: a string for a label of the
// reference language alone, else a table of the languages' labels; "" for
// none.
func settingLabelValue(labels map[string]string) string {
	var parts []string
	for _, l := range locale.All() {
		if v := labels[l.Code]; v != "" {
			parts = append(parts, l.Code+" = "+writeString(v))
		}
	}
	switch {
	case len(parts) == 0:
		return ""
	case len(parts) == 1 && labels[locale.Fallback] != "":
		return writeString(labels[locale.Fallback])
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

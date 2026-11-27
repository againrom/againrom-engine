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
	was, err := settingSourceRows(before)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	after, err := settingSourceRows(d.Settings.Settings)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	out, err := writeRows(source, was, after, func(key string) string { return key })
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	back, err := ParseSettings(out)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	again, err := settingSourceRows(back)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	if !sameRows(again, after) {
		return nil, fmt.Errorf("%s: the written rows do not read back as the rows given", rel)
	}
	return out, nil
}

// settingSourceRows is the settings as the row writer sees them: a setting is
// a plain table named by its key.
func settingSourceRows(list []Setting) ([]sourceRow, error) {
	rows := make([]sourceRow, len(list))
	for i, s := range list {
		var pairs []missionPair
		label, err := settingLabelValue(s.Labels)
		if err != nil {
			return nil, fmt.Errorf("setting %s: label: %w", s.Key, err)
		}
		if label != "" {
			pairs = append(pairs, missionPair{"label", label})
		}
		typ, err := writeString(s.Kind.String())
		if err != nil {
			return nil, fmt.Errorf("setting %s: type: %w", s.Key, err)
		}
		pairs = append(pairs, missionPair{"type", typ})
		switch s.Kind {
		case KindInt:
			pairs = append(pairs, missionPair{"default", strconv.FormatInt(s.Default.Int, 10)},
				missionPair{"min", strconv.FormatInt(s.Min, 10)}, missionPair{"max", strconv.FormatInt(s.Max, 10)})
		case KindBool:
			pairs = append(pairs, missionPair{"default", strconv.FormatBool(s.Default.Bool)})
		case KindChoice:
			def, err := writeString(s.Default.Str)
			if err != nil {
				return nil, fmt.Errorf("setting %s: default: %w", s.Key, err)
			}
			choices, err := writeStringList(s.Choices)
			if err != nil {
				return nil, fmt.Errorf("setting %s: choices: %w", s.Key, err)
			}
			pairs = append(pairs, missionPair{"default", def}, missionPair{"choices", choices})
		}
		rows[i] = sourceRow{table: s.Key, identity: s.Key, plain: true, pairs: pairs}
	}
	return rows, nil
}

// settingLabelValue is the label pair's value: a string for a label of the
// reference language alone, else a table of the languages' labels; "" for
// none.
func settingLabelValue(labels map[string]string) (string, error) {
	var parts []string
	for _, l := range locale.All() {
		if v := labels[l.Code]; v != "" {
			q, err := writeString(v)
			if err != nil {
				return "", fmt.Errorf("%s: %w", l.Code, err)
			}
			parts = append(parts, l.Code+" = "+q)
		}
	}
	switch {
	case len(parts) == 0:
		return "", nil
	case len(parts) == 1 && labels[locale.Fallback] != "":
		return writeString(labels[locale.Fallback])
	}
	return "{ " + strings.Join(parts, ", ") + " }", nil
}

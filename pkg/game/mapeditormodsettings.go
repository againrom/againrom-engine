package game

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// settingsAddKey is the Settings page's adder.
const settingsAddKey = "newsetting"

// settingsGate refuses a save while the project's settings file is one the
// game cannot read, naming the file and the row the reader names.
func (s *mapInspectionEdits) settingsGate() error {
	d := s.workspace.Project
	if d == nil {
		return nil
	}
	if _, ok := d.Files[mod.SettingsName]; !ok {
		return nil
	}
	if _, err := s.heldRows(d, nil, mod.SettingsName); err != nil {
		return fmt.Errorf(s.inspector.word("editor.settings.err.save"), mod.SettingsName, err)
	}
	return nil
}

// settingsFields is W's Settings page: the Add button, then each setting's
// label per language, type, default, bounds or choices, and removal.
func (s *mapInspectionEdits) settingsFields() []ui.InspectionField {
	x := s.inspector
	if !s.modPageHolds() {
		return nil
	}
	project := s.workspace.Project
	if project == nil {
		project = x.mod.document()
	}
	held, err := s.heldRows(project, nil, mod.SettingsName)
	if err != nil {
		return []ui.InspectionField{x.pageLine("error", "", err.Error())}
	}
	out := []ui.InspectionField{{Key: settingsAddKey, Label: x.word("editor.settings.add"), Meaning: x.word("editor.settings.add.help"), Action: true, Column: 1}}
	kinds := []ui.InspectionChoice{{Value: int64(mod.KindInt), Label: x.word("editor.settings.type.int")},
		{Value: int64(mod.KindBool), Label: x.word("editor.settings.type.bool")}, {Value: int64(mod.KindChoice), Label: x.word("editor.settings.type.choice")}}
	for i, st := range held.Settings.Settings {
		key := "setting." + strconv.Itoa(i) + "."
		start := len(out)
		for _, l := range textLanguages() {
			f := workspaceText(key+"label."+l, fmt.Sprintf(x.word("editor.settings.label"), languageLabel(l)), st.Labels[l])
			f.Meaning = x.word("editor.settings.label.help")
			out = append(out, f)
		}
		out = append(out, ui.InspectionField{Key: key + "type", Label: x.word("editor.settings.type"), Meaning: x.word("editor.settings.type.help"), Value: int64(st.Kind), Choices: kinds})
		switch st.Kind {
		case mod.KindInt:
			out = append(out, s.settingNumber(key+"default", x.word("editor.settings.default"), x.word("editor.settings.default.help"), st.Default.Int),
				s.settingNumber(key+"min", x.word("editor.settings.min"), x.word("editor.settings.min.help"), st.Min), s.settingNumber(key+"max", x.word("editor.settings.max"), x.word("editor.settings.max.help"), st.Max))
		case mod.KindBool:
			v := int64(0)
			if st.Default.Bool {
				v = 1
			}
			out = append(out, ui.InspectionField{Key: key + "default", Label: x.word("editor.settings.default"), Meaning: x.word("editor.settings.default.help"), Value: v,
				Choices: []ui.InspectionChoice{{Value: 0, Label: x.word("editor.settings.no")}, {Value: 1, Label: x.word("editor.settings.yes")}}})
		case mod.KindChoice:
			f := ui.InspectionField{Key: key + "default", Label: x.word("editor.settings.default"), Meaning: x.word("editor.settings.default.help"),
				Value: int64(slices.Index(st.Choices, st.Default.Str))}
			for j, c := range st.Choices {
				f.Choices = append(f.Choices, ui.InspectionChoice{Value: int64(j), Label: x.inspectionUTF8(c)})
			}
			choices := workspaceText(key+"choices", x.word("editor.settings.choices"), strings.Join(st.Choices, ", "))
			choices.Meaning = x.word("editor.settings.choices.help")
			out = append(out, f, choices)
		}
		out = append(out, ui.InspectionField{Key: key + "remove", Label: x.word("editor.settings.remove"), Meaning: x.word("editor.settings.remove.help"), Action: true, Column: 1})
		for k := start; k < len(out); k++ {
			out[k].Group = st.Key
		}
	}
	return out
}

// settingNumber is the text field of a number of a setting.
func (s *mapInspectionEdits) settingNumber(key, label, hint string, v int64) ui.InspectionField {
	f := workspaceText(key, label, strconv.FormatInt(v, 10))
	f.Meaning = hint
	return f
}

// editSettings is one draft step of the settings file in the project w.
func (s *mapInspectionEdits) editSettings(w *editorWorkspaceState, edit func(*mod.SettingData) error) error {
	return kindFiles{s: s, w: w}.editRows(nil, []string{mod.SettingsName}, func(d *mod.Data) error { return edit(&d.Settings) })
}

// setSettingsField makes one edit of the Settings page a draft step.
func (s *mapInspectionEdits) setSettingsField(key, value string) (*ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	x := s.inspector
	if err := x.modWritable(); err != nil {
		return nil, err
	}
	value = strings.TrimSpace(value)
	gone := fmt.Errorf(x.word("editor.settings.err.gone"), key)
	if key == settingsAddKey {
		return s.editKind(func(k kindFiles) error {
			return s.editSettings(k.w, func(d *mod.SettingData) error { s.addSetting(d); return nil })
		})
	}
	kind, rest, _ := strings.Cut(key, ".")
	index, field, _ := strings.Cut(rest, ".")
	i, err := strconv.Atoi(index)
	if err != nil || i < 0 || kind != "setting" {
		return nil, gone
	}
	return s.editKind(func(k kindFiles) error {
		return s.editSettings(k.w, func(d *mod.SettingData) error {
			if i >= len(d.Settings) {
				return gone
			}
			return s.editSetting(d, i, field, value, gone)
		})
	})
}

// addSetting appends an int setting of a key none holds, labelled in each
// language.
func (s *mapInspectionEdits) addSetting(d *mod.SettingData) {
	key := mod.RecordKey("setting", func(k string) bool {
		return slices.ContainsFunc(d.Settings, func(o mod.Setting) bool { return o.Key == k })
	})
	labels := map[string]string{}
	for _, l := range textLanguages() {
		labels[l] = words.For(l).Text("editor.settings.newlabel")
	}
	d.Settings = append(d.Settings, mod.Setting{Key: key, Labels: labels, Kind: mod.KindInt, Min: 0, Max: 10})
	d.Settings[len(d.Settings)-1].Default = mod.Value{Kind: mod.KindInt}
}

// settingNumberOf is the whole number text names.
func (x *MapInspector) settingNumberOf(text string) (int64, error) {
	n, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0, errors.New(x.word("editor.settings.err.number"))
	}
	return n, nil
}

// editSetting makes the edit of field of setting i. A change of a bound or of
// the choices moves the default into the new range.
func (s *mapInspectionEdits) editSetting(d *mod.SettingData, i int, field, value string, gone error) error {
	x := s.inspector
	st := &d.Settings[i]
	if field == "remove" {
		d.Settings = slices.Delete(d.Settings, i, i+1)
		return nil
	}
	if lang, ok := strings.CutPrefix(field, "label."); ok {
		if !slices.Contains(textLanguages(), lang) {
			return gone
		}
		labels := map[string]string{}
		for k, v := range st.Labels {
			labels[k] = v
		}
		labels[lang] = value
		st.Labels = labels
		return nil
	}
	switch field {
	case "type":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 || n > int(mod.KindChoice) || mod.Kind(n) == st.Kind {
			if err == nil && mod.Kind(n) == st.Kind {
				return nil
			}
			return gone
		}
		st.Kind, st.Min, st.Max, st.Choices = mod.Kind(n), 0, 0, nil
		switch st.Kind {
		case mod.KindInt:
			st.Max, st.Default = 10, mod.Value{Kind: mod.KindInt}
		case mod.KindBool:
			st.Default = mod.Value{Kind: mod.KindBool}
		case mod.KindChoice:
			st.Choices = []string{"first", "second"}
			st.Default = mod.Value{Kind: mod.KindChoice, Str: "first"}
		}
	case "default":
		switch st.Kind {
		case mod.KindInt:
			n, err := x.settingNumberOf(value)
			if err != nil {
				return err
			}
			if n < st.Min || n > st.Max {
				return fmt.Errorf(x.word("editor.settings.err.range"), st.Min, st.Max)
			}
			st.Default.Int = n
		case mod.KindBool:
			if value != "0" && value != "1" {
				return gone
			}
			st.Default.Bool = value == "1"
		case mod.KindChoice:
			n, err := strconv.Atoi(value)
			if err != nil || n < 0 || n >= len(st.Choices) {
				return gone
			}
			st.Default.Str = st.Choices[n]
		}
	case "min", "max":
		if st.Kind != mod.KindInt {
			return gone
		}
		n, err := x.settingNumberOf(value)
		if err != nil {
			return err
		}
		lo, hi := st.Min, st.Max
		if field == "min" {
			lo = n
		} else {
			hi = n
		}
		if lo > hi {
			return fmt.Errorf(x.word("editor.settings.err.bounds"), lo, hi)
		}
		st.Min, st.Max = lo, hi
		st.Default.Int = max(lo, min(hi, st.Default.Int))
	case "choices":
		if st.Kind != mod.KindChoice {
			return gone
		}
		var list []string
		for _, c := range strings.Split(value, ",") {
			c = strings.TrimSpace(c)
			if c == "" || slices.Contains(list, c) {
				return errors.New(x.word("editor.settings.err.choices"))
			}
			list = append(list, c)
		}
		st.Choices = list
		if !slices.Contains(list, st.Default.Str) {
			st.Default.Str = list[0]
		}
	default:
		return gone
	}
	return nil
}

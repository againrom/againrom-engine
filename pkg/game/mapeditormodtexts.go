package game

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/locale"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// The texts page's keys: a key's name is "name:<key>", its text in a
// language "<lang>:<key>" and its removal from one "drop.<lang>:<key>";
// "new" adds the typed key.
const (
	textsNameKey = "name:"
	textsDropKey = "drop."
	textsNewKey  = "new"
)

// The mission row fields that pick a text key.
const (
	missionTitleKey = "missiontitle"
	missionBriefKey = "missionbriefing"
)

// textLanguages is the codes of the languages a mod's strings files hold,
// English first.
func textLanguages() []string {
	var out []string
	for _, l := range locale.All() {
		out = append(out, l.Code)
	}
	slices.SortStableFunc(out, func(a, b string) int {
		switch {
		case a == locale.Fallback:
			return -1
		case b == locale.Fallback:
			return 1
		}
		return 0
	})
	return out
}

// languageLabel is a language code as the texts page shows it.
func languageLabel(code string) string { return strings.ToUpper(code) }

// strings is the project's text of each language, as its strings files
// read; a language without a file has none.
func (d *editorModDocument) strings() (map[string]mod.Strings, error) {
	out := map[string]mod.Strings{}
	for _, l := range textLanguages() {
		rel := mod.StringsFile(l)
		s, err := mod.ParseStrings(rel, d.Files[rel])
		if err != nil {
			return nil, err
		}
		out[l] = s
	}
	return out, nil
}

// textLookup is the one text lookup over the project's strings files for a
// game in lang: its own string, else the English one.
func (d *editorModDocument) textLookup(lang string) (func(key string) (string, bool), error) {
	return mod.DraftText(lang, func(rel string) ([]byte, bool) { b, ok := d.Files[rel]; return b, ok })
}

// textKeys is every key the languages of sets hold, sorted.
func textKeys(sets map[string]mod.Strings) []string {
	held := map[string]bool{}
	for _, s := range sets {
		for k := range s {
			held[k] = true
		}
	}
	return slices.Sorted(maps.Keys(held))
}

// textCitations is what cites each text key: the project's mission rows by
// row, which a rename carries along, and the folder's files of every other
// kind whose records cite text, by file, which the editor does not write.
func (s *mapInspectionEdits) textCitations(d *editorModDocument) (rows, files map[string][]string) {
	m := s.inspector.mod
	rows, files = map[string][]string{}, map[string][]string{}
	for _, r := range d.Missions {
		for _, k := range []string{r.TitleKey, r.BriefKey} {
			if k != "" && !slices.Contains(rows[k], r.RowID()) {
				rows[k] = append(rows[k], r.RowID())
			}
		}
	}
	for _, k := range mod.DataKinds() {
		for _, rel := range k.Files {
			if rel == mod.MissionsFile {
				continue
			}
			src, err := modDataFile(m.dir, rel)
			if err != nil || src == nil {
				continue
			}
			keys, _ := k.Cites(mod.DataReader{Mod: m.id, Dir: m.dir, Lang: s.inspector.pageLanguage()}, rel, src)
			for _, key := range keys {
				files[key] = append(files[key], rel)
			}
		}
	}
	return rows, files
}

// textsFields is W's texts page: a count of the keys and of those missing
// in each language, then each key with its text in every language, what
// cites it, and the row that adds a key; none with no mod folder open.
func (s *mapInspectionEdits) textsFields() []ui.InspectionField {
	x := s.inspector
	if x.mod == nil || x.mod.packed || s.prepareWorkspace() != nil {
		return nil
	}
	line := x.pageLine
	d := s.workspace.Project
	if d == nil {
		d = x.mod.document()
	}
	sets, err := d.strings()
	if err != nil {
		return []ui.InspectionField{line("error", "", err.Error())}
	}
	langs, keys := textLanguages(), textKeys(sets)
	rows, files := s.textCitations(d)
	missing := map[string]int{}
	out := []ui.InspectionField{{}}
	for _, k := range keys {
		var lacks []string
		start := len(out)
		out = append(out, workspaceText(textsNameKey+k, x.word("editor.texts.name"), k))
		for _, l := range langs {
			v, held := sets[l][k]
			f := workspaceText(l+":"+k, fmt.Sprintf(x.word("editor.texts.value"), languageLabel(l)), v)
			if !held {
				lacks = append(lacks, languageLabel(l))
				missing[l]++
				f.Meaning = x.word("editor.texts.missing")
			}
			out = append(out, f)
			if held {
				out = append(out, ui.InspectionField{Key: textsDropKey + l + ":" + k, Label: fmt.Sprintf(x.word("editor.texts.drop"), languageLabel(l)), Action: true, Column: 1})
			}
		}
		if by := append(slices.Clone(rows[k]), files[k]...); len(by) > 0 {
			out = append(out, line("used:"+k, "", fmt.Sprintf(x.word("editor.texts.used"), strings.Join(by, ", "))))
		}
		head := k
		if len(lacks) > 0 {
			head = fmt.Sprintf(x.word("editor.texts.lacks"), k, strings.Join(lacks, ", "))
		}
		for i := start; i < len(out); i++ {
			out[i].Group = head
		}
	}
	var counts []string
	for _, l := range langs {
		counts = append(counts, fmt.Sprintf("%s %d", languageLabel(l), missing[l]))
	}
	out[0] = line("summary", "", fmt.Sprintf(x.word("editor.texts.summary"), len(keys), strings.Join(counts, ", ")))
	f := workspaceText(textsNewKey, x.word("editor.texts.new"), "")
	f.Meaning = x.word("editor.texts.new.hint")
	return append(out, f)
}

// setTextsField makes one edit of the texts page a draft step; a key name
// the loader refuses is refused in its words, and a key a record cites is
// renamed through the mission rows or refused, naming what cites it.
func (s *mapInspectionEdits) setTextsField(key, value string) (*ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	x := s.inspector
	if err := x.modWritable(); err != nil {
		return nil, err
	}
	d := s.workspace.Project
	if d == nil {
		d = x.mod.document()
	}
	sets, err := d.strings()
	if err != nil {
		return nil, err
	}
	rows, files := s.textCitations(d)
	held := func(k string) bool { return slices.Contains(textKeys(sets), k) }
	name := func(k string) (string, error) {
		k = strings.TrimSpace(k)
		if err := mod.CheckTextKey(k); err != nil {
			return "", fmt.Errorf(x.word("editor.texts.err.key"), k, err)
		}
		if held(k) {
			return "", fmt.Errorf(x.word("editor.texts.err.taken"), k)
		}
		return k, nil
	}
	gone := fmt.Errorf(x.word("editor.texts.err.gone"), key)
	switch {
	case key == textsNewKey:
		k, err := name(value)
		if err != nil {
			return nil, err
		}
		return s.editTexts(func(sets map[string]mod.Strings, _ *editorModDocument) error {
			sets[locale.Fallback][k] = ""
			return nil
		})
	case strings.HasPrefix(key, textsNameKey):
		old := strings.TrimPrefix(key, textsNameKey)
		if !held(old) {
			return nil, gone
		}
		if strings.TrimSpace(value) == old {
			return nil, nil
		}
		k, err := name(value)
		if err != nil {
			return nil, err
		}
		if len(files[old]) > 0 {
			return nil, fmt.Errorf(x.word("editor.texts.err.cited"), old, strings.Join(files[old], ", "))
		}
		return s.editTexts(func(sets map[string]mod.Strings, d *editorModDocument) error {
			for _, set := range sets {
				if v, ok := set[old]; ok {
					delete(set, old)
					set[k] = v
				}
			}
			for i := range d.Missions {
				r := &d.Missions[i]
				if r.TitleKey == old {
					r.TitleKey = k
				}
				if r.BriefKey == old {
					r.BriefKey = k
				}
			}
			return nil
		})
	case strings.HasPrefix(key, textsDropKey):
		l, k, _ := strings.Cut(strings.TrimPrefix(key, textsDropKey), ":")
		if _, ok := sets[l][k]; !ok {
			return nil, gone
		}
		if by := append(slices.Clone(rows[k]), files[k]...); len(by) > 0 {
			if _, en := sets[locale.Fallback][k]; l == locale.Fallback || !en {
				return nil, fmt.Errorf(x.word("editor.texts.err.used"), k, strings.Join(by, ", "))
			}
		}
		return s.editTexts(func(sets map[string]mod.Strings, _ *editorModDocument) error {
			delete(sets[l], k)
			return nil
		})
	}
	l, k, ok := strings.Cut(key, ":")
	if !ok || !slices.Contains(textLanguages(), l) || !held(k) {
		return nil, gone
	}
	return s.editTexts(func(sets map[string]mod.Strings, _ *editorModDocument) error {
		sets[l][k] = value
		return nil
	})
}

// editTexts is one draft step of the strings files: edit changes the text
// of each language and the project's rows, and the kind's writer writes
// the file of each language whose pairs the edit changed over the draft's
// bytes. A changed file with no pair and no other text leaves the mod; a
// language the edit left alone keeps its file as it was.
func (s *mapInspectionEdits) editTexts(edit func(map[string]mod.Strings, *editorModDocument) error) (*ui.InspectionDocument, error) {
	x := s.inspector
	return s.editKind(func(k kindFiles) error {
		w := k.w
		sets, err := w.Project.strings()
		if err != nil {
			return err
		}
		was := map[string]mod.Strings{}
		for l, p := range sets {
			was[l] = maps.Clone(p)
		}
		if err := edit(sets, w.Project); err != nil {
			return err
		}
		for _, l := range textLanguages() {
			if maps.Equal(was[l], sets[l]) {
				continue
			}
			rel := mod.StringsFile(l)
			kind, _ := mod.WrittenKind(rel)
			out, err := kind.Write(rel, w.Project.Files[rel], mod.Data{Strings: sets})
			if err != nil {
				return err
			}
			if len(sets[l]) == 0 && strings.TrimSpace(string(out)) == "" {
				err = k.drop(rel)
			} else {
				err = k.put(rel, out)
			}
			if err != nil {
				return err
			}
		}
		return w.Project.citedTexts(x.pageLanguage())
	})
}

// citedTexts makes each mission row's title and briefing the text of the
// keys they cite in the project's strings files, for a game in lang.
func (d *editorModDocument) citedTexts(lang string) error {
	look, err := d.textLookup(lang)
	if err != nil {
		return err
	}
	for i := range d.Missions {
		r := &d.Missions[i]
		if r.TitleKey != "" {
			r.Title, _ = look(r.TitleKey)
		}
		if r.BriefKey != "" {
			r.Brief, _ = look(r.BriefKey)
		}
	}
	return nil
}

// missionTextField is the selected row's pick of the key its title or
// briefing cites: none, or a key the project's strings files hold, each
// shown with its text in the page's language.
func (s *mapInspectionEdits) missionTextField(key string) (ui.InspectionField, bool) {
	x, d := s.inspector, s.workspace.Project
	if x.mod == nil || d == nil || d.Selected < 0 || d.Selected >= len(d.Missions) || d.Missions[d.Selected].Off {
		return ui.InspectionField{}, false
	}
	sets, err := d.strings()
	look, lerr := d.textLookup(x.pageLanguage())
	if err != nil || lerr != nil {
		return ui.InspectionField{}, false
	}
	cited := d.Missions[d.Selected].TitleKey
	if key == missionBriefKey {
		cited = d.Missions[d.Selected].BriefKey
	}
	f := ui.InspectionField{Key: key, Label: x.modFieldLabel(key), Column: 1, Choices: []ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.texts.none"))}}}
	for i, k := range textKeys(sets) {
		v, _ := look(k)
		f.Choices = append(f.Choices, ui.InspectionChoice{Value: int64(i + 1), Label: x.inspectionUTF8(k + ": " + v)})
		if k == cited {
			f.Value = int64(i + 1)
		}
	}
	return f, true
}

// pickMissionText makes the selected row's title or briefing cite the key
// of choice n of its field, 0 for none; a key English lacks is refused,
// since the loader reads every language's text from English at last.
func (s *mapInspectionEdits) pickMissionText(w *editorWorkspaceState, key, value string) error {
	x, d := s.inspector, w.Project
	if d == nil || d.Selected < 0 || d.Selected >= len(d.Missions) {
		return errors.New(x.word("editor.mod.err.nomission"))
	}
	sets, err := d.strings()
	if err != nil {
		return err
	}
	keys := textKeys(sets)
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n < 0 || n > len(keys) {
		return fmt.Errorf(x.word("editor.texts.err.gone"), value)
	}
	k := ""
	if n > 0 {
		k = keys[n-1]
		if _, ok := sets[locale.Fallback][k]; !ok {
			return fmt.Errorf(x.word("editor.texts.err.noen"), k)
		}
	}
	r := &d.Missions[d.Selected]
	if key == missionTitleKey {
		r.TitleKey, r.Title = k, ""
	} else {
		r.BriefKey, r.Brief = k, ""
	}
	return d.citedTexts(x.pageLanguage())
}

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

// The texts page's fields: a text in a language is "<lang>:<key>" and its
// removal from one "drop.<lang>:<key>"; "new" makes a text for the place
// of the choice it picks, its key made by makeTextKey.
const (
	textsDropKey = "drop."
	textsNewKey  = "new"
)

// missionTextNew is the value of a mission row's title or briefing choice
// that makes a new text for it.
const missionTextNew = -1

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

// textFiles are the strings files of the text languages; a language
// without its file reads with no text.
func textFiles() []string {
	var out []string
	for _, l := range textLanguages() {
		out = append(out, mod.StringsFile(l))
	}
	return out
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

// textPlaces is where each text key of the project d shows, in the page's
// words: a mission row's name or briefing, an item's name, or a file of
// another kind that cites it, which the editor does not write.
func (s *mapInspectionEdits) textPlaces(d *editorModDocument, skipFiles ...string) map[string][]string {
	x, m := s.inspector, s.inspector.mod
	out := map[string][]string{}
	add := func(k, format string, a ...any) {
		if place := fmt.Sprintf(format, a...); k != "" && !slices.Contains(out[k], place) {
			out[k] = append(out[k], place)
		}
	}
	for _, r := range d.Missions {
		add(r.TitleKey, x.word("editor.texts.at.title"), missionPlaceName(r))
		add(r.BriefKey, x.word("editor.texts.at.brief"), missionPlaceName(r))
	}
	if read, err := s.heldRows(s.workspace.Project, s.itemNames, mod.ItemsFile); err == nil {
		items := read.Items
		for _, r := range items.Rows {
			add(r.NameKey, x.word("editor.texts.at.item"), itemPlaceName(r.Name, r.Key))
		}
		for _, r := range items.Weapons {
			add(r.NameKey, x.word("editor.texts.at.item"), itemPlaceName(r.Name, r.Key))
		}
		for _, r := range items.Materials {
			add(r.NameKey, x.word("editor.texts.at.material"), r.Item, r.Material)
		}
	}
	for _, k := range mod.DataKinds() {
		for _, rel := range k.Files {
			if rel == mod.MissionsFile || rel == mod.ItemsFile || slices.Contains(skipFiles, rel) {
				continue
			}
			src, held := d.Files[rel]
			var err error
			if !held {
				src, err = modDataFile(m.dir, rel)
			}
			if err != nil || src == nil {
				continue
			}
			keys, _ := k.Cites(mod.DataReader{Mod: m.id, Dir: m.dir, Lang: x.pageLanguage()}, rel, src)
			for _, key := range keys {
				add(key, x.word("editor.texts.at.file"), rel)
			}
		}
	}
	return out
}

// missionPlaceName names mission row r where a text of it shows: its title
// once it has one, else its row.
func missionPlaceName(r mod.Mission) string {
	if r.Title != "" {
		return r.Title
	}
	return r.RowID()
}

// itemPlaceName names an added item where its name shows: the name it
// reads as, else its row.
func itemPlaceName(name, key string) string {
	if name != "" && name != "?" {
		return name
	}
	return key
}

// textPlace is one place a new text can be made for: its words, the base
// of its key and what cites the new key there.
type textPlace struct {
	label, base string
	cite        func(d *editorModDocument, key string)
}

// newTextPlaces are the places the texts page makes a new text for: the
// name and briefing of each mission row that shows none, then a text to
// show later.
func (s *mapInspectionEdits) newTextPlaces(d *editorModDocument) []textPlace {
	x := s.inspector
	var out []textPlace
	for i, r := range d.Missions {
		if r.Off {
			continue
		}
		for _, title := range []bool{true, false} {
			if (title && r.TitleKey != "") || (!title && r.BriefKey != "") {
				continue
			}
			word := x.word("editor.texts.at.brief")
			if title {
				word = x.word("editor.texts.at.title")
			}
			out = append(out, textPlace{label: fmt.Sprintf(word, missionPlaceName(r)), base: missionTextBase(r, title),
				cite: func(d *editorModDocument, key string) { citeMissionText(&d.Missions[i], title, key) }})
		}
	}
	return append(out, textPlace{label: x.word("editor.texts.at.later"), base: "text", cite: func(*editorModDocument, string) {}})
}

// missionTextBase is the base of the key of mission row r's name or
// briefing.
func missionTextBase(r mod.Mission, title bool) string {
	if title {
		return "mission." + r.RowID() + ".name"
	}
	return "mission." + r.RowID() + ".briefing"
}

// citeMissionText makes row r's name or briefing cite key.
func citeMissionText(r *mod.Mission, title bool, key string) {
	if title {
		r.TitleKey, r.Title = key, ""
	} else {
		r.BriefKey, r.Brief = key, ""
	}
}

// makeTextKey is the key the editor adds a text under: the record key rule
// over base, never a key sets hold.
func makeTextKey(sets map[string]mod.Strings, base string) string {
	held := textKeys(sets)
	return mod.RecordKey(base, func(k string) bool { return slices.Contains(held, k) })
}

// textsFields is W's texts page: what the page is for, a count of the
// texts and of those missing in each language, the choice that makes a
// new text for a place, then each text under where it shows, in every
// language; none with no mod folder open.
func (s *mapInspectionEdits) textsFields() []ui.InspectionField {
	x := s.inspector
	if !s.modPageHolds() {
		return nil
	}
	line := x.pageLine
	d := s.workspace.Project
	if d == nil {
		d = x.mod.document()
	}
	read, err := s.heldRows(d, nil, textFiles()...)
	if err != nil {
		return []ui.InspectionField{line("error", "", err.Error())}
	}
	sets := read.Strings
	langs, keys := textLanguages(), textKeys(sets)
	places := s.textPlaces(d)
	heads := map[string]string{}
	unshown := 0
	for _, k := range keys {
		if len(places[k]) == 0 {
			unshown++
			heads[k] = fmt.Sprintf(x.word("editor.texts.at.nowhere"), unshown)
		} else {
			heads[k] = strings.Join(places[k], "; ")
		}
	}
	slices.SortStableFunc(keys, func(a, b string) int { return strings.Compare(heads[a], heads[b]) })
	missing := map[string]int{}
	out := []ui.InspectionField{line("about", "", x.word("editor.texts.about")), {}}
	newText := ui.InspectionField{Key: textsNewKey, Label: x.word("editor.texts.new"), Meaning: x.word("editor.texts.new.hint"),
		Choices: []ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.texts.new.pick"))}}}
	for i, p := range s.newTextPlaces(d) {
		newText.Choices = append(newText.Choices, ui.InspectionChoice{Value: int64(i + 1), Label: x.inspectionUTF8(p.label)})
	}
	out = append(out, newText)
	for _, k := range keys {
		var lacks []string
		start := len(out)
		for _, l := range langs {
			v, held := sets[l][k]
			f := workspaceText(l+":"+k, fmt.Sprintf(x.word("editor.texts.value"), languageLabel(l)), v)
			f.Meaning = x.word("editor.texts.value.help")
			if !held {
				lacks = append(lacks, languageLabel(l))
				missing[l]++
				f.Meaning = x.word("editor.texts.missing")
			}
			out = append(out, f)
			if held {
				out = append(out, ui.InspectionField{Key: textsDropKey + l + ":" + k, Label: fmt.Sprintf(x.word("editor.texts.drop"), languageLabel(l)), Meaning: x.word("editor.texts.drop.help"), Action: true, Column: 1})
			}
		}
		head := heads[k]
		if len(lacks) > 0 {
			head = fmt.Sprintf(x.word("editor.texts.lacks"), head, strings.Join(lacks, ", "))
		}
		for i := start; i < len(out); i++ {
			out[i].Group = head
		}
	}
	var counts []string
	for _, l := range langs {
		counts = append(counts, fmt.Sprintf("%s %d", languageLabel(l), missing[l]))
	}
	out[1] = line("summary", "", fmt.Sprintf(x.word("editor.texts.summary"), len(keys), strings.Join(counts, ", ")))
	return out
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
	read, err := s.heldRows(d, nil, textFiles()...)
	if err != nil {
		return nil, err
	}
	sets := read.Strings
	held := func(k string) bool { return slices.Contains(textKeys(sets), k) }
	gone := fmt.Errorf(x.word("editor.texts.err.gone"), key)
	switch {
	case key == textsNewKey:
		places := s.newTextPlaces(d)
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || n < 0 || n > len(places) {
			return nil, gone
		}
		if n == 0 {
			return nil, nil
		}
		p := places[n-1]
		return s.editTexts(func(sets map[string]mod.Strings, d *editorModDocument) error {
			k := makeTextKey(sets, p.base)
			sets[locale.Fallback][k] = ""
			p.cite(d, k)
			return nil
		})
	case strings.HasPrefix(key, textsDropKey):
		l, k, _ := strings.Cut(strings.TrimPrefix(key, textsDropKey), ":")
		if _, ok := sets[l][k]; !ok {
			return nil, gone
		}
		if by := s.textPlaces(d)[k]; len(by) > 0 {
			if _, en := sets[locale.Fallback][k]; l == locale.Fallback || !en {
				return nil, fmt.Errorf(x.word("editor.texts.err.used"), strings.Join(by, "; "))
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

// editTexts is one draft step of the strings files through editRows: edit
// changes the text of each language and the project's rows; a language the
// edit left alone keeps its file as it was.
func (s *mapInspectionEdits) editTexts(edit func(map[string]mod.Strings, *editorModDocument) error) (*ui.InspectionDocument, error) {
	x := s.inspector
	return s.editKind(func(k kindFiles) error {
		if err := k.editRows(nil, textFiles(), func(d *mod.Data) error { return edit(d.Strings, k.w.Project) }); err != nil {
			return err
		}
		return k.w.Project.citedTexts(x.pageLanguage())
	})
}

func newModText(sets map[string]mod.Strings, kind string, values map[string]string) string {
	key := makeTextKey(sets, kind)
	for _, l := range textLanguages() {
		if sets[l] == nil {
			sets[l] = mod.Strings{}
		}
		sets[l][key] = values[l]
	}
	return key
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

// missionTextField is the selected row's pick of the text its title or
// briefing shows: none, a text the project's strings files hold, shown by
// its text in the page's language, or a new text.
func (s *mapInspectionEdits) missionTextField(key string) (ui.InspectionField, bool) {
	x, d := s.inspector, s.workspace.Project
	if x.mod == nil || d == nil || d.Selected < 0 || d.Selected >= len(d.Missions) || d.Missions[d.Selected].Off {
		return ui.InspectionField{}, false
	}
	read, err := s.heldRows(d, nil, textFiles()...)
	look, lerr := d.textLookup(x.pageLanguage())
	if err != nil || lerr != nil {
		return ui.InspectionField{}, false
	}
	sets := read.Strings
	cited := d.Missions[d.Selected].TitleKey
	if key == missionBriefKey {
		cited = d.Missions[d.Selected].BriefKey
	}
	f := ui.InspectionField{Key: key, Label: x.modFieldLabel(key), Meaning: x.word("editor.mod." + key + ".help"), Column: 1,
		Choices: []ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.texts.none"))}}}
	for i, k := range textKeys(sets) {
		v, _ := look(k)
		if v == "" {
			v = x.word("editor.texts.empty")
		}
		f.Choices = append(f.Choices, ui.InspectionChoice{Value: int64(i + 1), Label: x.inspectionUTF8(v)})
		if k == cited {
			f.Value = int64(i + 1)
		}
	}
	f.Choices = append(f.Choices, ui.InspectionChoice{Value: missionTextNew, Label: x.inspectionUTF8(x.word("editor.texts.new.choice"))})
	return f, true
}

// pickMissionText makes the selected row's title or briefing cite the key
// of choice n of its field, 0 for none, missionTextNew for a new empty
// English text whose key makeTextKey makes; a key English lacks is
// refused, since the loader reads every language's text from English at
// last.
func (s *mapInspectionEdits) pickMissionText(w *editorWorkspaceState, key, value string) error {
	x, d := s.inspector, w.Project
	if d == nil || d.Selected < 0 || d.Selected >= len(d.Missions) {
		return errors.New(x.word("editor.mod.err.nomission"))
	}
	read, err := s.heldRows(d, nil, textFiles()...)
	if err != nil {
		return err
	}
	sets := read.Strings
	keys := textKeys(sets)
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n < missionTextNew || n > len(keys) {
		return fmt.Errorf(x.word("editor.texts.err.gone"), value)
	}
	r, title := &d.Missions[d.Selected], key == missionTitleKey
	k := ""
	switch {
	case n == missionTextNew:
		err := kindFiles{s: s, w: w}.editRows(nil, textFiles(), func(t *mod.Data) error {
			k = makeTextKey(t.Strings, missionTextBase(*r, title))
			t.Strings[locale.Fallback][k] = ""
			return nil
		})
		if err != nil {
			return err
		}
	case n > 0:
		k = keys[n-1]
		if _, ok := sets[locale.Fallback][k]; !ok {
			return errors.New(x.word("editor.texts.err.noen"))
		}
	}
	citeMissionText(r, title, k)
	return d.citedTexts(x.pageLanguage())
}

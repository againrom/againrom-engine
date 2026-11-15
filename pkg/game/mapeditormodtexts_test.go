package game

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
)

// textsField is the texts page's field key; false when the page has none.
func textsField(s *mapInspectionEdits, key string) (ui.InspectionField, bool) {
	for _, f := range s.WorkspaceFields(ui.ModTextsScope) {
		if f.Key == key {
			return f, true
		}
	}
	return ui.InspectionField{}, false
}

// setTexts makes one edit on the texts page, failing the test on a refusal.
func setTexts(t *testing.T, s *mapInspectionEdits, key, value string) {
	t.Helper()
	if _, err := s.SetWorkspaceField(ui.ModTextsScope, key, value); err != nil {
		t.Fatalf("%s = %q: %v", key, value, err)
	}
}

// refusedTexts fails the test unless the texts page refuses the edit with
// words holding want and leaves the draft as it was.
func refusedTexts(t *testing.T, s *mapInspectionEdits, key, value, want string) {
	t.Helper()
	before := cloneWorkspace(s.workspace)
	_, err := s.SetWorkspaceField(ui.ModTextsScope, key, value)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("%s = %q: %v, want a refusal naming %q", key, value, err, want)
	}
	if !reflect.DeepEqual(s.workspace, before) {
		t.Fatalf("%s = %q: the refusal changed the draft", key, value)
	}
}

// textsSummary is the texts page's count line.
func textsSummary(t *testing.T, s *mapInspectionEdits) string {
	t.Helper()
	f, ok := textsField(s, "summary")
	if !ok {
		t.Fatal("the texts page has no count line")
	}
	return f.TextValue
}

// pickTitle makes the selected mission row's title or briefing cite key
// through W's field, the choice that names it.
func pickTitle(t *testing.T, s *mapInspectionEdits, field, key string) {
	t.Helper()
	for _, f := range s.WorkspaceFields("package") {
		if f.Key != field {
			continue
		}
		for _, c := range f.Choices {
			if strings.HasPrefix(c.Label, key+": ") {
				if _, err := s.SetWorkspaceField("package", field, strconv.FormatInt(c.Value, 10)); err != nil {
					t.Fatalf("%s = %s: %v", field, key, err)
				}
				return
			}
		}
		t.Fatalf("%s offers no key %q: %+v", field, key, f.Choices)
	}
	t.Fatalf("W has no field %s", field)
}

// selectedRow is the project's selected mission row.
func selectedRow(s *mapInspectionEdits) mod.Mission {
	d := s.workspace.Project
	return d.Missions[d.Selected]
}

// The page lists each key with its text in English and Russian, marks and
// counts a key one language lacks, and refuses a key name the loader
// refuses and a taken one in their words, with no draft.
func TestTextsPageMarksAKeyMissingInOneLanguageAndRefusesABadKey(t *testing.T) {
	s, _ := terrainPage(t)
	if got := textsSummary(t, s); !strings.HasPrefix(got, "0 keys") {
		t.Fatalf("a mod with no strings: %q", got)
	}
	setTexts(t, s, "new", "pass.title")
	setTexts(t, s, "en:pass.title", "The Pass")
	if got := textsSummary(t, s); got != "1 keys; missing: EN 0, RU 1" {
		t.Fatalf("the count line %q", got)
	}
	f, ok := textsField(s, "ru:pass.title")
	if !ok || f.Meaning == "" || !strings.Contains(f.Group, "missing in RU") {
		t.Fatalf("the Russian field of a key Russian lacks: %+v", f)
	}
	if en, _ := textsField(s, "en:pass.title"); en.TextValue != "The Pass" || en.Meaning != "" {
		t.Fatalf("the English field %+v", en)
	}
	if _, ok := textsField(s, "drop.ru:pass.title"); ok {
		t.Fatal("the page offers to remove a key from a language that lacks it")
	}
	setTexts(t, s, "ru:pass.title", "Перевал")
	if got := textsSummary(t, s); got != "1 keys; missing: EN 0, RU 0" {
		t.Fatalf("the count line %q", got)
	}
	if f, _ := textsField(s, "ru:pass.title"); f.TextValue != "Перевал" || strings.Contains(f.Group, "missing") {
		t.Fatalf("the Russian field %+v", f)
	}
	refusedTexts(t, s, "new", "  ", mod.CheckTextKey("").Error())
	refusedTexts(t, s, "new", "pass.title", "already has text key")
	refusedTexts(t, s, "name:pass.title", "", mod.CheckTextKey("").Error())
	refusedTexts(t, s, "en:no.such", "x", "no field")
	b := s.workspace.Project.Files[mod.StringsFile("ru")]
	if back, err := mod.ParseStrings(mod.StringsFile("ru"), b); err != nil || back["pass.title"] != "Перевал" {
		t.Fatalf("the Russian draft %q: %v", b, err)
	}
}

// Adding, editing and renaming a key are draft steps Undo and Redo walk; a
// rename carries the mission row that cites the key, and the row's title
// field shows the key's text.
func TestTextsPageUndoesRedoesAndCarriesARenameThroughTheMissionRow(t *testing.T) {
	s, _ := terrainPage(t)
	setTexts(t, s, "new", "pass.title")
	setTexts(t, s, "en:pass.title", "The Pass")
	pickTitle(t, s, missionTitleKey, "pass.title")
	if r := selectedRow(s); r.TitleKey != "pass.title" || r.Title != "The Pass" {
		t.Fatalf("the row after the pick: %q %q", r.TitleKey, r.Title)
	}
	if f, ok := textsField(s, "used:pass.title"); !ok || !strings.Contains(f.TextValue, selectedRow(s).RowID()) {
		t.Fatalf("the used line %+v", f)
	}
	setTexts(t, s, "name:pass.title", "pass.name")
	if r := selectedRow(s); r.TitleKey != "pass.name" || r.Title != "The Pass" {
		t.Fatalf("the row after the rename: %q %q", r.TitleKey, r.Title)
	}
	if _, ok := textsField(s, "en:pass.title"); ok {
		t.Fatal("the old key stays after the rename")
	}
	setTexts(t, s, "en:pass.name", "High Pass")
	if r := selectedRow(s); r.Title != "High Pass" {
		t.Fatalf("the row's title after an edit of its key: %q", r.Title)
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if f, _ := textsField(s, "en:pass.name"); f.TextValue != "The Pass" || selectedRow(s).Title != "The Pass" {
		t.Fatalf("after undoing the edit: %+v, row %q", f, selectedRow(s).Title)
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if _, ok := textsField(s, "en:pass.title"); !ok || selectedRow(s).TitleKey != "pass.title" {
		t.Fatalf("after undoing the rename: row %q", selectedRow(s).TitleKey)
	}
	for range 2 {
		if _, err := s.Redo(); err != nil {
			t.Fatal(err)
		}
	}
	if f, _ := textsField(s, "en:pass.name"); f.TextValue != "High Pass" || selectedRow(s).TitleKey != "pass.name" {
		t.Fatalf("after redo: %+v, row %q", f, selectedRow(s).TitleKey)
	}
}

// A key a mission row cites cannot leave English, and leaves Russian while
// English holds it; a key a file of another kind cites is not renamed, and
// the refusal names the file. A title cannot pick a key English lacks.
func TestTextsPageRefusesWhatWouldLeaveACitationUnread(t *testing.T) {
	s, dir := terrainPage(t)
	writeFileAll(t, filepath.Join(dir, filepath.FromSlash(mod.ItemsFile)), []byte("[[weapon]]\nkey        = \"star\"\nname       = \"coat.name\"\nbase       = \"Dagger\"\n"))
	setTexts(t, s, "new", "coat.name")
	setTexts(t, s, "new", "pass.title")
	setTexts(t, s, "ru:pass.title", "Перевал")
	pickTitle(t, s, missionTitleKey, "pass.title")
	refusedTexts(t, s, "drop.en:pass.title", "", selectedRow(s).RowID())
	refusedTexts(t, s, "name:coat.name", "coat.label", mod.ItemsFile)
	refusedTexts(t, s, "drop.en:coat.name", "", mod.ItemsFile)
	if f, ok := textsField(s, "used:coat.name"); !ok || !strings.Contains(f.TextValue, mod.ItemsFile) {
		t.Fatalf("the used line of a key the items cite: %+v", f)
	}
	setTexts(t, s, "drop.ru:pass.title", "")
	if f, _ := textsField(s, "ru:pass.title"); f.Meaning == "" {
		t.Fatalf("the Russian field after its text left: %+v", f)
	}
	setTexts(t, s, "new", "only.ru")
	setTexts(t, s, "ru:only.ru", "Только")
	setTexts(t, s, "drop.en:only.ru", "")
	before, tried := cloneWorkspace(s.workspace), false
	for _, f := range s.WorkspaceFields("package") {
		if f.Key == missionBriefKey {
			for _, c := range f.Choices {
				if strings.HasPrefix(c.Label, "only.ru: ") {
					tried = true
					if _, err := s.SetWorkspaceField("package", missionBriefKey, strconv.FormatInt(c.Value, 10)); err == nil || !strings.Contains(err.Error(), "no English") {
						t.Fatalf("a briefing picked a key English lacks: %v", err)
					}
				}
			}
		}
	}
	if !tried {
		t.Fatal("the briefing field offers no key English lacks")
	}
	if !reflect.DeepEqual(s.workspace, before) {
		t.Fatal("the refused pick changed the draft")
	}
}

// A mod whose texts and mission title are made on the page alone saves
// through the project writer: both strings files, the row citing the key,
// a hand comment kept, the entry file unchanged. The starter's route reads
// the title in English and Russian, and the published mod passes the
// launch's check on both installs.
func TestTextsPageModSavesPublishesAndShowsTheTitle(t *testing.T) {
	s, dir := terrainPage(t)
	made := tree(t, dir)
	setTexts(t, s, "new", "pass.title")
	setTexts(t, s, "en:pass.title", "The Pass")
	setTexts(t, s, "ru:pass.title", "Перевал")
	setTexts(t, s, "new", "pass.brief")
	setTexts(t, s, "en:pass.brief", "Cross \"the\" pass.")
	pickTitle(t, s, missionTitleKey, "pass.title")
	pickTitle(t, s, missionBriefKey, "pass.brief")
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	saved := tree(t, dir)
	if saved[mod.EntryName] != made[mod.EntryName] {
		t.Fatalf("the save changed the entry file:\n%s", saved[mod.EntryName])
	}
	for _, l := range textLanguages() {
		rel := mod.StringsFile(l)
		if _, ok := saved[rel]; !ok {
			t.Fatalf("the save wrote no %s: %v", rel, keysOf(saved))
		}
	}
	if rows := saved[mod.MissionsFile]; !strings.Contains(rows, "\"pass.title\"") || !strings.Contains(rows, "\"pass.brief\"") {
		t.Fatalf("the saved row does not cite the key:\n%s", rows)
	}
	en := filepath.Join(dir, filepath.FromSlash(mod.StringsFile("en")))
	writeFileAll(t, en, append([]byte("# made on the page\n"), saved[mod.StringsFile("en")]...))
	s = openInW(t, s, dir)
	setTexts(t, s, "en:pass.title", "The High Pass")
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(en); !strings.HasPrefix(string(b), "# made on the page\n\"pass.title\" = ") || !strings.Contains(string(b), "\"pass.brief\" = \"Cross \\\"the\\\" pass.\"\n") || !strings.Contains(string(b), "\"pass.title\" = \"The High Pass\"\n") {
		t.Fatalf("the English file after the second save:\n%s", b)
	}
	for lang, want := range map[string][2]string{"en": {"The High Pass", "Cross \"the\" pass."}, "ru": {"Перевал", "Cross \"the\" pass."}} {
		d, _, err := mod.DescribeMissions(dir, "authored-map", lang)
		if err != nil {
			t.Fatal(err)
		}
		i := slices.IndexFunc(d.Missions, func(m mod.Mission) bool { return m.TitleKey == "pass.title" })
		if i < 0 || d.Missions[i].Title != want[0] || d.Missions[i].Brief != want[1] {
			t.Fatalf("%s: the starter's rows %+v", lang, d.Missions)
		}
	}
	out := t.TempDir()
	if _, err := s.SetWorkspaceField("package", "output", out); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetWorkspaceField("package", "publish", ""); err != nil {
		t.Fatal(err)
	}
	entries, err := mod.Resolve(out, []string{"authored-map"})
	if err != nil || len(entries) != 1 {
		t.Fatal(entries, err)
	}
	for base, want := range map[string]string{"rom1-en": "The High Pass", "rom1-ru": "Перевал"} {
		res, err := modrt.Load(entries, base, nil, modrt.Options{})
		if err != nil {
			t.Fatalf("%s: the check refused the published mod: %v", base, err)
		}
		i := slices.IndexFunc(res.Missions.Missions, func(m mod.Mission) bool { return m.TitleKey == "pass.title" })
		if i < 0 || res.Missions.Missions[i].Title != want {
			t.Fatalf("%s: the published rows %+v", base, res.Missions.Missions)
		}
	}
}

// An English edit leaves a Russian strings file whose pairs it did not
// change as it was, empty or blank-lined.
func TestTextsPageEditKeepsAnUntouchedEmptyLanguageFile(t *testing.T) {
	for _, body := range []string{"", "\r\n\r\n"} {
		s, dir := terrainPage(t)
		ru := filepath.Join(dir, filepath.FromSlash(mod.StringsFile("ru")))
		writeFileAll(t, ru, []byte(body))
		s = openInW(t, s, dir)
		setTexts(t, s, "new", "a")
		setTexts(t, s, "en:a", "one")
		if _, err := s.SaveIntoMod(); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(ru)
		if err != nil || string(b) != body {
			t.Fatalf("the Russian file %q after an English edit: %q, %v", body, b, err)
		}
	}
}

// Adding a key, saving and undoing takes the strings file the save made out
// of the folder at the next save; editing a value, saving and undoing writes
// the old value back; the same after a redo and a second undo.
func TestTextsPageUndoPastASaveRestoresTheStrings(t *testing.T) {
	s, dir := terrainPage(t)
	ru := filepath.Join(dir, filepath.FromSlash(mod.StringsFile("ru")))
	setTexts(t, s, "new", "a")
	setTexts(t, s, "ru:a", "один")
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ru); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ru); !os.IsNotExist(err) {
		t.Fatalf("the Russian file the undone step made stays: %v", err)
	}
	setTexts(t, s, "ru:a", "один")
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	setTexts(t, s, "ru:a", "два")
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	for _, step := range []func() (*ui.InspectionDocument, error){s.Undo, s.Redo, s.Undo} {
		if _, err := step(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(ru); !strings.Contains(string(b), "\"a\" = \"один\"") {
		t.Fatalf("the Russian file after an undo past a save:\n%s", b)
	}
}

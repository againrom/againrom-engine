package game

import (
	"fmt"
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

// heldKeys is every text key of the open project.
func heldKeys(t *testing.T, s *mapInspectionEdits) []string {
	t.Helper()
	d := s.workspace.Project
	if d == nil {
		d = s.inspector.mod.document()
	}
	sets, err := heldTexts(s, d)
	if err != nil {
		t.Fatal(err)
	}
	return textKeys(sets)
}

// madeKey is the one key of after that before lacks.
func madeKey(t *testing.T, before, after []string) string {
	t.Helper()
	var made []string
	for _, k := range after {
		if !slices.Contains(before, k) {
			made = append(made, k)
		}
	}
	if len(made) != 1 {
		t.Fatalf("keys made: %q", made)
	}
	return made[0]
}

// newText makes a text on the texts page for the place whose words hold
// place, and is the key the editor made for it.
func newText(t *testing.T, s *mapInspectionEdits, place string) string {
	t.Helper()
	f, ok := textsField(s, textsNewKey)
	if !ok || f.Text {
		t.Fatalf("the new text row %+v is no choice", f)
	}
	for _, c := range f.Choices {
		if c.Value > 0 && strings.Contains(c.Label, place) {
			before := heldKeys(t, s)
			setTexts(t, s, textsNewKey, strconv.FormatInt(c.Value, 10))
			return madeKey(t, before, heldKeys(t, s))
		}
	}
	t.Fatalf("the new text row offers no place %q: %+v", place, f.Choices)
	return ""
}

// newMissionText makes a new text for the selected mission row's title or
// briefing through its field on the Missions page, and is its key.
func newMissionText(t *testing.T, s *mapInspectionEdits, field string) string {
	t.Helper()
	before := heldKeys(t, s)
	if _, err := s.SetWorkspaceField("package", field, strconv.Itoa(missionTextNew)); err != nil {
		t.Fatalf("%s: new text: %v", field, err)
	}
	return madeKey(t, before, heldKeys(t, s))
}

// pickKey makes the selected mission row's title or briefing cite key
// through W's field, by the choice that stands for it.
func pickKey(s *mapInspectionEdits, field, key string) error {
	sets, err := heldTexts(s, s.workspace.Project)
	if err != nil {
		return err
	}
	n := slices.Index(textKeys(sets), key)
	if n < 0 {
		return fmt.Errorf("no key %q", key)
	}
	_, err = s.SetWorkspaceField("package", field, strconv.Itoa(n+1))
	return err
}

// selectedRow is the project's selected mission row.
func selectedRow(s *mapInspectionEdits) mod.Mission {
	d := s.workspace.Project
	return d.Missions[d.Selected]
}

// noKeyField fails the test when a field of the texts page holds key as its
// value or a label names a key: no key is a field the modder fills.
func noKeyField(t *testing.T, s *mapInspectionEdits, key string) {
	t.Helper()
	for _, f := range s.WorkspaceFields(ui.ModTextsScope) {
		if f.TextValue == key || strings.Contains(strings.ToLower(f.Label), "key") {
			t.Fatalf("a field shows the key %q: %+v", key, f)
		}
	}
}

// The page says first what it is for, lists each text under where it shows
// with its text in English and Russian, marks and counts a text one
// language lacks, and refuses a place it does not offer.
func TestTextsPageMarksATextMissingInOneLanguage(t *testing.T) {
	s, _ := terrainPage(t)
	if f := s.WorkspaceFields(ui.ModTextsScope); len(f) == 0 || f[0].Key != "about" || f[0].TextValue == "" {
		t.Fatalf("the page does not open with what it is for: %+v", f)
	}
	if got := textsSummary(t, s); !strings.HasPrefix(got, "0 texts") {
		t.Fatalf("a mod with no strings: %q", got)
	}
	k := newText(t, s, "A text to show later")
	setTexts(t, s, "en:"+k, "The Pass")
	if got := textsSummary(t, s); got != "1 texts; missing: EN 0, RU 1" {
		t.Fatalf("the count line %q", got)
	}
	f, ok := textsField(s, "ru:"+k)
	if !ok || f.Meaning == "" || !strings.Contains(f.Group, "missing in RU") || !strings.HasPrefix(f.Group, "Not shown anywhere yet") {
		t.Fatalf("the Russian field of a text Russian lacks: %+v", f)
	}
	if en, _ := textsField(s, "en:"+k); en.TextValue != "The Pass" || en.Meaning == "" {
		t.Fatalf("the English field %+v", en)
	}
	if drop, ok := textsField(s, "drop.en:"+k); !ok || drop.Meaning == "" {
		t.Fatalf("the button that removes the English text has no help: %+v", drop)
	}
	if _, ok := textsField(s, "drop.ru:"+k); ok {
		t.Fatal("the page offers to remove a text from a language that lacks it")
	}
	setTexts(t, s, "ru:"+k, "Перевал")
	if got := textsSummary(t, s); got != "1 texts; missing: EN 0, RU 0" {
		t.Fatalf("the count line %q", got)
	}
	if f, _ := textsField(s, "ru:"+k); f.TextValue != "Перевал" || strings.Contains(f.Group, "missing") {
		t.Fatalf("the Russian field %+v", f)
	}
	noKeyField(t, s, k)
	refusedTexts(t, s, textsNewKey, "99", "no field")
	refusedTexts(t, s, "en:no.such", "x", "no field")
	if again := newText(t, s, "A text to show later"); again != k+"_2" {
		t.Fatalf("the second text to show later is %q, want the key maker's %q", again, k+"_2")
	}
	b := s.workspace.Project.Files[mod.StringsFile("ru")]
	if back, err := mod.ParseStrings(mod.StringsFile("ru"), b); err != nil || back[k] != "Перевал" {
		t.Fatalf("the Russian draft %q: %v", b, err)
	}
}

// A mission name made from the Missions page shows on the texts page under
// the mission's name, with no key field; its words show as the row's
// title. Making it, typing it and making the briefing from the texts page
// are draft steps Undo and Redo walk.
func TestTextsPageShowsAMissionNameMadeFromMissions(t *testing.T) {
	s, _ := terrainPage(t)
	k := newMissionText(t, s, missionTitleKey)
	row := selectedRow(s)
	if want := makeTextKey(nil, missionTextBase(row, true)); row.TitleKey != k || k != want {
		t.Fatalf("the row cites %q, the made key is %q, want %q", row.TitleKey, k, want)
	}
	en, ok := textsField(s, "en:"+k)
	if want := "Mission " + row.RowID() + ": name"; !ok || !strings.HasPrefix(en.Group, want) {
		t.Fatalf("the new name's field %+v, want it under %q", en, want)
	}
	noKeyField(t, s, k)
	setTexts(t, s, "en:"+k, "The Pass")
	if r := selectedRow(s); r.Title != "The Pass" {
		t.Fatalf("the row's title after typing it: %q", r.Title)
	}
	if f, _ := textsField(s, "en:"+k); !strings.HasPrefix(f.Group, "Mission The Pass: name") {
		t.Fatalf("the name's place after typing it: %q", f.Group)
	}
	nf, _ := textsField(s, textsNewKey)
	for _, c := range nf.Choices {
		if strings.Contains(c.Label, ": name") {
			t.Fatalf("the new text row still offers the named mission's name: %+v", nf.Choices)
		}
	}
	b := newText(t, s, "Mission The Pass: briefing")
	if r := selectedRow(s); r.BriefKey != b {
		t.Fatalf("the briefing made on the texts page: row cites %q, made %q", r.BriefKey, b)
	}
	for range 3 {
		if _, err := s.Undo(); err != nil {
			t.Fatal(err)
		}
	}
	if r := selectedRow(s); r.TitleKey != "" || r.BriefKey != "" || len(heldKeys(t, s)) != 0 {
		t.Fatalf("after three undos: row %q %q, keys %q", r.TitleKey, r.BriefKey, heldKeys(t, s))
	}
	for range 3 {
		if _, err := s.Redo(); err != nil {
			t.Fatal(err)
		}
	}
	if r := selectedRow(s); r.TitleKey != k || r.BriefKey != b || r.Title != "The Pass" {
		t.Fatalf("after redo: row %q %q %q", r.TitleKey, r.BriefKey, r.Title)
	}
}

// A text a mission row shows cannot leave English, and leaves Russian while
// English holds it. A title cannot pick a text English lacks.
func TestTextsPageRefusesWhatWouldLeaveACitationUnread(t *testing.T) {
	s, _ := terrainPage(t)
	k := newMissionText(t, s, missionTitleKey)
	setTexts(t, s, "ru:"+k, "Перевал")
	refusedTexts(t, s, "drop.en:"+k, "", "Mission "+selectedRow(s).RowID()+": name")
	setTexts(t, s, "drop.ru:"+k, "")
	if f, _ := textsField(s, "ru:"+k); f.Meaning == "" {
		t.Fatalf("the Russian field after its text left: %+v", f)
	}
	only := newText(t, s, "A text to show later")
	setTexts(t, s, "ru:"+only, "Только")
	setTexts(t, s, "drop.en:"+only, "")
	before := cloneWorkspace(s.workspace)
	if err := pickKey(s, missionBriefKey, only); err == nil || !strings.Contains(err.Error(), "has no English") {
		t.Fatalf("a briefing picked a text English lacks: %v", err)
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
	title := newMissionText(t, s, missionTitleKey)
	setTexts(t, s, "en:"+title, "The Pass")
	setTexts(t, s, "ru:"+title, "Перевал")
	brief := newMissionText(t, s, missionBriefKey)
	setTexts(t, s, "en:"+brief, "Cross \"the\" pass.")
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
	if rows := saved[mod.MissionsFile]; !strings.Contains(rows, strconv.Quote(title)) || !strings.Contains(rows, strconv.Quote(brief)) {
		t.Fatalf("the saved row does not cite the keys:\n%s", rows)
	}
	en := filepath.Join(dir, filepath.FromSlash(mod.StringsFile("en")))
	writeFileAll(t, en, append([]byte("# made on the page\n"), saved[mod.StringsFile("en")]...))
	s = openInW(t, s, dir)
	setTexts(t, s, "en:"+title, "The High Pass")
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(en); !strings.HasPrefix(string(b), "# made on the page\n") || !strings.Contains(string(b), strconv.Quote(brief)+" = \"Cross \\\"the\\\" pass.\"\n") || !strings.Contains(string(b), strconv.Quote(title)+" = \"The High Pass\"\n") {
		t.Fatalf("the English file after the second save:\n%s", b)
	}
	for lang, want := range map[string][2]string{"en": {"The High Pass", "Cross \"the\" pass."}, "ru": {"Перевал", "Cross \"the\" pass."}} {
		d, _, err := mod.DescribeMissions(dir, "authored-map", lang)
		if err != nil {
			t.Fatal(err)
		}
		i := slices.IndexFunc(d.Missions, func(m mod.Mission) bool { return m.TitleKey == title })
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
		i := slices.IndexFunc(res.Missions.Missions, func(m mod.Mission) bool { return m.TitleKey == title })
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
		a := newText(t, s, "A text to show later")
		setTexts(t, s, "en:"+a, "one")
		if _, err := s.SaveIntoMod(); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(ru)
		if err != nil || string(b) != body {
			t.Fatalf("the Russian file %q after an English edit: %q, %v", body, b, err)
		}
	}
}

// Adding a text, saving and undoing takes the strings file the save made
// out of the folder at the next save; editing a value, saving and undoing
// writes the old value back; the same after a redo and a second undo.
func TestTextsPageUndoPastASaveRestoresTheStrings(t *testing.T) {
	s, dir := terrainPage(t)
	ru := filepath.Join(dir, filepath.FromSlash(mod.StringsFile("ru")))
	a := newText(t, s, "A text to show later")
	setTexts(t, s, "ru:"+a, "один")
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
	setTexts(t, s, "ru:"+a, "один")
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	setTexts(t, s, "ru:"+a, "два")
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
	if b, _ := os.ReadFile(ru); !strings.Contains(string(b), strconv.Quote(a)+" = \"один\"") && !strings.Contains(string(b), a+" = \"один\"") {
		t.Fatalf("the Russian file after an undo past a save:\n%s", b)
	}
}

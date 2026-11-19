package game

import (
	"image"
	"image/color"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/witness"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// pageStep makes one edit of the page scope and checks it is one undo step.
func pageStep(t *testing.T, s *mapInspectionEdits, scope, key, value string) {
	t.Helper()
	steps := len(s.undoWorkspace)
	if _, err := s.SetWorkspaceField(scope, key, value); err != nil {
		t.Fatalf("%s %s = %q: %v", scope, key, value, err)
	}
	if len(s.undoWorkspace) != steps+1 {
		t.Fatalf("%s made %d undo steps", key, len(s.undoWorkspace)-steps)
	}
}

// pageRefused checks the edit is refused with text and made no undo step.
func pageRefused(t *testing.T, s *mapInspectionEdits, scope, key, value, text string) {
	t.Helper()
	steps := len(s.undoWorkspace)
	_, err := s.SetWorkspaceField(scope, key, value)
	if err == nil || !strings.Contains(err.Error(), text) || len(s.undoWorkspace) != steps {
		t.Fatalf("%s %s = %q: error %v, want %q and no step", scope, key, value, err, text)
	}
}

// seedProject puts files in the project as the mod's own, before an edit.
func seedProject(t *testing.T, x *MapInspector, s *mapInspectionEdits, files map[string]string) {
	t.Helper()
	if err := s.prepareWorkspace(); err != nil {
		t.Fatal(err)
	}
	if s.workspace.Project == nil {
		s.workspace.Project = x.mod.document()
	}
	for name, text := range files {
		s.workspace.Project.Files[name] = []byte(text)
	}
}

func pageField(t *testing.T, s *mapInspectionEdits, scope, key string) ui.InspectionField {
	t.Helper()
	for _, f := range s.WorkspaceFields(scope) {
		if f.Key == key {
			return f
		}
	}
	t.Fatalf("%s has no field %s", scope, key)
	return ui.InspectionField{}
}

func undoAll(t *testing.T, s *mapInspectionEdits) {
	t.Helper()
	for range len(s.undoWorkspace) {
		if _, err := s.Undo(); err != nil {
			t.Fatal(err)
		}
	}
}

func heldSettings(t *testing.T, s *mapInspectionEdits) []mod.Setting {
	t.Helper()
	d, err := s.heldRows(s.workspace.Project, nil, mod.SettingsName)
	if err != nil {
		t.Fatal(err)
	}
	return d.Settings.Settings
}

// The Settings page adds, changes and removes settings, each edit one undo
// step, and the file it makes reads back through the settings reader.
func TestModSettingsPageEditsSettings(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			_, s, folder := charactersPage(t, lang)
			before := tree(t, folder)
			if f := s.WorkspaceFields(ui.ModSettingsScope); len(f) != 1 || f[0].Key != settingsAddKey || !f[0].Action {
				t.Fatalf("the empty page holds %+v", f)
			}
			if !maps.Equal(tree(t, folder), before) {
				t.Fatal("an empty page wrote")
			}
			pageStep(t, s, ui.ModSettingsScope, settingsAddKey, "")
			got := heldSettings(t, s)
			if len(got) != 1 || got[0].Kind != mod.KindInt || got[0].Max != 10 || got[0].Labels["en"] == "" || got[0].Labels["ru"] == "" {
				t.Fatalf("the row: %+v", got)
			}
			pageStep(t, s, ui.ModSettingsScope, "setting.0.default", "4")
			pageStep(t, s, ui.ModSettingsScope, "setting.0.max", "3")
			if got := heldSettings(t, s)[0]; got.Default.Int != 3 || got.Max != 3 {
				t.Fatalf("a lower bound leaves the default out: %+v", got)
			}
			pageRefused(t, s, ui.ModSettingsScope, "setting.0.min", "9", "9")
			pageRefused(t, s, ui.ModSettingsScope, "setting.0.default", "x", "")
			pageStep(t, s, ui.ModSettingsScope, "setting.0.label.ru", "Скорость")
			pageStep(t, s, ui.ModSettingsScope, "setting.0.type", "2")
			pageStep(t, s, ui.ModSettingsScope, "setting.0.choices", "slow, fast")
			pageStep(t, s, ui.ModSettingsScope, "setting.0.default", "1")
			pageRefused(t, s, ui.ModSettingsScope, "setting.0.choices", "a, , b", "")
			got = heldSettings(t, s)
			if got[0].Kind != mod.KindChoice || got[0].Default.Str != "fast" || len(got[0].Choices) != 2 || got[0].Labels["ru"] != "Скорость" {
				t.Fatalf("the changed row: %+v", got[0])
			}
			if err := s.settingsGate(); err != nil {
				t.Fatal(err)
			}
			parsed, err := mod.ParseSettings(s.workspace.Project.Files[mod.SettingsName])
			if err != nil || len(parsed) != 1 || parsed[0].Key != got[0].Key {
				t.Fatalf("the file reads back as %+v, %v", parsed, err)
			}
			pageStep(t, s, ui.ModSettingsScope, "setting.0.remove", "")
			if _, held := s.workspace.Project.Files[mod.SettingsName]; held {
				t.Fatal("removing the last setting left the file")
			}
			undoAll(t, s)
			if len(s.workspace.Project.Files[mod.SettingsName]) != 0 || !maps.Equal(tree(t, folder), before) {
				t.Fatal("undo left a setting")
			}
		})
	}
}

// A settings file kept as it is stays byte-equal through a no-change edit and
// a save gate, and a file the game cannot read is refused naming the setting.
func TestModSettingsPageKeepsFileAndGatesSave(t *testing.T) {
	x, s, _ := charactersPage(t, "en")
	src := "# knobs\n[speed]  # how fast\nlabel = \"Speed\"\ntype = \"int\"\ndefault = 3\nmin = 1\nmax = 9\n"
	seedProject(t, x, s, map[string]string{mod.SettingsName: src})
	pageSame(t, s, ui.ModSettingsScope, "setting.0.label.en", "Speed")
	pageSame(t, s, ui.ModSettingsScope, "setting.0.default", "3")
	if got := string(s.workspace.Project.Files[mod.SettingsName]); got != src {
		t.Fatalf("no-change edits wrote %q", got)
	}
	if err := s.settingsGate(); err != nil {
		t.Fatal(err)
	}
	s.workspace.Project.Files[mod.SettingsName] = []byte(strings.Replace(src, "default = 3", "default = 30", 1))
	if err := s.settingsGate(); err == nil || !strings.Contains(err.Error(), "speed") {
		t.Fatalf("the gate: %v", err)
	}
}

func heldScreens(t *testing.T, s *mapInspectionEdits) []mod.Screen {
	t.Helper()
	d, err := s.heldRows(s.workspace.Project, s.screenStructure, mod.ScreensFile)
	if err != nil {
		t.Fatal(err)
	}
	return d.Screens.Screens
}

// The Screens page adds, changes and removes screens and their entries, each
// edit one undo step; the texts a new screen cites exist in every language.
func TestModScreensPageEditsScreens(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			_, s, folder := charactersPage(t, lang)
			before := tree(t, folder)
			if f := s.WorkspaceFields(ui.ModScreensScope); len(f) != 2 || f[0].Key != screensAddKey || f[1].Key != screensActionKey {
				t.Fatalf("the empty page holds %+v", f)
			}
			pageStep(t, s, ui.ModScreensScope, screensAddKey, "")
			got := heldScreens(t, s)
			if len(got) != 1 || got[0].Kind != mod.ScreenInfo || !got[0].Main || len(got[0].TextKeys) != 1 {
				t.Fatalf("the row: %+v", got)
			}
			for _, l := range textLanguages() {
				sets, err := s.heldRows(s.workspace.Project, nil, mod.StringsFile(l))
				if err != nil || sets.Strings[l][got[0].TitleKey] == "" || sets.Strings[l][got[0].TextKeys[0]] == "" {
					t.Fatalf("texts of %s: %v %v", l, sets.Strings[l], err)
				}
			}
			if err := s.screensGate(); err != nil {
				t.Fatal(err)
			}
			pageStep(t, s, ui.ModScreensScope, "screen.0.place", "2")
			pageStep(t, s, ui.ModScreensScope, "screen.0.addline", "")
			pageStep(t, s, ui.ModScreensScope, "screen.0.kind", "2")
			pageStep(t, s, ui.ModScreensScope, "screen.0.addline", "")
			pageStep(t, s, ui.ModScreensScope, "screen.0.key", "facts")
			got = heldScreens(t, s)
			if got[0].Key != "facts" || got[0].Kind != mod.ScreenTable || !got[0].Main || !got[0].Game || len(got[0].RowKeys) != 3 {
				t.Fatalf("the changed row: %+v", got[0])
			}
			pageRefused(t, s, ui.ModScreensScope, "screen.0.key", "Bad Key", "key")
			pageStep(t, s, ui.ModScreensScope, "screen.0.rows.3.remove", "")
			pageStep(t, s, ui.ModScreensScope, "screen.0.rows.2.remove", "")
			pageRefused(t, s, ui.ModScreensScope, "screen.0.rows.1.remove", "", "")
			pageStep(t, s, ui.ModScreensScope, screensActionKey, "")
			pageStep(t, s, ui.ModScreensScope, "screen.1.action", "1")
			if got := heldScreens(t, s); len(got) != 2 || got[1].Kind != mod.ActionRestart || got[1].MenuKey == "" {
				t.Fatalf("the action: %+v", got)
			}
			pageStep(t, s, ui.ModScreensScope, "screen.1.remove", "")
			pageStep(t, s, ui.ModScreensScope, "screen.0.remove", "")
			if _, held := s.workspace.Project.Files[mod.ScreensFile]; held {
				t.Fatal("removing every screen left the file")
			}
			undoAll(t, s)
			if len(s.workspace.Project.Files[mod.ScreensFile]) != 0 || !maps.Equal(tree(t, folder), before) {
				t.Fatal("undo left a screen")
			}
		})
	}
}

// A file kept as it is stays byte-equal, a menu with no room refuses the
// screen, and a save names the screen whose text is missing.
func TestModScreensPageKeepsFileSlotsAndGatesSave(t *testing.T) {
	x, s, _ := charactersPage(t, "en")
	src := "# screens\n[[screen]]\nkey = \"about\"  # note\nkind = \"info\"\ntitle = \"t\"\nmenu = \"m\"\nplace = \"game\"\ntext = [\"p\"]\n"
	strs := "t = \"Title\"\nm = \"Menu\"\np = \"Text\"\n"
	seedProject(t, x, s, map[string]string{mod.ScreensFile: src, mod.StringsFile("en"): strs, mod.StringsFile("ru"): strs})
	pageSame(t, s, ui.ModScreensScope, "screen.0.title", "3")
	pageSame(t, s, ui.ModScreensScope, "screen.0.place", "1")
	if got := string(s.workspace.Project.Files[mod.ScreensFile]); got != src {
		t.Fatalf("no-change edits wrote %q", got)
	}
	if err := s.screensGate(); err != nil {
		t.Fatal(err)
	}
	pageStep(t, s, ui.ModScreensScope, screensAddKey, "")
	pageStep(t, s, ui.ModScreensScope, screensAddKey, "")
	pageStep(t, s, ui.ModScreensScope, screensAddKey, "")
	pageStep(t, s, ui.ModScreensScope, screensAddKey, "")
	pageRefused(t, s, ui.ModScreensScope, screensAddKey, "", "place")
	pageRefused(t, s, ui.ModScreensScope, "screen.1.place", "1", "slots")
	delete(s.workspace.Project.Files, mod.StringsFile("ru"))
	s.workspace.Project.Files[mod.StringsFile("en")] = []byte("t = \"Title\"\nm = \"Menu\"\n")
	if err := s.screensGate(); err == nil || !strings.Contains(err.Error(), "about") || !strings.Contains(err.Error(), "\"p\"") {
		t.Fatalf("the gate: %v", err)
	}
}

func facePNGSized(t *testing.T, w, h int) string { return facePNGTint(t, w, h, 90) }

func facePNGTint(t *testing.T, w, h int, tint byte) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{R: byte(x * 3), G: byte(y * 2), B: tint, A: 255})
		}
	}
	p := filepath.Join(t.TempDir(), "face.png")
	if err := os.WriteFile(p, witness.Bytes(t, img), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func heldSpeakers(t *testing.T, s *mapInspectionEdits) []mod.Speaker {
	t.Helper()
	d, err := s.heldRows(s.workspace.Project, nil, mod.SpeakersFile)
	if err != nil {
		t.Fatal(err)
	}
	return d.Speakers.Speakers
}

// The Speakers page adds, renames, gives a new picture to and removes a
// speaker, each edit one undo step; a speaker with no picture refuses the
// save, and the picture comes through the mod's asset import.
func TestModSpeakersPageEditsSpeakers(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			x, s, folder := charactersPage(t, lang)
			size := speakerPictureSize()
			before := tree(t, folder)
			pageRefused(t, s, ui.ModSpeakersScope, speakersAddKey, facePNGSized(t, size.X+1, size.Y), "")
			pageRefused(t, s, ui.ModSpeakersScope, speakersAddKey, "", "")
			pageStep(t, s, ui.ModSpeakersScope, speakersAddKey, facePNGSized(t, size.X, size.Y))
			got := heldSpeakers(t, s)
			if len(got) != 1 || got[0].ID < mod.FirstModSpeaker || got[0].Picture != speakerPictureFile(got[0].ID) || len(got[0].PictureBytes) == 0 {
				t.Fatalf("the row: %+v", got)
			}
			id := got[0].ID
			wantFile := speakerPictureFile(id)
			pageSame(t, s, ui.ModSpeakersScope, "speaker.0.picture", facePNGSized(t, size.X, size.Y))
			first := string(s.workspace.Project.Files[wantFile])
			pageStep(t, s, ui.ModSpeakersScope, "speaker.0.picture", facePNGTint(t, size.X, size.Y, 200))
			if string(s.workspace.Project.Files[wantFile]) == first {
				t.Fatal("the picture did not change")
			}
			pageStep(t, s, ui.ModSpeakersScope, "speaker.0.name", "Karl")
			got = heldSpeakers(t, s)
			if got[0].Name != "Karl" || got[0].Picture != wantFile || len(s.workspace.Project.Files[wantFile]) == 0 {
				t.Fatalf("the changed row: %+v", got[0])
			}
			pageRefused(t, s, ui.ModSpeakersScope, "speaker.0.name", "", "")
			if err := s.speakersGate(); err != nil {
				t.Fatal(err)
			}
			if p := pageField(t, s, ui.ModSpeakersScope, "speaker.0.name"); p.TextValue != "Karl" {
				t.Fatalf("name field %+v", p)
			}
			_ = x
			pageStep(t, s, ui.ModSpeakersScope, "speaker.0.remove", "")
			if _, held := s.workspace.Project.Files[mod.SpeakersFile]; held {
				t.Fatal("removing the last speaker left the file")
			}
			if _, held := s.workspace.Project.Files[wantFile]; held {
				t.Fatal("removing the speaker left its picture")
			}
			undoAll(t, s)
			if len(s.workspace.Project.Files[mod.SpeakersFile]) != 0 || !maps.Equal(tree(t, folder), before) {
				t.Fatal("undo left a speaker")
			}
		})
	}
}

// A speaker a message names is not removed: the refusal names where it is
// used; a file kept as it is stays byte-equal.
func TestModSpeakersPageRefusesRemovingAUsedSpeaker(t *testing.T) {
	x, s, _ := charactersPage(t, "en")
	size := speakerPictureSize()
	src := "# faces\n[[speaker]]\nid = 20007\nname = \"Karl\"  # lord\npicture = \"assets/speakers/20007.png\"\nmap = \"maps/m.alm\"\nunit = 3\n\n[[speaker]]\nid = 20008\noff = true\n"
	seedProject(t, x, s, map[string]string{mod.SpeakersFile: src, "maps/m.text/event01.txt": "<part=1,npc=20007>Hello"})
	pic, err := os.ReadFile(facePNGSized(t, size.X, size.Y))
	if err != nil {
		t.Fatal(err)
	}
	s.workspace.Project.Files["assets/speakers/20007.png"] = pic
	pageStep(t, s, ui.ModSpeakersScope, "speaker.0.name", "Karl")
	if got := string(s.workspace.Project.Files[mod.SpeakersFile]); got != src {
		t.Fatalf("a no-change edit wrote %q", got)
	}
	pageRefused(t, s, ui.ModSpeakersScope, "speaker.0.remove", "", "maps/m.text/event01.txt")
	pageStep(t, s, ui.ModSpeakersScope, "speaker.1.remove", "")
	s.workspace.Project.Files["maps/m.text/event01.txt"] = []byte("<part=1,npc=20070>Hello")
	pageStep(t, s, ui.ModSpeakersScope, "speaker.0.name", "Ann")
	pageStep(t, s, ui.ModSpeakersScope, "speaker.0.remove", "")
	if _, held := s.workspace.Project.Files["assets/speakers/20007.png"]; held {
		t.Fatal("the picture stayed")
	}
}

// pageSame makes an edit that changes nothing: it succeeds and the project
// keeps its bytes.
func pageSame(t *testing.T, s *mapInspectionEdits, scope, key, value string) {
	t.Helper()
	if _, err := s.SetWorkspaceField(scope, key, value); err != nil {
		t.Fatalf("%s %s = %q: %v", scope, key, value, err)
	}
}

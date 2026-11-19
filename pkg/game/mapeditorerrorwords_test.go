package game

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// The files whose errors reach an editor window note: the map inspection
// host (every mapinspection file), the speaker host, the test play host and
// the mod, spells and terrain hosts.
func editorHostWordFiles(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob("mapinspection*.go")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range files {
		if !strings.HasSuffix(f, "_test.go") {
			out = append(out, f)
		}
	}
	if len(out) < 20 {
		t.Fatalf("the scan found only %d host files", len(out))
	}
	return append(out, "modspeakers.go", "mapedittestplay.go", "mapedittestplay_files.go", "mapeditormod.go",
		"mapeditormodcontext.go", "mapeditormodpublish.go", "mapeditormodspells.go", "mapeditormodspellpictures.go", "mapeditormodterrain.go")
}

// Errors that never reach a window as a note: a name the editor itself
// offered and so cannot send wrongly, a state the host built itself, or an
// install fault a log line reports. Each is named by its file and its text.
var editorHostEnglishErrors = map[string][]string{
	"mapinspectionadd.go":          {"the editor adds no %s record", "the new %s lacks a value"},
	"mapinspectionbrush.go":        {"no brush plane %d"},
	"mapinspectionedit.go":         {"a %s record cannot be edited", "edit steps are out of order", "no new node for the field %q", "no %s field %q"},
	"mapinspectioninventory.go":    {"unknown item recipe field"},
	"mapinspectionitemrecipe.go":   {"unknown item recipe field"},
	"mapinspectionpreview.go":      {"incomplete placement"},
	"mapinspectionscript.go":       {"no script arm %d", "no structure %d"},
	"mapinspectionscriptauthor.go": {"no script kind %q", "a trigger is not copied here"},
	"mapinspectionscriptform.go":   {"no new node for the field %q"},
	"mapinspectionsheet.go":        {"the world holds no actor for placement %d"},
	"mapinspectionunitsettings.go": {"unit %d has no actor"},
	"mapinspectionworkspace.go":    {"unknown page field %q", "unknown workspace %q", "unknown map metadata %q"},
	"modspeakers.go":               {"the install has no speaker records for the mods to add to"},
}

var (
	formatVerb = regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z]`)
	englishRun = regexp.MustCompile(`[A-Za-z]{2,}`)
)

// A refusal of an editor host file is a word key, not an English literal,
// unless the file and its text are named above.
func TestEditorHostErrorsHoldNoLiteral(t *testing.T) {
	used := map[string]bool{}
	for _, file := range editorHostWordFiles(t) {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			at := strings.Index(line, `errors.New("`)
			if at < 0 {
				at = strings.Index(line, `fmt.Errorf("`)
			}
			if at < 0 {
				continue
			}
			// A format that holds only verbs and marks puts words together
			// that were read in the editor's language already.
			literal, _, _ := strings.Cut(line[at+strings.Index(line[at:], `"`)+1:], `"`)
			if !englishRun.MatchString(formatVerb.ReplaceAllString(literal, "")) {
				continue
			}
			allowed := false
			for _, text := range editorHostEnglishErrors[file] {
				if strings.Contains(line[at:], `"`+text+`"`) {
					allowed, used[file+" "+text] = true, true
				}
			}
			if !allowed {
				t.Errorf("%s:%d holds an English literal error: %s", file, i+1, strings.TrimSpace(line))
			}
		}
	}
	for file, texts := range editorHostEnglishErrors {
		for _, text := range texts {
			if !used[file+" "+text] {
				t.Errorf("%s no longer holds %q: drop it from the allow list", file, text)
			}
		}
	}
}

// A refusal of each host file group reads in the editor's language, from
// the words table, in English and in Russian.
func TestEditorHostRefusalsAreInTheEditorsLanguage(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			s, _, book := modAssetProject(t, lang)
			for _, c := range []struct {
				name string
				do   func() error
				key  string
				args []any
			}{
				{"add", func() error {
					return s.CheckNewRecord(ui.InspectionNewRecord{Kind: ui.KindEnchantment, Values: []ui.InspectionFieldValue{{Key: "spell"}}}, "spell")
				}, "editor.err.spellnone", nil},
				{"authoring", func() error { _, err := s.AddItemRecipe(ui.KindUnit, 0, 1); return err }, "editor.err.recipepick", nil},
				{"groups", func() error { _, err := s.SetUnitsGroup(nil, -1); return err }, "editor.err.groupnumber", []any{-1, uint32(math.MaxUint32)}},
				{"selection", func() error { _, _, err := s.CopyRecords(nil); return err }, "editor.err.copynone", nil},
				{"workspace", func() error { _, err := s.SetWorkspaceField("pages", "page", "zzz"); return err }, "editor.err.pagename", []any{unitNamesPage}},
				{"save", func() error { _, err := s.SaveAs(""); return err }, "editor.err.saveas.path", nil},
				{"script", func() error { _, err := s.CopyTrigger(99); return err }, "editor.err.notrigger", []any{100}},
			} {
				err := c.do()
				want := fmt.Sprintf(book.Text(c.key), c.args...)
				if err == nil || err.Error() != want {
					t.Errorf("%s: the refusal is %v, want %q", c.name, err, want)
				}
				if lang == "ru" && err != nil && !cyrillic.MatchString(err.Error()) {
					t.Errorf("%s: the Russian refusal reads %q", c.name, err)
				}
			}
		})
	}
}

// A mod file refusal of the library is worded at the host boundary by its
// file and line; the library's own text stays with the error for a log.
func TestEditorLibraryRefusalIsWordedAtTheHost(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			s, _, book := modAssetProject(t, lang)
			for _, lib := range []error{
				&mod.ItemFileError{File: "data/speakers.toml", Line: 7, Msg: "speaker record 3 has no name"},
				(mod.Source{Mod: "demo", File: "data/speakers.toml", Line: 7}).Errorf("speaker record 3 has no name"),
			} {
				got := s.inspector.libraryRefusal(lib)
				want := fmt.Sprintf(book.Text("editor.err.modfile"), "data/speakers.toml", 7, "speaker record 3 has no name")
				if got.Error() != want {
					t.Errorf("%T: the host says %q, want %q", lib, got, want)
				}
				if !errors.Is(got, lib) || !strings.Contains(lib.Error(), "has no name") {
					t.Errorf("%T: the library error is lost under the worded one", lib)
				}
			}
		})
	}
}

// A refusal of the test play, the mod file and the spells host reads in the
// editor's language, and a library error that an edit returns is worded at
// the host boundary with its own text kept under it.
func TestEditorTestPlayAndModRefusalsAreInTheEditorsLanguage(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			s, _, book := modAssetProject(t, lang)
			file := filepath.Join(t.TempDir(), "plain")
			if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			_, owned := missionPackageOwnership(file, testPlayRecord, nil)
			library := errors.New("record 4 cannot move")
			_, changed := s.change(func(*mapedit.Editor) (bool, error) { return false, library })
			_, party := s.TestPlayParty(-1, nil)
			_, started := s.TestPlay()
			for _, c := range []struct {
				name string
				got  error
				key  string
				args []any
			}{
				{"test play party", party, "editor.err.tp.partyhero", nil},
				{"test play start", started, "editor.err.tp.nogame", nil},
				{"test mod files", s.inspector.inLanguage(owned), "editor.err.tp.notowned", []any{file}},
				{"test folder overlap", s.inspector.inLanguage(checkMissionPackageOverlap(file, testPlayRecord, []packageGuard{{dir: file, name: wordRef("editor.err.tp.guard.map")}})), "editor.err.tp.overlap",
					[]any{book.Text("editor.err.tp.noun.test"), file, book.Text("editor.err.tp.rel.is"), book.Text("editor.err.tp.guard.map"), file}},
				{"spells", s.inspector.inLanguage(setSpellRowField(&mod.SpellRow{}, "mana", "x")), "editor.err.spell.number", []any{"mana", "x"}},
				{"library error", changed, "editor.err.library", []any{library.Error()}},
			} {
				want := fmt.Sprintf(book.Text(c.key), c.args...)
				if c.got == nil || c.got.Error() != want {
					t.Errorf("%s: the refusal is %v, want %q", c.name, c.got, want)
				}
				if lang == "ru" && c.got != nil && !cyrillic.MatchString(c.got.Error()) {
					t.Errorf("%s: the Russian refusal reads %q", c.name, c.got)
				}
			}
			if !errors.Is(changed, library) {
				t.Errorf("the library error is lost under the worded one")
			}
		})
	}
}

package game

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/witness"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// workbenchFixture is the mod window over a new map in lang, with a program
// that resolved modsFolder as its mods folder.
func workbenchFixture(t *testing.T, lang, modsFolder string) (*testPlayFixture, *ui.MapEditor, *mapInspectionEdits) {
	t.Helper()
	f, e := workspaceFixture(t)
	f.x.archives.Base.Profile.Language = map[string]string{"en": "english", "ru": "russian"}[lang]
	f.x.testPlay.ModsFolder = modsFolder
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	return f, e, e.Document().Edits.(*mapInspectionEdits)
}

func modWindowField(t *testing.T, e *ui.MapEditor, key string) ui.InspectionField {
	t.Helper()
	for _, f := range e.ObjectFields() {
		if f.Key == key {
			return f
		}
	}
	t.Fatalf("the mod window has no %s field", key)
	return ui.InspectionField{}
}

// A mod with no folder shows the working folder the profile's workbench
// proposes and the mods folder the program resolved; Save makes the first and
// / Publish writes the packed mod into the second, with no path typed. With
// / AGAINROM_EDITOR_FRAMES set the window is written there as a PNG and its rows
// as text.
func TestModWindowProposesTheWorkbenchAndPublishesIntoTheModsFolder(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			mods := filepath.Join(t.TempDir(), "mods")
			f, e, s := workbenchFixture(t, lang, mods)
			book := words.For(f.x.archives.Base.Profile.Language)
			working := filepath.Join(f.profile, "workbench", "authored-map")
			folder, output := modWindowField(t, e, "folder"), modWindowField(t, e, "output")
			if folder.TextValue != working || folder.Label != book.Text("editor.mod.folder") {
				t.Fatalf("working folder %q (%q), want %q (%q)", folder.TextValue, folder.Label, working, book.Text("editor.mod.folder"))
			}
			if output.TextValue != mods || output.Label != book.Text("editor.mod.output") {
				t.Fatalf("Publish folder %q (%q), want %q (%q)", output.TextValue, output.Label, mods, book.Text("editor.mod.output"))
			}
			if lang == "ru" && (folder.Label == words.For("english").Text("editor.mod.folder") || output.Label == words.For("english").Text("editor.mod.output")) {
				t.Fatal("the Russian mod window shows the English labels")
			}
			if _, err := os.Lstat(filepath.Join(f.profile, "workbench")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("showing the proposal made the workbench", err)
			}
			if dir := os.Getenv("AGAINROM_EDITOR_FRAMES"); dir != "" {
				pic, _, err := e.HeadlessFrame()
				if err != nil {
					t.Fatal(err)
				}
				witness.PNG(t, filepath.Join(dir, "editor-mod-window-proposed-folders-"+lang+".png"), pic)
				// The fixture font draws no letters: the rows go beside the picture.
				var rows strings.Builder
				for _, field := range e.ObjectFields() {
					fmt.Fprintf(&rows, "%s: %s\n", field.Label, field.TextValue)
				}
				writeTestPlayInput(t, filepath.Join(dir, "editor-mod-window-proposed-folders-"+lang+".txt"), []byte(rows.String()))
			}
			if got, want := s.ModOutput(), filepath.Join(mods, "mod-authored-map.res"); got != want {
				t.Fatalf("Publish target %q, want %q", got, want)
			}
			if _, err := s.SetWorkspaceField("package", "save", ""); err != nil {
				t.Fatal(err)
			}
			s = e.Document().Edits.(*mapInspectionEdits)
			if s.inspector.mod == nil || !sameSaveName(s.inspector.mod.dir, working) || !s.inspector.mod.owned {
				t.Fatalf("Save did not make the proposed working mod: %+v", s.inspector.mod)
			}
			if _, err := os.Stat(filepath.Join(working, mod.ManifestName)); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(s.SaveNote(), book.Text("editor.mod.saved")) {
				t.Fatalf("Save note %q", s.SaveNote())
			}
			if _, err := s.SetWorkspaceField("package", "publish", ""); err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(mods, "mod-authored-map.res")
			if st, err := os.Stat(file); err != nil || !st.Mode().IsRegular() {
				t.Fatalf("Publish wrote no %s: %v", file, err)
			}
			if s.inspector.modOutput != "" || !strings.Contains(s.SaveNote(), book.Text("editor.mod.published")) {
				t.Fatalf("Publish typed a path or left no receipt: %q %q", s.inspector.modOutput, s.SaveNote())
			}
			listed, err := mod.Scan(mods)
			if err != nil || len(listed) != 1 || listed[0].Err != nil || listed[0].Manifest.ID != "authored-map" {
				t.Fatalf("the mods folder lists %+v, %v", listed, err)
			}
			// The file is there now: Publish asks before it replaces it.
			var ask *ui.WorkspaceConfirmation
			if _, err := s.SetWorkspaceField("package", "publish", ""); !errors.As(err, &ask) || ask.Action != "publish-replace" {
				t.Fatal("a second Publish did not ask Replace", err)
			}
			// A path can still be typed, and a blank field returns to the
			// resolved folder.
			typed := t.TempDir()
			if _, err := s.SetWorkspaceField("package", "output", typed); err != nil {
				t.Fatal(err)
			}
			if _, err := s.SetWorkspaceField("package", "publish", ""); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(typed, "mod-authored-map.res")); err != nil {
				t.Fatal("Publish did not write into the typed folder", err)
			}
			if _, err := s.SetWorkspaceField("package", "output", ""); err != nil || s.ModOutput() != file {
				t.Fatalf("a blank Publish folder gives %q, %v", s.ModOutput(), err)
			}
		})
	}
}

// An id that Windows keeps no folder for proposes no working folder, as an
// unusable id proposes none.
func TestModWindowProposesNoFolderWindowsCannotKeep(t *testing.T) {
	f, _, _ := workbenchFixture(t, "en", filepath.Join(t.TempDir(), "mods"))
	for id, want := range map[string]bool{"authored-map": true, "con": false, "nul.map": false, "com1": false, "lpt9.x": false,
		"map.": false, "console": true, "com10": true, "a.con": true, "bad id": false} {
		if got := f.x.proposedModFolder(id) != ""; got != want {
			t.Errorf("id %q proposed a folder: %v, want %v", id, got, want)
		}
	}
}

// Save does not write over a folder that already stands where the working
// folder is proposed.
func TestModWindowSaveRefusesATakenProposedFolder(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			f, _, s := workbenchFixture(t, lang, filepath.Join(t.TempDir(), "mods"))
			working := filepath.Join(f.profile, "workbench", "authored-map")
			if err := os.MkdirAll(working, 0o755); err != nil {
				t.Fatal(err)
			}
			_, err := s.SetWorkspaceField("package", "save", "")
			want := strings.SplitN(words.For(f.x.archives.Base.Profile.Language).Text("editor.mod.err.exists"), "%s", 2)[1]
			if err == nil || !strings.Contains(err.Error(), want) || !strings.Contains(err.Error(), working) {
				t.Fatalf("Save over a taken folder: %v", err)
			}
			if len(tree(t, working)) != 0 {
				t.Fatal("the refused Save wrote into the folder")
			}
		})
	}
}

// Every refusal Publish had stands when no path is typed.
func TestModWindowPublishRefusalsStandWithNoPathTyped(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, kind := range []string{"no-copy", "no-folder", "install", "inside", "taken", "changed", "unsaved", "no-map"} {
			t.Run(lang+"/"+kind, func(t *testing.T) {
				mods := filepath.Join(t.TempDir(), "mods")
				f, e, s := workbenchFixture(t, lang, mods)
				book := words.For(f.x.archives.Base.Profile.Language)
				working := filepath.Join(f.profile, "workbench", "authored-map")
				if kind != "no-copy" {
					if _, err := s.SetWorkspaceField("package", "save", ""); err != nil {
						t.Fatal(err)
					}
					s = e.Document().Edits.(*mapInspectionEdits)
				}
				want, confirm := "", ""
				switch kind {
				case "no-copy":
					want = book.Text("editor.mod.err.nocopy")
				case "no-folder":
					f.x.testPlay.ModsFolder = ""
					want = book.Text("editor.mod.err.nooutput")
				case "install":
					mods = filepath.Join(f.root, "mods")
					f.x.testPlay.ModsFolder = mods
					want = f.root
				case "inside":
					mods = filepath.Join(working, "out")
					f.x.testPlay.ModsFolder = mods
					want = book.Text("editor.mod.err.outputinside")
				case "taken":
					if err := os.MkdirAll(filepath.Join(mods, "authored-map"), 0o755); err != nil {
						t.Fatal(err)
					}
					want = "authored-map"
				case "changed":
					manifest, err := os.ReadFile(filepath.Join(working, mod.ManifestName))
					if err != nil {
						t.Fatal(err)
					}
					writeTestPlayInput(t, filepath.Join(working, mod.ManifestName), append(manifest, "# written outside the editor\n"...))
					want = book.Text("editor.mod.err.changed")
				case "unsaved":
					if _, err := s.SetWorkspaceField("package", "gold", "73"); err != nil {
						t.Fatal(err)
					}
					confirm = "publish-save"
				case "no-map":
					// A new map opened over the working mod is none of its missions.
					_ = e.Key("escape")
					if err := e.Key("new"); err != nil {
						t.Fatal(err)
					}
					_ = e.Key("clear")
					e.TextInput("32x32")
					for range 2 {
						if err := e.Key("enter"); err != nil {
							t.Fatal(err)
						}
					}
					s = e.Document().Edits.(*mapInspectionEdits)
					if s.inspector.mod == nil {
						t.Fatal("the new map closed the working mod")
					}
					want = book.Text("editor.mod.err.addmap")
				}
				_, err := s.SetWorkspaceField("package", "publish", "")
				var ask *ui.WorkspaceConfirmation
				switch {
				case confirm != "":
					if !errors.As(err, &ask) || ask.Action != confirm {
						t.Fatalf("Publish did not ask %s: %v", confirm, err)
					}
				case err == nil || errors.As(err, &ask) || !strings.Contains(err.Error(), want):
					t.Fatalf("Publish answered %v, want a refusal holding %q", err, want)
				}
				if _, err := os.Lstat(filepath.Join(mods, "mod-authored-map.res")); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("a refused Publish left a packed mod", err)
				}
				// A refusal and a question not answered make no mods folder.
				if kind != "taken" {
					if _, err := os.Lstat(mods); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("a refused Publish made the folder %s: %v", mods, err)
					}
				}
			})
		}
	}
}

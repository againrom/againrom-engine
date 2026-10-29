package game

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
)

func TestEditorPageEditKeepsLanguageInsideTextsPath(t *testing.T) {
	testEditorModPageEdit(t, "declared")
}

func TestEditorPageEditReachesMapSiblingPages(t *testing.T) {
	for _, route := range []string{"sibling", "sibling-and-declared"} {
		t.Run(route, func(t *testing.T) { testEditorModPageEdit(t, route) })
	}
}

func testEditorModPageEdit(t *testing.T, route string) {
	t.Helper()
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			f, e := workspaceFixture(t)
			language := map[string]string{"en": "english 0", "ru": "russian 1"}[lang]
			writeTestPlayInput(t, filepath.Join(f.root, MainArchive), synth.Archive([]synth.File{{Path: "id", Data: []byte(language)}}))
			for _, name := range []string{GraphicsArchive, WorldArchive, MoviesArchive} {
				writeTestPlayInput(t, filepath.Join(f.root, name), synth.Archive(nil))
			}
			f.x.archives.Base.Profile.Language = map[string]string{"en": "english", "ru": "russian"}[lang]
			if got := BaseID(InspectInstall(f.root)); got != "rom1-"+lang {
				t.Fatalf("fixture language = %q", got)
			}
			declarations := "[[mission]]\nkey = 'first'\nnumber = 25\nmap = 'first.alm'\nafter = 20\nparty = [{character = 'npc22'}]\n"
			if route != "sibling" {
				declarations += "texts = 'mission/{lang}/text' # keep template\n"
			}
			dir := writeEditorMod(t, t.TempDir(), declarations)
			writeTestPlayInput(t, filepath.Join(dir, "first.alm"), e.Document().SourceBytes)
			for _, l := range []string{"en", "ru"} {
				if route != "declared" {
					writeTestPlayInput(t, filepath.Join(dir, "first.text", l, "event01.txt"), []byte("event "+l))
					writeTestPlayInput(t, filepath.Join(dir, "first.text", l, "title.txt"), []byte("title "+l))
				}
				if route != "sibling" {
					writeTestPlayInput(t, filepath.Join(dir, "mission", l, "text", "event01.txt"), []byte("event "+l))
					writeTestPlayInput(t, filepath.Join(dir, "mission", l, "text", "title.txt"), []byte("title "+l))
				}
			}
			before := tree(t, dir)
			if err := workspaceField(e, "package", 0, dir); err != nil {
				t.Fatal(err)
			}
			copyDir := filepath.Join(t.TempDir(), "ambush")
			if err := workspaceField(e, "package", 0, copyDir); err != nil {
				t.Fatal(err)
			}
			workspacePage(t, e, "event01.txt", "edited "+lang)
			s := e.Document().Edits.(*mapInspectionEdits)
			check := func(folder string) {
				t.Helper()
				for _, l := range []string{"en", "ru"} {
					result, err := modrt.Load([]mod.Entry{mod.Load(folder)}, "rom1-"+l, nil, modrt.Options{})
					if err != nil || len(result.Missions.Missions) != 1 {
						t.Fatalf("load %s %s: %v", folder, l, err)
					}
					pages := result.Missions.Missions[0].TextFiles
					want := "event " + l
					if l == lang {
						want = "edited " + l
					}
					if string(pages["event01.txt"]) != want || string(pages["title.txt"]) != "title "+l {
						t.Errorf("%s %s: pages = %q, want event %q and title %q", folder, l, pages, want, "title "+l)
					}
				}
				written, err := os.ReadFile(filepath.Join(folder, mod.MissionsFile))
				if err != nil || !bytes.Equal(written, []byte(declarations)) {
					t.Errorf("page edit changed declaration: %v\n%s", err, written)
				}
			}
			if _, err := s.TestPlay(); err != nil {
				t.Fatal(err)
			}
			check(filepath.Join(f.profile, TestPlayModID, "ambush"))
			if len(f.started) != 1 || !s.Dirty() {
				t.Fatal("F5 did not use the recording launch or changed the checkpoint")
			}
			if _, err := s.selectWorkspaceMission(1); err != nil {
				t.Fatal(err)
			}
			if got := string(s.workspace.Pages["event01.txt"]); got != "edited "+lang {
				t.Errorf("mission switch lost pending page: %q", got)
			}
			if _, err := s.SaveIntoMod(); err != nil {
				t.Fatal(err)
			}
			check(copyDir)
			doc, err := s.SetWorkspaceField("package", "folder-open", copyDir)
			if err != nil {
				t.Fatal(err)
			}
			if got := string(doc.Edits.(*mapInspectionEdits).workspace.Pages["event01.txt"]); got != "edited "+lang {
				t.Errorf("reopened page = %q", got)
			}
			if !reflect.DeepEqual(before, tree(t, dir)) {
				t.Fatal("source mod changed")
			}
		})
	}
}

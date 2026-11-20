package game

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/witness"
	"againrom/pkg/mod"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// moveOldProject is the older-layout mod of form opened as the editor's
// project, its speaker picture a real picture on the old path
// speakers/karl.png.
func moveOldProject(t *testing.T, formName string) (*mapInspectionEdits, string, map[string]string) {
	t.Helper()
	_, s, dir, moves := moveOldEditor(t, formName)
	return s, dir, moves
}

// moveOldEditor is moveOldProject with the editor that holds the project.
func moveOldEditor(t *testing.T, formName string) (*ui.MapEditor, *mapInspectionEdits, string, map[string]string) {
	t.Helper()
	for _, form := range oldLayoutForms {
		if form.name != formName {
			continue
		}
		f, e := workspaceFixture(t)
		dir := writeEditorMod(t, t.TempDir(), form.decl)
		for name, body := range form.files {
			file := filepath.Join(dir, filepath.FromSlash(name))
			b := []byte(body)
			switch body {
			case "map":
				b = e.Document().SourceBytes
			case "png":
				writeTestPlayInput(t, file, nil)
				b = speakerPNG(t, file, color.RGBA{R: 9, G: 8, B: 7, A: 255})
			}
			writeTestPlayInput(t, file, b)
		}
		return e, selectProject(t, f, e, dir, form.opened), dir, form.moves
	}
	t.Fatalf("no form %s", formName)
	return nil, nil, "", nil
}

// The mod window's button row holds the button and pressing it moves the
// files; with AGAINROM_MOVEOLD_FRAMES set the window is drawn at 1920x1080
// into that folder before the press.
func TestModWindowMoveOldPathsButton(t *testing.T) {
	if root := os.Getenv("AGAINROM_ASSETS"); root != "" && os.Getenv("AGAINROM_MOVEOLD_FRAMES") != "" {
		a, err := OpenArchives(root)
		if err != nil {
			t.Fatal(err)
		}
		font, err := LoadFont(a.Containers, DefaultFont, FontShades)
		if err != nil {
			t.Fatal(err)
		}
		font.Selector = LanguageSelector(a.Containers)
		body, err := LoadFont(a.Containers, DocumentFont, FontCoverage)
		if err != nil {
			body = font
		}
		body.Selector = font.Selector
		workspaceFonts = []*text.Font{font, body}
		t.Cleanup(func() { workspaceFonts = nil })
	}
	e, s, dir, _ := moveOldEditor(t, "assembled")
	e.Layout(1920, 1080)
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	if !e.FocusModField(modMoveOldKey) {
		t.Fatal("the mod window has no Move old paths button")
	}
	_ = s
	if out := os.Getenv("AGAINROM_MOVEOLD_FRAMES"); out != "" {
		pic := e.ChromeImage()
		if !pic.Bounds().Eq(image.Rect(0, 0, 1920, 1080)) {
			t.Fatalf("the window is drawn %v", pic.Bounds())
		}
		witness.PNG(t, filepath.Join(out, "mod-window-moveold-1920x1080.png"), pic)
	}
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "assets", "speakers", "karl.png")); err != nil {
		t.Errorf("the button moved nothing: %v", err)
	}
}

// The button moves the mission files into the maps folder and the speaker
// picture into assets/speakers, rewrites the rows that name them and nothing
// else; the check then names no old path, every language reads the same maps
// and pages, and the packed mod holds the moved files.
func TestMoveOldPathsCarriesEveryRowFileIntoTheTree(t *testing.T) {
	for _, name := range []string{"package", "added", "assembled"} {
		t.Run(name, func(t *testing.T) {
			s, dir, moves := moveOldProject(t, name)
			if name == "assembled" {
				moves = map[string]string{"speakers/karl.png": "assets/speakers/karl.png"}
				for from, to := range oldLayoutForms[2].moves {
					moves[from] = to
				}
			}
			before := tree(t, dir)
			viewBefore, warned := layoutView(t, mod.Load(dir))
			if len(oldPaths(warned)) == 0 {
				t.Fatalf("the check names no old path before the move: %q", warned)
			}
			if _, err := s.SetWorkspaceField("package", modMoveOldKey, ""); err != nil {
				t.Fatal(err)
			}
			after := tree(t, dir)
			for from, to := range moves {
				if body, ok := before[from]; !ok || after[to] != body || after[from] != "" {
					t.Errorf("%s did not move to %s", from, to)
				}
			}
			if len(after) != len(before) {
				t.Errorf("the move left %d files of %d", len(after), len(before))
			}
			if name == "assembled" {
				want := "[[speaker]]\nid = 20001\nname = 'Karl'\npicture = \"assets/speakers/karl.png\"\nmap = \"maps/en/map.alm\" # bound\nunit = 7\n"
				if after[mod.SpeakersFile] != want {
					t.Errorf("the speakers file is\n%s\nwant\n%s", after[mod.SpeakersFile], want)
				}
			}
			opened, err := mod.OpenTree(dir)
			if err != nil {
				t.Fatal(err)
			}
			described, err := opened.Describe()
			if err != nil {
				t.Fatal(err)
			}
			for _, w := range described.Warnings {
				if strings.Contains(w, "uses an old path") {
					t.Errorf("the check still warns: %s", w)
				}
			}
			if len(described.OldPaths) != 0 {
				t.Errorf("old paths remain: %v", described.OldPaths)
			}
			for _, to := range moves {
				if !described.Admits(to) {
					t.Errorf("the tree does not admit %s", to)
				}
			}
			viewAfter, warned := layoutView(t, mod.Load(dir))
			if len(oldPaths(warned)) != 0 || !reflect.DeepEqual(viewAfter, viewBefore) {
				t.Errorf("the mod reads\n%q %q\nafter the move, before\n%q", viewAfter, warned, viewBefore)
			}
			if packed, _ := layoutView(t, packedEntry(t, dir)); !reflect.DeepEqual(packed, viewBefore) {
				t.Errorf("the packed mod reads\n%q\nwant\n%q", packed, viewBefore)
			}
			// A second press finds nothing to move and changes nothing.
			if _, err := s.SetWorkspaceField("package", modMoveOldKey, ""); err != nil {
				t.Fatal(err)
			}
			if s.saveNote != s.inspector.word("editor.mod.nomove") || !reflect.DeepEqual(after, tree(t, dir)) {
				t.Errorf("the second press said %q or changed the mod", s.saveNote)
			}
		})
	}
}

// A target that holds a different file refuses the whole move, naming both
// paths, and writes and removes nothing.
func TestMoveOldPathsRefusesToWriteOverADifferentFile(t *testing.T) {
	s, dir, _ := moveOldProject(t, "assembled")
	writeTestPlayInput(t, filepath.Join(dir, "assets", "speakers", "karl.png"), []byte("another picture"))
	before := tree(t, dir)
	_, err := s.SetWorkspaceField("package", modMoveOldKey, "")
	if err == nil || !strings.Contains(err.Error(), "speakers/karl.png") || !strings.Contains(err.Error(), "assets/speakers/karl.png") {
		t.Fatalf("the move answers %v, want a refusal naming both paths", err)
	}
	if !reflect.DeepEqual(before, tree(t, dir)) {
		t.Error("a refused move changed the mod")
	}
}

package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/internal/witness"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/town"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// assetFile writes a synthetic file named name and answers its path and bytes.
func assetFile(t *testing.T, name string, data []byte) (string, []byte) {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return p, data
}

// assetPNG is a synthetic PNG of the size given in one colour.
func assetPNG(t *testing.T, w, h int, ink color.RGBA) []byte {
	t.Helper()
	pic := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(pic.Pix); i += 4 {
		pic.Pix[i], pic.Pix[i+1], pic.Pix[i+2], pic.Pix[i+3] = ink.R, ink.G, ink.B, ink.A
	}
	return witness.Bytes(t, pic)
}

// modAssetProject is a mod's project open in the editor in the language
// given, with the word book of that language.
func modAssetProject(t *testing.T, lang string) (*mapInspectionEdits, string, words.Book) {
	t.Helper()
	s, dir := modSpeakerProject(t)
	language := map[string]string{"en": "english", "ru": "russian"}[lang]
	s.inspector.archives.Base.Profile.Language = language
	return s, dir, words.For(language)
}

// An item picture is imported as a draft at assets/items under its own name
// in lower case: one undo step, nothing written before the save, and the
// save writes it there. A second import of the name asks before it replaces.
func TestImportItemPictureIsOneDraftUntilSave(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			s, dir, book := modAssetProject(t, lang)
			before := tree(t, dir)
			steps := len(s.undoWorkspace)
			draft := workspaceBytes(s.workspace)
			source, data := assetFile(t, "Sword.PNG", assetPNG(t, 8, 8, color.RGBA{R: 200, A: 255}))
			const rel = "assets/items/sword.png"

			if _, err := s.SetWorkspaceField("package", modAssetImportKey+"items", source); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, tree(t, dir)) {
				t.Fatal("the import wrote into the mod folder")
			}
			if len(s.undoWorkspace) != steps+1 || !s.Dirty() {
				t.Fatalf("undo steps %d, want %d; dirty %v", len(s.undoWorkspace), steps+1, s.Dirty())
			}
			if !bytes.Equal(s.workspace.Project.Files[rel], data) {
				t.Fatalf("the project holds the drafts %v", slices.Sorted(mapKeys(s.workspace.Project.Files)))
			}
			if note := s.SaveNote(); !strings.Contains(note, rel) || !strings.Contains(note, strings.SplitN(book.Text("editor.asset.imported"), " ", 2)[0]) {
				t.Fatalf("the note is %q", note)
			}
			if _, err := s.Undo(); err != nil {
				t.Fatal(err)
			}
			if len(s.undoWorkspace) != steps || !bytes.Equal(draft, workspaceBytes(s.workspace)) {
				t.Fatal("undo did not remove the draft")
			}
			if _, err := s.Redo(); err != nil {
				t.Fatal(err)
			}
			if _, err := s.SaveIntoMod(); err != nil {
				t.Fatal(err)
			}
			if got := tree(t, dir)[rel]; got != string(data) {
				t.Fatalf("the save wrote %d bytes at %s, want %d", len(got), rel, len(data))
			}

			// The same name again: a question, then the replacement.
			again, next := assetFile(t, "sword.png", assetPNG(t, 9, 9, color.RGBA{G: 200, A: 255}))
			steps = len(s.undoWorkspace)
			_, err := s.SetWorkspaceField("package", modAssetImportKey+"items", again)
			var ask *ui.WorkspaceConfirmation
			if !errors.As(err, &ask) || ask.Action != modAssetReplaceKey || !strings.Contains(ask.Message, rel) || ask.Confirm != book.Text("editor.mod.replacefile") {
				t.Fatalf("the second import answered %v", err)
			}
			if want := strings.SplitN(book.Text("editor.asset.replace"), "%s", 2)[0]; !strings.HasPrefix(ask.Message, want) {
				t.Fatalf("the question %q is not in the editor's language", ask.Message)
			}
			if len(s.undoWorkspace) != steps || s.Dirty() {
				t.Fatal("the question changed the project")
			}
			if _, err := s.SetWorkspaceField("package", modAssetReplaceKey, "confirmed"); err != nil {
				t.Fatal(err)
			}
			if len(s.undoWorkspace) != steps+1 || !bytes.Equal(s.workspace.Project.Files[rel], next) {
				t.Fatal("the answer did not replace the file in one step")
			}
			if _, err := s.SaveIntoMod(); err != nil {
				t.Fatal(err)
			}
			if got := tree(t, dir)[rel]; got != string(next) {
				t.Fatal("the save did not write the replacement")
			}
			// Undoing both imports and saving takes the file out again.
			for range 2 {
				if _, err := s.Undo(); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := s.SaveIntoMod(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(dir, mod.AssetsFolder)); !os.IsNotExist(err) {
				t.Fatalf("undoing the imports and saving left the file or its folder in the mod: %v", err)
			}
		})
	}
}

func mapKeys(m map[string][]byte) func(func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

// A file the kind's loader refuses is refused at the import, in the loader's
// own words, and leaves no draft and no undo step.
func TestImportRefusesWhatTheKindsLoaderRefuses(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			s, dir, book := modAssetProject(t, lang)
			before := tree(t, dir)
			steps := len(s.undoWorkspace)
			draft := workspaceBytes(s.workspace)
			wide := assetPNG(t, maxModIconSide+1, 4, color.RGBA{A: 255})
			small := assetPNG(t, 4, 4, color.RGBA{A: 255})
			_, itemWords := modIconSprite(wide)
			_, faceWords := decodeSpeakerPicture(small)
			_, sheetWords := decodeBodySheet([]byte("text"))
			_, townWords := town.Parse([]byte("{"))
			_, artWords := decodeModPicture([]byte("text"), bmpDepth([]byte("text")))
			for _, tc := range []struct {
				kind, name string
				data       []byte
				want       string
			}{
				{"items", "wide.png", wide, itemWords.Error()},
				{"speakers", "small.png", small, faceWords.Error()},
				{"bodies", "sheet.png", []byte("text"), sheetWords.Error()},
				{"towns", "east.json", []byte("{"), townWords.Error()},
				{"towns", "square.bmp", []byte("text"), artWords.Error()},
				{"items", "notes.txt", small, strings.SplitN(book.Text("editor.asset.err.type"), "%s", 3)[1]},
				{"spells", "icon.png", small, strings.SplitN(book.Text("editor.asset.err.kind"), "%q", 2)[0]},
				{"items", ".hidden.png", small, "dot"},
			} {
				source, _ := assetFile(t, tc.name, tc.data)
				_, err := s.SetWorkspaceField("package", modAssetImportKey+tc.kind, source)
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Errorf("%s %s: the import answered %v, want %q", tc.kind, tc.name, err, tc.want)
				}
			}
			if len(s.undoWorkspace) != steps || !bytes.Equal(draft, workspaceBytes(s.workspace)) {
				t.Error("a refused import left a draft or an undo step")
			}
			if !reflect.DeepEqual(before, tree(t, dir)) {
				t.Error("a refused import wrote into the mod folder")
			}
		})
	}
}

// The speaker's face is this import: the record and the face at
// assets/speakers/<id>.png are one undo step, and a face the message
// window cannot show is refused in the decoder's words.
func TestSpeakerFaceIsTheImport(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			s, dir, _ := modAssetProject(t, lang)
			before := tree(t, dir)
			steps := len(s.undoWorkspace)
			draft := workspaceBytes(s.workspace)
			small, data := assetFile(t, "small.png", assetPNG(t, 4, 4, color.RGBA{A: 255}))
			_, want := decodeSpeakerPicture(data)
			if _, err := s.AddSpeaker(speakerLady, "Lady", small); err == nil || err.Error() != want.Error() {
				t.Fatalf("the small face answered %v, want %v", err, want)
			}
			if len(s.undoWorkspace) != steps || !bytes.Equal(draft, workspaceBytes(s.workspace)) {
				t.Fatal("the refused face left a draft")
			}
			source := facePNG(t)
			face, _ := os.ReadFile(source)
			id, err := s.AddSpeaker(speakerLady, "Lady", source)
			if err != nil {
				t.Fatal(err)
			}
			rel := mod.AssetsFolder + "/speakers/" + itoa(id) + ".png"
			if len(s.undoWorkspace) != steps+1 || !bytes.Equal(s.workspace.Project.Files[rel], face) || !reflect.DeepEqual(before, tree(t, dir)) {
				t.Fatalf("the face is not one draft at %s", rel)
			}
			if !strings.Contains(string(s.workspace.Project.Files[mod.SpeakersFile]), `"`+rel+`"`) {
				t.Fatalf("the record does not name %s: %s", rel, s.workspace.Project.Files[mod.SpeakersFile])
			}
			if _, err := s.Undo(); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(draft, workspaceBytes(s.workspace)) {
				t.Fatal("undo did not remove the record and the face")
			}
			if _, err := s.Redo(); err != nil {
				t.Fatal(err)
			}
			if _, err := s.SaveIntoMod(); err != nil {
				t.Fatal(err)
			}
			if got := tree(t, dir)[rel]; got != string(face) {
				t.Fatalf("the save wrote %d bytes at %s", len(got), rel)
			}
			if lines := s.inspector.mod.checkLines; slices.ContainsFunc(lines, func(l string) bool { return strings.HasPrefix(l, "warning:") }) {
				t.Fatalf("the saved mod's check warns: %q", lines)
			}
		})
	}
}

func itoa(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// Every sort of file the tree keeps under assets is a row of the import, and
// each row imports: the kinds with no page of their own are reached as data
// of the one builder.
func TestImportTakesEverySortTheTreeKeepsUnderAssets(t *testing.T) {
	want := map[string]bool{}
	for _, k := range mod.TreeKinds() {
		place, exts := k.AssetPlace()
		for _, ext := range exts {
			want[place+" "+ext] = true
		}
	}
	got := map[string]bool{}
	for _, a := range modAssets {
		got[mod.AssetsFolder+"/"+a.folder+" "+a.ext] = true
		if k, ok := mod.AssetKind(a.folder); !ok || k.Name != a.ref.Kind {
			t.Errorf("%s%s: the tree's kind there is %q, the row's rule is of %q", a.folder, a.ext, k.Name, a.ref.Kind)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("the import takes %v; the tree keeps %v", got, want)
	}
	if kinds := modAssetKinds(); !reflect.DeepEqual(kinds, []string{"items", "bodies", "speakers", "towns"}) {
		t.Fatalf("the import's kinds are %v", kinds)
	}

	s, dir, _ := modAssetProject(t, "en")
	description, _ := json.Marshal(map[string]any{"art": []town.ArtSpec{{Name: "a", Format: "picture", Key: mod.ModArtPrefix + "assets/towns/square.bmp"}}})
	files := map[string][]byte{
		"assets/items/ring.png":    assetPNG(t, 6, 6, color.RGBA{B: 200, A: 255}),
		"assets/bodies/hammer.png": assetPNG(t, 16, 16, color.RGBA{B: 200, A: 255}),
		"assets/towns/east.json":   description,
		"assets/towns/square.bmp":  synth.BMP24(image.NewRGBA(image.Rect(0, 0, 4, 4))),
	}
	for _, rel := range slices.Sorted(mapKeys(files)) {
		kind := strings.Split(rel, "/")[1]
		source, data := assetFile(t, strings.ToUpper(filepath.Base(rel)), files[rel])
		steps := len(s.undoWorkspace)
		got, _, err := s.ImportModAsset(kind, source, false)
		if err != nil || got != rel || len(s.undoWorkspace) != steps+1 || !bytes.Equal(s.workspace.Project.Files[rel], data) {
			t.Fatalf("%s: imported as %q in %d steps: %v", rel, got, len(s.undoWorkspace)-steps, err)
		}
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	held := tree(t, dir)
	for rel, data := range files {
		if held[rel] != string(data) {
			t.Errorf("the save did not write %s", rel)
		}
	}
}

// The imported name is the picked file's in lower case, cut to the bound of
// one packed name with its extension kept.
func TestImportedNameIsLowerCaseWithinTheNameBound(t *testing.T) {
	long := strings.Repeat("N", mod.MaxNameBytes) + ".PNG"
	for source, want := range map[string]string{
		filepath.Join("C:", "Pictures", "Sword.PNG"): "sword.png",
		long:                              strings.Repeat("n", mod.MaxNameBytes-4) + ".png",
		"Щит.png":                         "щит.png",
		strings.Repeat("я", 200) + ".png": strings.Repeat("я", 125) + ".png",
	} {
		if got := modAssetName(source); got != want || len(got) > mod.MaxNameBytes {
			t.Errorf("%q is imported as %q (%d bytes), want %q", source, got, len(got), want)
		}
	}
}

// The check a player runs and the editor's overview of the mod both warn of
// a file under assets no row names and of a named file outside assets; a mod
// whose rows name its assets shows neither.
func TestAssetWarningsReachTheCheckAndTheEditor(t *testing.T) {
	face := assetPNG(t, speakerPictureSize().X, speakerPictureSize().Y, color.RGBA{R: 9, A: 255})
	entry := "def init(game, settings):\n    game.data.add(\"data/speakers.toml\")\n"
	for _, tc := range []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{"old", map[string]string{
			mod.SpeakersFile:        speakerRecordText(mod.FirstModSpeaker, "speakers/karl.png"),
			"speakers/karl.png":     string(face),
			"assets/items/lost.png": string(assetPNG(t, 4, 4, color.RGBA{A: 255})),
		}, []string{
			"data/speakers.toml:4: picture speakers/karl.png uses an old path; use assets/speakers/<name>.png",
			"assets/items/lost.png: no row names this file of items; the game does not read it",
		}},
		{"clean", map[string]string{
			mod.SpeakersFile:           speakerRecordText(mod.FirstModSpeaker, "assets/speakers/karl.png"),
			"assets/speakers/karl.png": string(face),
		}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeNamedEditorMod(t, t.TempDir(), "ambush", "")
			writeModText(t, dir, mod.EntryName, entry)
			for rel, body := range tc.files {
				writeModText(t, dir, rel, body)
			}
			warnings := func(lines []string) []string {
				var out []string
				for _, l := range lines {
					if rest, ok := strings.CutPrefix(l, "warning: "); ok {
						out = append(out, rest[strings.Index(rest, ": ")+2:])
					}
				}
				slices.Sort(out)
				return out
			}
			want := slices.Clone(tc.want)
			slices.Sort(want)
			result, err := modrt.Load([]mod.Entry{mod.Load(dir)}, "rom1-en", nil, modrt.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if got := warnings(ModCheckLines(result)); !reflect.DeepEqual(got, want) {
				t.Errorf("the check warns %q, want %q", got, want)
			}
			_, s := speakerPickEditor(t)
			if err := s.inspector.SetMod(dir); err != nil {
				t.Fatal(err)
			}
			if got := warnings(s.inspector.mod.checkLines); !reflect.DeepEqual(got, want) {
				t.Errorf("the editor's check warns %q, want %q", got, want)
			}
		})
	}
}

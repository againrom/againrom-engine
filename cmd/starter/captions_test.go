package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
	"unicode"

	"againrom/pkg/game"
)

// sameInBothLanguages are the rows whose Russian text is the English text: a
// bare format verb, a file name, a product mark.
var sameInBothLanguages = map[captionID]bool{
	cText: true, cModState: true, cSettingsToml: true, cBaseOK: true,
}

func verbs(s string) string { return strings.Join(verbPattern.FindAllString(s, -1), " ") }

func hasCyrillic(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Cyrillic, r) {
			return true
		}
	}
	return false
}

func TestCaptionTableIsComplete(t *testing.T) {
	seen := map[captionID]bool{}
	for _, r := range captionRows {
		if seen[r.id] {
			t.Errorf("caption %q is listed twice", r.id)
		}
		seen[r.id] = true
		en, ru := r.by["en"], r.by["ru"]
		if en == "" || ru == "" {
			t.Errorf("caption %q lacks a language: en %q ru %q", r.id, en, ru)
		}
		if verbs(en) != verbs(ru) {
			t.Errorf("caption %q: verbs differ between en %q and ru %q", r.id, en, ru)
		}
		if !sameInBothLanguages[r.id] && (!hasCyrillic(ru) || en == ru) {
			t.Errorf("caption %q is not translated: %q", r.id, ru)
		}
	}
}

// TestEveryCaptionIsUsed fails for a table row no code asks for and for a
// caption name the table does not define.
func TestEveryCaptionIsUsed(t *testing.T) {
	fset := token.NewFileSet()
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	defined := map[string]bool{}
	used := map[string]bool{}
	for _, e := range files {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.ValueSpec:
				if name == "captions.go" {
					if id, ok := n.Type.(*ast.Ident); ok && id.Name == "captionID" {
						for _, nm := range n.Names {
							defined[nm.Name] = true
						}
					}
				}
			case *ast.Ident:
				if name != "captions.go" || n.Obj == nil {
					used[n.Name] = true
				}
			case *ast.CallExpr:
				if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "row" && name == "captions.go" && len(n.Args) > 0 {
					if a, ok := n.Args[0].(*ast.Ident); ok {
						used["row:"+a.Name] = true
					}
				}
			}
			return true
		})
	}
	if len(defined) != len(captionRows) {
		t.Errorf("%d caption names, %d rows", len(defined), len(captionRows))
	}
	for id := range defined {
		if !used[id] {
			t.Errorf("caption %s is in the table and nothing asks for it", id)
		}
		if !used["row:"+id] {
			t.Errorf("caption %s has no row", id)
		}
	}
}

// texts lists what the window draws as words: every text, button, toggle and
// row of the layout.
func windowTexts(a *app) []string {
	var out []string
	for _, it := range a.layout() {
		out = append(out, it.text)
	}
	return out
}

func rigWithLanguage(t *testing.T, language string) *rig {
	r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n")
	r.fakes.validRoots[r.root] = game.InstallInfo{Language: language}
	return r
}

// windowCaptions are the captions the layout always shows, with the control
// each belongs to, in the order they are checked.
var windowCaptions = []captionID{
	cHeadBase, cHeadMods, cHeadModSettings,
	cAddPath, cAdd, cBrowse, cRemove, cFolder, cRescan, cTickHint, cSelectMod,
	cPlay, cCheck, cSaveSettings, cMapEditor, cSettings,
}

// settingsCaptions are the captions the settings view shows.
var settingsCaptions = []captionID{
	cHeadParams, cHeadAdvanced, cHeadStarter, cSound, cVolume, cMovies,
	cMissionMovies, cMarkers, cSavesFolder, cMissionNumber, cMapPicker, cChicken, cSkill,
	cExtraArgs, cCloseAfterPlay, cGameProgram, cLoadUnmarked,
	cPlay, cCheck, cSaveSettings, cMapEditor, cClose,
}

func TestWindowShowsEveryCaptionInBothLanguages(t *testing.T) {
	for _, tc := range []struct{ language, lang string }{{"english", "en"}, {"russian", "ru"}} {
		a := rigWithLanguage(t, tc.language).app()
		if a.lang() != tc.lang {
			t.Fatalf("%s base: language %q", tc.language, a.lang())
		}
		for _, view := range []struct {
			v   viewKind
			ids []captionID
		}{{viewMain, windowCaptions}, {viewSettings, settingsCaptions}} {
			a.setView(view.v)
			checkWindowCaptions(t, a, tc.lang, view.ids)
		}
	}
}

func checkWindowCaptions(t *testing.T, a *app, lang string, ids []captionID) {
	t.Helper()
	shown := windowTexts(a)
	for _, id := range ids {
		want := caption(lang, id)
		found := false
		for _, line := range shown {
			if line == want {
				found = true
			}
		}
		if !found {
			t.Errorf("%s window lacks %q (%s)", lang, want, id)
		}
	}
	if lang == "ru" {
		for _, line := range shown {
			for _, id := range ids {
				if en := caption("en", id); line == en && en != caption("ru", id) {
					t.Errorf("ru window shows the English caption %q", line)
				}
			}
		}
	}
}

func TestStatusAndErrorLinesFollowTheBaseLanguage(t *testing.T) {
	r := rigWithLanguage(t, "russian")
	a := r.app()
	a.s.Volume = "loud"
	if _, err := a.command(); err == nil || a.errText(err) != caption("ru", cBadVolume, "loud") {
		t.Fatalf("command error %v", err)
	}
	if err := (&captionError{id: cBadVolume, args: []any{"loud"}}).Error(); err != caption("en", cBadVolume, "loud") {
		t.Fatalf("plain error text %q", err)
	}
	a.play()
	if want := caption("ru", cCannotPlay, caption("ru", cBadVolume, "loud")); a.status != want {
		t.Fatalf("status %q, want %q", a.status, want)
	}
	a.s.Volume = ""
	a.addText = ""
	a.addBase()
	if a.status != caption("ru", cTypeFolder) {
		t.Fatalf("status %q", a.status)
	}
	e := rigWithLanguage(t, "english").app()
	e.s.Volume = "loud"
	e.play()
	if want := "cannot play: volume \"loud\" is not a number from 0 to 100"; e.status != want {
		t.Fatalf("english status %q, want %q", e.status, want)
	}
}

func TestFirstProblemIsWordedInTheBaseLanguage(t *testing.T) {
	r := newRig(t, "[starter]\nlast-base = en\nnot a pair\n[bases]\nen = {root}\n")
	r.fakes.validRoots[r.root] = game.InstallInfo{Language: "russian"}
	a := r.app()
	if !a.statusBad || !strings.HasPrefix(a.status, "установки ") {
		t.Fatalf("status %q", a.status)
	}
}

func TestRussianCaptionsFitTheirControls(t *testing.T) {
	a := rigWithLanguage(t, "russian").app()
	for _, v := range []viewKind{viewMain, viewSettings} {
		a.setView(v)
		for _, it := range a.layout() {
			switch it.kind {
			case iButton:
				if w := textWidth(it.text); w > it.rect.Dx()-8 {
					t.Errorf("button %q is %d px wide in %d", it.text, w, it.rect.Dx()-8)
				}
			case iText:
				if it.rect.Dx() == 300 && textWidth(it.text) > 300 {
					t.Errorf("label %q is %d px wide in 300", it.text, textWidth(it.text))
				}
			}
		}
	}
}

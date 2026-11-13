package ui

import (
	"errors"
	"fmt"
	"image"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/pathdialog"
	"againrom/pkg/words"
)

// modOpenEditor is W open on a workspace host whose path window answers
// path, or answers err; the requests it was asked land in requests.
func modOpenEditor(t *testing.T, path string, err error) (*MapEditor, *modesCalls, chan pathdialog.Request) {
	t.Helper()
	e, calls := newModesEditor(t, "workspace")
	requests := make(chan pathdialog.Request, 1)
	bindEditorDialog(e, func(r pathdialog.Request) (pathdialog.Result, error) {
		requests <- r
		return pathdialog.Result{Path: path}, err
	}, t.TempDir())
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	return e, calls, requests
}

// W's Open mod button, by key and by click, asks the host's path window for a
// mod.toml or a packed .res and sends the chosen path down the folder field's
// route; with unsaved changes that route asks the shared question first.
func TestModWindowOpenButtonSendsTheChosenModDownTheFolderRoute(t *testing.T) {
	chosen := filepath.Join(t.TempDir(), "other-mod", "mod.toml")
	for _, door := range []string{"key", "click", "unsaved"} {
		e, calls, requests := modOpenEditor(t, chosen, nil)
		if door == "unsaved" {
			e.doc.Edits.(*modesWorkspace).leave = &WorkspaceLeave{Folder: chosen}
		}
		if door == "click" {
			s, _, _ := e.modSpec()
			modesClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Buttons[modButtonIndex("open")]))
		} else if !e.FocusModField("open") || e.Key("enter") != nil {
			t.Fatal("W has no Open mod button")
		}
		finishEditorDialog(t, e)
		r := <-requests
		if r.Kind != pathdialog.OpenFile || r.Title != e.plainWord("editor.mod.openmod") || len(r.Filters) != 1 || !slices.Equal(r.Filters[0].Extensions, []string{"toml", "res"}) {
			t.Fatalf("%s: request %+v", door, r)
		}
		want := fmt.Sprintf("Workspace package folder %q", chosen)
		if door == "unsaved" {
			if !asked(e) || !e.prompt.then.workspace || slices.Contains(calls.lines, want) {
				t.Fatalf("unsaved changes did not reach the shared question: %v", calls.lines)
			}
			continue
		}
		if !slices.Contains(calls.lines, want) || !e.ModWindowOpen() {
			t.Fatalf("%s: the chosen mod did not reach the folder field: %v", door, calls.lines)
		}
	}
}

// A click on W's folder field, Enter in it as shown and Ctrl+O each ask the
// path window for a mod.toml or a packed .res, starting in the editor's
// folder, and the chosen mod opens down the folder route. A typed folder and
// Enter still opens the typed path.
func TestModWindowFolderFieldAndCtrlOAskThePathWindow(t *testing.T) {
	chosen := filepath.Join(t.TempDir(), "other-mod", "mod.toml")
	for _, door := range []string{"click", "click-focused", "enter", "ctrl+o", "typed"} {
		e, calls, requests := modOpenEditor(t, chosen, nil)
		folder := slices.IndexFunc(e.obj.fields, func(f InspectionField) bool { return f.Key == "folder" })
		focus := "folder"
		if door == "click" {
			focus = "title"
		}
		if folder < 0 || !e.FocusModField(focus) {
			t.Fatal("W has no folder field")
		}
		switch door {
		case "click", "click-focused":
			r, ok := e.FindControl(ControlQuery{Role: ControlField, N: folder})
			if !ok {
				t.Fatal("the folder field is not shown")
			}
			modesClick(e, center(r))
		case "enter":
			_ = e.Key("enter")
		case "ctrl+o":
			updateEditorKeys(t, e, ebiten.KeyControl, ebiten.KeyO)
		case "typed":
			_ = e.Key("clear")
			e.TextInput("typed-mod")
			_ = e.Key("enter")
		}
		finishEditorDialog(t, e)
		if door == "typed" {
			if len(requests) != 0 || !slices.Contains(calls.lines, `Workspace package folder "typed-mod"`) {
				t.Fatalf("typed: the typed folder did not open: %d requests, %v", len(requests), calls.lines)
			}
			continue
		}
		if len(requests) != 1 {
			t.Fatalf("%s: %d path window requests", door, len(requests))
		}
		r := <-requests
		if r.Kind != pathdialog.OpenFile || r.StartDir != e.pathDialog.baseDir || len(r.Filters) != 1 || !slices.Equal(r.Filters[0].Extensions, []string{"toml", "res"}) {
			t.Fatalf("%s: request %+v", door, r)
		}
		if want := fmt.Sprintf("Workspace package folder %q", chosen); !slices.Contains(calls.lines, want) || !e.ModWindowOpen() {
			t.Fatalf("%s: the chosen mod did not open: %v", door, calls.lines)
		}
	}
	e, _, requests := modOpenEditor(t, "", errors.New("closed"))
	_ = e.Key("open")
	finishEditorDialog(t, e)
	if r := <-requests; !slices.Equal(r.Filters[0].Extensions, []string{"alm", "almx"}) {
		t.Fatalf("File > Open with W open asked %+v, not for a map", r)
	}
}

// Without a path window Open mod focuses the folder field and says the system
// has none; a window that fails says why. Both say to type the mod's path.
func TestModWindowOpenWithoutAPathWindowAsksForTheTypedPath(t *testing.T) {
	failed := errors.New("test dialog failure")
	for _, tc := range []struct {
		name string
		err  error
	}{{"none", nil}, {"unavailable", pathdialog.ErrUnavailable}, {"failed", failed}} {
		var e *MapEditor
		if tc.name != "none" {
			e, _, _ = modOpenEditor(t, "", tc.err)
		} else {
			e, _ = newModesEditor(t, "workspace")
			_ = e.Key("package")
		}
		if !e.FocusModField("open") || e.Key("enter") != nil {
			t.Fatal("W has no Open mod button")
		}
		finishEditorDialog(t, e)
		if e.obj.focus >= len(e.obj.fields) || e.obj.fields[e.obj.focus].Key != "folder" {
			t.Fatalf("%s: focus %d is not the folder field", tc.name, e.obj.focus)
		}
		want := e.plainWord("editor.mod.nodialog")
		if tc.err == failed {
			want = fmt.Sprintf(e.plainWord("editor.mod.typepath"), failed)
		}
		if e.modStatus != want {
			t.Fatalf("%s: W says %q, not %q", tc.name, e.modStatus, want)
		}
	}
}

// modesWordsEditor is newModesEditor in lang with the ten-pixel word font,
// encoding text as the installed fonts do.
func modesWordsEditor(t *testing.T, host, lang string, size image.Point) *MapEditor {
	t.Helper()
	e, _ := newModesEditor(t, host)
	e.font = wordsFont(lang)
	e.SetWords(words.For(lang))
	e.SetHostTextEncoder(func(s string) string {
		encoded, _ := textinput.EncodeText(s, e.font.Selector, textinput.Substitute)
		return encoded
	})
	e.Layout(size.X, size.Y)
	return e
}

// buttonProblems are the buttons of popup s whose caption runs past them, that
// stand outside the popup or that overlap another.
func buttonProblems(s popupSpec, l popupLayout, body interface{ Advance(string) int }) []string {
	rects, captions := slices.Clone(l.Buttons), slices.Clone(s.Buttons)
	if s.SwitchButton != "" && !l.SwitchButton.Empty() {
		rects, captions = append(rects, l.SwitchButton), append(captions, s.SwitchButton)
	}
	var problems []string
	for i, r := range rects[:min(len(rects), len(captions))] {
		if body.Advance(captions[i])+2 > r.Dx() {
			problems = append(problems, "caption "+captions[i]+" runs past its button")
		}
		if !r.In(l.Rect) {
			problems = append(problems, "button "+captions[i]+" stands outside the popup")
		}
		for j := range i {
			if rects[j].Overlaps(r) {
				problems = append(problems, "buttons "+captions[j]+" and "+captions[i]+" overlap")
			}
		}
	}
	return problems
}

// Every popup the editor builds, at the end of every modes path, in EN and
// RU, holds each button's caption inside its button, and no two buttons
// overlap; W also at 640x480.
func TestEditorPopupButtonsHoldTheirCaptionsInBothLanguages(t *testing.T) {
	seen := 0
	for _, lang := range []string{"en", "ru"} {
		for _, p := range modesPaths {
			host, sizes := "full", []image.Point{{1024, 768}}
			if strings.HasPrefix(p.name, "mod-") {
				host, sizes = "workspace", append(sizes, image.Pt(640, 480))
			}
			for _, size := range sizes {
				e := modesWordsEditor(t, host, lang, size)
				for _, step := range p.steps {
					modesAct(e, step)
				}
				for _, id := range []editorPopupID{popList, popObject, popPrompt, popTree, popProps, popToolbar} {
					s, ok := e.movableSpec(id)
					if !ok || dialogGivenForm(s) {
						continue
					}
					seen++
					for _, problem := range buttonProblems(s, layoutPopup(s, e.font, e.bodyFont()), e.bodyFont()) {
						t.Errorf("%s %s %v %q: %s", lang, p.name, size, s.Title, problem)
					}
				}
			}
		}
	}
	if seen < 2*len(modesPaths) {
		t.Fatalf("only %d popups seen", seen)
	}
}

// modWordFields are W's fields as the game hosts them, labelled with the word
// book's words in lang: the mission's own in the first column, the manifest's
// and the party's in the second.
func modWordFields(lang string) []InspectionField {
	b := words.For(lang)
	var fields []InspectionField
	for _, key := range []string{"folder", "id", "title", "source", "output"} {
		fields = append(fields, InspectionField{Key: key, Label: b.Text("editor.mod." + key), Text: true, TextValue: "value"})
	}
	for _, key := range []string{"version", "author", "description", "applies-to", "requires", "conflicts", "load-after", "kind", "number", "after", "guest", "chapter", "gold", "landing", "weapon", "shield", "armour"} {
		fields = append(fields, InspectionField{Key: key, Label: b.Text("editor.mod." + key), Text: true, TextValue: "value", Column: 1})
	}
	return append(fields, InspectionField{Key: "add", Label: b.Text("editor.mod.add"), Action: true})
}

// Every W field label is drawn whole: the label column follows the longest
// label, and a label longer than half the column takes a line of its own.
func TestModWindowLabelsAreWholeInBothLanguages(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, size := range []image.Point{{1024, 768}, {640, 480}} {
			e := modesWordsEditor(t, "workspace", lang, size)
			fields := modWordFields(lang)
			e.doc.Edits.(*modesWorkspace).fields = fields
			if err := e.Key("package"); err != nil {
				t.Fatal(err)
			}
			whole := map[string]bool{}
			for top := 0; ; top++ {
				e.objectTop = top
				s, _, _ := e.modSpec()
				l := layoutPopup(s, e.font, e.bodyFont())
				for _, rr := range l.Rows {
					row := s.Rows[rr.Index]
					if row.Label != "" && row.Kind != popupHeading && editorClip(e.bodyFont(), row.Label, rr.Label.Dx()-2) == row.Label {
						whole[row.Label] = true
					}
				}
				if e.objectTop != top || top >= popupRowCount(s) {
					break
				}
			}
			for _, f := range fields {
				if label := e.editorHostText(f.Label); !f.Action && !whole[label] {
					t.Errorf("%s %v: label %q is not drawn whole", lang, size, f.Label)
				}
			}
		}
	}
}

// W moves by its title bar, keeps its place while open and stays inside the
// window past an edge; a left border drag keeps its right edge.
func TestModWindowMovesByItsTitleAndStaysInside(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e := modesWordsEditor(t, "workspace", lang, image.Pt(1024, 768))
		if err := e.Key("package"); err != nil {
			t.Fatal(err)
		}
		d := image.Pt(200, 150)
		before, after := dragTitle(t, e, d)
		if after.Min != before.Min.Add(d) {
			t.Fatalf("%s: title %v moved to %v, not by %v", lang, before, after, d)
		}
		for _, key := range []string{"tab", "tab", "backtab"} {
			_ = e.Key(key)
		}
		e.Layout(1024, 768)
		if kept, _ := e.FindControl(ControlQuery{Role: ControlTitle}); kept.Min != after.Min {
			t.Fatalf("%s: W left %v for %v while open", lang, after, kept)
		}
		_, far := dragTitle(t, e, image.Pt(5000, 5000))
		bounds := e.popupBounds()
		if far.Min.X != bounds.Max.X-popupKeepWide || far.Min.Y >= bounds.Max.Y || far.Max.Y > bounds.Max.Y {
			t.Fatalf("%s: a drag past the corner left the title at %v in %v", lang, far, bounds)
		}
		_, _ = dragTitle(t, e, image.Pt(-600, -600))
		was, now := edgeDrag(t, e, leftEdge, image.Pt(-40, 0))
		if now.Max.X != was.Max.X || now.Dx() != was.Dx()+40 {
			t.Fatalf("%s: a left border drag sized %v to %v", lang, was, now)
		}
	}
}

// Every word W reads differs between EN and RU: W's own words carry no proper
// name or file name, which are the data's.
func TestModWindowWordsAreRussianInRussian(t *testing.T) {
	ids, err := words.IDs(words.Engine())
	if err != nil {
		t.Fatal(err)
	}
	en, ru := words.For("english"), words.For("russian")
	n := 0
	for _, id := range ids {
		if !strings.HasPrefix(id, "editor.mod.") && id != "editor.button.cancel" {
			continue
		}
		n++
		if en.Text(id) == ru.Text(id) {
			t.Errorf("%s reads %q in both languages", id, en.Text(id))
		}
	}
	if n < len(modButtons) {
		t.Fatalf("only %d W words", n)
	}
}

// Text typed into a W field and not entered is applied by Open mod, so the
// route asks about unsaved changes before another mod opens; Cancel leaves
// the mod open.
func TestModWindowOpenAppliesATypedFieldAndAsksAboutUnsavedChanges(t *testing.T) {
	chosen := filepath.Join(t.TempDir(), "other-mod", "mod.toml")
	e, calls, _ := modOpenEditor(t, chosen, nil)
	e.doc.Edits.(*modesWorkspace).dirtyOnEdit = true
	if !e.FocusModField("title") {
		t.Fatal("W has no name field")
	}
	e.TextInput("Typed")
	s, _, _ := e.modSpec()
	modesClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Buttons[modButtonIndex("open")]))
	finishEditorDialog(t, e)
	if !asked(e) || !e.prompt.then.workspace {
		t.Fatalf("no unsaved-changes question: %v", calls.lines)
	}
	if !slices.Contains(calls.lines, `Workspace package title "RiverTyped"`) {
		t.Fatalf("the typed text was not applied: %v", calls.lines)
	}
	opened := fmt.Sprintf("Workspace package folder %q", chosen)
	_ = e.Key("escape") // the question's Cancel
	if asked(e) || !e.ModWindowOpen() || slices.Contains(calls.lines, opened) || slices.Contains(calls.lines, `Workspace package folder-open "other-mod"`) {
		t.Fatalf("Cancel did not leave the mod as it was: asked %v window %v %v", asked(e), e.ModWindowOpen(), calls.lines)
	}
}

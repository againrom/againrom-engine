package ui

import (
	"errors"
	"fmt"
	"image"
	"strings"
	"testing"

	"againrom/pkg/pathdialog"
)

func modMapActionsEditor(t *testing.T, lang string, size image.Point) (*MapEditor, *modesWorkspace) {
	t.Helper()
	e := modesWordsEditor(t, "workspace", lang, size)
	p := e.doc.Edits.(*modesWorkspace)
	p.actionsOpen = true
	p.actions = []ModMapAction{
		{Label: "spare.alm: " + e.plainWord("editor.mod.map.unused"), Buttons: []ModMapButton{
			{Caption: e.plainWord("editor.mod.makemission"), Key: "makemission", Value: "maps/spare.alm"},
			{Caption: fmt.Sprintf(e.plainWord("editor.mod.makereplace"), 30), Key: "makemission", Value: "30|maps/spare.alm"}}},
		{Label: "7.alm: " + fmt.Sprintf(e.plainWord("editor.mod.map.byname"), 7), Buttons: []ModMapButton{
			{Caption: e.plainWord("editor.mod.makerow"), Key: "makerow", Value: "7.alm"}}},
	}
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	return e, p
}

func pageButton(t *testing.T, e *MapEditor, caption string) image.Point {
	t.Helper()
	for top := 0; top < 40; top++ {
		e.objectTop = top
		s, _, _ := e.modSpec()
		l := layoutPopup(s, e.font, e.bodyFont())
		for _, rr := range l.Rows {
			for k, b := range rr.Buttons {
				if k < len(s.Rows[rr.Index].Buttons) && s.Rows[rr.Index].Buttons[k] == caption && len(e.obj.modKeys[rr.Index]) > 0 {
					if body := e.bodyFont(); body.Advance(caption)+2 > b.Dx() || !b.In(l.Rect) {
						t.Errorf("button %q is cut: %v of %v", caption, b, l.Rect)
					}
					return center(b)
				}
			}
		}
	}
	t.Fatalf("the Missions page has no button %q", caption)
	return image.Point{}
}

func TestModWindowMissionsPageOffersTheMapActions(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, size := range []image.Point{{640, 480}, {1280, 800}} {
			t.Run(fmt.Sprintf("%s/%dx%d", lang, size.X, size.Y), func(t *testing.T) {
				e, p := modMapActionsEditor(t, lang, size)
				for _, c := range []struct{ caption, line string }{
					{e.word("editor.mod.makemission"), `Workspace package makemission "maps/spare.alm"`},
					{fmt.Sprintf(e.word("editor.mod.makereplace"), 30), `Workspace package makemission "30|maps/spare.alm"`},
					{e.word("editor.mod.makerow"), `Workspace package makerow "7.alm"`},
					{e.word("editor.mod.addmap"), `Workspace package addmap ""`},
				} {
					modesClick(e, pageButton(t, e, c.caption))
					if got := p.calls.lines[len(p.calls.lines)-1]; got != c.line {
						t.Fatalf("%q reached the host as %q, want %q", c.caption, got, c.line)
					}
				}
			})
		}
	}
}

func TestModWindowMissionsPageHasNoMapRowsWithoutTheHost(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	_ = e.Key("package")
	s, _, _ := e.modSpec()
	for _, r := range s.Rows {
		if r.Label == e.word("editor.mod.addmap.label") || r.Label == e.word("editor.mod.mapactions") {
			t.Fatalf("the page holds a map row %q without a host", r.Label)
		}
	}
}

func TestAddMapsFromFolderAsksTheSystemFolderWindow(t *testing.T) {
	e, p := modMapActionsEditor(t, "en", image.Pt(1280, 800))
	requests := make(chan pathdialog.Request, 1)
	bindEditorDialog(e, func(r pathdialog.Request) (pathdialog.Result, error) {
		requests <- r
		return pathdialog.Result{Path: "C:/maps"}, nil
	}, t.TempDir())
	modesClick(e, pageButton(t, e, e.word("editor.mod.addmaps")))
	finishEditorDialog(t, e)
	if r := <-requests; r.Kind != pathdialog.ChooseFolder {
		t.Fatalf("the window asked for kind %v", r.Kind)
	}
	if got := p.calls.lines[len(p.calls.lines)-1]; got != `Workspace package addmaps "C:/maps"` {
		t.Fatalf("the answer reached the host as %q", got)
	}
	if e.prompt.open() {
		t.Fatal("a typed field stands where the system window answered")
	}
}

func TestAddMapsFromFolderFallsBackToATypedFolder(t *testing.T) {
	for name, ask := range map[string]func(pathdialog.Request) (pathdialog.Result, error){
		"no window": nil,
		"unavailable": func(pathdialog.Request) (pathdialog.Result, error) {
			return pathdialog.Result{}, pathdialog.ErrUnavailable
		},
		"failed": func(pathdialog.Request) (pathdialog.Result, error) { return pathdialog.Result{}, errors.New("broken") },
	} {
		t.Run(name, func(t *testing.T) {
			e, p := modMapActionsEditor(t, "en", image.Pt(1280, 800))
			bindEditorDialog(e, ask, t.TempDir())
			modesClick(e, pageButton(t, e, e.word("editor.mod.addmaps")))
			finishEditorDialog(t, e)
			if e.prompt.kind != promptMapsFolder {
				t.Fatalf("no typed field asks: prompt %d", e.prompt.kind)
			}
			e.TextInput("D:/folder")
			if err := e.Key("enter"); err != nil {
				t.Fatal(err)
			}
			if got := p.calls.lines[len(p.calls.lines)-1]; got != `Workspace package addmaps "D:/folder"` {
				t.Fatalf("the typed folder reached the host as %q", got)
			}
			if strings.TrimSpace(e.promptTitle()) == "" {
				t.Fatal("the prompt has no title")
			}
		})
	}
}

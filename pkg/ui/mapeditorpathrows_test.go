package ui

import (
	"fmt"
	"image"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/pathdialog"
)

// pathRow is a row that asks the path window, with its request and its
// no-window fallback as they were on base.
type pathRow struct {
	name     string
	editor   func(t *testing.T) *MapEditor
	door     func(t *testing.T, e *MapEditor)
	want     func(e *MapEditor, base, maps string) pathdialog.Request
	fellBack func(e *MapEditor) bool
}

func modRowsEditor(t *testing.T) *MapEditor {
	t.Helper()
	e, _ := newModesEditor(t, "workspace")
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	return e
}

func modOpenRequest(e *MapEditor, start string) pathdialog.Request {
	return pathdialog.Request{Kind: pathdialog.OpenFile, Title: e.plainWord("editor.mod.openmod"), StartDir: start,
		Filters: []pathdialog.Filter{{Name: e.plainWord("editor.mod.openfilter"), Extensions: []string{"toml", "res"}}}}
}

func modOpenFellBack(e *MapEditor) bool {
	f, ok := e.modFocusField()
	// The instruction stands in the typed prompt, never in W's notes.
	return ok && f.Key == "folder" && e.prompt.kind == promptMapsFolder && e.prompt.scope == "folder" &&
		e.message == e.plainWord("editor.mod.nodialog") && e.modSaveState() == ""
}

func orMaps(base, maps string) string {
	if maps != "" {
		return maps
	}
	return base
}

var almFilters = []pathdialog.Filter{{Name: "ALM maps", Extensions: []string{"alm", "almx"}}, {Name: "All files", Extensions: []string{"*"}}}

func editorPathRows(saveDir string) []pathRow {
	return []pathRow{
		{
			name: "mod folder field, click", editor: modRowsEditor,
			door: func(t *testing.T, e *MapEditor) {
				folder := slices.IndexFunc(e.obj.fields, func(f InspectionField) bool { return f.Key == "folder" })
				if folder < 0 || !e.FocusModField("title") {
					t.Fatal("W has no folder field")
				}
				r, ok := e.FindControl(ControlQuery{Role: ControlField, N: folder})
				if !ok {
					t.Fatal("the folder field is not shown")
				}
				modesClick(e, center(r))
			},
			want:     func(e *MapEditor, base, maps string) pathdialog.Request { return modOpenRequest(e, orMaps(base, maps)) },
			fellBack: modOpenFellBack,
		},
		{
			name: "mod folder field, Enter", editor: modRowsEditor,
			door: func(t *testing.T, e *MapEditor) {
				if !e.FocusModField("folder") || e.Key("enter") != nil {
					t.Fatal("W has no folder field")
				}
			},
			want:     func(e *MapEditor, base, maps string) pathdialog.Request { return modOpenRequest(e, orMaps(base, maps)) },
			fellBack: modOpenFellBack,
		},
		{
			name: "Open mod button", editor: modRowsEditor,
			door: func(t *testing.T, e *MapEditor) {
				if !e.FocusModField("open") || e.Key("enter") != nil {
					t.Fatal("W has no Open mod button")
				}
			},
			want:     func(e *MapEditor, base, maps string) pathdialog.Request { return modOpenRequest(e, orMaps(base, maps)) },
			fellBack: modOpenFellBack,
		},
		{
			name: "Add folder of maps",
			editor: func(t *testing.T) *MapEditor {
				e, _ := modMapActionsEditor(t, "en", image.Pt(1280, 800))
				return e
			},
			door: func(t *testing.T, e *MapEditor) { modesClick(e, pageButton(t, e, e.word("editor.mod.addmaps"))) },
			want: func(e *MapEditor, base, maps string) pathdialog.Request {
				return pathdialog.Request{Kind: pathdialog.ChooseFolder, Title: e.plainWord("editor.mod.addmaps.title"), StartDir: orMaps(base, maps)}
			},
			fellBack: func(e *MapEditor) bool {
				return e.prompt.kind == promptMapsFolder && e.message == e.plainWord("editor.mod.addmaps.nodialog")
			},
		},
		{
			name: "speaker picture",
			editor: func(t *testing.T) *MapEditor {
				e, _, _ := speakerAddEditor(t, "en")
				return e
			},
			door: func(t *testing.T, e *MapEditor) {
				armSpeakerPrompt(t, e)
				e.TextInput("Lady")
				if err := e.Key("enter"); err != nil {
					t.Fatal(err)
				}
			},
			want: func(e *MapEditor, base, maps string) pathdialog.Request {
				return pathdialog.Request{Kind: pathdialog.OpenFile, Title: e.word("editor.speaker.add.picture"), StartDir: e.doc.Edits.(*speakerAddEdits).folder,
					Filters: []pathdialog.Filter{{Name: e.word("editor.speaker.add.pictures"), Extensions: []string{"png"}}}}
			},
			fellBack: func(e *MapEditor) bool {
				return e.message == fmt.Sprintf(e.word("editor.speaker.add.nodialog"), pathdialog.ErrUnavailable)
			},
		},
		{
			name:   "File > Open",
			editor: func(t *testing.T) *MapEditor { return testPlayEditor(t, &saveFolderEdits{folder: saveDir}) },
			door:   func(t *testing.T, e *MapEditor) { e.askPath(pathdialog.OpenFile) },
			want: func(e *MapEditor, base, maps string) pathdialog.Request {
				return pathdialog.Request{Kind: pathdialog.OpenFile, Title: "Open map", StartDir: orMaps(base, maps), Filters: almFilters}
			},
			fellBack: func(e *MapEditor) bool { return e.prompt.kind == promptOpen },
		},
		{
			name:   "File > Save As",
			editor: func(t *testing.T) *MapEditor { return testPlayEditor(t, &saveFolderEdits{folder: saveDir}) },
			door:   func(t *testing.T, e *MapEditor) { e.askPath(pathdialog.SaveFile) },
			want: func(e *MapEditor, base, maps string) pathdialog.Request {
				return pathdialog.Request{Kind: pathdialog.SaveFile, Title: "Save map as", StartDir: saveDir, Filters: almFilters, DefaultName: "20.alm"}
			},
			fellBack: func(e *MapEditor) bool { return e.prompt.kind == promptSave },
		},
	}
}

// Each row asks its base request with and without a maps folder and falls
// back as on base; Publish folder stays typed.
func TestEditorPathRowsAskAsOnBase(t *testing.T) {
	saveDir := t.TempDir()
	for _, row := range editorPathRows(saveDir) {
		for _, maps := range []string{"", filepath.Join(t.TempDir(), "maps")} {
			e := row.editor(t)
			base := t.TempDir()
			requests := make(chan pathdialog.Request, 4)
			bindEditorDialog(e, func(r pathdialog.Request) (pathdialog.Result, error) {
				requests <- r
				return pathdialog.Result{Cancelled: true}, nil
			}, base)
			e.SetMapsFolder(func() string { return maps })
			row.door(t, e)
			finishEditorDialog(t, e)
			if len(requests) != 1 {
				t.Fatalf("%s, maps %q: %d requests", row.name, maps, len(requests))
			}
			if got, want := <-requests, row.want(e, base, maps); !reflect.DeepEqual(got, want) {
				t.Errorf("%s, maps %q: request %+v, want %+v", row.name, maps, got, want)
			}
		}
		e := row.editor(t)
		bindEditorDialog(e, nil, t.TempDir())
		row.door(t, e)
		if !row.fellBack(e) {
			t.Errorf("%s: no window: prompt %v, message %q, status %q", row.name, e.prompt.kind, e.message, e.modStatus)
		}
	}
	if modPathRows["output"] != nil {
		t.Error("Publish folder asks a path window; it is typed")
	}
	e := modRowsEditor(t)
	folder := t.TempDir()
	for i := range e.obj.fields {
		if e.obj.fields[i].Key == "folder" {
			e.obj.fields[i].TextValue = filepath.Join(folder, "mod.toml")
		}
	}
	if got := e.questionRequest(modPathRows["open"](e)).StartDir; got != folder {
		t.Errorf("Open mod with an absolute folder typed starts in %q, want %q", got, folder)
	}
}

// The Publish folder row carries Choose: a press asks the system folder
// window from the folder the row holds and the field takes the folder
// chosen; a system without the window asks the typed prompt, its
// instruction inside the prompt and none in W's notes.
func TestModPublishFolderRowChoosesInTheFolderWindow(t *testing.T) {
	held, chosen := t.TempDir(), t.TempDir()
	choose := func(t *testing.T, ask pathdialog.Func) (*MapEditor, *modesCalls) {
		t.Helper()
		e, calls := newModesEditor(t, "workspace")
		host := e.doc.Edits.(*modesWorkspace)
		host.fields = append(host.fields, InspectionField{Key: "output", Label: "Publish folder", Text: true, TextValue: held},
			InspectionField{Key: "output.pick", Label: "Choose...", Action: true, Of: "output"})
		e.Layout(1920, 1080)
		bindEditorDialog(e, ask, t.TempDir())
		_ = e.Key("package")
		s, _, _ := e.modSpec()
		l := layoutPopup(s, e.font, e.bodyFont())
		for _, r := range l.Rows {
			if row := s.Rows[r.Index]; row.Label == "Publish folder" && len(r.Buttons) == 1 && slices.Equal(row.Buttons, []string{"Choose..."}) {
				modesClick(e, boxMidpoint(r.Buttons[0]))
				finishEditorDialog(t, e)
				return e, calls
			}
		}
		t.Fatalf("no Publish folder row with Choose: %+v", s.Rows)
		return nil, nil
	}
	requests := make(chan pathdialog.Request, 2)
	withWindow, calls := choose(t, func(r pathdialog.Request) (pathdialog.Result, error) {
		requests <- r
		return pathdialog.Result{Path: chosen}, nil
	})
	// With the system window the field is chosen, not typed.
	if !withWindow.FocusModField("output") {
		t.Fatal("no Publish folder field")
	}
	before := withWindow.obj.buf[withWindow.obj.focus]
	_ = withWindow.Key("clear")
	withWindow.TextInput("typed")
	if got := withWindow.obj.buf[withWindow.obj.focus]; got != before {
		t.Errorf("the Publish folder took typing with the system window: %q, was %q", got, before)
	}
	if r := <-requests; r.Kind != pathdialog.ChooseFolder || r.Title != "Publish folder" || r.StartDir != held {
		t.Errorf("the request %+v", r)
	}
	if got := fmt.Sprint(calls.lines); !strings.Contains(got, "package output") || !strings.Contains(got, filepath.Base(chosen)) {
		t.Errorf("the chosen folder did not reach the field: %s", got)
	}
	e, _ := choose(t, nil)
	if e.prompt.kind != promptMapsFolder || e.prompt.scope != "output" || e.message != e.plainWord("editor.mod.nodialog") || e.modSaveState() != "" {
		t.Errorf("no window: prompt %v %q, message %q, notes %q", e.prompt.kind, e.prompt.scope, e.message, e.modSaveState())
	}
	// Without the system window the field itself stays typable.
	e.dropPrompt()
	if !e.FocusModField("output") {
		t.Fatal("no Publish folder field")
	}
	_ = e.Key("clear")
	e.TextInput("typed")
	if got := e.obj.buf[e.obj.focus]; got != "typed" {
		t.Errorf("the Publish folder took no typing without the system window: %q", got)
	}
}

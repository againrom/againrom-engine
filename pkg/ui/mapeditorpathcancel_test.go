package ui

import (
	"bytes"
	"errors"
	"image"
	"testing"

	"againrom/pkg/pathdialog"
)

func TestEditorFailedPathDialogDropsCancelCheckpoint(t *testing.T) {
	for _, command := range []string{"open", "saveas"} {
		for _, dismiss := range []bool{false, true} {
			name := command + "/message-open"
			if dismiss {
				name = command + "/message-OK"
			}
			t.Run(name, func(t *testing.T) {
				e, h := popupEditor(t)
				if err := e.Open("popups"); err != nil {
					t.Fatal(err)
				}
				e.open = func(string) (*InspectionDocument, error) { return nil, errors.New("corrupt map") }
				bindEditorDialog(e, func(pathdialog.Request) (pathdialog.Result, error) {
					return pathdialog.Result{Path: "bad.alm"}, nil
				}, t.TempDir())
				_ = e.Key(command)
				finishEditorDialog(t, e)
				if e.message == "" {
					t.Fatal("missing path error")
				}
				if e.cancelRestore != nil {
					t.Error("failed path dialog retained Cancel checkpoint")
				}
				if dismiss {
					if r, ok := e.PromptButtonRect(0); ok {
						chromeClick(e, center(r))
					} else {
						t.Fatal("missing error OK")
					}
					if e.message != "" || e.cancelRestore != nil {
						t.Fatal("OK retained error or checkpoint")
					}
				} else {
					e.message = ""
				}
				e.ShowCell(image.Pt(45, 45))
				chromeClick(e, barPoint(t, e, "units"))
				chromeClick(e, treeMemberPoint(t, e, "units", 2))
				if e.selected != 2 {
					t.Fatalf("mouse did not select unit 2: %d", e.selected)
				}
				cam, selected := *e.doc.Viewer.cam, e.selected
				raw, mark := bytes.Clone(e.doc.SourceBytes), h.cursor
				objectLayout(t, e)
				e.obj.buf[0] = "999"
				chromeClick(e, objectButtonPoint(t, e, 1))
				if got := *e.doc.Viewer.cam; got.X != cam.X || got.Y != cam.Y || got.Zoom != cam.Zoom {
					t.Error("object Cancel moved camera to old dialog checkpoint")
				}
				if e.selected != selected {
					t.Errorf("object Cancel changed selection %d to %d", selected, e.selected)
				}
				if !bytes.Equal(raw, e.doc.SourceBytes) || h.cursor != mark || len(h.calls) != 0 {
					t.Fatal("object Cancel changed bytes or Undo")
				}
			})
		}
	}
}

func TestEditorMessageOKUsesCancelPath(t *testing.T) {
	e, _ := popupEditor(t)
	if err := e.Open("popups"); err != nil {
		t.Fatal(err)
	}
	e.rememberCancel()
	e.message = "failed operation"
	r, ok := e.PromptButtonRect(0)
	if !ok {
		t.Fatal("missing error OK")
	}
	chromeClick(e, center(r))
	if e.cancelRestore != nil || e.message != "" {
		t.Fatal("message OK retained Cancel checkpoint")
	}
}
